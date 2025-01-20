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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFsInfo(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		path      string
		induceErr bool
		expected  struct {
			available  int64
			capacity   int64
			usage      int64
			inodes     int64
			inodesFree int64
			inodesUsed int64
			err        error
		}
	}{
		{
			testname:  "Normal operation",
			path:      "/path",
			induceErr: false,
			expected: struct {
				available  int64
				capacity   int64
				usage      int64
				inodes     int64
				inodesFree int64
				inodesUsed int64
				err        error
			}{
				available:  1000,
				capacity:   2000,
				usage:      1000,
				inodes:     4,
				inodesFree: 2,
				inodesUsed: 2,
				err:        nil,
			},
		},
		{
			testname:  "Induced error",
			path:      "/path",
			induceErr: true,
			expected: struct {
				available  int64
				capacity   int64
				usage      int64
				inodes     int64
				inodesFree int64
				inodesUsed int64
				err        error
			}{
				available:  0,
				capacity:   0,
				usage:      0,
				inodes:     0,
				inodesFree: 0,
				inodesUsed: 0,
				err:        errors.New("filesystemInfo induced error: Failed to get filesystem stats"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceFilesystemInfoError = tt.induceErr
			available, capacity, usage, inodes, inodesFree, inodesUsed, err := fs.fsInfo(tt.ctx, tt.path)

			assert.Equal(t, tt.expected.available, available)
			assert.Equal(t, tt.expected.capacity, capacity)
			assert.Equal(t, tt.expected.usage, usage)
			assert.Equal(t, tt.expected.inodes, inodes)
			assert.Equal(t, tt.expected.inodesFree, inodesFree)
			assert.Equal(t, tt.expected.inodesUsed, inodesUsed)
			assert.Equal(t, tt.expected.err, err)
		})
	}
}

func TestDeviceRescan(t *testing.T) {
	tests := []struct {
		testname   string
		ctx        context.Context
		devicePath string
		induceErr  bool
		expected   struct {
			err error
		}
	}{
		{
			testname:   "Normal operation",
			devicePath: "/dev/sda",
			induceErr:  false,
			expected: struct {
				err error
			}{
				err: nil,
			},
		},
		{
			testname:   "Induced error",
			devicePath: "/dev/sda",
			induceErr:  true,
			expected: struct {
				err error
			}{
				err: errors.New("DeviceRescan induced error: Failed to rescan device"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceDeviceRescanError = tt.induceErr
			err := fs.DeviceRescan(tt.ctx, tt.devicePath)

			assert.Equal(t, tt.expected.err, err)
		})
	}
}

func TestFSGetMounts(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		induceErr bool
		expected  struct {
			mounts []Info
			err    error
		}
	}{
		{
			testname:  "Normal operation",
			induceErr: false,
			expected: struct {
				mounts []Info
				err    error
			}{
				mounts: nil,
				err:    nil,
			},
		},
		{
			testname:  "Induced error",
			induceErr: true,
			expected: struct {
				mounts []Info
				err    error
			}{
				mounts: nil,
				err:    errors.New("getMounts induced error"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceGetMountsError = tt.induceErr
			mounts, err := fs.GetMounts(tt.ctx)

			assert.Equal(t, tt.expected.err, err)
			assert.Equal(t, tt.expected.mounts, mounts)
		})
	}
}
