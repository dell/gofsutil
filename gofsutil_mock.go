package gofsutil

import (
	"context"
	"errors"
	"fmt"
)

var (
	GOFSMockMounts []Info
	GOFSMock       struct {
		InduceBindMountError bool
		InduceMountError     bool
		InduceGetMountsError bool
		InduceDevMountsError bool
		InduceUnmountError   bool
		InduceFormatError    bool
		InduceGetDiskFormatError bool
		InduceGetDiskFormatType string
	}
)

type mockfs struct {
	// ScanEntry is the function used to process mount table entries.
	ScanEntry EntryScanFunc
}

func (fs *mockfs) getDiskFormat(ctx context.Context, disk string) (string, error) {
	if GOFSMock.InduceGetDiskFormatError {
		GOFSMock.InduceMountError = false
		return "",  errors.New("getDiskFormat induced error")
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
	info := Info{Device: source, Path: target, Type: fsType, Opts: make([]string, 0)}
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
	info := Info{Device: source, Path: target, Opts: make([]string, 0)}
	for _, str := range opts {
		info.Opts = append(info.Opts, str)
	}
	GOFSMockMounts = append(GOFSMockMounts, info)
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
	info := Info{Device: source, Path: target, Opts: make([]string, 0)}
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
