package gofsutil

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ---- Test entrypoint - either runs tests (default) or acts as a helper process (if -helperProc is passed) ----

func TestMain(m *testing.M) {
	helperProc := flag.Bool("helperProc", false, "start a helper process instead of running tests")
	helperReadyDir := flag.String("helperReadyDir", "", "ready dir for helper process")
	helperNoTrap := flag.Bool("helperNoTrap", false, "helper process without SIGINT trap")

	flag.Parse()

	if *helperProc == true {
		// Act as the helper child process; do NOT run the test suite.
		if err := os.MkdirAll(*helperReadyDir, 0o755); err != nil {
			panic(err)
		}
		fmt.Println("== helper started")
		sigCh := make(chan os.Signal, 1)
		if *helperNoTrap == false {
			signal.Notify(sigCh, syscall.SIGINT)
		}
		select {
		case <-sigCh:
			fmt.Println("== helper got SIGINT")
			os.Exit(32)
		case <-time.After(time.Second * 30):
			fmt.Println("== helper finished")
			os.Exit(0)
		}
	}

	// Normal test execution
	os.Exit(m.Run())
}

// ---- Test utilities ---------------------------------------------------------

type capturingObserver struct {
	events []string
}

func (c *capturingObserver) OnEvent(msg string) {
	c.events = append(c.events, msg)
}

func setMockExec(f func(ctx context.Context, name string, args ...string) (int, error)) func() {
	orig := OSExecFn
	OSExecFn = f
	return func() { OSExecFn = orig }
}

func assertEvents(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected events.\n  got : %#v\n  want: %#v", got, want)
	}
}

// ---- GetFSChecker -----------------------------------------------------------

func TestGetFSChecker_Supported(t *testing.T) {
	obs := &capturingObserver{}

	for _, fsType := range []string{"ext4", "ext3", "ext2", "ext"} {
		ch, err := GetFSChecker("/dev/sda1", fsType, obs)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", fsType, err)
		}
		if ch == nil {
			t.Fatalf("expected non-nil checker for %s", fsType)
		}
		if _, ok := ch.(*extChecker); !ok {
			t.Fatalf("expected extChecker for %s, got %T", fsType, ch)
		}
	}

	ch2, err := GetFSChecker("/dev/sdb1", "xfs", obs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch2 == nil {
		t.Fatalf("expected non-nil checker for xfs")
	}
	if _, ok := ch2.(*xfsChecker); !ok {
		t.Fatalf("expected xfsChecker, got %T", ch2)
	}
}

func TestGetFSChecker_UnsupportedReturnsNil(t *testing.T) {
	obs := &capturingObserver{}
	ch, err := GetFSChecker("/dev/sdc1", "btrfs", obs)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if ch != nil {
		t.Fatalf("expected nil checker for unsupported fs")
	}
}

// ---- EXT: Check -------------------------------------------------------------

func TestEXTChecker_Check_NoErrors(t *testing.T) {
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) == 2 && args[0] == "-nf" {
			return 0, nil // no errors
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		StartedFSCheckEvent,
		FoundNoErrorsEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_Errors_NoRepair(t *testing.T) {
	// e2fsck -nf -> rc=1 with non-nil err (errors found), doRepair=false
	// err != nil required to bypass the err==nil->FoundNoErrors path
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			return 1, errors.New("exit status 1")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_Errors_DoRepair_Success(t *testing.T) {
	// Check first (rc=1, err non-nil), then repair (rc=0), then final check (rc=0)
	// e2fsck -nf -> rc=1, err non-nil (errors found)
	// e2fsck -p -> rc=0, err nil (repair succeeds)
	// e2fsck -nf -> rc=0, err nil -> FinishedFSRepairEvent
	callCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			callCount++
			if callCount == 1 {
				// First check finds errors
				return 1, errors.New("exit status 1")
			}
			// Second check after repair finds no errors
			return 0, nil
		}
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-p" {
			return 0, nil
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FinishedFSRepairEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_Errors_DoRepair_TimedOut(t *testing.T) {
	// Check first (rc=1, err non-nil), then repair times out (rc=32)
	// e2fsck -nf -> rc=1, err non-nil (errors found)
	// e2fsck -p -> rc=32 (canceled by user); check never runs
	callCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			callCount++
			if callCount == 1 {
				// First check finds errors
				return 1, errors.New("exit status 1")
			}
		}
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-p" {
			return 32, errors.New("signal: killed")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FSRepairTimedOutEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_Errors_DoRepair_Failed(t *testing.T) {
	// Check first (rc=1, err non-nil), then repair (rc=4, err nil), then final check still finds errors (rc=4)
	// e2fsck -nf -> rc=1, err non-nil (errors found)
	// e2fsck -p -> rc=4, err nil (repair fails silently)
	// e2fsck -nf -> rc=4, err non-nil (final check finds errors)
	callCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			callCount++
			if callCount == 1 {
				// First check finds errors
				return 1, errors.New("exit status 1")
			}
			// Second check after repair still finds errors
			return 4, errors.New("exit status 4")
		}
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-p" {
			return 4, nil // repair fails silently, no error
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FSRepairFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_DoRepair_Success_CheckStillFindsErrors(t *testing.T) {
	// Check first (rc=1, err non-nil), then repair succeeds (rc=0), but final check still finds errors (rc=1)
	// e2fsck -nf -> rc=1, err non-nil (errors found)
	// e2fsck -p -> rc=0, err nil (repair succeeds)
	// e2fsck -nf -> rc=1, err non-nil (final check still finds errors)
	callCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			callCount++
			if callCount == 1 {
				// First check finds errors
				return 1, errors.New("exit status 1")
			}
			// Second check after repair still finds errors
			return 1, errors.New("exit status 1")
		}
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-p" {
			return 0, nil // repair succeeds
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FSRepairFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_TimedOut(t *testing.T) {
	// e2fsck -nf -> rc=32, non-nil err (isCanceledByUser)
	// err != nil bypasses the err==nil path; isCanceledByUser() -> FSCheckTimedOutEvent
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			return 32, errors.New("signal: killed")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FSCheckTimedOutEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_Failed_Generic(t *testing.T) {
	// e2fsck -nf -> rc=8 and non-nil err -> generic failure path
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			return 8, errors.New("exit status 8")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FSCheckFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestEXTChecker_Check_FinalPass_GenericFailure(t *testing.T) {
	// Test the uncovered final error path: second pass with unexpected exit code
	// First pass: rc=1 (errors found) -> repair -> second pass: rc=8 (unexpected)
	callCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-nf" {
			callCount++
			if callCount == 1 {
				// First pass finds errors to trigger repair
				return 1, errors.New("exit status 1")
			}
			// Second pass returns unexpected exit code (not 1, 2, 4, or 32)
			return 8, errors.New("exit status 8")
		}
		if name == "e2fsck" && len(args) >= 1 && args[0] == "-p" {
			return 0, nil // repair succeeds
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sda1", "ext4", obs)
	err := ch.Check(context.Background(), true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FSRepairFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

// ---- XFS: Check + Replay + Repair ------------------------------------------

func TestXFSChecker_Check_NoErrors(t *testing.T) {
	// xfs_repair -n -> 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" && len(args) == 2 && args[0] == "-n" {
			return 0, nil
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		StartedFSCheckEvent,
		FoundNoErrorsEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_Repairable_NoRepair(t *testing.T) {
	// xfs_repair -n -> 1
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" && len(args) >= 1 && args[0] == "-n" {
			return 1, nil
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_Repairable_DoRepair_Success(t *testing.T) {
	// -n -> 1, repair -> rc=0
	call := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		switch name {
		case "xfs_repair":
			if len(args) >= 1 && args[0] == "-n" {
				call++
				return 1, nil
			}
			// repair without -n
			if len(args) == 1 {
				call++
				return 0, nil
			}
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FinishedFSRepairEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_Repairable_DoRepair_Failed(t *testing.T) {
	// -n -> 1, repair -> rc!=0 or err!=nil
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" && len(args) >= 1 && args[0] == "-n" {
			return 1, nil
		}
		if name == "xfs_repair" && len(args) == 1 {
			return 1, errors.New("repair failed")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FSRepairFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_DirtyLog_Replay_Success_Then_NoErrors(t *testing.T) {
	// first -n -> 2 (dirty log)
	// mount -> 0, umount -> 0
	// second -n -> 0
	checkCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		switch name {
		case "xfs_repair":
			if len(args) >= 1 && args[0] == "-n" {
				checkCount++
				if checkCount == 1 {
					return 2, nil
				}
				return 0, nil
			}
		case "mount":
			return 0, nil
		case "umount":
			return 0, nil
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		StartedFSCheckEvent,
		FoundDirtyLogEvent,
		StartLogReplayEvent,
		LogReplayDoneEvent,
		FoundNoErrorsEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_DirtyLog_Replay_Success_Then_Repairable_DoRepair_Success(t *testing.T) {
	// first -n -> 2
	// mount/umount -> 0
	// second -n -> 1 (repairable)
	// repair -> 0
	checkCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		switch name {
		case "xfs_repair":
			if len(args) >= 1 && args[0] == "-n" {
				checkCount++
				if checkCount == 1 {
					return 2, nil
				}
				return 1, nil
			}
			// actual repair
			if len(args) == 1 {
				return 0, nil
			}
		case "mount":
			return 0, nil
		case "umount":
			return 0, nil
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		StartedFSCheckEvent,
		FoundDirtyLogEvent,
		StartLogReplayEvent,
		LogReplayDoneEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FinishedFSRepairEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_DirtyLog_Replay_Success_Then_StillDirty(t *testing.T) {
	// first -n -> 2
	// mount/umount -> 0
	// second -n -> 2 again (still dirty)
	checkCount := 0
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		switch name {
		case "xfs_repair":
			if len(args) >= 1 && args[0] == "-n" {
				checkCount++
				return 2, nil
			}
		case "mount":
			return 0, nil
		case "umount":
			return 0, nil
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// Note: On second dirty log, the implementation returns an error *without*
	// emitting FSCheckFailedEvent. We assert exactly the emitted events.
	want := []string{
		StartedFSCheckEvent,
		FoundDirtyLogEvent,
		StartLogReplayEvent,
		LogReplayDoneEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_DirtyLog_Replay_MountFail(t *testing.T) {
	// -n -> 2; mount fails
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		switch name {
		case "xfs_repair":
			if len(args) >= 1 && args[0] == "-n" {
				return 2, nil
			}
		case "mount":
			return 1, errors.New("mount failed")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundDirtyLogEvent,
		StartLogReplayEvent,
		LogReplayFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_DirtyLog_Replay_UmountFail(t *testing.T) {
	// -n -> 2; mount ok; umount fails
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		switch name {
		case "xfs_repair":
			if len(args) >= 1 && args[0] == "-n" {
				return 2, nil
			}
		case "mount":
			return 0, nil
		case "umount":
			return 1, errors.New("umount failed")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundDirtyLogEvent,
		StartLogReplayEvent,
		LogReplayFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Check_Failed_Generic(t *testing.T) {
	// -n -> rc=7, err non-nil => generic failed branch
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" && len(args) >= 1 && args[0] == "-n" {
			return 7, errors.New("xfs_repair failed")
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)
	err := ch.Check(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	want := []string{
		StartedFSCheckEvent,
		FSCheckFailedEvent,
	}
	assertEvents(t, obs.events, want)
}

// nil observer -> NoopFSCheckObserver is used; xfs_repair -n returns 0
func TestXFSChecker_WithNilObserver(t *testing.T) {
	checkCalled := false
	restore := setMockExec(func(_ context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" && len(args) == 2 && args[0] == "-n" {
			checkCalled = true
			return 0, nil // happy path: no errors
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	// Pass observer=nil to force NoopFSCheckObserver
	checker, err := GetFSChecker("/dev/sdb1", "xfs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if checker == nil {
		t.Fatalf("expected non-nil checker")
	}

	// Run check - this should internally emit events to NoopFSCheckObserver.OnEvent()
	if err := checker.Check(context.Background(), false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checkCalled {
		t.Fatalf("expected check to be called")
	}
}

// ---- Test ExecOSCommand ---------------------------------------------------------

// 1) Success path: command exits with 0
func TestExecFn_Success(t *testing.T) {
	// Use /bin/sh -c "true" to guarantee rc=0
	rc, err := execOSCommand(context.Background(), "/bin/sh", "-c", "true")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d", rc)
	}
}

// 2) Non-zero exit code: verify we get rc from ExitError and err != nil
func TestExecFn_NonZeroExit(t *testing.T) {
	// exit 7 ensures a specific rc is propagated
	rc, err := execOSCommand(context.Background(), "/bin/sh", "-c", "exit 7")
	if err == nil {
		t.Fatalf("expected non-nil error, got nil")
	}
	if rc != 7 {
		t.Fatalf("expected rc=7, got %d", rc)
	}
}

// 3) Command not found: rc should be -1 (since it's not an ExitError) and err != nil
func TestExecFn_CommandNotFound(t *testing.T) {
	rc, err := execOSCommand(context.Background(), "this-command-should-not-exist-xyz")
	if err == nil {
		t.Fatalf("expected error for non-existent command, got nil")
	}
	if rc != -1 {
		t.Fatalf("expected rc=-1 for non-ExitError cases, got %d", rc)
	}
}

// 4) Non-zero exit code with stderr output: verify we get stderr from the command
func TestExecFn_ExitWithStderr(t *testing.T) {
	// command writes to stderr and exits with code 7
	rc, err := execOSCommand(context.Background(), "/bin/sh",
		"-c", "echo command-error-message >&2; exit 7")
	//"-c", "for i in $(seq 15); do echo error-line-$i >&2; done; exit 7"))
	if err == nil {
		t.Fatalf("expected non-nil error, got nil")
	}
	if rc != 7 {
		t.Fatalf("expected rc=7, got %d", rc)
	}
}

// 5) Deterministic cancellation after confirming the command started.
// The command (helper process) installs a SIGINT trap that exits with rc=32,
// then creates a "ready" dir and sleeps. We wait for the ready dir,
// then cancel the context and expect rc=32 with non-nil error.
func TestExecFn_CancelAfterStarted_TrapSIGINT_RC32(t *testing.T) {
	// Prepare a temp path for readiness signal.
	tmpDir := t.TempDir()
	readyDir := filepath.Join(tmpDir, "ready")

	// The helper process:
	// - create the ready file to signal it's actually started
	// - monitors INT signal
	// - exits 0 upon completion or 32 upon SIGINT receival

	// Context we will cancel after we see the ready dir created.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		rc  int
		err error
	}
	done := make(chan result, 1)

	// Run execOSCommand in a goroutine so we can wait for readiness.
	go func() {
		rc, err := execOSCommand(ctx, os.Args[0], "-helperProc=true", "-helperReadyDir="+readyDir)
		done <- result{rc: rc, err: err}
	}()

	// Wait deterministically until the helper created the dir (or time out the test).
	waitCtx, stopWait := context.WithTimeout(context.Background(), 15*time.Second)
	defer stopWait()

	// Poll for the ready dir existence.
	for {
		select {
		case <-waitCtx.Done():
			t.Fatalf("timed out waiting for helper readiness dir %s", readyDir)
		default:
			if _, err := os.Stat(readyDir); err == nil {
				goto READY
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

READY:
	t.Logf("The process has started, interrupting it by cancelling ctx")
	cancel()

	// Wait for the command to exit and verify rc and error.
	select {
	case r := <-done:
		if r.err == nil {
			t.Fatalf("expected non-nil error after context cancel, got nil (rc=%d)", r.rc)
		}
		if r.rc != 32 {
			t.Fatalf("expected rc=32 from SIGINT trap, got %d (err=%v)", r.rc, r.err)
		}
		// Context should be canceled
		if ctx.Err() == nil {
			t.Fatalf("expected ctx.Err() to be non-nil (context canceled), got nil")
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("command did not exit in time after cancellation")
	}
}

func TestXFSChecker_Check_TimedOut(t *testing.T) {
	// Simulate xfs_repair -n invocation executing a helper process that does NOT trap SIGINT.
	// When we cancel the context, execOSCommand sends SIGINT to the process group, the helper
	// dies due to the signal, and isProcKilled(err) must evaluate to true, emitting FSCheckTimedOutEvent.
	tmpDir := t.TempDir()
	readyDir := filepath.Join(tmpDir, "ready")

	restore := setMockExec(func(ctx context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" && len(args) == 2 && args[0] == "-n" {
			// Run the no-trap helper to be killed by signal.
			return execOSCommand(ctx, os.Args[0], "-helperProc=true", "-helperNoTrap=true", "-helperReadyDir="+readyDir)
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	// Create cancelable context; we'll cancel once helper signals readiness.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		err error
	}
	done := make(chan result, 1)

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)

	go func() {
		done <- result{err: ch.Check(ctx, false)}
	}()

	// Wait deterministically for helper readiness, then cancel.
	waitCtx, stopWait := context.WithTimeout(context.Background(), 15*time.Second)
	defer stopWait()
	for {
		select {
		case <-waitCtx.Done():
			t.Fatalf("timed out waiting for helper readiness dir %s", readyDir)
		default:
			if _, err := os.Stat(readyDir); err == nil {
				cancel()
				goto CANCELED_CHECK
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

CANCELED_CHECK:

	select {
	case r := <-done:
		if r.err == nil {
			t.Fatalf("expected error after timeout, got nil")
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("xfs check did not exit in time after cancellation")
	}

	want := []string{
		StartedFSCheckEvent,
		FSCheckTimedOutEvent,
	}
	assertEvents(t, obs.events, want)
}

func TestXFSChecker_Repair_TimedOut(t *testing.T) {
	// First phase: xfs_repair -n should report repairable (rc=1).
	// Second phase: actual repair should run helper without SIGINT trap so that
	// canceling the context results in a signal-terminated process, triggering
	// FSRepairTimedOutEvent via isProcKilled(err).
	tmpDir := t.TempDir()
	readyDir := filepath.Join(tmpDir, "ready-repair")

	// We need to distinguish between the -n check and the repair call.
	restore := setMockExec(func(ctx context.Context, name string, args ...string) (int, error) {
		if name == "xfs_repair" {
			// Check call
			if len(args) >= 1 && args[0] == "-n" {
				return 1, nil // repairable
			}
			// Repair call (no -n): run helper-no-trap to be killed by signal
			if len(args) == 1 {
				return execOSCommand(ctx, os.Args[0], "-helperProc=true", "-helperNoTrap=true", "-helperReadyDir="+readyDir)
			}
		}
		return -1, errors.New("unexpected call")
	})
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		err error
	}
	done := make(chan result, 1)

	obs := &capturingObserver{}
	ch, _ := GetFSChecker("/dev/sdb1", "xfs", obs)

	go func() {
		done <- result{err: ch.Check(ctx, true)}
	}()

	// Wait for the repair helper readiness, then cancel.
	waitCtx, stopWait := context.WithTimeout(context.Background(), 15*time.Second)
	defer stopWait()
	for {
		select {
		case <-waitCtx.Done():
			t.Fatalf("timed out waiting for helper readiness dir %s", readyDir)
		default:
			if _, err := os.Stat(readyDir); err == nil {
				cancel()
				goto CANCELED_REPAIR
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

CANCELED_REPAIR:

	select {
	case r := <-done:
		if r.err == nil {
			t.Fatalf("expected error after repair timeout, got nil")
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("xfs repair did not exit in time after cancellation")
	}

	want := []string{
		StartedFSCheckEvent,
		FoundErrorsEvent,
		StartFSRepairEvent,
		FSRepairTimedOutEvent,
	}
	assertEvents(t, obs.events, want)
}

// Helper to set FSCheckLinesToLog for a test and restore afterwards.
// Requires FSCheckLinesToLog to be a var, not const.
func withFSN(t *testing.T, n int, fn func()) {
	t.Helper()
	orig := FSCheckLinesToLog
	FSCheckLinesToLog = n
	defer func() { FSCheckLinesToLog = orig }()
	fn()
}

func TestCollectStderr_EmptyBuffer(t *testing.T) {
	withFSN(t, 3, func() {
		var errBuf bytes.Buffer // empty

		got := truncOutput(errBuf, "cmd")
		if got == nil {
			t.Fatalf("expected non-nil buffer")
		}
		if got.Len() != 0 {
			t.Fatalf("expected empty buffer, got %q", got.String())
		}
	})
}

func TestTruncOutput_WhitespaceOnly(t *testing.T) {
	withFSN(t, 3, func() {
		var errBuf bytes.Buffer
		errBuf.WriteString("   \n\t\n \n")

		got := truncOutput(errBuf, "cmd", "-v")
		if got.Len() != 0 {
			t.Fatalf("expected empty buffer for whitespace-only input, got %q", got.String())
		}
	})
}

func TestTruncOutput_FewerThanN(t *testing.T) {
	withFSN(t, 3, func() {
		var errBuf bytes.Buffer
		// Leading spaces preserved, trailing spaces removed on non-empty lines.
		errBuf.WriteString("  line1   \n\n\tline2\t\t\n")

		got := truncOutput(errBuf, "cmd", "a", "b")

		want := strings.Join([]string{
			"stderr from command: cmd a b",
			"  line1",
			"\tline2",
		}, "\n")

		if got.String() != want {
			t.Fatalf("unexpected output\n--- got ---\n%q\n--- want ---\n%q", got.String(), want)
		}
	})
}

func TestTruncOutput_ExactlyN(t *testing.T) {
	withFSN(t, 3, func() {
		var errBuf bytes.Buffer
		// Exactly 3 non-empty (after trim-space emptiness check).
		errBuf.WriteString("one \n  two\t \n\tthree\n")

		got := truncOutput(errBuf, "tool")

		want := strings.Join([]string{
			"stderr from command: tool",
			"one",   // trailing space trimmed
			"  two", // leading spaces preserved, trailing trimmed
			"\tthree",
		}, "\n")

		if got.String() != want {
			t.Fatalf("unexpected output\n--- got ---\n%q\n--- want ---\n%q", got.String(), want)
		}
	})
}

func TestTruncOutput_MoreThanN(t *testing.T) {
	withFSN(t, 3, func() {
		var errBuf bytes.Buffer
		// Non-empty trimmed lines (show leading variations): a, "  b", "\tc", "d", " e  "
		// Also include blanks to ensure they are skipped.
		errBuf.WriteString(" \n a \n  b\t \n\tc\t\t\n\n d\n e  \n")

		got := truncOutput(errBuf, "tool", "--flag")

		// Expect ellipsis and last 3 non-empty: "\tc" (leading tab, trailing trimmed),
		// "d" (no change), " e" (leading preserved, trailing trimmed).
		want := strings.Join([]string{
			"stderr from command: tool --flag",
			"...",
			"\tc",
			" d",
			" e",
		}, "\n")

		if got.String() != want {
			t.Fatalf("unexpected output for >N with ellipsis\n--- got ---\n%q\n--- want ---\n%q", got.String(), want)
		}
	})
}

// Test EXT checker with real filesystem images.
// The images are sample images found in the e2fsprogs test suite .

func TestEXTChecker_WithRealImages(t *testing.T) {
	// Check if e2fsck and gunzip are available in PATH
	if _, err := exec.LookPath("e2fsck"); err != nil {
		t.Skip("e2fsck binary not found in PATH, skipping test")
	}
	if _, err := exec.LookPath("gunzip"); err != nil {
		t.Skip("gunzip binary not found in PATH, skipping test")
	}
	if !canSetupLoopback() {
		t.Skip("insufficient permissions to setup loopback devices, skipping test")
	}

	// Find all img_*.gz files in tests/fsckdata
	fsckdataDir := filepath.Join("tests", "fsckdata")
	entries, err := os.ReadDir(fsckdataDir)
	if err != nil {
		t.Fatalf("failed to read fsckdata directory: %v", err)
	}

	// Process each archived image
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gz") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			testEXTCheckerWithImage(t, fsckdataDir, entry.Name())
		})
	}
}

func testEXTCheckerWithImage(t *testing.T, fsckdataDir, gzFile string) {
	t.Helper()

	// Extract expected exit code from filename
	// Pattern: img_[number]_....
	var expectedCode int
	n, err := fmt.Sscanf(gzFile, "img_%d_", &expectedCode)
	if err != nil || n != 1 {
		t.Fatalf("invalid filename format: %s", gzFile)
	}

	// Create temporary directory for this test
	tempDir := t.TempDir()
	imagePath := filepath.Join(fsckdataDir, gzFile)
	unpackedPath := filepath.Join(tempDir, "image")

	// Unpack the gz file
	t.Logf("Unpacking %s to %s", imagePath, unpackedPath)
	if err := unpackGz(imagePath, unpackedPath); err != nil {
		t.Fatalf("failed to unpack %s: %v", gzFile, err)
	}

	// Setup loopback device
	loopDev, err := setupLoopback(unpackedPath)
	if err != nil {
		t.Fatalf("failed to setup loopback device: %v", err)
	}
	defer func() {
		if err := detachLoopback(loopDev); err != nil {
			t.Logf("warning: failed to detach loopback %s: %v", loopDev, err)
		}
	}()

	// Create ext checker and run in check-and-repair mode
	obs := &capturingObserver{}
	checker, err := GetFSChecker(loopDev, "ext", obs)
	if err != nil {
		t.Fatalf("failed to get fs checker: %v", err)
	}

	// Run check with repair enabled
	err = checker.Check(context.Background(), true)

	// Verify expectations based on expected code
	if expectedCode == 0 {
		// Should succeed
		if err != nil {
			t.Fatalf("expected check to succeed for %s, got error: %v", gzFile, err)
		}
		t.Logf("✓ %s: check succeeded as expected", gzFile)
	} else {
		// Should fail with error containing the exit code
		if err == nil {
			t.Fatalf("expected check to fail for %s, but it succeeded", gzFile)
		}

		// Check if error message contains the expected exit code
		expectedErrPattern := fmt.Sprintf("\\(%d\\)", expectedCode)
		matched, regexErr := regexp.MatchString(expectedErrPattern, err.Error())
		if regexErr != nil {
			t.Fatalf("failed to match error pattern: %v", regexErr)
		}
		if !matched {
			t.Fatalf("error message should contain (%d), got: %s", expectedCode, err.Error())
		}
		t.Logf("✓ %s: check failed with expected error code %d", gzFile, expectedCode)
	}
}

// unpackGz unpacks a gz file to the target path
func unpackGz(gzPath, targetPath string) error {
	gzFile, err := os.Open(gzPath)
	if err != nil {
		return fmt.Errorf("failed to open gz file: %w", err)
	}
	defer gzFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	// Use gunzip command to unpack
	cmd := exec.Command("gunzip", "-c", gzPath)
	cmd.Stdout = targetFile
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unpack gz file: %w", err)
	}

	return nil
}

// canSetupLoopback checks if user has permissions to setup loopback devices
func canSetupLoopback() bool {
	// Try to find a free loop device
	cmd := exec.Command("losetup", "-f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Check if we got a valid loop device path
	loopDevice := strings.TrimSpace(string(output))
	if !strings.HasPrefix(loopDevice, "/dev/loop") {
		return false
	}

	// Test if we can actually access the device
	if _, err := os.Stat(loopDevice); err != nil {
		return false
	}

	return true
}

// setupLoopback creates a loopback device for the given image file
func setupLoopback(imagePath string) (string, error) {
	// Use losetup to create loopback device
	cmd := exec.Command("losetup", "-f", "--show", imagePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to setup loopback: %w", err)
	}

	loopDev := strings.TrimSpace(string(output))
	if loopDev == "" {
		return "", errors.New("empty loopback device path")
	}

	return loopDev, nil
}

// detachLoopback detaches a loopback device
func detachLoopback(loopDev string) error {
	cmd := exec.Command("losetup", "-d", loopDev)
	return cmd.Run()
}
