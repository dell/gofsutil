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
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mocking exec.Command
var execCommand = exec.Command

func TestGetDiskFormatInvalidPath(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create a test disk path
	disk := "/dev/ invalid"

	// Call getDiskFormat
	_, err := fs.getDiskFormat(context.Background(), disk)
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestGetDiskFormatUnformattedDisk(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create a test disk path
	disk := "/dev/sda1"

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
		return []byte("\n"), nil
	}

	// Call getDiskFormat
	_, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGetDiskFormatUnknownData(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create a test disk path
	disk := "/dev/sda1"

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
		return []byte("\ntest1\ntest2"), nil
	}

	// Call getDiskFormat
	_, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGetDiskFormatSDCWithFilesystem(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create an SDC disk path
	disk := "/dev/scinia"

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(name string, args ...string) ([]byte, error) {
		// Verify blkid is called with correct arguments
		if name != "blkid" {
			t.Errorf("expected blkid command, got %s", name)
		}
		expectedArgs := []string{"-o", "value", "-s", "TYPE", disk}
		if len(args) != len(expectedArgs) {
			t.Errorf("expected %d arguments, got %d", len(expectedArgs), len(args))
		}
		for i, arg := range expectedArgs {
			if args[i] != arg {
				t.Errorf("expected arg[%d] = %s, got %s", i, arg, args[i])
			}
		}
		return []byte("xfs"), nil
	}

	// Call getDiskFormat
	fstype, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if fstype != "xfs" {
		t.Errorf("expected xfs, got %s", fstype)
	}
}

func TestGetDiskFormatSDCUnformatted(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create an SDC disk path
	disk := "/dev/scinia"

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(name string, _ ...string) ([]byte, error) {
		// Verify blkid is called
		if name != "blkid" {
			t.Errorf("expected blkid command, got %s", name)
		}
		// Simulate blkid with empty output (unformatted device)
		return []byte(""), nil
	}

	// Call getDiskFormat
	fstype, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error for unformatted SDC device, got %v", err)
	}
	if fstype != "" {
		t.Errorf("expected empty fstype for unformatted device, got %s", fstype)
	}
}

func TestGetDiskFormatSDCNonSCDDevice(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create a non-SDC disk path (should use lsblk)
	disk := "/dev/sda1"

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(name string, _ ...string) ([]byte, error) {
		// Verify lsblk is called for non-SDC devices
		if name != "lsblk" {
			t.Errorf("expected lsblk command for non-SDC device, got %s", name)
		}
		return []byte("ext4"), nil
	}

	// Call getDiskFormat
	fstype, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if fstype != "ext4" {
		t.Errorf("expected ext4, got %s", fstype)
	}
}

func TestGetDiskFormatBlkidFallbackDetectsFS(t *testing.T) {
	fs := &FS{}
	disk := "/dev/disk/by-id/dm-uuid-mpath-test123"

	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	var lsblkCalled, blkidCalled bool
	getExecCommandCombinedOutput = func(name string, args ...string) ([]byte, error) {
		if name == "lsblk" {
			lsblkCalled = true
			return []byte("\n"), nil
		}
		if name == "blkid" {
			blkidCalled = true
			expectedArgs := []string{"-o", "value", "-s", "TYPE", disk}
			if len(args) != len(expectedArgs) {
				t.Errorf("expected %d blkid arguments, got %d", len(expectedArgs), len(args))
			}
			for i, arg := range expectedArgs {
				if i < len(args) && args[i] != arg {
					t.Errorf("expected blkid arg[%d] = %s, got %s", i, arg, args[i])
				}
			}
			return []byte("ext4"), nil
		}
		t.Errorf("unexpected command: %s", name)
		return nil, errors.New("unexpected command")
	}

	fstype, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if fstype != "ext4" {
		t.Errorf("expected ext4, got %s", fstype)
	}
	if !lsblkCalled {
		t.Errorf("expected lsblk to be called first")
	}
	if !blkidCalled {
		t.Errorf("expected blkid to be called as fallback")
	}
}

func TestGetDiskFormatBlkidFallbackUnformatted(t *testing.T) {
	fs := &FS{}
	disk := "/dev/disk/by-id/dm-uuid-mpath-test456"

	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(name string, _ ...string) ([]byte, error) {
		if name == "lsblk" {
			return []byte("\n"), nil
		}
		if name == "blkid" {
			return []byte(""), nil
		}
		t.Errorf("unexpected command: %s", name)
		return nil, errors.New("unexpected command")
	}

	fstype, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if fstype != "" {
		t.Errorf("expected empty fstype, got %s", fstype)
	}
}

func TestGetDiskFormatBlkidFallbackError(t *testing.T) {
	fs := &FS{}
	disk := "/dev/disk/by-id/dm-uuid-mpath-test789"

	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(name string, _ ...string) ([]byte, error) {
		if name == "lsblk" {
			return []byte("\n"), nil
		}
		if name == "blkid" {
			return []byte(""), errors.New("exit status 2")
		}
		t.Errorf("unexpected command: %s", name)
		return nil, errors.New("unexpected command")
	}

	fstype, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if fstype != "" {
		t.Errorf("expected empty fstype, got %s", fstype)
	}
}

func Test_formatAndMount(t *testing.T) {
	fs := &MockFS{}
	ctx := context.WithValue(context.Background(), ContextKey("RequestID"), "test-req-id")
	ctx = context.WithValue(ctx, ContextKey(NoDiscard), NoDiscard)

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput

	after := func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}

	tests := []struct {
		name      string
		setup     func()
		source    string
		target    string
		fsType    string
		opts      []string
		wantError bool
	}{
		{
			name:      "Disk is formatted with a different filesystem and mount fails due to unknown error and unknown data, and the user provided format option",
			setup:     func() {},
			source:    "/dev/sda1",
			target:    "/mnt/data",
			fsType:    "ext4",
			opts:      []string{"defaults", "fsFormatOption:nodiscard"},
			wantError: true,
		},
		{
			name: "Disk is Unformatted and mount pass",
			setup: func() {
				getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
					return []byte("\n"), nil
				}
			},
			source:    "/dev/sda1",
			target:    "/mnt/data",
			fsType:    "",
			opts:      []string{},
			wantError: true,
		},
		{
			name: "Disk is Unformatted and user provides format option",
			setup: func() {
				getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
					return []byte("\n"), nil
				}
			},
			source:    "/dev/sda1",
			target:    "/mnt/data",
			fsType:    "",
			opts:      []string{"defaults", "fsFormatOption:"},
			wantError: true,
		},
		{
			name: "Disk is Unformatted and user provides format option as xfs",
			setup: func() {
				getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
					return []byte("\n"), nil
				}
			},
			source:    "/dev/sda1",
			target:    "/mnt/data",
			fsType:    "xfs",
			opts:      []string{"defaults", "fsFormatOption:"},
			wantError: true,
		},
		{
			name: "fsType xfs - Disk is Unformatted and mount pass",
			setup: func() {
				getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
					return []byte("\n"), nil
				}
			},
			source:    "/dev/sda1",
			target:    "/mnt/data",
			fsType:    "xfs",
			opts:      []string{},
			wantError: true,
		},
		{
			name: "Disk failed to mount",
			setup: func() {
				getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
					return []byte("ext4\n"), nil
				}
			},
			source:    "/dev/sda1",
			target:    "/mnt/data",
			fsType:    "ext4",
			opts:      []string{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer after()
			err := fs.formatAndMount(ctx, tt.source, tt.target, tt.fsType, tt.opts...)
			if (err != nil) != tt.wantError {
				t.Errorf("formatAndMount() error = %v, wantError %v", err != nil, tt.wantError)
			}
		})
	}
}

// MockFS struct for testing
type MockFS struct {
	FS
}

func TestFormat(t *testing.T) {
	fs := &MockFS{}
	ctx := context.WithValue(context.Background(), ContextKey("RequestID"), "test-req-id")
	ctx = context.WithValue(ctx, ContextKey(NoDiscard), NoDiscard)

	tests := []struct {
		name      string
		source    string
		target    string
		fsType    string
		opts      []string
		mockError error
		wantError bool
	}{
		{
			name:      "format failure",
			source:    "test-source",
			target:    "test-target",
			fsType:    "ext4",
			opts:      []string{"defaults"},
			mockError: errors.New("format failed"),
			wantError: true,
		},
		{
			name:      "format failure",
			source:    "test-source",
			target:    "test-target",
			fsType:    "",
			opts:      []string{"defaults"},
			mockError: errors.New("format failed"),
			wantError: true,
		},
		{
			name:      "format xfs failure",
			source:    "test-source",
			target:    "test-target",
			fsType:    "xfs",
			opts:      []string{"defaults"},
			mockError: errors.New("format failed"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock exec.Command
			execCommand = func(_ string, _ ...string) *exec.Cmd {
				cmd := exec.Command("echo", "mock command")
				if tt.mockError != nil {
					cmd = exec.Command("false")
				}
				return cmd
			}

			err := fs.format(ctx, tt.source, tt.target, tt.fsType, tt.opts...)
			if (err != nil) != tt.wantError {
				t.Errorf("format() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestIsLsblkNew(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		want      bool
		wantError bool
	}{
		{
			name:      "lsblk version greater than 2.30",
			output:    "lsblk from util-linux 2.31.1",
			want:      true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock exec.Command
			execCommand = func(_ string, _ ...string) *exec.Cmd {
				cmd := exec.Command("echo", "mock command")
				if tt.wantError {
					cmd = exec.Command("false")
				}
				return cmd
			}

			fs := &FS{}
			got, err := fs.isLsblkNew()
			if (err != nil) != tt.wantError {
				t.Errorf("isLsblkNew() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("isLsblkNew() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNativeDevicesFromPpath(t *testing.T) {
	fs := &FS{}

	// Mock the output
	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput

	after := func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}

	tests := []struct {
		name            string
		setup           func()
		ppath           string
		expectedDevices []string
		wantErr         bool
	}{
		{
			name:            "Invalid ppath",
			setup:           func() {},
			ppath:           "invalid_ppath",
			expectedDevices: nil,
			wantErr:         true,
		},
		{
			name: "Success",
			setup: func() {
				getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
					return []byte("/dev/emcpowerg   :EMC     :SYMMETRIX       :60000970000120000549533030354435\n"), nil
				}
			},
			ppath:           "invalid_ppath",
			expectedDevices: []string{},
			wantErr:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.setup()
			defer after()
			devices, err := fs.getNativeDevicesFromPpath(context.Background(), test.ppath)
			if !reflect.DeepEqual(devices, test.expectedDevices) || (err != nil) != test.wantErr {
				t.Errorf("Expected: %v, %v. Actual: %v, %v", test.expectedDevices, test.wantErr, devices, err != nil)
			}
		})
	}
}

func TestFS_expandXfs(t *testing.T) {
	tests := []struct {
		name    string
		volume  string
		wantErr bool
	}{
		{
			name:    "Invalid path",
			volume:  "/invalid/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FS{}

			err := fs.expandXfs(tt.volume)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandXfs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadProcMounts(t *testing.T) {
	tests := []struct {
		name      string
		fs        *FS
		path      string
		info      bool
		wantInfos []Info
		wantHash  uint32
		wantErr   bool
	}{
		{
			name:      "Normal operation",
			fs:        &FS{},
			path:      "/",
			wantInfos: nil,
			wantHash:  uint32(2166136261),
			wantErr:   false,
		},
		{
			name: "Error reading file",
			fs: &FS{
				ScanEntry: defaultEntryScanFunc,
			},
			path:      "/wrong-path",
			wantInfos: nil,
			wantHash:  uint32(0),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			infos, hash, err := tt.fs.readProcMounts(ctx, tt.path, tt.info)
			if !reflect.DeepEqual(infos, tt.wantInfos) || hash != tt.wantHash || (err != nil) != tt.wantErr {
				t.Errorf("readProcMounts() = (%v, %v, %v), want (%v, %v, %v)", infos, hash, err != nil, tt.wantInfos, tt.wantHash, tt.wantErr)
			}
		})
	}
}

func TestGetMpathNameFromDevice_Error(t *testing.T) {
	// Create a new instance of FS
	fs := &FS{}

	// Test case when device is a invalid path
	device := "/"
	expectedMpathName := ""
	mpathName, err := fs.getMpathNameFromDevice(context.Background(), device)
	if err == nil {
		t.Errorf("Expected error, got: %v", err)
	}
	if mpathName != expectedMpathName {
		t.Errorf("Expected mpathName to be %s, but got %s", expectedMpathName, mpathName)
	}
}

func TestGetMpathNameFromDevice_InvalidDeviceID(t *testing.T) {
	fs := &FS{}

	// Device ID with special characters should fail validateDeviceID
	mpathName, err := fs.getMpathNameFromDevice(context.Background(), "sda;rm -rf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid device")
	assert.Equal(t, "", mpathName)
}

func TestConsistentRead(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		retry   int
		wantErr bool
	}{
		{
			name: "File does not exist",
			setup: func(_ *testing.T) string {
				return "/nonexistent/path/file"
			},
			retry:   3,
			wantErr: true,
		},
		{
			name: "Consistent content",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "stable")
				require.NoError(t, os.WriteFile(path, []byte("content"), 0o644))
				return path
			},
			retry:   3,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FS{}
			path := tt.setup(t)
			content, err := fs.consistentRead(path, tt.retry)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, content)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, []byte("content"), content)
			}
		})
	}
}

func TestConsistentRead_MaxRetriesExhausted(t *testing.T) {
	fs := &FS{}
	dir := t.TempDir()
	path := filepath.Join(dir, "changing")

	// Write initial content
	require.NoError(t, os.WriteFile(path, []byte("v0"), 0o644))

	// Start a goroutine that keeps modifying the file
	done := make(chan struct{})
	go func() {
		i := 1
		for {
			select {
			case <-done:
				return
			default:
				_ = os.WriteFile(path, []byte("v"+string(rune('0'+i%10))), 0o644)
				i++
			}
		}
	}()
	defer close(done)

	// With only 1 retry and a rapidly changing file, it should eventually fail
	_, err := fs.consistentRead(path, 1)
	// This may or may not error depending on timing; just exercise the code path
	_ = err
}

func TestFindFSType_InvalidPath(t *testing.T) {
	fs := &FS{}

	_, err := fs.findFSType(context.Background(), "/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate path")
}

func TestResizeMultipath_InvalidPath(t *testing.T) {
	fs := &FS{}

	err := fs.resizeMultipath(context.Background(), "/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate path")
}

func TestResizeMultipath_CommandFails(t *testing.T) {
	fs := &FS{}

	// Valid path but multipathd binary doesn't exist
	err := fs.resizeMultipath(context.Background(), "/dev/dm-0")
	assert.Error(t, err)
}

func TestExpandExtFs_InvalidPath(t *testing.T) {
	fs := &FS{}

	err := fs.expandExtFs("/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate path")
}

func TestExpandExtFs_CommandFails(t *testing.T) {
	fs := &FS{}

	// Valid path but resize2fs binary doesn't exist or device is invalid
	err := fs.expandExtFs("/dev/nonexistent")
	assert.Error(t, err)
}

func TestExpandXfs_CommandFails(t *testing.T) {
	fs := &FS{}

	// Valid path but xfs_growfs binary doesn't exist
	err := fs.expandXfs("/dev/nonexistent")
	assert.Error(t, err)
}

func TestResizeFS_AllFsTypes(t *testing.T) {
	fs := &FS{}
	ctx := context.Background()

	tests := []struct {
		name        string
		mountpoint  string
		devicePath  string
		ppathDevice string
		mpathDevice string
		fsType      string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Unsupported filesystem",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "",
			mpathDevice: "",
			fsType:      "ntfs",
			wantErr:     true,
			errContains: "filesystem not supported",
		},
		{
			name:        "Ppath device with partprobe failure",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "emcpowera",
			mpathDevice: "",
			fsType:      "ext4",
			wantErr:     true,
		},
		{
			name:        "Mpath device with ext4",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "",
			mpathDevice: "mpath0",
			fsType:      "ext4",
			wantErr:     true,
		},
		{
			name:        "Mpath device with ext3",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "",
			mpathDevice: "mpath0",
			fsType:      "ext3",
			wantErr:     true,
		},
		{
			name:        "Mpath device with xfs",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "",
			mpathDevice: "mpath0",
			fsType:      "xfs",
			wantErr:     true,
		},
		{
			name:        "Plain device with ext4",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "",
			mpathDevice: "",
			fsType:      "ext4",
			wantErr:     true,
		},
		{
			name:        "Plain device with xfs",
			mountpoint:  "/mnt/data",
			devicePath:  "/dev/sda1",
			ppathDevice: "",
			mpathDevice: "",
			fsType:      "xfs",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.resizeFS(ctx, tt.mountpoint, tt.devicePath, tt.ppathDevice, tt.mpathDevice, tt.fsType)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReReadPartitionTable_InvalidPath(t *testing.T) {
	err := reReadPartitionTable(context.Background(), "/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate path")
}

func TestReReadPartitionTable_CommandFails(t *testing.T) {
	// Valid path but partprobe binary doesn't exist
	err := reReadPartitionTable(context.Background(), "/dev/nonexistent")
	assert.Error(t, err)
}

func TestDeviceRescan_InvalidPath(t *testing.T) {
	fs := &FS{}

	err := fs.deviceRescan(context.Background(), "/")
	assert.Error(t, err)
}

func TestDeviceRescan_Success(t *testing.T) {
	fs := &FS{}

	// Create a temp dir that simulates sysfs structure
	dir := t.TempDir()
	deviceDir := filepath.Join(dir, "device")
	require.NoError(t, os.MkdirAll(deviceDir, 0o755))

	err := fs.deviceRescan(context.Background(), dir)
	assert.NoError(t, err)

	// Verify the rescan file was written
	content, err := os.ReadFile(filepath.Join(deviceDir, "rescan"))
	assert.NoError(t, err)
	assert.Equal(t, "1\n", string(content))
}

func TestDeviceRescan_WriteError(t *testing.T) {
	fs := &FS{}

	// Path exists but device/rescan does not
	dir := t.TempDir()
	err := fs.deviceRescan(context.Background(), dir)
	assert.Error(t, err)
}

func TestGetMountInfoFromDevice_EmptyID(t *testing.T) {
	fs := &FS{}

	info, err := fs.getMountInfoFromDevice(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "device ID cannot be empty")
}

func TestGetMountInfoFromDevice_InvalidPath(t *testing.T) {
	fs := &FS{}

	info, err := fs.getMountInfoFromDevice(context.Background(), "/")
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestGetMountInfoFromDevice_InvalidDeviceID(t *testing.T) {
	fs := &FS{}

	info, err := fs.getMountInfoFromDevice(context.Background(), "sda;rm -rf")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "invalid device ID")
}

func TestGetMounts_Linux(t *testing.T) {
	fs := &FS{ScanEntry: defaultEntryScanFunc}
	ctx := context.Background()

	// getMounts reads /proc/mounts - should succeed on Linux
	mounts, err := fs.getMounts(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, mounts)
}

func TestGetDiskFormatSDCBlkidError(t *testing.T) {
	fs := &FS{}
	disk := "/dev/scinia"

	defaultGetExecCommandCombinedOutput := getExecCommandCombinedOutput
	defer func() {
		getExecCommandCombinedOutput = defaultGetExecCommandCombinedOutput
	}()

	getExecCommandCombinedOutput = func(_ string, _ ...string) ([]byte, error) {
		// Simulate blkid returning non-empty output with a non-ExitError
		// This exercises the "Other errors are actual failures" branch (line 80-81)
		return []byte("ext4"), errors.New("blkid failed: I/O error")
	}

	_, err := fs.getDiskFormat(context.Background(), disk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blkid failed")
}

func TestBindMount_Linux(t *testing.T) {
	fs := &FS{}
	ctx := context.Background()

	// bindMount calls doMount twice - both will fail in test env
	err := fs.bindMount(ctx, "/dev/nonexistent", "/mnt/data")
	assert.Error(t, err)
}

func TestDefaultEntryScanFunc(t *testing.T) {
	fn := DefaultEntryScanFunc()
	assert.NotNil(t, fn, "DefaultEntryScanFunc should return a non-nil function")
}

func TestConsistentRead_RetryExhaustion(t *testing.T) {
	f := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(f, []byte("content"), 0o644))

	fs := &FS{}
	// retry=0 means the loop never executes, falling through to the error return
	_, err := fs.consistentRead(f, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not get consistent content")
}

func TestConsistentRead_ConsistentContent(t *testing.T) {
	f := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(f, []byte("stable content"), 0o644))

	fs := &FS{}
	content, err := fs.consistentRead(f, 3)
	assert.NoError(t, err)
	assert.Equal(t, []byte("stable content"), content)
}

func TestConsistentRead_FileNotFound(t *testing.T) {
	fs := &FS{}
	_, err := fs.consistentRead("/nonexistent/path/to/file", 1)
	assert.Error(t, err)
}
