/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMount(t *testing.T) {
	mounts, err := GetMounts(context.TODO())
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	for _, m := range mounts {
		t.Logf("%+v", m)
	}
}

func TestGetSysBlockDevicesForVolumeWWNs(t *testing.T) {
	tempDir := t.TempDir()
	sysBlockDir = tempDir

	// Ensure the directory is cleaned up after the test
	defer func() {
		require.NoError(t, os.RemoveAll(sysBlockDir))
		sysBlockDir = "/sys/block"
	}()

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
			result, err := GetSysBlockDevicesForVolumeWWN(context.Background(), tt.wwn)
			assert.Nil(t, err)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestGetNVMeController(t *testing.T) {
	tempDir := t.TempDir()
	sysBlockDir = tempDir

	// Ensure the directory is cleaned up after the test
	defer func() {
		require.NoError(t, os.RemoveAll(sysBlockDir))
		sysBlockDir = "/sys/block"
	}()

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
			controller, err := GetNVMeController(test.device)
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
