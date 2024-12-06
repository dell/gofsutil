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
		{
			testname: "Valid mount command",
			mntCmnd:  "mount",
			source:   "dev",
			target:   "usr",
			fstype:   "ext4",
			opts:     []string{"key=value", "variable"},
			expect:   errors.New("mount failed: exit status 32\nmounting arguments: -t ext4 -o key=value,variable dev usr\noutput: mount: usr: mount point does not exist.\n"),
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
		{
			testname:  "Target hosts",
			targets:   []string{"iqn.2016-06.io.k8s", "iqn.2017-06.io.k8s", "0x500000"},
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

func TestIsBind(t *testing.T) {
	tests := []struct {
		testname string
		ctx      context.Context
		opts     []string
		expect   bool
	}{
		{
			testname: "Opts",
			opts:     []string{"a", "bind", "remount"},
			expect:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{SysBlockDir: "string"}
			_, err := fs.isBind(tt.ctx, tt.opts...)
			assert.Equal(t, tt.expect, err)
		})
	}
}
