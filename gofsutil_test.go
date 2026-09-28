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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// Expect an error since /mnt/test doesn't exist and findmnt will fail
	if err == nil {
		t.Errorf("expected error for non-existent mountpoint, got nil")
	}
	t.Logf("Filesystem type: %s, error: %v", result, err)
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
	tests := []struct {
		name      string
		ctx       context.Context
		symPath   string
		shouldErr bool
	}{
		{
			name:      "Context with symlink path",
			ctx:       context.Background(),
			symPath:   "/test/symlink",
			shouldErr: true,
		},
		{
			name:      "With /tmp path",
			ctx:       context.Background(),
			symPath:   "/tmp",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EvalSymlinks(tt.ctx, &tt.symPath)
			if (err != nil) != tt.shouldErr {
				t.Errorf("TestEvalSymlinks failed for %s: expected error: %v, got: %v", tt.name, tt.shouldErr, err)
			}
		})
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
	timeout := time.Duration(10)
	chroot := ""
	arguments := []string{"-o", "defaults"}

	if _, err := MultipathCommand(ctx, timeout, chroot, arguments...); err == nil {
		t.Errorf("MultipathCommand should have failed with empty chroot")
	}
}

func TestTargetIPLUNToDevicePath_Error(t *testing.T) {
	// Test case: TargetIPLUNToDevicePath with non-existent bypathdir
	ctx := context.Background()
	targetIP := "1.1.1.1"
	lunID := 0

	origBypathdir := bypathdir
	bypathdir = "/nonexistent/path/that/does/not/exist"
	defer func() { bypathdir = origBypathdir }()

	if _, err := TargetIPLUNToDevicePath(ctx, targetIP, lunID); err == nil {
		t.Errorf("TargetIPLUNToDevicePath error expected")
	}
}

func TestGetFCHostPortWWNs_Error(t *testing.T) {
	// Test case: GetFCHostPortWWNs with with invalid context
	var ctx context.Context

	tempDir := t.TempDir()
	fcHostsDir = tempDir
	require.NoError(t, os.MkdirAll(fcHostsDir, 0o755))

	// Ensure the directory is cleaned up after the test
	defer func() {
		require.NoError(t, os.RemoveAll(fcHostsDir))
		fcHostsDir = "/sys/class/fc_host"
	}()

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

func TestMockGetDevMounts(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Normal operation
	GOFSMock.InduceDevMountsError = false
	GOFSMockMounts = []Info{{Device: "/dev/sda", Path: "/mnt/data"}}
	mounts, err := fs.GetDevMounts(ctx, "/dev/sda")
	assert.NoError(t, err)
	assert.NotEmpty(t, mounts)

	// Induced error
	GOFSMock.InduceDevMountsError = true
	_, err = fs.GetDevMounts(ctx, "/dev/sda")
	assert.Error(t, err)
	GOFSMock.InduceDevMountsError = false
}

func TestMockValidateDevice(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	_, err := fs.ValidateDevice(ctx, "/dev/sda")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestMockReadProcMounts(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	_, _, err := fs.readProcMounts(ctx, "/", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestMockRescanSCSIHost(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Normal operation
	GOFSMock.InduceRescanError = false
	err := fs.RescanSCSIHost(ctx, []string{}, "0")
	assert.NoError(t, err)

	// Induced error
	GOFSMock.InduceRescanError = true
	err = fs.RescanSCSIHost(ctx, []string{}, "0")
	assert.Error(t, err)
	GOFSMock.InduceRescanError = false
}

func TestMockRescanSCSIHost_WithCallback(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	var capturedScan string
	GOFSRescanCallback = func(scanString string) {
		capturedScan = scanString
	}
	defer func() { GOFSRescanCallback = nil }()

	err := fs.RescanSCSIHost(ctx, []string{}, "5")
	assert.NoError(t, err)
	assert.Equal(t, "5", capturedScan)
}

func TestMockRemoveBlockDevice(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Induced error
	GOFSMock.InduceRemoveBlockDeviceError = true
	err := fs.RemoveBlockDevice(ctx, "/dev/sda")
	assert.Error(t, err)
	GOFSMock.InduceRemoveBlockDeviceError = false

	// Normal operation with mock WWN entries
	GOFSMockWWNToDevice = map[string]string{
		"wwn1": "/dev/sda",
		"wwn2": "/dev/sdb",
	}
	err = fs.RemoveBlockDevice(ctx, "/dev/sda")
	assert.NoError(t, err)
	// wwn1 should be removed
	_, exists := GOFSMockWWNToDevice["wwn1"]
	assert.False(t, exists)
}

func TestGetDevice(t *testing.T) {
	// Test with nonexistent path - returns original string
	result := getDevice("/nonexistent/path")
	assert.Equal(t, "/nonexistent/path", result)

	// Test with valid path - resolves symlinks
	dir := t.TempDir()
	result = getDevice(dir)
	assert.NotEmpty(t, result)
}

func TestMockFstrim(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Normal operation
	GOFSMock.InduceFstrimError = false
	GOFSMockFstrimResult = nil
	result, err := fs.Fstrim(ctx, "/mnt/data")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1073741824), result.BytesTrimmed)

	// With custom result
	customResult := &FstrimResult{BytesTrimmed: 100}
	GOFSMockFstrimResult = customResult
	result, err = fs.Fstrim(ctx, "/mnt/data")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), result.BytesTrimmed)
	GOFSMockFstrimResult = nil

	// Induced error
	GOFSMock.InduceFstrimError = true
	_, err = fs.Fstrim(ctx, "/mnt/data")
	assert.Error(t, err)
	GOFSMock.InduceFstrimError = false
}

func TestMockBlkdiscard(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Normal operation
	GOFSMock.InduceBlkdiscardError = false
	GOFSMockBlkdiscardResult = nil
	result, err := fs.Blkdiscard(ctx, "/dev/sda")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Induced error
	GOFSMock.InduceBlkdiscardError = true
	_, err = fs.Blkdiscard(ctx, "/dev/sda")
	assert.Error(t, err)
	GOFSMock.InduceBlkdiscardError = false
}

func TestMockCheckDiscardSupport(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Normal operation
	GOFSMock.InduceCheckDiscardSupportError = false
	GOFSMockDiscardCapability = nil
	result, err := fs.CheckDiscardSupport(ctx, "/dev/sda")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Supported)

	// Induced error
	GOFSMock.InduceCheckDiscardSupportError = true
	_, err = fs.CheckDiscardSupport(ctx, "/dev/sda")
	assert.Error(t, err)
	GOFSMock.InduceCheckDiscardSupportError = false
}

func TestMockMount_WithExistingSource(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	origMounts := GOFSMockMounts
	defer func() { GOFSMockMounts = origMounts }()

	// Pre-populate GOFSMockMounts with a source path
	GOFSMockMounts = []Info{{Device: "/dev/sda1", Path: "/mnt/source"}}
	GOFSMock.InduceMountError = false

	// Mount with source = "/mnt/source" which matches Path of existing mount
	err := fs.Mount(ctx, "/mnt/source", "/mnt/target", "ext4")
	assert.NoError(t, err)

	// Verify the new mount has the Source set from the existing mount
	found := false
	for _, m := range GOFSMockMounts {
		if m.Path == "/mnt/target" {
			found = true
			assert.Equal(t, "/dev/sda1", m.Source)
			assert.Equal(t, "devtmpfs", m.Device)
		}
	}
	assert.True(t, found, "Expected mount to /mnt/target")
}

func TestMockWWNToDevicePath(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Setup mock WWN mapping
	GOFSMockWWNToDevice = map[string]string{
		"abc123": "/dev/sda",
	}
	GOFSWWNPath = "/dev/disk/by-id/wwn-0x"

	// Normal operation
	GOFSMock.InduceWWNToDevicePathError = false
	byID, devPath, err := fs.WWNToDevicePath(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "/dev/disk/by-id/wwn-0xabc123", byID)
	assert.Equal(t, "/dev/sda", devPath)

	// With nil map (auto-initialized)
	GOFSMockWWNToDevice = nil
	byID, devPath, err = fs.WWNToDevicePath(ctx, "xyz")
	assert.NoError(t, err)
	assert.Equal(t, "/dev/disk/by-id/wwn-0xxyz", byID)
	assert.Empty(t, devPath)

	// Induced error
	GOFSMock.InduceWWNToDevicePathError = true
	_, _, err = fs.WWNToDevicePath(ctx, "abc123")
	assert.Error(t, err)
	GOFSMock.InduceWWNToDevicePathError = false
}

func TestMockTargetIPLUNToDevicePath(t *testing.T) {
	ctx := context.Background()
	fs := &mockfs{}

	// Setup mock target mapping
	GOFSMockTargetIPLUNToDevice = map[string]string{
		"ip-1.1.1.1:-lun-0": "/dev/sda",
	}

	// Normal operation - matching entry
	GOFSMock.InduceTargetIPLUNToDeviceError = false
	result, err := fs.TargetIPLUNToDevicePath(ctx, "1.1.1.1", 0)
	assert.NoError(t, err)
	assert.Equal(t, "/dev/sda", result["ip-1.1.1.1:-lun-0"])

	// Non-matching entry
	result, err = fs.TargetIPLUNToDevicePath(ctx, "2.2.2.2", 1)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// With nil map (auto-initialized)
	GOFSMockTargetIPLUNToDevice = nil
	result, err = fs.TargetIPLUNToDevicePath(ctx, "1.1.1.1", 0)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// Induced error
	GOFSMock.InduceTargetIPLUNToDeviceError = true
	_, err = fs.TargetIPLUNToDevicePath(ctx, "1.1.1.1", 0)
	assert.Error(t, err)
	GOFSMock.InduceTargetIPLUNToDeviceError = false
}

func TestGetDevice_WithSymlink(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "realfile")
	require.NoError(t, os.WriteFile(realFile, []byte("test"), 0o644))

	link := filepath.Join(dir, "symlink")
	require.NoError(t, os.Symlink(realFile, link))

	result := getDevice(link)
	assert.Equal(t, realFile, result)
}

func TestGetDevice_EvalSymlinksError(t *testing.T) {
	dir := t.TempDir()
	// Create a dangling symlink (target doesn't exist)
	link := filepath.Join(dir, "dangling")
	require.NoError(t, os.Symlink(filepath.Join(dir, "nonexistent"), link))

	// Lstat succeeds but EvalSymlinks fails on dangling symlink
	result := getDevice(link)
	assert.Equal(t, link, result)
}
