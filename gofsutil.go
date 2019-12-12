package gofsutil

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

// FSinterface has the methods support by gofsutils.
type FSinterface interface {
	// Architecture specific implementations
	getDiskFormat(ctx context.Context, disk string) (string, error)
	format(ctx context.Context, source, target, fsType string, opts ...string) error
	formatAndMount(ctx context.Context, source, target, fsType string, opts ...string) error
	bindMount(ctx context.Context, source, target string, opts ...string) error
	getMounts(ctx context.Context) ([]Info, error)
	readProcMounts(ctx context.Context, path string, info bool) ([]Info, uint32, error)
	mount(ctx context.Context, source, target, fsType string, opts ...string) error
	unmount(ctx context.Context, target string) error
	getDevMounts(ctx context.Context, dev string) ([]Info, error)
	validateDevice(ctx context.Context, source string) (string, error)
	wwnToDevicePath(ctx context.Context, wwn string) (string, string, error)
	rescanSCSIHost(ctx context.Context, targets []string, lun string) error
	removeBlockDevice(ctx context.Context, blockDevicePath string) error
	targetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error)
	multipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error)
	getFCHostPortWWNs(ctx context.Context) ([]string, error)
	issueLIPToAllFCHosts(ctx context.Context) error
	getSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error)

	// Architecture agnostic implementations, generally just wrappers
	GetDiskFormat(ctx context.Context, disk string) (string, error)
	Format(ctx context.Context, source, target, fsType string, options ...string) error
	FormatAndMount(ctx context.Context, source, target, fsType string, options ...string) error
	Mount(ctx context.Context, source, target, fsType string, options ...string) error
	BindMount(ctx context.Context, source, target string, options ...string) error
	Unmount(ctx context.Context, target string) error
	GetMounts(ctx context.Context) ([]Info, error)
	GetDevMounts(ctx context.Context, dev string) ([]Info, error)
	ValidateDevice(ctx context.Context, source string) (string, error)
	WWNToDevicePath(ctx context.Context, wwn string) (string, string, error)
	RescanSCSIHost(ctx context.Context, targets []string, lun string) error
	RemoveBlockDevice(ctx context.Context, blockDevicePath string) error
	TargetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error)
	MultipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error)
	GetFCHostPortWWNs(ctx context.Context) ([]string, error)
	IssueLIPToAllFCHosts(ctx context.Context) error
	GetSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error)
}

var (
	MultipathDevDiskByIDPrefix = "/dev/disk/by-id/dm-uuid-mpath-3"
)

var (
	// ErrNotImplemented is returned when a platform does not implement
	// the contextual function.
	ErrNotImplemented = errors.New("not implemented")

	// fs is the default FS instance.
	fs FSinterface = &FS{ScanEntry: defaultEntryScanFunc}
)

// Type of variable containing context-keys
type ContextKey string

// Context option for using the nodiscard flag on mkfs
const NoDiscard = "NoDiscard"

// UseMockFS creates a mock file system for testing. This then is used
// with gofsutil_mock.go methods so that you can implement mock testing
// for calls using gofsutils.
func UseMockFS() {
	fs = &mockfs{ScanEntry: defaultEntryScanFunc}
}

// GetDiskFormat uses 'lsblk' to see if the given disk is unformatted.
func GetDiskFormat(ctx context.Context, disk string) (string, error) {
	return fs.GetDiskFormat(ctx, disk)
}

// FormatAndMount uses unix utils to format and mount the given disk.
func FormatAndMount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string) error {

	return fs.FormatAndMount(ctx, source, target, fsType, opts...)
}

// Format uses unix utils to format the given disk.
func Format(
	ctx context.Context,
	source, target, fsType string,
	opts ...string) error {

	return fs.Format(ctx, source, target, fsType, opts...)
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
func Mount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string) error {

	return fs.Mount(ctx, source, target, fsType, opts...)
}

// BindMount behaves like Mount was called with a "bind" flag set
// in the options list.
func BindMount(
	ctx context.Context,
	source, target string,
	opts ...string) error {

	return fs.BindMount(ctx, source, target, opts...)
}

// Unmount unmounts the target.
func Unmount(ctx context.Context, target string) error {
	return fs.Unmount(ctx, target)
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
func GetMounts(ctx context.Context) ([]Info, error) {
	return fs.GetMounts(ctx)
}

// GetDevMounts returns a slice of all mounts for the provided device.
func GetDevMounts(ctx context.Context, dev string) ([]Info, error) {
	return fs.GetDevMounts(ctx, dev)
}

// EvalSymlinks evaluates the provided path and updates it to remove
// any symlinks in its structure, replacing them with the actual path
// components.
func EvalSymlinks(ctx context.Context, symPath *string) error {
	realPath, err := filepath.EvalSymlinks(*symPath)
	if err != nil {
		return err
	}
	*symPath = realPath
	return nil
}

// ValidateDevice evalutes the specified path and determines whether
// or not it is a valid device. If true then the provided path is
// evaluated and returned as an absolute path without any symlinks.
// Otherwise an empty string is returned.
func ValidateDevice(ctx context.Context, source string) (string, error) {
	return fs.ValidateDevice(ctx, source)
}

// WWNToDevicePath returns the device path corresponding to a LUN's WWN
// (World Wide Name). A null path is returned if the device isn't found.
func WWNToDevicePath(ctx context.Context, wwn string) (string, error) {
	_, path, err := fs.WWNToDevicePath(ctx, wwn)
	return path, err
}

// WWNToDevicePathX returns the symlink and device path corresponding to a LUN's WWN
// (World Wide Name). A null path is returned if the device isn't found.
func WWNToDevicePathX(ctx context.Context, wwn string) (string, string, error) {
	return fs.WWNToDevicePath(ctx, wwn)
}

// RescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// FC port WWN or iscsi iqn target(s) are rescanned.
// Targets must either begin with 0x50 for FC or iqn. for Iscsi.
// If lun is specified, then the rescan is for that particular volume.
func RescanSCSIHost(ctx context.Context, targets []string, lun string) error {
	return fs.RescanSCSIHost(ctx, targets, lun)
}

// RemoveBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func RemoveBlockDevice(ctx context.Context, blockDevicePath string) error {
	return fs.RemoveBlockDevice(ctx, blockDevicePath)
}

// Execute the multipath command with a timeout and various arguments.
// Optionally a chroot directory can be specified for changing root directory.
// This only works in a container or another environment where it can chroot to /noderoot.
func MultipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error) {
	return fs.MultipathCommand(ctx, timeoutSeconds, chroot, arguments...)
}

// TargetIPLUNToDevicePath returns the /dev/devxxx path when presented with an ISCSI target IP
// and a LUN id. It returns the entry name in /dev/disk/by-path and the device path, along with error.
func TargetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error) {
	return fs.TargetIPLUNToDevicePath(ctx, targetIP, lunID)
}

// GetFCHostPortWWNs returns the Fibrechannel Port WWNs of the local host.
func GetFCHostPortWWNs(ctx context.Context) ([]string, error) {
	return fs.GetFCHostPortWWNs(ctx)
}

// IssueLIPToAllFCHosts issues the LIP command to all FC hosts.
func IssueLIPToAllFCHosts(ctx context.Context) error {
	return fs.IssueLIPToAllFCHosts(ctx)
}

// GetSysBlockDevicesForVolumeWWN given a volumeWWN will return a list of devices in /sys/block for that WWN (e.g. sdx, sdaa)
func GetSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error) {
	return fs.GetSysBlockDevicesForVolumeWWN(ctx, volumeWWN)
}
