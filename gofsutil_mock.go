package gofsutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	// GOFSMockMounts and the other variables in gofsutils_mock.go
	// allow the user to manipulate the data returned in the mock
	// mode or return induced errors.
	GOFSMockMounts []Info
	// GOFSMockFCHostWWNs is a list of port WWNs on this host's FC NICs
	GOFSMockFCHostWWNs []string
	// GOFSMockWWNToDevice allows you to return a device for a WWN.
	GOFSMockWWNToDevice map[string]string
	// GOFSWWNPath gives a path for the WWN entry (e.g. /dev/disk/by-id/wwn-0x)
	GOFSWWNPath string
	// GOFSMockTargetIPLUNToDevice map[string]string
	// assumes key is of form ip-<targetIP>:-lun<decimal_lun_id>
	GOFSMockTargetIPLUNToDevice map[string]string
	// GOFSRescanCallback is a function called when a rescan is processed.
	GOFSRescanCallback func(scan string)
	// GOFSMockMountInfo contains mount information for filesystem volumes
	GOFSMockMountInfo *DeviceMountInfo

	// GOFSMock allows you to induce errors in the various routine.
	GOFSMock struct {
		InduceBindMountError              bool
		InduceMountError                  bool
		InduceGetMountsError              bool
		InduceDevMountsError              bool
		InduceUnmountError                bool
		InduceFormatError                 bool
		InduceGetDiskFormatError          bool
		InduceWWNToDevicePathError        bool
		InduceTargetIPLUNToDeviceError    bool
		InduceRemoveBlockDeviceError      bool
		InduceMultipathCommandError       bool
		InduceFCHostWWNsError             bool
		InduceRescanError                 bool
		InduceIssueLipError               bool
		InduceGetSysBlockDevicesError     bool
		InduceGetDiskFormatType           string
		InduceGetMountInfoFromDeviceError bool
		InduceDeviceRescanError           bool
		InduceResizeMultipathError        bool
		InduceFSTypeError                 bool
		InduceResizeFSError               bool
		InduceGetMpathNameFromDeviceError bool
		InduceFilesystemInfoError         bool
	}
)

type mockfs struct {
	// ScanEntry is the function used to process mount table entries.
	ScanEntry EntryScanFunc
}

func (fs *mockfs) getDiskFormat(ctx context.Context, disk string) (string, error) {
	if GOFSMock.InduceGetDiskFormatError {
		GOFSMock.InduceMountError = false
		return "", errors.New("getDiskFormat induced error")
	}
	if GOFSMock.InduceGetDiskFormatType != "" {
		GOFSMock.InduceMountError = false
		return GOFSMock.InduceGetDiskFormatType, nil
	}
	for _, info := range GOFSMockMounts {
		if info.Device == disk {
			return info.Type, nil
		}
	}
	return "", nil
}

func (fs *mockfs) formatAndMount(ctx context.Context, source, target, fsType string, opts ...string) error {
	if GOFSMock.InduceBindMountError {
		GOFSMock.InduceMountError = false
		return errors.New("bindMount induced error")
	}
	fmt.Printf(">>>formatAndMount source %s target %s fstype %s opts %v\n", source, target, fsType, opts)
	info := Info{Device: getDevice(source), Path: target, Type: fsType, Opts: make([]string, 0)}
	for _, str := range opts {
		info.Opts = append(info.Opts, str)
	}
	GOFSMockMounts = append(GOFSMockMounts, info)
	return nil
}

func (fs *mockfs) format(ctx context.Context, source, target, fsType string, opts ...string) error {
	if GOFSMock.InduceFormatError {
		return errors.New("format induced error")
	}
	fmt.Printf(">>>format source %s target %s fstype %s opts %v\n", source, target, fsType, opts)
	for _, info := range GOFSMockMounts {
		if info.Device == source {
			info.Type = fsType
		}
	}
	return nil
}

func (fs *mockfs) bindMount(ctx context.Context, source, target string, opts ...string) error {
	if GOFSMock.InduceBindMountError {
		return errors.New("bindMount induced error")
	}
	fmt.Printf(">>>bindMount source %s target %s opts %v\n", source, target, opts)
	info := Info{Device: getDevice(source), Path: target, Opts: make([]string, 0)}
	for _, str := range opts {
		info.Opts = append(info.Opts, str)
	}
	GOFSMockMounts = append(GOFSMockMounts, info)
	return nil
}

func (fs *mockfs) DeviceRescan(ctx context.Context, devicePath string) error {
	return fs.deviceRescan(ctx, devicePath)
}

func (fs *mockfs) deviceRescan(ctx context.Context, devicePath string) error {
	if GOFSMock.InduceDeviceRescanError {
		return errors.New("DeviceRescan induced error: Failed to rescan device")
	}
	return nil
}

func (fs *mockfs) ResizeFS(ctx context.Context, volumePath, devicePath, ppathDevice, mpathDevice, fsType string) error {
	return fs.resizeFS(ctx, volumePath, devicePath, ppathDevice, mpathDevice, fsType)
}

func (fs *mockfs) resizeFS(ctx context.Context, volumePath, devicePath, ppathDevice, mpathDevice, fsType string) error {
	if GOFSMock.InduceResizeFSError {
		return errors.New("resizeFS induced error:	Failed to resize device")
	}
	return nil
}

func (fs *mockfs) FindFSType(ctx context.Context, mountpoint string) (fsType string, err error) {
	return fs.findFSType(ctx, mountpoint)
}

func (fs *mockfs) findFSType(ctx context.Context, mountpoint string) (fsType string, err error) {
	if GOFSMock.InduceFSTypeError {
		return "", errors.New("getMounts induced error: Failed to fetch filesystem as no mount info")
	}
	return "xfs", nil
}

func (fs *mockfs) GetMountInfoFromDevice(ctx context.Context, devID string) (*DeviceMountInfo, error) {
	return fs.getMountInfoFromDevice(ctx, devID)
}

func (fs *mockfs) getMountInfoFromDevice(ctx context.Context, devID string) (*DeviceMountInfo, error) {
	if GOFSMock.InduceGetMountInfoFromDeviceError {
		return GOFSMockMountInfo, errors.New("getMounts induced error: Failed to find mount information")
	}
	mntPoint := "/noderoot/var/lib/kubelet/pods/abc-123/volumes/k8.io/pmax-0123/mount"
	GOFSMockMountInfo = &DeviceMountInfo{
		DeviceNames: []string{"sda", "sdb"},
		MPathName:   "mpathb",
		MountPoint:  mntPoint,
	}
	return GOFSMockMountInfo, nil
}

func (fs *mockfs) GetMpathNameFromDevice(ctx context.Context, devID string) (string, error) {
	return fs.getMpathNameFromDevice(ctx, devID)
}

func (fs *mockfs) getMpathNameFromDevice(ctx context.Context, devID string) (string, error) {
	if GOFSMock.InduceGetMpathNameFromDeviceError {
		return "", errors.New("getMpathNameFromDevice induced error: Failed to find mount information")
	}

	return "mpatha", nil
}

func (fs *mockfs) FsInfo(ctx context.Context, path string) (int64, int64, int64, int64, int64, int64, error) {
	return fs.fsInfo(ctx, path)
}

func (fs *mockfs) fsInfo(ctx context.Context, path string) (int64, int64, int64, int64, int64, int64, error) {
	if GOFSMock.InduceFilesystemInfoError {
		return 0, 0, 0, 0, 0, 0, errors.New("filesystemInfo induced error: Failed to get fileystem stats")
	}
	return 1000, 2000, 1000, 4, 2, 2, nil
}

func (fs *mockfs) ResizeMultipath(ctx context.Context, deviceName string) error {
	return fs.resizeMultipath(ctx, deviceName)
}

func (fs *mockfs) resizeMultipath(ctx context.Context, deviceName string) error {
	if GOFSMock.InduceResizeMultipathError {
		return errors.New("resize multipath induced error: Failed to resize multipath mount device")
	}
	return nil
}

func (fs *mockfs) getMounts(ctx context.Context) ([]Info, error) {
	if GOFSMock.InduceGetMountsError {
		return GOFSMockMounts, errors.New("getMounts induced error")
	}
	return GOFSMockMounts, nil
}

func (fs *mockfs) readProcMounts(ctx context.Context,
	path string,
	info bool) ([]Info, uint32, error) {
	return nil, 0, errors.New("not implemented")
}

func (fs *mockfs) mount(ctx context.Context, source, target, fsType string, opts ...string) error {
	if GOFSMock.InduceMountError {
		return errors.New("mount induced error")
	}
	fmt.Printf(">>>mount source %s target %s fstype %s opts %v\n", source, target, fsType, opts)
	info := Info{Device: getDevice(source), Path: target, Opts: make([]string, 0)}
	for _, str := range opts {
		info.Opts = append(info.Opts, str)
	}

	// Try to determine the root source.
	for _, infox := range GOFSMockMounts {
		if infox.Path == source {
			info.Source = infox.Device
			info.Device = "devtmpfs"
		}
	}
	fmt.Printf(">>>mount Device %s Path %s Source %s\n", info.Device, info.Path, info.Source)
	GOFSMockMounts = append(GOFSMockMounts, info)
	return nil
}

func (fs *mockfs) unmount(ctx context.Context, target string) error {
	if GOFSMock.InduceUnmountError {
		return errors.New("unmount induced error")
	}
	for i, mnt := range GOFSMockMounts {
		if mnt.Path == target {
			copy(GOFSMockMounts[i:], GOFSMockMounts[i+1:])
			GOFSMockMounts = GOFSMockMounts[:len(GOFSMockMounts)-1]
		}
	}
	return nil
}

func (fs *mockfs) getDevMounts(ctx context.Context, dev string) ([]Info, error) {
	if GOFSMock.InduceDevMountsError {
		return GOFSMockMounts, errors.New("dev mount induced error")
	}
	return GOFSMockMounts, nil
}

func (fs *mockfs) validateDevice(
	ctx context.Context, source string) (string, error) {
	return "", errors.New("not implemented")
}

// ====================================================================
// Architecture agnostic code for the mock implementation

// GetDiskFormat uses 'lsblk' to see if the given disk is unformatted.
func (fs *mockfs) GetDiskFormat(ctx context.Context, disk string) (string, error) {
	return fs.getDiskFormat(ctx, disk)
}

// FormatAndMount uses unix utils to format and mount the given disk.
func (fs *mockfs) FormatAndMount(
	ctx context.Context,
	source, target, fsType string,
	options ...string) error {

	return fs.formatAndMount(ctx, source, target, fsType, options...)
}

// Format uses unix utils to format the given disk.
func (fs *mockfs) Format(
	ctx context.Context,
	source, target, fsType string,
	options ...string) error {

	return fs.format(ctx, source, target, fsType, options...)
}

// Mount mounts source to target as fstype with given options.
//
// The parameters 'source' and 'fstype' must be empty strings in case they
// are not required, e.g. for remount, or for an auto filesystem type where
// the kernel handles fstype automatically.
//
// The 'options' parameter is a list of options. Please see mount(8) for
// more information. If no options are required then please invoke Mount
// with an empty or nil argument.
func (fs *mockfs) Mount(
	ctx context.Context,
	source, target, fsType string,
	options ...string) error {

	return fs.mount(ctx, source, target, fsType, options...)
}

// BindMount behaves like Mount was called with a "bind" flag set
// in the options list.
func (fs *mockfs) BindMount(
	ctx context.Context,
	source, target string,
	options ...string) error {

	if options == nil {
		options = []string{"bind"}
	} else {
		options = append(options, "bind")
	}
	return fs.mount(ctx, source, target, "", options...)
}

// Unmount unmounts the target.
func (fs *mockfs) Unmount(ctx context.Context, target string) error {
	return fs.unmount(ctx, target)
}

// GetMounts returns a slice of all the mounted filesystems.
//
// * Linux hosts use mount_namespaces to obtain mount information.
//
//   Support for mount_namespaces was introduced to the Linux kernel
//   in 2.2.26 (http://man7.org/linux/man-pages/man5/proc.5.html) on
//   2004/02/04.
//
//   The kernel documents the contents of "/proc/<pid>/mountinfo" at
//   https://www.kernel.org/doc/Documentation/filesystems/proc.txt.
//
// * Darwin hosts parse the output of the "mount" command to obtain
//   mount information.
func (fs *mockfs) GetMounts(ctx context.Context) ([]Info, error) {
	return fs.getMounts(ctx)
}

// GetDevMounts returns a slice of all mounts for the provided device.
func (fs *mockfs) GetDevMounts(ctx context.Context, dev string) ([]Info, error) {
	return fs.getDevMounts(ctx, dev)
}

// ValidateDevice evalutes the specified path and determines whether
// or not it is a valid device. If true then the provided path is
// evaluated and returned as an absolute path without any symlinks.
// Otherwise an empty string is returned.
func (fs *mockfs) ValidateDevice(
	ctx context.Context, source string) (string, error) {

	return fs.validateDevice(ctx, source)
}

// wwnToDevicePath lookups a mock WWN (no prefix) to a device path.
func (fs *mockfs) wwnToDevicePath(
	ctx context.Context, wwn string) (string, string, error) {
	if GOFSMockWWNToDevice == nil {
		GOFSMockWWNToDevice = make(map[string]string)
	}
	devPath := GOFSMockWWNToDevice[wwn]
	if GOFSMock.InduceWWNToDevicePathError {
		return "", "", errors.New("induced error")
	}
	return GOFSWWNPath + wwn, devPath, nil
}

func (fs *mockfs) WWNToDevicePath(
	ctx context.Context, wwn string) (string, string, error) {
	return fs.wwnToDevicePath(ctx, wwn)
}

// RescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// iqn target(s) are rescanned.
// If lun is specified, then the rescan is for that particular volume.
func (fs *mockfs) RescanSCSIHost(ctx context.Context, targets []string, lun string) error {
	return fs.rescanSCSIHost(ctx, targets, lun)
}

// Execute the multipath command with a timeout and various arguments.
// Optionally a chroot directory can be specified for changing root directory.
// This only works in a container or another environment where it can chroot to /noderoot.
func (fs *mockfs) MultipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error) {
	return fs.multipathCommand(ctx, timeoutSeconds, chroot, arguments...)
}

// rescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// iqn target(s) are rescanned.
// If lun is specified, then the rescan is for that particular volume.
func (fs *mockfs) rescanSCSIHost(ctx context.Context, targets []string, lun string) error {
	if GOFSMock.InduceRescanError {
		return errors.New("induced rescan error")
	}
	if GOFSRescanCallback != nil {
		scanString := fmt.Sprintf("%s", lun)
		GOFSRescanCallback(scanString)
	}
	return nil
}

// RemoveBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func (fs *mockfs) RemoveBlockDevice(ctx context.Context, blockDevicePath string) error {
	if GOFSMock.InduceRemoveBlockDeviceError {
		return errors.New("remove block device induced error")
	}
	return fs.removeBlockDevice(ctx, blockDevicePath)
}

// removeBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func (fs *mockfs) removeBlockDevice(ctx context.Context, blockDevicePath string) error {
	fmt.Printf(">>>removeBlockDevice %s %#v", blockDevicePath, GOFSMockWWNToDevice)
	for key, value := range GOFSMockWWNToDevice {
		if value == blockDevicePath {
			// Remove from the device table
			delete(GOFSMockWWNToDevice, key)
		}
		_ = os.Remove(blockDevicePath)
	}
	return nil
}

// getDevice returns the actual device pointed to by a
// symlink if applicable, otherwise the original string.
func getDevice(path string) string {
	_, err := os.Lstat(path)
	if err != nil {
		return path
	}

	// eval any symlinks and make sure it points to a device
	d, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}

	result := strings.Replace(d, "\\", "/", -1)
	fmt.Printf(">>>getDevice: %s -> %s\n", path, result)
	return result
}

// Execute the multipath command with a timeout and various arguments.
// Optionally a chroot directory can be specified for changing root directory.
// This only works in a container or another environment where it can chroot to /noderoot.
func (fs *mockfs) multipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error) {
	if GOFSMock.InduceMultipathCommandError {
		return make([]byte, 0), errors.New("multipath command induced error")
	}
	GOFSMockWWNToDevice = make(map[string]string)
	return make([]byte, 0), nil
}

// TargetIPLUNToDevicePath returns the /dev/devxxx path when presented with an ISCSI target IP
// and a LUN id. It returns the entry names in /dev/disk/by-path and the corresponding device path, along with error.
func (fs *mockfs) TargetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error) {
	return fs.targetIPLUNToDevicePath(ctx, targetIP, lunID)
}

// TargetIPLUNToDevicePath returns the /dev/devxxx path when presented with an ISCSI target IP
// and a LUN id. It returns the entry names in /dev/disk/by-path and their associated device paths, along with error.
func (fs *mockfs) targetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error) {
	result := make(map[string]string, 0)
	key := fmt.Sprintf("ip-%s:-lun-%d", targetIP, lunID)
	if GOFSMockTargetIPLUNToDevice == nil {
		GOFSMockTargetIPLUNToDevice = make(map[string]string)
	}
	if GOFSMock.InduceTargetIPLUNToDeviceError {
		return result, errors.New("induced error")
	}
	path := GOFSMockTargetIPLUNToDevice[key]
	result[key] = path
	return result, nil
}

func (fs *mockfs) GetFCHostPortWWNs(ctx context.Context) ([]string, error) {
	return fs.getFCHostPortWWNs(ctx)
}

// getFCHostPortWWNs returns the port WWN addresses of local FC adapters.
func (fs *mockfs) getFCHostPortWWNs(ctx context.Context) ([]string, error) {
	portWWNs := GOFSMockFCHostWWNs
	if GOFSMock.InduceFCHostWWNsError {
		return portWWNs, errors.New("induced error")
	}
	return portWWNs, nil
}

// IssueLIPToAllFCHosts issues the LIP command to all FC hosts.
func (fs *mockfs) IssueLIPToAllFCHosts(ctx context.Context) error {
	return fs.issueLIPToAllFCHosts(ctx)
}

// issueLIPToAllFCHosts issues the LIP command to all FC hosts.
func (fs *mockfs) issueLIPToAllFCHosts(ctx context.Context) error {
	if GOFSMock.InduceIssueLipError {
		return errors.New("induced error")
	}
	return nil
}

// GetSysBlockDevicesForVolumeWWN given a volumeWWN will return a list of devices in /sys/block for that WWN (e.g. sdx, sdaa)
func (fs *mockfs) GetSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error) {
	return fs.getSysBlockDevicesForVolumeWWN(ctx, volumeWWN)
}

// GetSysBlockDevicesForVolumeWWN given a volumeWWN will return a list of devices in /sys/block for that WWN (e.g. sdx, sdaa)
func (fs *mockfs) getSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error) {
	result := make([]string, 0)
	if GOFSMock.InduceGetSysBlockDevicesError {
		return result, errors.New("induced error")
	}
	for key, value := range GOFSMockWWNToDevice {
		if key == volumeWWN {
			split := strings.Split(value, "/")
			result = append(result, split[len(split)-1])
		}
	}
	return result, nil
}
