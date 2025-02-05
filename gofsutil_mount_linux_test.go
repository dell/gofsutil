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
	"testing"
)

// Mocking exec.Command
var execCommand = exec.Command

// func TestGetDiskFormatValidPath(t *testing.T) {
// 	// Create a test FS
// 	fs := &FS{}

// 	// Create a test disk path
// 	disk := "/dev/sda1"

// 	// Mock the lsblk output
// 	output := "ext4\n"
// 	origCmd := execCommand
// 	execCommand = func(name string, args ...string) *exec.Cmd {
// 		if name == "lsblk" {
// 			cmd := &exec.Cmd{
// 				Path:   name,
// 				Args:   append([]string{name}, args...),
// 				Stdout: &bytes.Buffer{},
// 				Stderr: &bytes.Buffer{},
// 			}
// 			cmd.Stdout.(*bytes.Buffer).WriteString(output)
// 			return cmd
// 		}
// 		return origCmd(name, args...)
// 	}
// 	defer func() {
// 		execCommand = origCmd
// 	}()

// 	// Call getDiskFormat
// 	_, err := fs.getDiskFormat(context.Background(), disk)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

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

// func TestGetDiskFormatUnformattedDisk(t *testing.T) {
// 	// Create a test FS
// 	fs := &FS{}

// 	// Create a test disk path
// 	disk := "/dev/sda1"

// 	// Mock the lsblk output
// 	output := "\n"
// 	cmd := exec.Command("lsblk", "-n", "-o", "FSTYPE", disk)
// 	cmd.Stdout = ioutil.Discard
// 	cmd.Stderr = ioutil.Discard
// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	origCmd := exec.Command
// 	execCommand = func(name string, args ...string) *exec.Cmd {
// 		if name == "lsblk" {
// 			cmd := &exec.Cmd{
// 				Path:   name,
// 				Args:   append([]string{name}, args...),
// 				Stdout: &bytes.Buffer{},
// 				Stderr: &bytes.Buffer{},
// 			}
// 			cmd.Stdout.(*bytes.Buffer).WriteString(output)
// 			return cmd
// 		}
// 		return origCmd(name, args...)
// 	}
// 	defer func() {
// 		execCommand = origCmd
// 	}()
// 	// Call getDiskFormat
// 	_, err = fs.getDiskFormat(context.Background(), disk)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

// func TestFormatAndMountSuccess(t *testing.T) {
// 	fs := &FS{}
// 	ctx := context.Background()
// 	source := "test-source"
// 	target := "test-target"
// 	fsType := "ext4"
// 	opts := []string{"defaults"}

// 	err := fs.formatAndMount(ctx, source, target, fsType, opts...)
// 	if err != nil {
// 		t.Errorf("expected nil, got %v", err)
// 	}
// }

// func TestFormatAndMountNoFsFormatOption(t *testing.T) {
// 	fs := &FS{}
// 	ctx := context.Background()
// 	source := "test-source"
// 	target := "test-target"
// 	fsType := "ext4"
// 	opts := []string{"defaults"}

// 	err := fs.formatAndMount(ctx, source, target, fsType, opts...)
// 	if err != nil {
// 		t.Errorf("expected nil, got %v", err)
// 	}
// }

// func TestFormatAndMountWithFsFormatOption(t *testing.T) {
// 	fs := &FS{}
// 	ctx := context.Background()
// 	source := "test-source"
// 	target := "test-target"
// 	fsType := "ext4"
// 	opts := []string{"defaults", "fsFormatOption:-F"}

// 	err := fs.formatAndMount(ctx, source, target, fsType, opts...)
// 	if err != nil {
// 		t.Errorf("expected nil, got %v", err)
// 	}
// }

// func TestFormatAndMountNoDiscard(t *testing.T) {
// 	fs := &FS{}
// 	ctx := context.Background()
// 	source := "test-source"
// 	target := "test-target"
// 	fsType := "ext4"
// 	opts := []string{"defaults"}

// 	// Simulate NoDiscard option
// 	ctx = context.WithValue(ctx, ContextKey(NoDiscard), NoDiscard)

// 	err := fs.formatAndMount(ctx, source, target, fsType, opts...)
// 	if err != nil {
// 		t.Errorf("expected nil, got %v", err)
// 	}
// }

// MockFS struct for testing
type MockFS struct {
	FS
}

// func (fs *MockFS) validateMountArgs(source, target, fsType string, opts ...string) error {
// 	// Mock validation logic
// 	if source == "" || target == "" || fsType == "" {
// 		return errors.New("invalid arguments")
// 	}
// 	return nil
// }

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
		// {
		// 	name:      "successful format",
		// 	source:    "test-source",
		// 	target:    "test-target",
		// 	fsType:    "ext4",
		// 	opts:      []string{"defaults"},
		// 	mockError: nil,
		// 	wantError: false,
		// },
		{
			name:      "format failure",
			source:    "test-source",
			target:    "test-target",
			fsType:    "ext4",
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
		// {
		// 	name:      "lsblk version less than 2.30",
		// 	output:    "lsblk from util-linux 2.29.2",
		// 	want:      false,
		// 	wantError: false,
		// },
		// {
		// 	name:      "lsblk command error",
		// 	output:    "",
		// 	want:      false,
		// 	wantError: true,
		// },
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
