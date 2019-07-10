package gofsutil

import (
	"context"
	"errors"
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
	ctx context.Context, wwn string) (string, error) {
	return "", errors.New("not implemented")
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
