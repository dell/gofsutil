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
