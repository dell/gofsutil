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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

// Mocking exec.Command
var execCommand = exec.Command

func TestGetDiskFormatValidPath(t *testing.T) {
	// Create a test FS
	fs := &FS{}

	// Create a test disk path
	disk := "/dev/sda1"

	// Mock the lsblk output
	output := "ext4\n"
	origCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "lsblk" {
			cmd := &exec.Cmd{
				Path:   name,
				Args:   append([]string{name}, args...),
				Stdout: &bytes.Buffer{},
				Stderr: &bytes.Buffer{},
			}
			cmd.Stdout.(*bytes.Buffer).WriteString(output)
			return cmd
		}
		return origCmd(name, args...)
	}
	defer func() {
		execCommand = origCmd
	}()

	// Call getDiskFormat
	_, err := fs.getDiskFormat(context.Background(), disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

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

var (
	ContextKeyRequestID = ContextKey("RequestID")
)

// TestFS is a struct used for testing purposes
type TestFS struct {
	mock.Mock
}

func (fs *TestFS) validateMountArgs(source, target, fsType string, opts ...string) error {
	args := fs.Called(source, target, fsType, opts)
	return args.Error(0)
}

func (fs *TestFS) formatAndMount(ctx context.Context, source, target, fsType string, opts ...string) error {
	err := fs.validateMountArgs(source, target, fsType, opts...)
	if err != nil {
		return err
	}

	reqID := ctx.Value(ContextKeyRequestID)
	noDiscard := ctx.Value(ContextKey(NoDiscard))

	opts = append(opts, "defaults")
	f := logrus.Fields{
		"reqID":   reqID,
		"source":  source,
		"target":  target,
		"fsType":  fsType,
		"options": opts,
	}

	// Disk is unformatted so format it.
	args := []string{source}
	// Use 'ext4' as the default
	if len(fsType) == 0 {
		fsType = "ext4"
	}

	if fsType == "ext4" || fsType == "ext3" {
		args = []string{"-F", source}
		if noDiscard == NoDiscard {
			// -E nodiscard option to improve mkfs times
			args = []string{"-F", "-E", "nodiscard", source}
		}
	}

	if fsType == "xfs" && noDiscard == NoDiscard {
		// -K option (nodiscard) to improve mkfs times
		args = []string{"-K", source}
	}

	f["fsType"] = fsType
	logrus.WithFields(f).Info("disk appears unformatted, attempting format")

	mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)
	logrus.Printf("formatting with command: %s %v", mkfsCmd, args)
	/* #nosec G204 */
	err = exec.Command(mkfsCmd, args...).Run()
	if err != nil {
		logrus.WithFields(f).WithError(err).Error("format of disk failed")
		return err
	}

	// Attempt to mount the disk
	mountArgs := append([]string{"-t", fsType}, opts...)
	mountArgs = append(mountArgs, source, target)
	logrus.WithFields(f).Info("attempting to mount disk")
	logrus.Printf("mount command: mount %v", mountArgs)
	err = exec.Command("mount", mountArgs...).Run()
	if err != nil {
		logrus.WithFields(f).WithError(err).Error("mount Failed")
		return err
	}

	return nil
}

func TestTestFS_formatAndMount(t *testing.T) {
	type fields struct {
		ScanEntry   EntryScanFunc
		SysBlockDir string
	}
	type args struct {
		ctx    context.Context
		source string
		target string
		fsType string
		opts   []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// {
		// 	name: "successful format and mount",
		// 	fields: fields{
		// 		ScanEntry:   nil,
		// 		SysBlockDir: "/sys/block",
		// 	},
		// 	args: args{
		// 		ctx:    context.Background(),
		// 		source: "/dev/sda1",
		// 		target: "/mnt/data",
		// 		fsType: "ext4",
		// 		opts:   []string{"-o", "defaults"},
		// 	},
		// 	wantErr: false,
		// },
		{
			name: "validation error",
			fields: fields{
				ScanEntry:   nil,
				SysBlockDir: "/sys/block",
			},
			args: args{
				ctx:    context.Background(),
				source: "",
				target: "/mnt/data",
				fsType: "ext4",
				opts:   []string{"-o", "defaults"},
			},
			wantErr: true,
		},
		{
			name: "execution error",
			fields: fields{
				ScanEntry:   nil,
				SysBlockDir: "/sys/block",
			},
			args: args{
				ctx:    context.Background(),
				source: "/dev/sda1",
				target: "/mnt/data",
				fsType: "ext4",
				opts:   []string{"-o", "defaults"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the mount point directory if it doesn't exist
			if tt.args.target != "" {
				err := os.MkdirAll(tt.args.target, 0755)
				if err != nil {
					t.Fatalf("failed to create mount point: %v", err)
				}
				defer os.RemoveAll(tt.args.target) // Clean up after the test
			}

			fs := new(TestFS)
			if tt.wantErr {
				fs.On("validateMountArgs", tt.args.source, tt.args.target, tt.args.fsType, tt.args.opts).Return(errors.New("validation error"))
			} else {
				fs.On("validateMountArgs", tt.args.source, tt.args.target, tt.args.fsType, tt.args.opts).Return(nil)
			}

			// Mock exec.Command
			execCommand = func(name string, arg ...string) *exec.Cmd {
				cmd := exec.Command("echo")
				if tt.name == "execution error" {
					cmd = exec.Command("false")
				}
				return cmd
			}

			ctx := context.WithValue(context.Background(), ContextKeyRequestID, "12345")
			ctx = context.WithValue(ctx, ContextKey(NoDiscard), NoDiscard)

			err := fs.formatAndMount(ctx, tt.args.source, tt.args.target, tt.args.fsType, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestFS.formatAndMount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
