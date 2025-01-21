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
	"fmt"
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
			available, capacity, usage, inodes, inodesFree, inodesUsed, err := fs.FsInfo(tt.ctx, tt.path)

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

func TestFSDeviceRescan(t *testing.T) {
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
				mounts: []Info{
					{
						Path: "/mnt/volume1",
						Type: "ext4",
						Opts: []string{"rw", "relatime"},
					},
					{
						Path: "/mnt/volume2",
						Type: "xfs",
						Opts: []string{"rw", "noexec"},
					},
				},
				err: nil,
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
			GOFSMockMounts = []Info{
				{
					Path: "/mnt/volume1",
					Type: "ext4",
					Opts: []string{"rw", "relatime"},
				},
				{
					Path: "/mnt/volume2",
					Type: "xfs",
					Opts: []string{"rw", "noexec"},
				},
			}

			mounts, err := fs.GetMounts(tt.ctx)

			assert.Equal(t, tt.expected.err, err)
			assert.Equal(t, tt.expected.mounts, mounts)
		})
	}
}

func TestFSRemoveBlockDevice(t *testing.T) {
	tests := []struct {
		testname        string
		ctx             context.Context
		blockDevicePath string
		induceErr       bool
		expectedErr     error
	}{
		{
			testname:        "Normal operation",
			blockDevicePath: "/dev/sda1",
			induceErr:       false,
			expectedErr:     nil,
		},
		{
			testname:        "Induced error",
			blockDevicePath: "/dev/sda1",
			induceErr:       true,
			expectedErr:     errors.New("remove block device induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceRemoveBlockDeviceError = tt.induceErr
			err := fs.RemoveBlockDevice(tt.ctx, tt.blockDevicePath)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestFSWWNToDevicePath(t *testing.T) {
	tests := []struct {
		testname     string
		ctx          context.Context
		wwn          string
		induceErr    bool
		expectedDev  string
		expectedPath string
		expectedErr  error
	}{
		{
			testname:     "Normal operation",
			wwn:          "wwn-0x5000c500a0b1c2d3",
			induceErr:    false,
			expectedDev:  "/dev/sda/wwn-0x5000c500a0b1c2d3",
			expectedPath: "/dev/sda",
			expectedErr:  nil,
		},
		{
			testname:     "Induced error",
			wwn:          "wwn-0x5000c500a0b1c2d3",
			induceErr:    true,
			expectedDev:  "",
			expectedPath: "",
			expectedErr:  errors.New("induced error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceWWNToDevicePathError = tt.induceErr
			GOFSMockWWNToDevice = map[string]string{
				"wwn-0x5000c500a0b1c2d3": "/dev/sda",
			}
			GOFSWWNPath = "/dev/sda/"
			dev, path, err := fs.WWNToDevicePath(tt.ctx, tt.wwn)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedDev, dev)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestFSGetDiskFormat(t *testing.T) {
	tests := []struct {
		testname     string
		ctx          context.Context
		disk         string
		induceErr    bool
		induceType   string
		expectedType string
		expectedErr  error
	}{
		{
			testname:     "Normal operation",
			disk:         "/dev/sda",
			induceErr:    false,
			induceType:   "",
			expectedType: "ext4",
			expectedErr:  nil,
		},
		{
			testname:     "Induced error",
			disk:         "/dev/sda",
			induceErr:    true,
			induceType:   "",
			expectedType: "",
			expectedErr:  errors.New("getDiskFormat induced error"),
		},
		{
			testname:     "Induced type",
			disk:         "/dev/sda",
			induceErr:    false,
			induceType:   "xfs",
			expectedType: "xfs",
			expectedErr:  nil,
		},
		{
			testname:     "Disk not found",
			disk:         "/dev/sdb",
			induceErr:    false,
			induceType:   "",
			expectedType: "",
			expectedErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceGetDiskFormatError = tt.induceErr
			GOFSMock.InduceGetDiskFormatType = tt.induceType
			GOFSMockMounts = []Info{
				{
					Device: "/dev/sda",
					Type:   "ext4",
				},
			}

			format, err := fs.GetDiskFormat(tt.ctx, tt.disk)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedType, format)
		})
	}
}

func TestFSMount(t *testing.T) {
	tests := []struct {
		testname       string
		ctx            context.Context
		source         string
		target         string
		fsType         string
		options        []string
		induceErr      bool
		expectedErr    error
		expectedMounts []Info
	}{
		{
			testname:    "Normal operation",
			source:      "/dev/sda1",
			target:      "/mnt/volume1",
			fsType:      "ext4",
			options:     []string{"rw", "relatime"},
			induceErr:   false,
			expectedErr: nil,
			expectedMounts: []Info{
				{
					Device: "/dev/sda1",
					Path:   "/mnt/volume1",
					Opts:   []string{"rw", "relatime"},
				},
			},
		},
		{
			testname:       "Induced error",
			source:         "/dev/sda1",
			target:         "/mnt/volume1",
			fsType:         "ext4",
			options:        []string{"rw", "relatime"},
			induceErr:      true,
			expectedErr:    errors.New("mount induced error"),
			expectedMounts: []Info{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceMountError = tt.induceErr
			GOFSMockMounts = []Info{}

			err := fs.Mount(tt.ctx, tt.source, tt.target, tt.fsType, tt.options...)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedMounts, GOFSMockMounts)
		})
	}
}

func TestFSUnmount(t *testing.T) {
	tests := []struct {
		testname       string
		ctx            context.Context
		target         string
		induceErr      bool
		expectedErr    error
		expectedMounts []Info
	}{
		{
			testname:    "Normal operation",
			target:      "/mnt/volume1",
			induceErr:   false,
			expectedErr: nil,
			expectedMounts: []Info{
				{
					Device: "/dev/sda2",
					Path:   "/mnt/volume2",
					Opts:   []string{"rw", "noexec"},
				},
			},
		},
		{
			testname:    "Induced error",
			target:      "/mnt/volume1",
			induceErr:   true,
			expectedErr: errors.New("unmount induced error"),
			expectedMounts: []Info{
				{
					Device: "/dev/sda1",
					Path:   "/mnt/volume1",
					Opts:   []string{"rw", "relatime"},
				},
				{
					Device: "/dev/sda2",
					Path:   "/mnt/volume2",
					Opts:   []string{"rw", "noexec"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceUnmountError = tt.induceErr
			GOFSMockMounts = []Info{
				{
					Device: "/dev/sda1",
					Path:   "/mnt/volume1",
					Opts:   []string{"rw", "relatime"},
				},
				{
					Device: "/dev/sda2",
					Path:   "/mnt/volume2",
					Opts:   []string{"rw", "noexec"},
				},
			}

			err := fs.unmount(tt.ctx, tt.target)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedMounts, GOFSMockMounts)
		})
	}
}

func TestFSFindFSType(t *testing.T) {
	tests := []struct {
		testname     string
		ctx          context.Context
		mountpoint   string
		induceErr    bool
		expectedType string
		expectedErr  error
	}{
		{
			testname:     "Normal operation",
			mountpoint:   "/mnt/volume1",
			induceErr:    false,
			expectedType: "xfs",
			expectedErr:  nil,
		},
		{
			testname:     "Induced error",
			mountpoint:   "/mnt/volume1",
			induceErr:    true,
			expectedType: "",
			expectedErr:  errors.New("getMounts induced error: Failed to fetch filesystem as no mount info"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceFSTypeError = tt.induceErr

			fsType, err := fs.FindFSType(tt.ctx, tt.mountpoint)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedType, fsType)
		})
	}
}

func TestFSGetMountInfoFromDevice(t *testing.T) {
	tests := []struct {
		testname     string
		ctx          context.Context
		devID        string
		induceErr    bool
		expectedInfo *DeviceMountInfo
		expectedErr  error
	}{
		{
			testname:  "Normal operation",
			devID:     "sda",
			induceErr: false,
			expectedInfo: &DeviceMountInfo{
				DeviceNames: []string{"sda", "sdb"},
				MPathName:   "mpathb",
				MountPoint:  "/noderoot/var/lib/kubelet/pods/abc-123/volumes/k8.io/pmax-0123/mount",
			},
			expectedErr: nil,
		},
		{
			testname:     "Induced error",
			devID:        "sda",
			induceErr:    true,
			expectedInfo: nil,
			expectedErr:  errors.New("getMounts induced error: Failed to find mount information"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceGetMountInfoFromDeviceError = tt.induceErr

			info, err := fs.GetMountInfoFromDevice(tt.ctx, tt.devID)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedInfo, info)
		})
	}
}

func TestFSGetNVMeController(t *testing.T) {
	tests := []struct {
		testname           string
		device             string
		induceErr          bool
		expectedController string
		expectedErr        error
	}{
		{
			testname:           "Normal operation",
			device:             "nvme0n1",
			induceErr:          false,
			expectedController: "controller0",
			expectedErr:        nil,
		},
		{
			testname:           "Induced error",
			device:             "nvme0n1",
			induceErr:          true,
			expectedController: "",
			expectedErr:        errors.New("induced error"),
		},
		{
			testname:           "Device does not exist",
			device:             "nvme1n1",
			induceErr:          false,
			expectedController: "",
			expectedErr:        fmt.Errorf("device nvme1n1 does not exist"),
		},
		{
			testname:           "Controller not found",
			device:             "nvme0n2",
			induceErr:          false,
			expectedController: "",
			expectedErr:        fmt.Errorf("controller not found for device nvme0n2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceGetNVMeControllerError = tt.induceErr
			GONVMEValidDevices = map[string]bool{
				"nvme0n1": true,
				"nvme0n2": true, // Ensure the device exists
			}
			GONVMEDeviceToControllerMap = map[string]string{
				"nvme0n1": "controller0",
			}

			controller, err := fs.GetNVMeController(tt.device)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedController, controller)
		})
	}
}

func TestFSGetSysBlockDevicesForVolumeWWN(t *testing.T) {
	tests := []struct {
		testname        string
		ctx             context.Context
		volumeWWN       string
		induceErr       bool
		expectedDevices []string
		expectedErr     error
	}{
		{
			testname:        "Normal operation",
			volumeWWN:       "wwn-0x5000c500a0b1c2d3",
			induceErr:       false,
			expectedDevices: []string{"sda"},
			expectedErr:     nil,
		},
		{
			testname:        "Induced error",
			volumeWWN:       "wwn-0x5000c500a0b1c2d3",
			induceErr:       true,
			expectedDevices: []string{},
			expectedErr:     errors.New("induced error"),
		},
		{
			testname:        "Volume WWN not found",
			volumeWWN:       "wwn-0x5000c500a0b1c2d4",
			induceErr:       false,
			expectedDevices: []string{},
			expectedErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceGetSysBlockDevicesError = tt.induceErr
			GOFSMockWWNToDevice = map[string]string{
				"wwn-0x5000c500a0b1c2d3": "/dev/sda",
			}

			devices, err := fs.GetSysBlockDevicesForVolumeWWN(tt.ctx, tt.volumeWWN)

			assert.Equal(t, tt.expectedErr, err)
			assert.ElementsMatch(t, tt.expectedDevices, devices)
		})
	}
}

func TestFSTargetIPLUNToDevicePath(t *testing.T) {
	tests := []struct {
		testname       string
		ctx            context.Context
		targetIP       string
		lunID          int
		induceErr      bool
		expectedResult map[string]string
		expectedErr    error
	}{
		{
			testname:  "Normal operation",
			targetIP:  "192.xxx.1.1",
			lunID:     1,
			induceErr: false,
			expectedResult: map[string]string{
				"ip-192.xxx.1.1:-lun-1": "/dev/sdx",
			},
			expectedErr: nil,
		},
		{
			testname:       "Induced error",
			targetIP:       "192.xxx.1.1",
			lunID:          1,
			induceErr:      true,
			expectedResult: map[string]string{},
			expectedErr:    errors.New("induced error"),
		},
		{
			testname:       "Target IP and LUN not found",
			targetIP:       "192.xxx.1.2",
			lunID:          2,
			induceErr:      false,
			expectedResult: map[string]string{},
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceTargetIPLUNToDeviceError = tt.induceErr
			GOFSMockTargetIPLUNToDevice = map[string]string{
				"ip-192.xxx.1.1:-lun-1": "/dev/sdx",
			}

			result, err := fs.TargetIPLUNToDevicePath(tt.ctx, tt.targetIP, tt.lunID)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
