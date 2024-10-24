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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dell/gofsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			err := os.MkdirAll(filepath.Dir(deviceWwidFile), 0755)
			require.Nil(t, err)
			if strings.HasPrefix(tt.deviceName, "nvme") {
				err = os.WriteFile(deviceWwidFile, []byte(tt.nguid), 0644)
			} else {
				err = os.WriteFile(deviceWwidFile, []byte(tt.wwn), 0644)
			}
			require.Nil(t, err)

			// Call the function with the test input
			result, err := gofsutil.GetSysBlockDevicesForVolumeWWN(context.Background(), tt.wwn)
			assert.Nil(t, err)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestDeviceRescan(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		name           string
		deviceName     string
		rescanFilePath []string
		wantError      bool
	}{
		{
			name:           "nvme device",
			deviceName:     "nvme0n1",
			rescanFilePath: []string{"device", "rescan_controller"},
			wantError:      false,
		},
		{
			name:           "iscsi device",
			deviceName:     "sda1",
			rescanFilePath: []string{"device", "rescan"},
			wantError:      false,
		},
		{
			name:       "invalid device",
			deviceName: "invalid",
			wantError:  true,
		},
		{
			name:       "invalid path",
			deviceName: "/",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the necessary directories and files
			path := []string{tempDir, tt.deviceName}
			path = append(path, tt.rescanFilePath...)
			deviceRescanFile := filepath.Join(path...)
			devicePath := tt.deviceName
			if !strings.Contains(tt.name, "invalid") {
				devicePath = filepath.Join(tempDir, tt.deviceName)
				err := os.MkdirAll(filepath.Dir(deviceRescanFile), 0755)
				require.Nil(t, err)
				err = os.WriteFile(deviceRescanFile, nil, 0644)
				require.Nil(t, err)
			}

			// Call the function with the test input
			err := gofsutil.DeviceRescan(context.Background(), devicePath)
			if (err != nil) != tt.wantError {
				t.Errorf("DeviceRescan() error = %v, wantError %v", err, tt.wantError)
			} else if err == nil {
				data, err := os.ReadFile(deviceRescanFile)
				dataStr := strings.TrimSpace(string(data))
				require.Nil(t, err)
				if !strings.EqualFold(dataStr, "1") {
					t.Errorf("DeviceRescan() file data = %s, want 1", dataStr)
				}
			}
		})
	}
}
