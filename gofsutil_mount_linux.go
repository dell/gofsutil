package gofsutil

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	procMountsPath = "/proc/self/mountinfo"
	// procMountsRetries is number of times to retry for a consistent
	// read of procMountsPath.
	procMountsRetries = 30
)

var (
	bindRemountOpts = []string{"remount"}
)

// getDiskFormat uses 'lsblk' to see if the given disk is unformatted
func (fs *FS) getDiskFormat(ctx context.Context, disk string) (string, error) {

	args := []string{"-n", "-o", "FSTYPE", disk}

	f := log.Fields{
		"disk": disk,
	}
	log.WithFields(f).WithField("args", args).Info(
		"checking if disk is formatted using lsblk")
	buf, err := exec.Command("lsblk", args...).CombinedOutput()
	out := string(buf)
	log.WithField("output", out).Debug("lsblk output")

	if err != nil {
		log.WithFields(f).WithError(err).Error(
			"failed to determine if disk is formatted")
		return "", err
	}

	// Split lsblk output into lines. Unformatted devices should contain only
	// "\n". Beware of "\n\n", that's a device with one empty partition.
	out = strings.TrimSuffix(out, "\n") // Avoid last empty line
	lines := strings.Split(out, "\n")
	if lines[0] != "" {
		// The device is formatted
		return lines[0], nil
	}

	if len(lines) == 1 {
		// The device is unformatted and has no dependent devices
		return "", nil
	}

	// The device has dependent devices, most probably partitions (LVM, LUKS
	// and MD RAID are reported as FSTYPE and caught above).
	return "unknown data, probably partitions", nil
}

// Log the CSI or other type of Request ID
const RequestID = "RequestID"

// formatAndMount uses unix utils to format and mount the given disk
func (fs *FS) formatAndMount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string) error {
	reqID := ctx.Value(ContextKey(RequestID))
	noDiscard := ctx.Value(ContextKey(NoDiscard))

	opts = append(opts, "defaults")
	f := log.Fields{
		"reqID":   reqID,
		"source":  source,
		"target":  target,
		"fsType":  fsType,
		"options": opts,
	}

	// Try to mount the disk
	log.WithFields(f).Info("attempting to mount disk")
	mountErr := fs.mount(ctx, source, target, fsType, opts...)
	if mountErr == nil {
		return nil
	}
	log.WithField("mountErr", mountErr.Error()).Info("Mount attempt failed")

	// Mount failed. This indicates either that the disk is unformatted or
	// it contains an unexpected filesystem.
	existingFormat, err := fs.getDiskFormat(ctx, source)
	if err != nil {
		log.WithFields(f).Info("error determining disk format")
		return err
	}
	f = log.Fields{
		"reqID":          reqID,
		"source":         source,
		"existingFormat": existingFormat,
	}
	log.WithFields(f).Info("getDiskFormat returned after initial mount failed")
	if existingFormat == "" {
		log.WithFields(f).Info("disk is unformatted")
		// Disk is unformatted so format it.
		args := []string{source}
		// Use 'ext4' as the default
		if len(fsType) == 0 {
			fsType = "ext4"
		}

		if fsType == "ext4" || fsType == "ext3" {
			args = []string{"-F", source}
			if noDiscard == NoDiscard {
				// -E nodiscard option to improve mkfs times
				args = []string{"-F", "-E", "nodiscard", source}
			}
		}

		if fsType == "xfs" && noDiscard == NoDiscard {
			// -K option (nodiscard) to improve mkfs times
			args = []string{"-K", source}
		}

		f["fsType"] = fsType
		log.WithFields(f).Info(
			"disk appears unformatted, attempting format")

		mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)
		if err := exec.Command(mkfsCmd, args...).Run(); err != nil {
			log.WithFields(f).WithError(err).Error(
				"format of disk failed")
		}

		// the disk has been formatted successfully try to mount it again.
		log.WithFields(f).Info(
			"disk successfully formatted")
		return fs.mount(ctx, source, target, fsType, opts...)
	}

	// Disk is already formatted and failed to mount
	if len(fsType) == 0 || fsType == existingFormat {
		log.WithField("ExistingFormat", existingFormat).Info("Disk failed to mount")
		// This is mount error
		return mountErr
	}

	// Block device is formatted with unexpected filesystem
	return fmt.Errorf(
		"failed to mount volume as %q; already contains %s: error: %v",
		fsType, existingFormat, mountErr)
}

// format uses unix utils to format and mount the given disk
func (fs *FS) format(
	ctx context.Context,
	source, target, fsType string,
	opts ...string) error {
	reqID := ctx.Value(ContextKey("RequestID"))
	noDiscard := ctx.Value(ContextKey(NoDiscard))

	opts = append(opts, "defaults")
	f := log.Fields{
		"reqID":   reqID,
		"source":  source,
		"target":  target,
		"fsType":  fsType,
		"options": opts,
	}

	// Disk is unformatted so format it.
	args := []string{source}
	// Use 'ext4' as the default
	if len(fsType) == 0 {
		fsType = "ext4"
	}

	if fsType == "ext4" || fsType == "ext3" {
		args = []string{"-F", source}
		if noDiscard == NoDiscard {
			// -E nodiscard option to improve mkfs times
			args = []string{"-F", "-E", "nodiscard", source}
		}
	}

	if fsType == "xfs" && noDiscard == NoDiscard {
		// -K option (nodiscard) to improve mkfs times
		args = []string{"-K", source}
	}

	f["fsType"] = fsType
	log.WithFields(f).Info(
		"disk appears unformatted, attempting format")

	mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)
	log.Printf("formatting with command: %s %v", mkfsCmd, args)
	err := exec.Command(mkfsCmd, args...).Run()
	if err != nil {
		log.WithFields(f).WithError(err).Error(
			"format of disk failed")
		return err
	}
	return nil
}

// bindMount performs a bind mount
func (fs *FS) bindMount(
	ctx context.Context,
	source, target string,
	opts ...string) error {

	err := fs.doMount(ctx, "mount", source, target, "", "bind")
	if err != nil {
		return err
	}
	return fs.doMount(ctx, "mount", source, target, "", opts...)
}

func (fs *FS) consistentRead(filename string, retry int) ([]byte, error) {
	oldContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	for i := 0; i < retry; i++ {
		newContent, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		if bytes.Compare(oldContent, newContent) == 0 {
			return newContent, nil
		}
		// Files are different, continue reading
		oldContent = newContent
	}
	return nil, fmt.Errorf("could not get consistent content of %s after %d attempts", filename, retry)
}

// getMounts returns a slice of all the mounted filesystems
func (fs *FS) getMounts(ctx context.Context) ([]Info, error) {
	infos := make([]Info, 0)
	content, err := fs.consistentRead(procMountsPath, procMountsRetries)
	if err != nil {
		return infos, err
	}
	buffer := bytes.NewBuffer(content)
	infos, _, err = ReadProcMountsFrom(ctx, buffer, true, ProcMountsFields, fs.ScanEntry)
	return infos, err
}

// readProcMounts reads procMountsInfo and produce a hash
// of the contents and a list of the mounts as Info objects.
func (fs *FS) readProcMounts(
	ctx context.Context,
	path string,
	info bool) ([]Info, uint32, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	return ReadProcMountsFrom(ctx, file, !info, ProcMountsFields, fs.ScanEntry)
}
