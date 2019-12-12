package gofsutil

import (
	"context"
	"errors"
	"time"
)

var info []Info

func (fs *FS) getDiskFormat(ctx context.Context, disk string) (string, error) {
	return "", errors.New("not implemented")
}

func (fs *FS) formatAndMount(ctx context.Context, source, target, fsType string, opts ...string) error {
	return errors.New("not implemented")
}

func (fs *FS) format(ctx context.Context, source, target, fsType string, opts ...string) error {
	return errors.New("not implemented")
}

func (fs *FS) bindMount(ctx context.Context, source, target string, opts ...string) error {
	return errors.New("not implemented")
}

func (fs *FS) getMounts(ctx context.Context) ([]Info, error) {

	return info, errors.New("not implemented")
}

func (fs *FS) readProcMounts(ctx context.Context,
	path string,
	info bool) ([]Info, uint32, error) {
	return nil, 0, errors.New("not implemented")
}

func (fs *FS) mount(ctx context.Context, source, target, fsType string, opts ...string) error {
	return errors.New("not implemented")
	return nil
}

func (fs *FS) unmount(ctx context.Context, target string) error {
	return errors.New("not implemented")
}

func (fs *FS) getDevMounts(ctx context.Context, dev string) ([]Info, error) {
	return info, errors.New("not implemented")
}

func (fs *FS) validateDevice(
	ctx context.Context, source string) (string, error) {
	return "", errors.New("not implemented")
}

func (fs *FS) wwnToDevicePath(
	ctx context.Context, wwn string) (string, string, error) {
	return "", "", errors.New("not implemented")
}

// targetIPLUNToDevicePath returns all the /dev/disk/by-path entries for a give targetIP and lunID
func (fs *FS) targetIPLUNToDevicePath(ctx context.Context, targetIP string, lunID int) (map[string]string, error) {
	result := make(map[string]string, 0)
	return result, errors.New("not implemented")
}

// rescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// iqn target(s) are rescanned.
// If lun is specified, then the rescan is for that particular volume.
func (fs *FS) rescanSCSIHost(ctx context.Context, targets []string, lun string) error {
	return errors.New("not implemented")
}

// RemoveBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func (fs *FS) removeBlockDevice(ctx context.Context, blockDevicePath string) error {
	return errors.New("not implemented")
}

// Execute the multipath command with a timeout and various arguments.
// Optionally a chroot directory can be specified for changing root directory.
// This only works in a container or another environment where it can chroot to /noderoot.
func (fs *FS) multipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error) {
	result := make([]byte, 0)
	return result, errors.New("not implemented")
}

func (fs *FS) getFCHostPortWWNs(context.Context) ([]string, error) {
	result := make([]string, 0)
	return result, errors.New("not implemented")
}

// issueLIPToAllFCHosts issues the LIP command to all FC hosts.
func (fs *FS) issueLIPToAllFCHosts(ctx context.Context) error {
	return errors.New("not implemented")
}

// getSysBlockDevicesForVolumeWWN given a volumeWWN will return a list of devices in /sys/block for that WWN (e.g. sdx, sdaa)
func (fs *FS) getSysBlockDevicesForVolumeWWN(ctx context.Context, volumeWWN string) ([]string, error) {
	result := make([]string, 0)
	return result, errors.New("not implemented")
}
