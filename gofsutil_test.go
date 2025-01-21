package gofsutil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDiskFormat(t *testing.T) {
	ctx := context.Background()
	disk := "/dev/sda1"

	result, err := GetDiskFormat(ctx, disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Disk format: %s", result)
}

func TestFormatAndMount(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		source      string
		target      string
		fsType      string
		opts        []string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			fsType:      "ext4",
			opts:        []string{"-o", "defaults"},
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			fsType:      "ext4",
			opts:        []string{"-o", "defaults"},
			induceErr:   true,
			expectedErr: errors.New("bindMount induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceBindMountError = tt.induceErr
			err := fs.FormatAndMount(tt.ctx, tt.source, tt.target, tt.fsType, tt.opts...)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestMount(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		source      string
		target      string
		fsType      string
		opts        []string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			fsType:      "ext4",
			opts:        []string{"-o", "defaults"},
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			fsType:      "ext4",
			opts:        []string{"-o", "defaults"},
			induceErr:   true,
			expectedErr: errors.New("mount induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceMountError = tt.induceErr
			err := fs.Mount(tt.ctx, tt.source, tt.target, tt.fsType, tt.opts...)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestBindMount(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		source      string
		target      string
		opts        []string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			opts:        []string{"-o", "defaults"},
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			opts:        []string{"-o", "defaults"},
			induceErr:   true,
			expectedErr: errors.New("bindMount induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceBindMountError = tt.induceErr
			err := fs.bindMount(tt.ctx, tt.source, tt.target, tt.opts...)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestUnmount(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		target      string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Induced error",
			target:      "/mnt/data",
			induceErr:   true,
			expectedErr: errors.New("unmount induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceUnmountError = tt.induceErr
			err := fs.Unmount(tt.ctx, tt.target)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

// func TestGetMountInfoFromDevice(t *testing.T) {
// 	ctx := context.Background()
// 	devID := "sda1"

// 	result, err := GetMountInfoFromDevice(ctx, devID)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// 	t.Logf("Mount info: %+v", result)
// }

func TestGetMpathNameFromDevice(t *testing.T) {
	ctx := context.Background()
	device := "sda1"

	result, err := GetMpathNameFromDevice(ctx, device)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Mpath name: %s", result)
}

func TestResizeFS(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		volumePath  string
		devicePath  string
		ppathDevice string
		mpathDevice string
		fsType      string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			volumePath:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "/dev/mapper/ppath",
			mpathDevice: "/dev/mapper/mpath",
			fsType:      "ext4",
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			volumePath:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "/dev/mapper/ppath",
			mpathDevice: "/dev/mapper/mpath",
			fsType:      "ext4",
			induceErr:   true,
			expectedErr: errors.New("resizeFS induced error:	Failed to resize device"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceResizeFSError = tt.induceErr
			err := fs.ResizeFS(tt.ctx, tt.volumePath, tt.devicePath, tt.ppathDevice, tt.mpathDevice, tt.fsType)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestResizeMultipath(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		deviceName  string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			deviceName:  "/dev/mapper/mpath",
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			deviceName:  "/dev/mapper/mpath",
			induceErr:   true,
			expectedErr: errors.New("resize multipath induced error: Failed to resize multipath mount device"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceResizeMultipathError = tt.induceErr
			err := fs.ResizeMultipath(tt.ctx, tt.deviceName)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestFindFSType(t *testing.T) {
	ctx := context.Background()
	mountpoint := "/mnt/test"

	result, err := FindFSType(ctx, mountpoint)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Filesystem type: %s", result)
}

func TestDeviceRescan(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		devicePath  string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			devicePath:  "/dev/sda",
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			devicePath:  "/dev/sda",
			induceErr:   true,
			expectedErr: errors.New("DeviceRescan induced error: Failed to rescan device"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceDeviceRescanError = tt.induceErr
			err := fs.DeviceRescan(tt.ctx, tt.devicePath)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestGetMounts(t *testing.T) {
	ctx := context.Background()

	result, err := GetMounts(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Mounts: %+v", result)
}

func TestGetSysBlockDevicesForVolumeWWN(t *testing.T) {
	ctx := context.Background()
	volumeWWN := "60000970000120000549533030354435"

	result, err := GetSysBlockDevicesForVolumeWWN(ctx, volumeWWN)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Sys block devices: %+v", result)
}

func TestMethodFormat(t *testing.T) {
	tests := []struct {
		testname    string
		ctx         context.Context
		source      string
		target      string
		fsType      string
		opts        []string
		induceErr   bool
		expectedErr error
	}{
		{
			testname:    "Normal operation",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			fsType:      "ext4",
			opts:        []string{"-o", "defaults"},
			induceErr:   false,
			expectedErr: nil,
		},
		{
			testname:    "Induced error",
			source:      "/dev/sda1",
			target:      "/mnt/data",
			fsType:      "ext4",
			opts:        []string{"-o", "defaults"},
			induceErr:   true,
			expectedErr: errors.New("format induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceFormatError = tt.induceErr
			err := fs.Format(tt.ctx, tt.source, tt.target, tt.fsType, tt.opts...)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
