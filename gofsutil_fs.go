package gofsutil

import (
	"context"
	"time"

	"golang.org/x/sys/unix"
)

// FS provides many filesystem-specific functions, such as mount, format, etc.
type FS struct {

	// ScanEntry is the function used to process mount table entries.
	ScanEntry EntryScanFunc
}

// GetDiskFormat uses 'lsblk' to see if the given disk is unformatted.
func (fs *FS) GetDiskFormat(ctx context.Context, disk string) (string, error) {
	return fs.getDiskFormat(ctx, disk)
}

// FormatAndMount uses unix utils to format and mount the given disk.
func (fs *FS) FormatAndMount(
	ctx context.Context,
	source, target, fsType string,
	options ...string) error {

	return fs.formatAndMount(ctx, source, target, fsType, options...)
}

// Format uses unix utils to format the given disk.
func (fs *FS) Format(
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
func (fs *FS) Mount(
	ctx context.Context,
	source, target, fsType string,
	options ...string) error {

	return fs.mount(ctx, source, target, fsType, options...)
}

// BindMount behaves like Mount was called with a "bind" flag set
// in the options list.
func (fs *FS) BindMount(
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
func (fs *FS) Unmount(ctx context.Context, target string) error {
	return fs.unmount(ctx, target)
}

//GetMountInfoFromDevice retrieves mount information associated with the volume
func (fs *FS) GetMountInfoFromDevice(ctx context.Context, devID string) (*DeviceMountInfo, error) {
	return fs.getMountInfoFromDevice(ctx, devID)
}

//GetMpathNameFromDevice retrieves mpath device name from device name
func (fs *FS) GetMpathNameFromDevice(ctx context.Context, device string) (string, error) {
	return fs.getMpathNameFromDevice(ctx, device)
}

//ResizeFS expands the filesystem to the new size of underlying device
func (fs *FS) ResizeFS(
	ctx context.Context,
	volumePath, devicePath,
	mpathDevice, fsType string) error {
	return fs.resizeFS(ctx, volumePath, devicePath, mpathDevice, fsType)
}

//FindFSType fetches the filesystem type on mountpoint
func (fs *FS) FindFSType(
	ctx context.Context, mountpoint string) (fsType string, err error) {
	return fs.findFSType(ctx, mountpoint)
}

//ResizeMultipath resizes the multipath devices mounted on FS
func (fs *FS) ResizeMultipath(ctx context.Context, deviceName string) error {
	return fs.resizeMultipath(ctx, deviceName)
}

//DeviceRescan rescan the device for size alterations
func (fs *FS) DeviceRescan(ctx context.Context,
	devicePath string) error {
	return fs.deviceRescan(ctx, devicePath)
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
func (fs *FS) GetMounts(ctx context.Context) ([]Info, error) {
	return fs.getMounts(ctx)
}

// GetDevMounts returns a slice of all mounts for the provided device.
func (fs *FS) GetDevMounts(ctx context.Context, dev string) ([]Info, error) {
	return fs.getDevMounts(ctx, dev)
}

// ValidateDevice evalutes the specified path and determines whether
// or not it is a valid device. If true then the provided path is
// evaluated and returned as an absolute path without any symlinks.
// Otherwise an empty string is returned.
func (fs *FS) ValidateDevice(
	ctx context.Context, source string) (string, error) {

	return fs.validateDevice(ctx, source)
}

// WWNToDevicePath returns the symlink and device path given a LUN's WWN.
func (fs *FS) WWNToDevicePath(ctx context.Context, wwn string) (string, string, error) {
	return fs.wwnToDevicePath(ctx, wwn)
}

// RescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// iqn target(s) are rescanned.
// If lun is specified, then the rescan is for that particular volume.
func (fs *FS) RescanSCSIHost(ctx context.Context, targets []string, lun string) error {
	return fs.rescanSCSIHost(ctx, targets, lun)
}

// RemoveBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func (fs *FS) RemoveBlockDevice(ctx context.Context, blockDevicePath string) error {
	return fs.removeBlockDevice(ctx, blockDevicePath)
}

// Execute the multipath command with a timeout and various arguments.
// Optionally a chroot directory can be specified for changing root directory.
// This only works in a container or another environment where it can chroot to /noderoot.
func (fs *FS) MultipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error) {
	return fs.multipathCommand(ctx, timeoutSeconds, chroot, arguments...)
}

// Info linux returns (available bytes, byte capacity, byte usage, total inodes, inodes free, inode usage, error)
// for the filesystem that path resides upon.
func fsInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	statfs := &unix.Statfs_t{}
	err := unix.Statfs(path, statfs)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	// Available is blocks available * fragment size
	available := int64(statfs.Bavail) * int64(statfs.Bsize)

	// Capacity is total block count * fragment size
	capacity := int64(statfs.Blocks) * int64(statfs.Bsize)

	// Usage is block being used * fragment size (aka block size).
	usage := (int64(statfs.Blocks) - int64(statfs.Bfree)) * int64(statfs.Bsize)

	inodes := int64(statfs.Files)
	inodesFree := int64(statfs.Ffree)
	inodesUsed := inodes - inodesFree

	return available, capacity, usage, inodes, inodesFree, inodesUsed, nil
}

// TargetIPLUNToDevicePath returns the /dev/devxxx path when presented with an ISCSI target IP
// and a LUN id. It returns the entry name in /dev/disk/by-path and the device path, along with error.
func (fs *FS) TargetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error) {
	return fs.targetIPLUNToDevicePath(ctx, targetIP, lunID)
}

// GetFCHostPortWWNs returns the port WWN addresses of local FC adapters.
func (fs *FS) GetFCHostPortWWNs(ctx context.Context) ([]string, error) {
	return fs.getFCHostPortWWNs(ctx)
}

// IssueLIPToAllFCHosts issues the LIP command to all FC hosts.
func (fs *FS) IssueLIPToAllFCHosts(ctx context.Context) error {
	return fs.issueLIPToAllFCHosts(ctx)
}

// GetSysBlockDevicesForVolumeWWN given a volumeWWN will return a list of devices in /sys/block for that WWN (e.g. sdx, sdaa)
func (fs *FS) GetSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error) {
	return fs.getSysBlockDevicesForVolumeWWN(ctx, volumeWWN)
}

// FsInfo given the path of the filesystem will return its stats
func (fs *FS) FsInfo(ctx context.Context, path string) (int64, int64, int64, int64, int64, int64, error) {
	return fs.fsInfo(ctx, path)
}
