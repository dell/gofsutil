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
	"os/exec"
	"reflect"
	"testing"
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
