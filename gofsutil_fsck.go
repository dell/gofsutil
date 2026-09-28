package gofsutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	log "github.com/dell/csmlog"
)

const (
	StartedFSCheckEvent  = "Starting file system check"
	FoundNoErrorsEvent   = "Found no errors in file system"
	FoundErrorsEvent     = "Found errors in file system"
	FoundDirtyLogEvent   = "Found dirty log in file system"
	FSCheckTimedOutEvent = "File system check timed out"
	FSCheckFailedEvent   = "File system check failed"

	StartFSRepairEvent    = "Starting file system repair"
	FinishedFSRepairEvent = "File system errors fixed"
	FSRepairTimedOutEvent = "File system repair timed out"
	FSRepairFailedEvent   = "File system errors could not be fixed"
	StartLogReplayEvent   = "Starting file system log replay"
	LogReplayFailedEvent  = "File system log replay failed"
	LogReplayDoneEvent    = "File system log replay done"
)

var FSCheckLinesToLog = 20

// e2fsck exit codes: https://www.man7.org/linux/man-pages/man8/e2fsck.8.html
type extExitCode int

func (e extExitCode) isFoundErrors() bool {
	// 1 or 2 were never observed, but according to e2fsck documentation
	// they indicate presense of repairable errors.
	return e&1 != 0 || e&2 != 0 || e&4 != 0
}

func (e extExitCode) isCanceledByUser() bool {
	return e&32 != 0
}

// xfsrepair exit codes: https://www.man7.org/linux/man-pages/man8/xfs_repair.8.html
const (
	xfsCodeNoErrorsFound         = 0
	xfsCodeFoundRepairableErrors = 1
	xfsCodeFoundDirtyLog         = 2
	xfsCodeErrorsRepaired        = 0
)

// FSChecker can do file system error checks using "e2fsck -nf" for ext and "xfs_repair -n" for xfs.
// It can also fix repairable errors using "e2fsck -p" or "xfs_repair" respectively.
// FSChecker instance for a given file system is created using GetFSChecker.
type FSChecker interface {
	// Check checks the fs for errors. If repairable errors are found and doRepair is true,
	// Check tries to do a safe repair. If repairable errors are found and doRepair is false,
	// Check returns an error. If unrepairable errors are found, or repair fails, Check returns an error.
	// If the context passed to Check times out or is closed before fs check and repair completes,
	// the process in interrupted and Check returns an error.
	Check(ctx context.Context, doRepair bool) error
	// repair is called when needed by the Check function to fix repairable file system errors.
	repair(ctx context.Context) error
}

// Allows monitoring fs check and repair lifecycle
type FSCheckObserver interface {
	OnEvent(message string)
}

type fsCheckerCommon struct {
	devPath  string
	observer FSCheckObserver
}

type (
	extChecker fsCheckerCommon
	xfsChecker fsCheckerCommon
)

type NoopFSCheckObserver struct{}

func (n *NoopFSCheckObserver) OnEvent(_ string) {}

func GetFSChecker(devPath, fsType string, observer FSCheckObserver) (FSChecker, error) {
	if observer == nil {
		observer = &NoopFSCheckObserver{}
	}

	ch := &fsCheckerCommon{
		devPath:  devPath,
		observer: observer,
	}

	fsType = strings.ToLower(fsType)

	switch fsType {
	case "ext4", "ext3", "ext2", "ext":
		return (*extChecker)(ch), nil
	case "xfs":
		return (*xfsChecker)(ch), nil
	}

	return nil, fmt.Errorf("unsupported fs type: %s", fsType)
}

func (ch *extChecker) Check(ctx context.Context, doRepair bool) error {
	defer logDuration("ext check/repair", time.Now())

	ch.observer.OnEvent(StartedFSCheckEvent)

	// Do up to 2 iterations: first for check and optional repair, second for post-repair check
	for pass := 0; pass < 2; pass++ {
		// Check for file system errors
		rc, err := OSExecFn(ctx, "e2fsck", "-nf", ch.devPath)
		code := extExitCode(rc)

		if err == nil {
			// Implies exit code 0
			if pass == 0 {
				ch.observer.OnEvent(FoundNoErrorsEvent)
			} else {
				ch.observer.OnEvent(FinishedFSRepairEvent)
			}
			return nil

		} else if code.isCanceledByUser() || isProcKilled(err) {
			ch.observer.OnEvent(FSCheckTimedOutEvent)
			return fmt.Errorf("checking file system on %s interrupted due to context timeout (%d)", ch.devPath, rc)

		} else if code.isFoundErrors() {
			if pass == 0 {
				// First pass found fs errors
				ch.observer.OnEvent(FoundErrorsEvent)
				if doRepair {
					err := ch.repair(ctx)
					if err != nil {
						// Error out on timeout
						return err
					}
					// Do another pass for final check
					continue
				}
				return fmt.Errorf("file system on %s has errors (%d)", ch.devPath, rc)
			}

			// Final (post-repair) pass still found fs errors
			ch.observer.OnEvent(FSRepairFailedEvent)
			return fmt.Errorf("repairing file system on %s failed (%d): %w", ch.devPath, rc, err)
		}

		if pass == 0 {
			ch.observer.OnEvent(FSCheckFailedEvent)
			return fmt.Errorf("checking file system on %s failed (%d): %w", ch.devPath, rc, err)
		}
		ch.observer.OnEvent(FSRepairFailedEvent)
		return fmt.Errorf("repairing file system on %s failed (%d): %w", ch.devPath, rc, err)
	}

	// Should never get here, but compiler requires this.
	return fmt.Errorf("unexpected error")
}

func (ch *extChecker) repair(ctx context.Context) error {
	defer logDuration("ext repair", time.Now())

	ch.observer.OnEvent(StartFSRepairEvent)

	// Safely repair ext with the preen option. Not all errors can be fixed
	// in this mode, but this will be verified during the final full check.

	rc, err := OSExecFn(ctx, "e2fsck", "-p", ch.devPath)
	code := extExitCode(rc)

	log.Debugf("e2fsck -p %s: exit code %d", ch.devPath, rc)

	if code.isCanceledByUser() || isProcKilled(err) {
		ch.observer.OnEvent(FSRepairTimedOutEvent)
		return fmt.Errorf("repairing file system on %s interrupted due to context timeout (%d)", ch.devPath, rc)
	}
	// All other errors and exit codes in the preen mode are not indicative of success
	// or failure (see tests/fsckdata/README.md), so we'll rely on the final errors check.

	return nil
}

func (ch *xfsChecker) Check(ctx context.Context, doRepair bool) error {
	defer logDuration("xfs check/repair", time.Now())

	ch.observer.OnEvent(StartedFSCheckEvent)

	// Normally do one pass. Second pass is needed if dirty log is found.
	for pass := 0; pass < 2; pass++ {

		rc, err := OSExecFn(ctx, "xfs_repair", "-n", ch.devPath)

		if err == nil && rc == xfsCodeNoErrorsFound {
			ch.observer.OnEvent(FoundNoErrorsEvent)
			return nil

		} else if rc == xfsCodeFoundRepairableErrors {
			ch.observer.OnEvent(FoundErrorsEvent)
			if doRepair {
				return ch.repair(ctx)
			}
			return fmt.Errorf("file system on %s has errors that can be fixed (%d)", ch.devPath, rc)

		} else if rc == xfsCodeFoundDirtyLog {
			if pass == 0 {
				ch.observer.OnEvent(FoundDirtyLogEvent)
				err := ch.replayLog(ctx)
				if err != nil {
					return fmt.Errorf("failed to replay log of file system on %s: %w", ch.devPath, err)
				}
				// Re-run fs check again and possibly do repair
				continue
			}
			return fmt.Errorf("file system on %s still has dirty log after replay (%d)", ch.devPath, rc)

		} else if isProcKilled(err) {
			ch.observer.OnEvent(FSCheckTimedOutEvent)
			return fmt.Errorf("checking file system on %s interrupted due to context timeout: %w", ch.devPath, err)
		}

		ch.observer.OnEvent(FSCheckFailedEvent)
		return fmt.Errorf("checking file system on %s failed (%d): %w", ch.devPath, rc, err)
	}

	// Should never get here, but compiler requires this.
	return fmt.Errorf("unexpected error")
}

func (ch *xfsChecker) repair(ctx context.Context) error {
	defer logDuration("xfs repair", time.Now())

	ch.observer.OnEvent(StartFSRepairEvent)

	rc, err := OSExecFn(ctx, "xfs_repair", ch.devPath)

	if err == nil && rc == xfsCodeErrorsRepaired {
		ch.observer.OnEvent(FinishedFSRepairEvent)
		return nil
	} else if isProcKilled(err) {
		ch.observer.OnEvent(FSRepairTimedOutEvent)
		return fmt.Errorf("repairing file system on %s interrupted due to context timeout: %w", ch.devPath, err)
	}

	ch.observer.OnEvent(FSRepairFailedEvent)
	return fmt.Errorf("repairing file system on %s failed (%d): %w", ch.devPath, rc, err)
}

// try mount-unmounting to a temp dir to cause the log replay by kernel
func (ch *xfsChecker) replayLog(ctx context.Context) (retErr error) {
	ch.observer.OnEvent(StartLogReplayEvent)
	defer func() {
		if retErr != nil {
			ch.observer.OnEvent(LogReplayFailedEvent)
		} else {
			ch.observer.OnEvent(LogReplayDoneEvent)
		}
	}()

	fsMounted := false
	tmpMountPoint, err := os.MkdirTemp("", "replay")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		// SAFETY: Don't attempt to remove tmpMountPoint if the fs is still mounted to it.
		if !fsMounted {
			// SAFETY: Intentionally not using RemoveAll, since this directory is expected to be empty after unmount.
			err := os.Remove(tmpMountPoint)
			if err != nil {
				log.Errorf("Failed to remove %s: %v", tmpMountPoint, err)
			}
		}
	}()

	// SAFETY: Mounting fs read-only to prevent accidental data alteration.
	rc, err := OSExecFn(ctx, "mount", "-o", "ro", ch.devPath, tmpMountPoint)

	if err == nil && rc == 0 {
		fsMounted = true
		rc, err := OSExecFn(ctx, "umount", tmpMountPoint)

		if err == nil && rc == 0 {
			fsMounted = false
			return nil
		}
		return fmt.Errorf("failed to unmount file system on %s to %s (%d): %w", ch.devPath, tmpMountPoint, rc, err)
	}
	return fmt.Errorf("failed to mount file system %s to %s (%d): %w", ch.devPath, tmpMountPoint, rc, err)
}

// Mockable function for testing
var OSExecFn = execOSCommand

func execOSCommand(ctx context.Context, name string, args ...string) (rc int, err error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G702

	// Start the child process in a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Give the Cancel (SIGINT) some time to work before a hard kill
	cmd.WaitDelay = 2 * time.Second

	cmd.Cancel = func() error {
		// Send SIGINT to the child's process group to also signal its children.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}

	// Collect all output in one buffer
	errBuffer := bytes.Buffer{}
	cmd.Stdout = &errBuffer
	cmd.Stderr = &errBuffer

	err = cmd.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			rc = exitError.ExitCode()
		} else {
			rc = -1
		}
		out := truncOutput(errBuffer, name, args...)
		if out.Len() > 0 {
			log.Error(out.String())
		}
	}

	return rc, err
}

func isProcKilled(err error) bool {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				// The process terminated due to an unhandled signal.
				// Normally, ws.Signal() would be syscall.SIGINT or syscall.SIGKILL
				// that we sent upon the context expiration. But we will not check
				// for specific signal here, since the process can be killed externally,
				// and we still want to interpret this as interruption.
				return true
			}
		}
	}
	return false
}

func logDuration(stage string, startTime time.Time) {
	log.Infof("%s took %.3fs", stage, time.Since(startTime).Seconds())
}

// Collect up to FSCheckLinesToLog last lines in the output. Empty lines are ignored.
func truncOutput(errBuf bytes.Buffer, name string, args ...string) *bytes.Buffer {
	outBuf := bytes.Buffer{}

	if errBuf.Len() == 0 {
		return &outBuf
	}

	n := FSCheckLinesToLog

	// Split by newline and filter out empty lines
	allLines := strings.Split(errBuf.String(), "\n")

	lines := make([]string, 0, n)

	// Iterate all err lines in the reverse order and filter out empty lines
	for i := len(allLines) - 1; i >= 0; i-- {
		if len(lines) >= n {
			lines = append(lines, "...")
			break
		}
		if trimmed := strings.TrimRight(allLines[i], " \t"); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	if len(lines) == 0 {
		return &outBuf
	}

	cmdStr := name
	if len(args) > 0 {
		cmdStr += " " + strings.Join(args, " ")
	}

	outBuf.WriteString("stderr from command: " + cmdStr + "\n")

	// Write the collected lines to outBuf taking into account the reverse order
	for i := len(lines) - 1; i >= 0; i-- {
		outBuf.WriteString(lines[i])
		if i > 0 {
			outBuf.WriteByte('\n')
		}
	}

	return &outBuf
}
