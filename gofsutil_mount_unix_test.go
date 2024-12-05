//go:build linux || darwin
// +build linux darwin

// Copyright © 2022-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	// "time"
)

func TestValidateMountArgs(t *testing.T) {
	tests := []struct {
		testname string
		source   string
		target   string
		fstype   string
		opts     []string
		expect   error
	}{
		{
			testname: "Invalid souce path",
			source:   "/",
			target:   "",
			fstype:   "",
			opts:     []string{"a", "b"},
			expect:   errors.New("Path: / is invalid"),
		},
		{
			testname: "Invalid target path",
			source:   "source",
			target:   "/",
			fstype:   "",
			opts:     []string{"a", "b"},
			expect:   errors.New("Path: / is invalid"),
		},
		{
			testname: "Invalid fstype",
			source:   "source",
			target:   "target",
			fstype:   "fstype",
			opts:     []string{"a", "b"},
			expect:   errors.New("FsType: fstype is invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			err := fs.validateMountArgs(tt.source, tt.target, tt.fstype, tt.opts...)
			assert.Equal(t, tt.expect, err)
		})
	}
}

func TestDoMount(t *testing.T) {
	tests := []struct {
		testname string
		ctx      context.Context
		mntCmnd  string
		source   string
		target   string
		fstype   string
		opts     []string
		expect   error
	}{
		{
			testname: "Invalid mount args",
			mntCmnd:  "invalid",
			source:   "/",
			target:   "",
			fstype:   "",
			opts:     []string{"a", "b"},
			expect:   errors.New("Path: / is invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			err := fs.doMount(tt.ctx, tt.mntCmnd, tt.source, tt.target, tt.fstype, tt.opts...)
			assert.Equal(t, tt.expect, err)
		})
	}
}

func TestUnMount(t *testing.T) {
	tests := []struct {
		testname string
		ctx      context.Context
		target   string
		expect   error
	}{
		{
			testname: "Invalid path",
			target:   "/",
			expect:   errors.New("Path: / is invalid"),
		},
		{
			testname: "Invalid arguments",
			target:   "/abc",
			expect:   errors.New("unmount failed: no such file or directory\nunmounting arguments: /abc"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			err := fs.unmount(tt.ctx, tt.target)
			assert.Equal(t, tt.expect, err)
		})
	}
}

func TestGetDevMounts(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		dev       string
		expectErr error
	}{
		{
			testname:  "Invalid dev",
			dev:       "abc",
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			_, err := fs.getDevMounts(tt.ctx, tt.dev)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestValidateDevice(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		source    string
		expectErr error
	}{
		{
			testname:  "Invalid dev",
			source:    "/dev",
			expectErr: errors.New("invalid device: /dev"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			_, err := fs.validateDevice(tt.ctx, tt.source)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestTargetIPLUNToDevicePath(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		targetIP  string
		lunID     int
		expectErr error
	}{
		{
			testname:  "Invalid lunid",
			targetIP:  "10.0.0.100",
			lunID:     1234,
			expectErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			_, err := fs.targetIPLUNToDevicePath(tt.ctx, tt.targetIP, tt.lunID)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestRescanSCSIHost(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		targets   []string
		lun       string
		expectErr error
	}{
		{
			testname:  "Invalid targets",
			targets:   []string{"a", "b"},
			lun:       "1234",
			expectErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			err := fs.rescanSCSIHost(tt.ctx, tt.targets, tt.lun)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestGetFCTargetHosts(t *testing.T) {
	tests := []struct {
		testname  string
		targets   []string
		expectErr error
	}{
		{
			testname:  "Invalid target hosts",
			targets:   []string{"a", "b"},
			expectErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			_, err := getFCTargetHosts(tt.targets)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestGetIscsiTargetHosts(t *testing.T) {
	tests := []struct {
		testname  string
		targets   []string
		expectErr error
	}{
		{
			testname: "Invalid target hosts",
			targets:  []string{"a", "b"},
			expectErr: &os.PathError{
				Op:   "open",                     // Operation that caused the error
				Path: "/sys/class/iscsi_session", // Path where the error occurred
				Err:  syscall.ENOENT,             // Error code (e.g., 0x2 corresponds to ENOENT - "No such file or directory")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			_, err := getIscsiTargetHosts(tt.targets)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestRemoveBlockDevice(t *testing.T) {
	tests := []struct {
		testname        string
		ctx             context.Context
		blockDevicePath string
		expectErr       error
	}{
		{
			testname:        "Invalid Block device path",
			blockDevicePath: "/abc",
			expectErr:       errors.New("Cannot read /sys/block/abc/device/state: open /sys/block/abc/device/state: no such file or directory"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			err := fs.removeBlockDevice(tt.ctx, tt.blockDevicePath)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

// func TestMultipathCommand(t *testing.T) {

// 	tests := []struct {
// 		testname       string
// 		ctx			   context.Context
// 		timeoutSeconds time.Duration
// 		chroot 		   string
// 		arguments	   []string
// 		expectErr	   error
// 	}{
// 		{
// 			testname:       "Invalid Block device path",
// 			timeoutSeconds:	time.Duration(10),
// 			chroot:         "",
// 			arguments:		[]string{"-A", "-iR",},
// 			expectErr:		errors.New("Ca"),
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.testname, func(t *testing.T) {
// 			fs := FS{SysBlockDir: "string"}
// 			_,err := fs.multipathCommand(tt.ctx, tt.timeoutSeconds, tt.chroot, tt.arguments)
// 			assert.Equal(t, tt.expectErr, err)
// 		})
// 	}
// }

func TestGetFCHostPortWWNs(t *testing.T) {
	fs := FS{SysBlockDir: "string"}
	expectErr := &os.PathError{
		Op:   "open",               // Operation that caused the error
		Path: "/sys/class/fc_host", // Path where the error occurred
		Err:  syscall.ENOENT,       // Error code (e.g., 0x2 corresponds to ENOENT - "No such file or directory")
	}
	_, err := fs.getFCHostPortWWNs(context.Background())
	assert.Equal(t, expectErr, err)
}

func TestIssueLIPToAllFCHosts(t *testing.T) {
	fs := FS{SysBlockDir: "string"}
	err := fs.issueLIPToAllFCHosts(context.Background())
	assert.Equal(t, nil, err)
}
