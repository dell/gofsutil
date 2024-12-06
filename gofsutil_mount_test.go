// Copyright © 2022 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package gofsutil_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"errors"
	"github.com/dell/gofsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"syscall"
)

func TestBindMount(t *testing.T) {
	src, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	tgt, err := os.MkdirTemp("", "")
	if err != nil {
		os.RemoveAll(src)
		t.Fatal(err)
	}
	if err := gofsutil.EvalSymlinks(context.TODO(), &src); err != nil {
		os.RemoveAll(tgt)
		os.RemoveAll(src)
		t.Fatal(err)
	}
	if err := gofsutil.EvalSymlinks(context.TODO(), &tgt); err != nil {
		os.RemoveAll(tgt)
		os.RemoveAll(src)
		t.Fatal(err)
	}
	defer func() {
		gofsutil.Unmount(context.TODO(), tgt)
		os.RemoveAll(tgt)
		os.RemoveAll(src)
	}()
	if err := gofsutil.BindMount(context.TODO(), src, tgt); err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	t.Logf("bind mount success: source=%s, target=%s", src, tgt)
	mounts, err := gofsutil.GetMounts(context.TODO())
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	success := false
	for _, m := range mounts {
		if m.Source == src && m.Path == tgt {
			success = true
		}
		t.Logf("%+v", m)
	}
	if !success {
		t.Errorf("unable to find bind mount: src=%s, tgt=%s", src, tgt)
		t.Fail()
	}
}

func TestGetMounts(t *testing.T) {
	mounts, err := gofsutil.GetMounts(context.TODO())
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	for _, m := range mounts {
		t.Logf("%+v", m)
	}
}

func TestGetSysBlockDevicesForVolumeWWN(t *testing.T) {
	tempDir := t.TempDir()
	gofsutil.UseMockSysBlockDir(tempDir)

	tests := []struct {
		name           string
		wwn            string
		nguid          string
		deviceName     string
		deviceWwidPath []string
		expect         []string
		errString      string
	}{
		{
			name:           "iscsi block device",
			wwn:            "example-volume-wwn",
			deviceName:     "sdx",
			deviceWwidPath: []string{"device", "wwid"},
			expect:         []string{"sdx"},
			errString:      "",
		},
		{
			name:           "PowerStore nvme block device",
			wwn:            "naa.68ccf098001111a2222b3d4444a1b23c",
			nguid:          "eui.1111a2222b3d44448ccf096800a1b23c",
			deviceName:     "nvme0n1",
			deviceWwidPath: []string{"wwid"},
			expect:         []string{"nvme0n1"},
			errString:      "",
		},
		{
			name:           "PowerMax nvme block device",
			wwn:            "naa.60000970000120001263533030313434",
			nguid:          "eui.12635330303134340000976000012000",
			deviceName:     "nvme0n2",
			deviceWwidPath: []string{"wwid"},
			expect:         []string{"nvme0n2"},
			errString:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the necessary directories and files
			path := []string{tempDir, tt.deviceName}
			path = append(path, tt.deviceWwidPath...)
			deviceWwidFile := filepath.Join(path...)
			err := os.MkdirAll(filepath.Dir(deviceWwidFile), 0o755)
			require.Nil(t, err)
			if strings.HasPrefix(tt.deviceName, "nvme") {
				err = os.WriteFile(deviceWwidFile, []byte(tt.nguid), 0o600)
			} else {
				err = os.WriteFile(deviceWwidFile, []byte(tt.wwn), 0o600)
			}
			require.Nil(t, err)

			// Call the function with the test input
			result, err := gofsutil.GetSysBlockDevicesForVolumeWWN(context.Background(), tt.wwn)
			assert.Nil(t, err)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestGetNVMeController(t *testing.T) {
	tempDir := t.TempDir()
	gofsutil.UseMockSysBlockDir(tempDir)

	tests := map[string]struct {
		device      string
		controller  string
		path        []string
		expectedErr error
	}{
		"device exists and is an NVMe controller": {
			device:      "nvme0n1",
			controller:  "nvme0",
			path:        []string{"virtual", "nvme-fabrics", "ctl", "nvme0", "nvme0n1"},
			expectedErr: nil,
		},
		"device exists but is not an NVMe controller": {
			device:      "nvme1n1",
			controller:  "",
			path:        []string{"virtual", "nvme-fabrics", "nvme-subsystem", "nvme-subsys0", "nvme1n1"},
			expectedErr: nil,
		},
		"device exists but NVMe controller not found": {
			device:      "nvme2n1",
			controller:  "",
			path:        []string{"virtual", "nvme-fabrics", "ctl", "nvme2n1"},
			expectedErr: fmt.Errorf("controller not found for device nvme2n1"),
		},
		"device does not exist": {
			device:      "nonexistent",
			controller:  "",
			expectedErr: fmt.Errorf("device %s does not exist", "nonexistent"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if name != "device does not exist" {
				// Create the necessary directories and files
				realPath := []string{tempDir}
				realPath = append(realPath, test.path...)
				err := os.MkdirAll(filepath.Join(realPath...), 0o755)
				require.NoError(t, err)

				sysBlockNVMeDeviceDir := filepath.Join(tempDir, test.device)
				err = os.Symlink(filepath.Join(realPath...), sysBlockNVMeDeviceDir)
				require.NoError(t, err)
			}

			// Call the function with the test input
			controller, err := gofsutil.GetNVMeController(test.device)
			if test.expectedErr != nil && err == nil {
				t.Errorf("getNVMeController() did not return error, expected %v", test.expectedErr)
			} else if test.expectedErr == nil && err != nil {
				t.Errorf("getNVMeController() returned error %v, expected no error", err)
			} else if err != nil && err.Error() != test.expectedErr.Error() {
				t.Errorf("getNVMeController() returned error %v, expected %v", err, test.expectedErr)
			}
			if controller != test.controller {
				t.Errorf("getNVMeController() = %v, expected %v", controller, test.controller)
			}
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
			_, err := gofsutil.GetDevMounts(tt.ctx, tt.dev)
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
			_, err := gofsutil.ValidateDevice(tt.ctx, tt.source)
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
			_, err := gofsutil.TargetIPLUNToDevicePath(tt.ctx, tt.targetIP, tt.lunID)
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
		{
			testname:  "Targets",
			targets:   []string{"iqn.2016-06.io.k8s", "iqn.2017-06.io.k8s","0x500000"},
			lun:       "",
			expectErr: &os.PathError{
				Op:   "open",                     // Operation that caused the error
				Path: "/sys/class/iscsi_session", // Path where the error occurred
				Err:  syscall.ENOENT,             // Error code (e.g., 0x2 corresponds to ENOENT - "No such file or directory")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			err := gofsutil.RescanSCSIHost(tt.ctx, tt.targets, tt.lun)
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
			err := gofsutil.RemoveBlockDevice(tt.ctx, tt.blockDevicePath)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestGetFCHostPortWWNs(t *testing.T) {
	expectErr := &os.PathError{
		Op:   "open",               // Operation that caused the error
		Path: "/sys/class/fc_host", // Path where the error occurred
		Err:  syscall.ENOENT,       // Error code (e.g., 0x2 corresponds to ENOENT - "No such file or directory")
	}
	_, err := gofsutil.GetFCHostPortWWNs(context.Background())
	assert.Equal(t, expectErr, err)
}

func TestIssueLIPToAllFCHosts(t *testing.T) {
	err := gofsutil.IssueLIPToAllFCHosts(context.Background())
	assert.Equal(t, nil, err)
}

