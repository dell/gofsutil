// Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gofsutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/dell/csmlog"
)

const (
	procMountsPath = "/proc/self/mountinfo"
	// procMountsRetries is number of times to retry for a consistent
	// read of procMountsPath.
	procMountsRetries = 30
	ppinqtool         = "pp_inq"
)

var (
	bindRemountOpts              = []string{"remount"}
	getExecCommandCombinedOutput = func(name string, arg ...string) ([]byte, error) {
		/* #nosec G204 */
		return exec.Command(name, arg...).CombinedOutput()
	}
)

// getDiskFormat uses 'lsblk' to see if the given disk is unformatted
// For SDC devices (/dev/scini*), it uses 'blkid' instead as lsblk doesn't work with character devices
func (fs *FS) getDiskFormat(_ context.Context, disk string) (string, error) {
	path := filepath.Clean(disk)
	if err := validatePath(path); err != nil {
		return "", err
	}

	f := log.Fields{
		"disk": disk,
	}

	// Check if this is an SDC device (character device /dev/scini*)
	// SDC devices are character devices, not block devices, so lsblk doesn't work
	if strings.HasPrefix(disk, "/dev/scini") {
		log.WithFields(f).Info("checking if SDC disk is formatted using blkid")
		buf, err := getExecCommandCombinedOutput("blkid", "-o", "value", "-s", "TYPE", disk)
		out := strings.TrimSpace(string(buf))
		log.WithFields(log.Fields{"output": out}).Debug("blkid output")

		if err != nil {
			// blkid returns exit code 2 when no filesystem is found (unformatted)
			// This is expected for unformatted devices
			var exitCode int
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}

			if exitCode == 2 || out == "" {
				// Exit code 2 or empty output means unformatted device
				log.WithFields(f).WithFields(log.Fields{"exitCode": exitCode}).Debug("blkid indicates unformatted SDC device")
				return "", nil
			}

			// Other errors are actual failures
			log.WithFields(f).WithFields(log.Fields{"exitCode": exitCode, log.FieldError: err.Error()}).Error("blkid failed for SDC device")
			return "", err
		}

		if out != "" {
			// The device is formatted
			log.WithFields(f).WithFields(log.Fields{"fstype": out}).Info("SDC device has filesystem")
			return out, nil
		}

		// Empty output means unformatted
		return "", nil
	}

	// For non-SDC devices, use lsblk as before
	args := []string{"-n", "-o", "FSTYPE", disk}

	log.WithFields(f).WithFields(log.Fields{"args": args}).Info(
		"checking if disk is formatted using lsblk")
	buf, err := getExecCommandCombinedOutput("lsblk", args...)
	out := string(buf)
	log.WithFields(log.Fields{"output": out}).Debug("lsblk output")

	if err != nil {
		log.WithFields(f).WithFields(log.Fields{log.FieldError: err.Error()}).Error(
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
		// lsblk returned empty for a single device. This could mean the device
		// is truly unformatted, OR lsblk could not read the FS type (e.g.
		// device-mapper symlinks inside containers where sysfs is incomplete).
		// Fall back to blkid which reads the superblock directly.
		blkidArgs := []string{"-o", "value", "-s", "TYPE", disk}
		log.WithFields(f).WithFields(log.Fields{"args": blkidArgs}).Info(
			"lsblk returned empty; probing with blkid as fallback")
		blkBuf, blkErr := getExecCommandCombinedOutput("blkid", blkidArgs...)
		blkOut := strings.TrimSpace(string(blkBuf))
		log.WithFields(log.Fields{"output": blkOut}).Debug("blkid fallback output")

		if blkErr == nil && blkOut != "" {
			// blkid found a filesystem that lsblk missed
			log.WithFields(f).WithFields(log.Fields{"fstype": blkOut}).Info(
				"blkid fallback detected filesystem")
			return blkOut, nil
		}

		// blkid also returned empty or errored — device is truly unformatted
		return "", nil
	}

	// The device has dependent devices, most probably partitions (LVM, LUKS
	// and MD RAID are reported as FSTYPE and caught above).
	return "unknown data, probably partitions", nil
}

// RequestID is for logging the CSI or other type of Request ID
const RequestID = "RequestID"

// formatAndMount uses unix utils to format and mount the given disk
func (fs *FS) formatAndMount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string,
) error {
	err := fs.validateMountArgs(source, target, fsType, opts...)
	if err != nil {
		return err
	}

	reqID := ctx.Value(ContextKey(RequestID))
	noDiscard := ctx.Value(ContextKey(NoDiscard))

	// retrive and remove fsFormatOption from opts if it is passed in
	var fsFormatOptionString string
	var fsFormatOption []string
	if len(opts) > 0 {
		fsFormatOptionString = opts[len(opts)-1]
		if strings.HasPrefix(fsFormatOptionString, "fsFormatOption:") {
			fsFormatOptionString = strings.TrimPrefix(fsFormatOptionString, "fsFormatOption:")
			fsFormatOption = strings.Split(fsFormatOptionString, " ")
			opts = opts[0 : len(opts)-1]
		}
	}

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
	log.WithFields(log.Fields{"mountErr": mountErr.Error()}).Info("Mount attempt failed")

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

		// if no fs format option is provided
		if len(fsFormatOption) == 0 {
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

			if fsType == "xfs" {
				args = append(args, "-m", "crc=0")
			}
		} else {
			// user provides format option
			if noDiscard == NoDiscard {
				if fsType == "ext4" || fsType == "ext3" {
					args = append(fsFormatOption, "-E", "nodiscard", source)
				}

				if fsType == "xfs" {
					args = append(fsFormatOption, "-K", source)
				}
			} else {
				args = append(fsFormatOption, source)
			}
		}

		f["fsType"] = fsType
		log.WithFields(f).Info(
			"disk appears unformatted, attempting format")

		log.Infof("mkfs args: %v", args)

		mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)
		out, err := exec.Command(mkfsCmd, args...).CombinedOutput() // #nosec G204
		if err != nil {
			log.WithFields(f).WithFields(log.Fields{log.FieldError: err.Error(), "out": string(out)}).Error(
				"format of disk failed")
		} else {
			log.WithFields(f).Info("disk successfully formatted")
		}

		// a format of the disk has been attempted, so try mounting it again
		log.WithFields(f).Info("re-attempting disk mount")
		return fs.mount(ctx, source, target, fsType, opts...)
	}

	// Disk is already formatted and failed to mount
	if len(fsType) == 0 || fsType == existingFormat {
		log.WithFields(log.Fields{"ExistingFormat": existingFormat}).Info("Disk failed to mount")
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
	opts ...string,
) error {
	err := fs.validateMountArgs(source, target, fsType, opts...)
	if err != nil {
		return err
	}

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
	log.Infof("formatting with command: %s %v", mkfsCmd, args)
	/* #nosec G204 */
	out, err := exec.Command(mkfsCmd, args...).CombinedOutput()
	if err != nil {
		log.WithFields(f).WithFields(log.Fields{log.FieldError: err.Error(), "out": string(out)}).Error(
			"format of disk failed")
		return err
	}
	return nil
}

// bindMount performs a bind mount
func (fs *FS) bindMount(
	ctx context.Context,
	source, target string,
	opts ...string,
) error {
	err := fs.doMount(ctx, "mount", source, target, "", "bind")
	if err != nil {
		return err
	}
	return fs.doMount(ctx, "mount", source, target, "", opts...)
}

// isLsblkNew returns true if lsblk version is greater than 2.3 and false otherwise
func (fs *FS) isLsblkNew() (bool, error) {
	lsblkNew := false
	// Use exec.Command with argv-style arguments instead of bash -c to avoid shell injection
	bufcheck, errcheck := exec.Command("lsblk", "-V").Output()
	if errcheck != nil {
		return lsblkNew, errcheck
	}
	outputcheck := string(bufcheck)
	versionRegx := regexp.MustCompile(`linux (?P<vers>\d+\.\d+)\.*`)
	match := versionRegx.FindStringSubmatch(outputcheck)
	subMatchMap := make(map[string]string)
	for i, name := range versionRegx.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}
	if s, err := strconv.ParseFloat(subMatchMap["vers"], 64); err == nil {
		log.Debugf("lsblk version: %v", s)
		if s > 2.30 { // need to check exact version
			lsblkNew = true
		}
	}
	return lsblkNew, nil
}

func (fs *FS) getMpathNameFromDevice(
	_ context.Context, device string,
) (string, error) {
	path := filepath.Clean(device)
	if err := validatePath(path); err != nil {
		return "", err
	}

	// Validate device to prevent OS command injection (CWE-78)
	if err := validateDeviceID(device); err != nil {
		return "", fmt.Errorf("invalid device for getMpathNameFromDevice: %w", err)
	}

	var args []string
	lsblkNew, err := fs.isLsblkNew()
	if err != nil {
		return "", err
	}
	if lsblkNew {
		args = []string{"-Px", "MODE"}
	} else {
		args = []string{"-P"}
	}

	// Use exec.Command with argv-style arguments instead of bash -c
	buf, err := exec.Command("lsblk", args...).Output()
	if err != nil {
		return "", nil // lsblk may fail if device doesn't exist, return empty
	}
	output := string(buf)

	// Filter output in Go instead of using awk/grep in shell
	mpathDeviceRegx := regexp.MustCompile(`NAME="(\S+)"`)
	lines := strings.Split(output, "\n")
	foundDevice := false
	linesToCheck := 0

	for _, line := range lines {
		if strings.Contains(line, device) {
			foundDevice = true
			linesToCheck = 2
		}
		if foundDevice && linesToCheck > 0 {
			if strings.Contains(line, `TYPE="mpath"`) {
				match := mpathDeviceRegx.FindStringSubmatch(line)
				if len(match) > 1 {
					return match[1], nil
				}
			}
			linesToCheck--
		}
	}

	return "", nil
}

func (fs *FS) getNativeDevicesFromPpath(
	ctx context.Context, ppath string,
) ([]string, error) {
	log.Infof("powerpath - trying to find native devices for ppath: %s", ppath)
	var devices []string
	var deviceWWN string

	deviceName := fmt.Sprintf("/dev/%s", ppath)
	cmd := fmt.Sprintf("%s/%s", "/noderoot/sbin", ppinqtool)
	log.Debugf("pp_inq cmd: %s", cmd)
	args := []string{"-wwn", "-dev", deviceName}
	out, err := getExecCommandCombinedOutput(cmd, args...)
	if err != nil {
		log.Errorf("Error powermt display %s: %v", deviceName, err)
		return devices, err
	}
	op := strings.Split(string(out), "\n")
	log.Debugf("pp_inq output for %s %+v", ppath, op)
	/*  Output for pp_inq -wwn -dev /dev/ppath
	L#1 Inquiry utility, Version V9.2-2602 (Rev 0.0)
		----------------------------------------------------------------------------
		DEVICE           :VEND    :PROD            :WWN
		----------------------------------------------------------------------------
	L#7	/dev/emcpowerg   :EMC     :SYMMETRIX       :60000970000120000549533030354435
	*/
	for _, line := range op {
		if strings.Contains(line, "emcpower") {
			tokens := strings.Fields(line)
			deviceWWN = strings.Replace(tokens[3], ":", "", 1)
			log.Debugf("found device wwn %s", deviceWWN)
			break
		}
	}
	devices, err = GetSysBlockDevicesForVolumeWWN(ctx, deviceWWN)
	if err != nil {
		return nil, err
	}
	log.Debugf("found devices: %+v for ppath %s with WWN %s", devices, ppath, deviceWWN)
	return devices, nil
}

// getMountInfoFromDevice gets mount info for the given device
// It first checks the existence of powerpath device, if not then checks for multipath, if not then checks for single device.
func (fs *FS) getMountInfoFromDevice(
	ctx context.Context, devID string,
) (*DeviceMountInfo, error) {
	if devID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}
	path := filepath.Clean(devID)
	if err := validatePath(path); err != nil {
		return nil, err
	}

	// Validate devID to prevent OS command injection (CWE-78)
	// This is critical as devID is used in pattern matching against lsblk output
	if err := validateDeviceID(devID); err != nil {
		return nil, fmt.Errorf("invalid device ID for getMountInfoFromDevice: %w", err)
	}

	var output string
	lsblkNew, err := fs.isLsblkNew()
	if err != nil {
		return nil, err
	}
	// check if devID has powerpath devices
	// Use exec.Command with argv-style arguments instead of bash -c to avoid shell injection
	// Get lsblk output and filter in Go instead of using awk
	buf, err := exec.Command("lsblk", "--pairs", "--output", "NAME,MAJ:MIN,RM,SIZE,RO,TYPE,MOUNTPOINT").Output()
	if err != nil {
		return nil, err
	}
	lsblkOutput := string(buf)
	lines := strings.Split(lsblkOutput, "\n")

	// Filter for powerpath devices in Go instead of awk - use regexp.QuoteMeta for safe pattern matching
	ppathRegex := regexp.MustCompile(`emcpower.+` + regexp.QuoteMeta(devID))
	var ppathMatches []string
	for _, line := range lines {
		if ppathRegex.MatchString(line) {
			ppathMatches = append(ppathMatches, line)
		}
	}
	output = strings.Join(ppathMatches, "\n")
	if output == "" {
		// output is nil, powerpath device not found, continuing for multipath or single device
		log.Info("powerpath command output is nil, continuing for multipath or single device")

		// Filter for multipath devices in Go instead of awk
		mpathRegex := regexp.MustCompile(`mpath.+` + regexp.QuoteMeta(devID))
		var mpathMatches []string
		for _, line := range lines {
			if mpathRegex.MatchString(line) {
				mpathMatches = append(mpathMatches, line)
			}
		}
		output = strings.Join(mpathMatches, "\n")
		log.Debugf("multipath exec command output is : %+v", output)
		var lsblkArgs []string
		if output != "" {
			log.Info("Multipath device found")
			if lsblkNew {
				lsblkArgs = []string{"--pairs", "--sort", "MODE", "--output", "NAME,MAJ:MIN,RM,SIZE,RO,TYPE,MOUNTPOINT"}
			} else {
				lsblkArgs = []string{"--pairs", "--output", "NAME,MAJ:MIN,RM,SIZE,RO,TYPE,MOUNTPOINT"}
			}
		} else {
			log.Info("Multipath device not found")
			// multipath device not found, continue as single device
			lsblkArgs = []string{"--pairs", "--output", "NAME,MAJ:MIN,RM,SIZE,RO,TYPE,MOUNTPOINT"}
		}

		// Execute lsblk with argv-style arguments instead of bash -c
		buf, err = exec.Command("lsblk", lsblkArgs...).Output()
		if err != nil {
			return nil, err
		}
		lsblkOutput = string(buf)
		lines = strings.Split(lsblkOutput, "\n")

		// Filter output in Go instead of awk - use regexp.QuoteMeta for safe pattern matching
		devRegex := regexp.MustCompile(regexp.QuoteMeta(devID))
		if output != "" {
			// Multipath device found - replicate awk: '/{devID}/{if (a && a !~ /{devID}/) print a; print} {a=$0}'
			var matchedLines []string
			var prevLine string
			for _, line := range lines {
				if devRegex.MatchString(line) {
					if prevLine != "" && !devRegex.MatchString(prevLine) {
						matchedLines = append(matchedLines, prevLine)
					}
					matchedLines = append(matchedLines, line)
				}
				prevLine = line
			}
			output = strings.Join(matchedLines, "\n")
		} else {
			// Single device - replicate awk: '/{devID}/ {print $0}'
			var matchedLines []string
			for _, line := range lines {
				if devRegex.MatchString(line) {
					matchedLines = append(matchedLines, line)
				}
			}
			output = strings.Join(matchedLines, "\n")
		}
		log.Debugf("command output is : %+v", output)
	}

	if output == "" {
		return nil, fmt.Errorf("device not found")
	}
	sdDeviceRegx := regexp.MustCompile(`NAME=\"sd\S+\"`)
	nvmeDeviceRegx := regexp.MustCompile(`NAME=\"nvme\S+\"`)
	mpathDeviceRegx := regexp.MustCompile(`NAME=\"mpath\S+\"`)
	ppathDeviceRegx := regexp.MustCompile(`NAME=\"emcpower\S+\"`)
	mountRegx := regexp.MustCompile(`MOUNTPOINT=\"\S+\"`)
	deviceTypeRegx := regexp.MustCompile(`TYPE=\"mpath"`)
	deviceNameRegx := regexp.MustCompile(`NAME=\"\S+\"`)
	mountPoint := mountRegx.FindString(output)
	devices := sdDeviceRegx.FindAllString(output, 99999)
	nvmeDevices := nvmeDeviceRegx.FindAllString(output, 99999)
	mpath := mpathDeviceRegx.FindString(output)
	ppath := ppathDeviceRegx.FindString(output)
	mountInfo := new(DeviceMountInfo)
	mountInfo.MountPoint = strings.Split(mountPoint, "\"")[1]
	for _, device := range devices {
		mountInfo.DeviceNames = append(mountInfo.DeviceNames, strings.Split(device, "\"")[1])
	}
	for _, device := range nvmeDevices {
		mountInfo.DeviceNames = append(mountInfo.DeviceNames, strings.Split(device, "\"")[1])
	}
	if ppath != "" {
		log.Infof("found ppath: %s", ppath)
		mountInfo.PPathName = strings.Split(ppath, "\"")[1]
		// find native devices for given ppath
		mountInfo.DeviceNames, err = fs.getNativeDevicesFromPpath(ctx, mountInfo.PPathName)
		if err != nil {
			return nil, err
		}
	} else if mpath != "" {
		mountInfo.MPathName = strings.Split(mpath, "\"")[1]
	} else {
		// In case the mpath device is of the form /dev/mapper/3600601xxxxxxx
		// we check if TYPE is "mpath" then we pick the first mapper device name from NAME
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

// FindFSType fetches the filesystem type on mountpoint
func (fs *FS) findFSType(
	_ context.Context, mountpoint string,
) (fsType string, err error) {
	path := filepath.Clean(mountpoint)
	if err := validatePath(path); err != nil {
		return "", fmt.Errorf("failed to validate path: %s error %v", mountpoint, err)
	}

	// Use exec.Command with argv-style arguments instead of bash -c to avoid shell injection
	buf, err := exec.Command("findmnt", "-n", "-o", "FSTYPE", path).Output()
	if err != nil {
		return "", fmt.Errorf("failed to find mount information for (%s) error (%v)", mountpoint, err)
	}
	fsType = strings.TrimSpace(string(buf))
	return fsType, err
}

func (fs *FS) resizeMultipath(_ context.Context, deviceName string) error {
	path := filepath.Clean(deviceName)
	if err := validatePath(path); err != nil {
		return fmt.Errorf("failed to validate path: %s error %v", deviceName, err)
	}

	args := []string{"resize", "map", path}
	/* #nosec G204 */
	out, err := exec.Command("multipathd", args...).CombinedOutput()
	log.WithFields(log.Fields{"output": string(out)}).Debug("Multipath resize output")
	if err != nil {
		return fmt.Errorf("failed to resize multipath mount device on (%s) error (%v)", deviceName, err)
	}
	log.Infof("Filesystem on %s resized successfully", deviceName)
	return nil
}

// resizeFS expands the filesystem to the new size of underlying device
// For XFS filesystem needs filesystem mount point
// For EXT4 needs devicepath
// For EXT3 needs devicepath
// For multipath device, fs resize needs "/device/mapper/mpathname"
// For powerpath device, fs resize needs "/dev/emcpowera"
func (fs *FS) resizeFS(
	ctx context.Context, mountpoint,
	devicePath, ppathDevice, mpathDevice, fsType string,
) error {
	if ppathDevice != "" {
		devicePath = "/dev/" + ppathDevice
		err := reReadPartitionTable(ctx, devicePath)
		if err != nil {
			return err
		}
	}

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
		err = fmt.Errorf("filesystem not supported to resize")
	}
	return err
}

// reReadPartitionTable re-read the partition table of the pseudo device.
func reReadPartitionTable(_ context.Context, devicePath string) error {
	path := filepath.Clean(devicePath)
	if err := validatePath(path); err != nil {
		return fmt.Errorf("failed to validate path: %s error %v", devicePath, err)
	}
	args := []string{path}
	_, err := exec.Command("partprobe", args...).CombinedOutput() // #nosec G204
	if err != nil {
		log.Errorf("Failed to execute partprobe on %s: %s", devicePath, err.Error())
		return err
	}
	return nil
}

func (fs *FS) expandExtFs(devicePath string) error {
	path := filepath.Clean(devicePath)
	if err := validatePath(path); err != nil {
		return fmt.Errorf("failed to validate path: %s error %v", devicePath, err)
	}
	/* #nosec G204 */
	out, err := exec.Command("resize2fs", path).CombinedOutput()
	log.WithFields(log.Fields{"output": string(out)}).Debug("Ext fs resize output")
	if err != nil {
		return fmt.Errorf("ext fs: failed to resize device (%s) error (%v)", devicePath, err)
	}
	log.Infof("Ext fs: Device %s resized successfully", devicePath)
	return nil
}

func (fs *FS) expandXfs(volumePath string) error {
	path := filepath.Clean(volumePath)
	if err := validatePath(path); err != nil {
		return fmt.Errorf("failed to validate path: %s error %v", volumePath, err)
	}
	args := []string{"-d", path}
	/* #nosec G204 */
	out, err := exec.Command("xfs_growfs", args...).CombinedOutput()
	log.WithFields(log.Fields{"output": string(out)}).Debug("XFS resize output")
	if err != nil {
		return fmt.Errorf("xfs: failed to resize device (%s) error (%v)", volumePath, err)
	}
	log.Infof("Xfs: Device %s resized successfully", volumePath)
	return nil
}

// DeviceRescan rescan the device for size alterations
func (fs *FS) deviceRescan(_ context.Context,
	devicePath string,
) error {
	path := filepath.Clean(devicePath)
	if err := validatePath(path); err != nil {
		return err
	}
	device := path + "/device/rescan"
	log.Infof("Executing rescan command on device (%s)", devicePath)
	// Write directly to sysfs instead of executing bash -c "echo 1 > ...".
	out := "1\n"
	err := os.WriteFile(device, []byte(out), 0o644)
	log.WithFields(log.Fields{"output": out}).Debug("Rescan output")
	if err != nil {
		log.Errorf("Failed to rescan device with error (%s)", err.Error())
		return err
	}
	log.Infof("Successful rescan on device (%s)", devicePath)
	return nil
}

func (fs *FS) consistentRead(filename string, retry int) ([]byte, error) {
	oldContent, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	for i := 0; i < retry; i++ {
		newContent, err := os.ReadFile(filepath.Clean(filename))
		if err != nil {
			return nil, err
		}
		if bytes.Equal(oldContent, newContent) {
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
	info bool,
) ([]Info, uint32, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = file.Close()
	}()
	return ReadProcMountsFrom(ctx, file, !info, ProcMountsFields, fs.ScanEntry)
}
