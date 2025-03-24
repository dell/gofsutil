// Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDiskFormat(t *testing.T) {
	// Test case: GetDiskFormat with invalid disk
	ctx := context.Background()
	disk := ""

	_, err := GetDiskFormat(ctx, disk)
	if err == nil {
		t.Errorf("GetDiskFormat should have failed with empty disk")
	}
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

func TestFormatAndMount_Error(t *testing.T) {
	// Test case: FormatAndMount with invalid source
	ctx := context.Background()
	source := ""
	target := "/mnt/data"
	fsType := "ext4"
	opts := []string{"-o", "defaults"}

	if err := FormatAndMount(ctx, source, target, fsType, opts...); err == nil {
		t.Errorf("FormatAndMount should have failed with empty source")
	}
}

func TestFormat_Error(t *testing.T) {
	// Test case: FormatAndMount with invalid source
	ctx := context.Background()
	source := ""
	target := "/mnt/data"
	fsType := "ext4"
	opts := []string{"-o", "defaults"}

	if err := Format(ctx, source, target, fsType, opts...); err == nil {
		t.Errorf("Format should have failed with empty source")
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

func TestMount_Error(t *testing.T) {
	// Test case: Mount with invalid source
	ctx := context.Background()
	source := ""
	target := "/mnt/data"
	fsType := "ext4"
	opts := []string{"-o", "defaults"}

	if err := Mount(ctx, source, target, fsType, opts...); err == nil {
		t.Errorf("Mount should have failed with empty source")
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

func TestBindMount_Error(t *testing.T) {
	// Test case: Mount with invalid source
	ctx := context.Background()
	source := ""
	target := "/mnt/data"
	opts := []string{"-o", "defaults"}

	if err := BindMount(ctx, source, target, opts...); err == nil {
		t.Errorf("BindMount should have failed with empty source")
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

func TestUnmount_Error(t *testing.T) {
	// Test case: Unmount with invalid target
	ctx := context.Background()
	target := ""
	if err := Unmount(ctx, target); err == nil {
		t.Errorf("Unmount should have failed with empty target")
	}
}

func TestGetMountInfoFromDevice(t *testing.T) {
	ctx := context.Background()
	devID := "/dev/sda1"

	result, err := GetMountInfoFromDevice(ctx, devID)
	if err == nil {
		t.Errorf("expected error, got %v", err)
	}
	t.Logf("Mount info: %+v", result)
}

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

func TestResizeFS_Error(t *testing.T) {
	// Test case: ResizeFS with invalid volumePath
	ctx := context.Background()
	volumePath := ""
	devicePath := "/dev/sda1"
	ppathDevice := "/dev/mapper/ppath"
	mpathDevice := "/dev/mapper/mpath"
	fsType := "ext4"

	if err := ResizeFS(ctx, volumePath, devicePath, ppathDevice, mpathDevice, fsType); err == nil {
		t.Errorf("ResizeFS should have failed with empty volumePath")
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

func TestResizeMultipath_Error(t *testing.T) {
	// Test case: ResizeMultipath with invalid deviceName
	ctx := context.Background()
	deviceName := ""

	if err := ResizeMultipath(ctx, deviceName); err == nil {
		t.Errorf("ResizeMultipath should have failed with empty deviceName")
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

func TestDeviceRescan_Error(t *testing.T) {
	// Test case: DeviceRescan with invalid devicePath
	ctx := context.Background()
	devicePath := ""

	if err := DeviceRescan(ctx, devicePath); err == nil {
		t.Errorf("DeviceRescan should have failed with empty devicePath")
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

func TestGetDevMounts_NoError(t *testing.T) {
	ctx := context.Background()
	dev := "abc"

	result, err := GetDevMounts(ctx, dev)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Get Dev Mounts: %+v", result)
}

func TestEvalSymlinks(t *testing.T) {
	// Test case: EvalSymlinks with invalid context
	ctx := context.Background()
	ctx = nil
	symPath := "/test/symlink"

	if err := EvalSymlinks(ctx, &symPath); err == nil {
		t.Errorf("EvalSymlinks should have failed with nil context")
	}
}

func TestWWNToDevicePath_Error(t *testing.T) {
	// Test case: WWNToDevicePath with invalid wwn
	ctx := context.Background()
	wwn := ""

	_, err := WWNToDevicePath(ctx, wwn)
	if err == nil {
		t.Errorf("WWNToDevicePath should have failed with empty wwn")
	}
}

func TestWWNToDevicePathX(t *testing.T) {
	// Test case: WWNToDevicePathX with invalid wwn
	ctx := context.Background()
	wwn := ""

	_, _, err := WWNToDevicePathX(ctx, wwn)
	if err == nil {
		t.Errorf("WWNToDevicePathX should have failed with empty wwn")
	}
}

func TestMultipathCommand_Error(t *testing.T) {
	// Test case: MultipathCommand with invalid chroot
	ctx := context.Background()
	timeoutSeconds := time.Duration(10)
	chroot := ""
	arguments := []string{"-o", "defaults"}

	if _, err := MultipathCommand(ctx, timeoutSeconds, chroot, arguments...); err == nil {
		t.Errorf("MultipathCommand should have failed with empty chroot")
	}
}

func TestTargetIPLUNToDevicePath_Error(t *testing.T) {
	// Test case: TargetIPLUNToDevicePath with invalid targetIP
	ctx := context.Background()
	targetIP := "1.1.1.1"
	lunID := 0

	if _, err := TargetIPLUNToDevicePath(ctx, targetIP, lunID); err == nil {
		t.Errorf("TargetIPLUNToDevicePath error expectd")
	}
}

func TestGetFCHostPortWWNs_Error(t *testing.T) {
	// Test case: GetFCHostPortWWNs with with invalid context
	ctx := context.Background()
	ctx = nil

	if _, err := GetFCHostPortWWNs(ctx); err != nil {
		t.Errorf("GetFCHostPortWWNs failed with err: %v", err)
	}
}

func TestIssueLIPToAllFCHosts_Error(t *testing.T) {
	// Test case: IssueLIPToAllFCHosts with with invalid context
	ctx := context.Background()

	if err := IssueLIPToAllFCHosts(ctx); err != nil {
		t.Errorf("IssueLIPToAllFCHosts failed: %v", err)
	}
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

func TestFsInfo_Error(t *testing.T) {
	// Test case: FsInfo with with invalid path
	ctx := context.Background()
	path := ""

	if _, _, _, _, _, _, err := FsInfo(ctx, path); err == nil {
		t.Errorf("FsInfo should have failed with empty path")
	}
}

// func TestUseMockFS(t *testing.T) {
// 	tests := []struct {
// 		name string
// 	}{
// 		{
// 			name: "UseMockFS",
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(_ *testing.T) {
// 			UseMockFS()
// 		})
// 	}
// }
