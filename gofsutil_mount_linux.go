package gofsutil

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"path/filepath"

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
	/* #nosec G204 */
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
		/* #nosec G204 */
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
	/* #nosec G204 */
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

func (fs *FS) getMountInfoFromDevice(
	ctx context.Context, devID string) (*DeviceMountInfo, error) {
	var cmd string
	/* #nosec G204 */
        checkCmd := "lsblk -P | awk '/mpath.+" + devID + "/ {print $0}'"
	/* #nosec G204 */
        buf, err := exec.Command("bash", "-c", checkCmd).Output()
        if err != nil {
                return nil, err
        }
        output := string(buf)
        fmt.Println("Check: "+ checkCmd)
        if output != "" {
                cmd = "lsblk -P | awk '/" + devID + "/{if (a && a !~ /" + devID + "/) print a; print} {a=$0}'"
        } else {
		/* #nosec G204 */
                cmd = "lsblk -P | awk '/" + devID + "/ {print $0}'"
        }
        fmt.Println(cmd)
	/* #nosec G204 */
        buf, err = exec.Command("bash", "-c", cmd).Output()
        if err != nil {
                return nil, err
        }
        output = string(buf)
        if output == "" {
                return nil, fmt.Errorf("Device not found")
        }
	sdDeviceRegx := regexp.MustCompile(`NAME=\"sd\S+\"`)
	mpathDeviceRegx := regexp.MustCompile(`NAME=\"mpath\S+\"`)
	mountRegx := regexp.MustCompile(`MOUNTPOINT=\"\S+\"`)
	deviceTypeRegx := regexp.MustCompile(`TYPE=\"mpath"`)
	deviceNameRegx := regexp.MustCompile(`NAME=\"\S+\"`)
	mountPoint := mountRegx.FindString(output)
	devices := sdDeviceRegx.FindAllString(output, 99999)
	mpath := mpathDeviceRegx.FindString(output)
	mountInfo := new(DeviceMountInfo)
	mountInfo.MountPoint = strings.Split(mountPoint, "\"")[1]

	for _, device := range devices {
		mountInfo.DeviceNames = append(mountInfo.DeviceNames, strings.Split(device, "\"")[1])
	}
	if mpath != "" {
		mountInfo.MPathName = strings.Split(mpath, "\"")[1]
	} else {
		//In case the mpath device is of the form /dev/mapper/3600601xxxxxxx 
		//we check if TYPE is "mpath" then we pick the first mapper device name from NAME
		for _, deviceInfo := range strings.Split(output, "\n") {
			deviceType := deviceTypeRegx.FindString(deviceInfo)
			if deviceType != "" {
				name := deviceNameRegx.FindString(deviceInfo)
				if name != "" {
					mountInfo.MPathName = strings.Split(name, "\"")[1]
					break
				}
			}
		}
	}
	return mountInfo, nil
}

//FindFSType fetches the filesystem type on mountpoint
func (fs *FS) findFSType(
	ctx context.Context, mountpoint string) (fsType string, err error) {
	cmd := "findmnt -n " + mountpoint + " | awk '{print $3}'"
	/* #nosec G204 */
	buf, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("Failed to find mount information for (%s) error (%v)", mountpoint, err)
	}
	fsType = strings.TrimSuffix(string(buf), "\n")
	return
}

func (fs *FS) resizeMultipath(ctx context.Context, deviceName string) error {
	args := []string{"resize", "map", deviceName}
	/* #nosec G204 */
	out, err := exec.Command("multipathd", args...).CombinedOutput()
	log.WithField("output", string(out)).Debug("Multipath resize output")
	if err != nil {
		return fmt.Errorf("Failed to resize multipath mount device on (%s) error (%v)", deviceName, err)
	}
	log.Infof("Filesystem on %s resized successfully", deviceName)
	return nil
}

//resizeFS expands the filesystem to the new size of underlying device
//For XFS filesystem needs filesystem mount point
//For EXT4 needs devicepath
//For EXT3 needs devicepath
//For multipath device, fs resize needs "/device/mapper/mpathname"
func (fs *FS) resizeFS(
	ctx context.Context, mountpoint,
	devicePath, mpathDevice, fsType string) error {
	if mpathDevice != "" {
		devicePath = "/dev/mapper/" + mpathDevice
		mountpoint = devicePath
	}
	var err error
	switch fsType {
	case "ext4":
		err = fs.expandExtFs(devicePath)
	case "ext3":
		err = fs.expandExtFs(devicePath)
	case "xfs":
		err = fs.expandXfs(mountpoint)
	default:
		err = fmt.Errorf("Filesystem not supported to resize")
	}
	return err
}

func (fs *FS) expandExtFs(devicePath string) error {
	/* #nosec G204 */
	out, err := exec.Command("resize2fs", devicePath).CombinedOutput()
	log.WithField("output", string(out)).Debug("Ext fs resize output")
	if err != nil {
		return fmt.Errorf("Ext fs: Failed to resize device (%s) error (%v)", devicePath, err)
	}
	log.Infof("Ext fs: Device %s resized successfully", devicePath)
	return nil
}

func (fs *FS) expandXfs(volumePath string) error {
	args := []string{"-d", volumePath}
	/* #nosec G204 */
	out, err := exec.Command("xfs_growfs", args...).CombinedOutput()
	log.WithField("output", string(out)).Debug("XFS resize output")
	if err != nil {
		return fmt.Errorf("Xfs: Failed to resize device (%s) error (%v)", volumePath, err)
	}
	log.Infof("Xfs: Device %s resized successfully", volumePath)
	return nil
}

//DeviceRescan rescan the device for size alterations
func (fs *FS) deviceRescan(ctx context.Context,
	devicePath string) error {
	device := devicePath + "/device/rescan"
	args := []string{"-c", "echo 1 > " + device}
	log.Infof("Executing rescan command on device (%s)", devicePath)
	/* #nosec G204 */
	buf, err := exec.Command("bash", args...).CombinedOutput()
	out := string(buf)
	log.WithField("output", out).Debug("Rescan output")
	if err != nil {
		log.Error("Failed to rescan device with error (%s)", err.Error())
		return err
	}
	log.Info("Successful rescan on device (%s)", devicePath)
	return nil
}

func (fs *FS) consistentRead(filename string, retry int) ([]byte, error) {
	oldContent, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	for i := 0; i < retry; i++ { 
		newContent, err := ioutil.ReadFile(filepath.Clean(filename))
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
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil,0, err
	}
	defer func() error { if err := file.Close(); err != nil { return err } else {return nil} } ()
	return ReadProcMountsFrom(ctx, file, !info, ProcMountsFields, fs.ScanEntry)
}
