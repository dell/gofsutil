//go:build linux || darwin
// +build linux darwin

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

package gofsutil

import (
	"context"
	// "fmt"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// func TestFCRescanSCSIHost(t *testing.T) {
// 	var targets []string
// 	// Scan the remote ports to find the array port WWNs
// 	fcRemotePortsDir := "/sys/class/fc_remote_ports"
// 	remotePortEntries, err := os.ReadDir(fcRemotePortsDir)
// 	if err != nil {
// 		t.Errorf("error reading %s: %s", fcRemotePortsDir, err)
// 	}
// 	for _, remotePort := range remotePortEntries {
// 		if !strings.HasPrefix(remotePort.Name(), "rport-") {
// 			continue
// 		}

// 		if !strings.HasPrefix(remotePort.Name(), "rport-") {
// 			continue
// 		}

// 		arrayPortNameBytes, err := os.ReadFile(fcRemotePortsDir + "/" + remotePort.Name() + "/" + "port_name")
// 		if err != nil {
// 			continue
// 		}
// 		arrayPortName := strings.TrimSpace(string(arrayPortNameBytes))
// 		if !strings.HasPrefix(arrayPortName, gofsutil.FCPortPrefix) {
// 			continue
// 		}
// 		targets = append(targets, arrayPortName)

// 	}

// 	if len(targets) > 0 {
// 		err := gofsutil.RescanSCSIHost(context.Background(), targets, "1")
// 		if err != nil {
// 			t.Errorf("RescanSCSIHost failed: %s", err)
// 		}
// 	}
// }

func TestMountArgs(t *testing.T) {
	tests := []struct {
		src    string
		tgt    string
		fst    string
		opts   []string
		result string
	}{
		{
			src:    "localhost:/data",
			tgt:    "/mnt",
			fst:    "nfs",
			result: "-t nfs localhost:/data /mnt",
		},
		{
			src:    "localhost:/data",
			tgt:    "/mnt",
			result: "localhost:/data /mnt",
		},
		{
			src:    "localhost:/data",
			tgt:    "/mnt",
			fst:    "nfs",
			opts:   []string{"tcp", "vers=4"},
			result: "-t nfs -o tcp,vers=4 localhost:/data /mnt",
		},
		{
			src:    "/dev/disk/mydisk",
			tgt:    "/mnt/mydisk",
			fst:    "xfs",
			opts:   []string{"ro", "noatime", "ro"},
			result: "-t xfs -o ro,noatime /dev/disk/mydisk /mnt/mydisk",
		},
		{
			src:    "/dev/sdc",
			tgt:    "/mnt",
			opts:   []string{"rw", "", "noatime"},
			result: "-o rw,noatime /dev/sdc /mnt",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			opts := MakeMountArgs(
				context.TODO(), tt.src, tt.tgt, tt.fst, tt.opts...)
			optsStr := strings.Join(opts, " ")
			if optsStr != tt.result {
				t.Errorf("Formatting of mount args incorrect, got: %s want: %s",
					optsStr, tt.result)
			}
		})
	}
}

func TestWWNToDevicePath(t *testing.T) {
	tempDir := t.TempDir()
	multipathDevDiskByID = tempDir
	MultipathDevDiskByIDPrefix = filepath.Join(tempDir, "dm-uuid-mpath-3")
	fs := &FS{}

	tests := []struct {
		name            string
		wwn             string
		symlinkPath     string
		devicePath      string
		expectedSymlink string
		expectedDevice  string
	}{
		{
			name:            "Multipath device",
			wwn:             "36057097000019790004653302024d444",
			symlinkPath:     filepath.Join(tempDir, "dm-uuid-mpath-336057097000019790004653302024d444"),
			devicePath:      "/dev/mapper/mpatha",
			expectedSymlink: filepath.Join(tempDir, "dm-uuid-mpath-336057097000019790004653302024d444"),
			expectedDevice:  "/dev/mpatha",
		},
		{
			name:            "NVMe device",
			wwn:             "12636210324d0000300000000000f001",
			symlinkPath:     filepath.Join(tempDir, "nvme-eui.12636210324d0000300000000000f001"),
			devicePath:      "/dev/nvme0n1",
			expectedSymlink: filepath.Join(tempDir, "nvme-eui.12636210324d0000300000000000f001"),
			expectedDevice:  "/dev/nvme0n1",
		},
		{
			name:            "Normal device",
			wwn:             "60000970000120001263533030313434",
			symlinkPath:     filepath.Join(tempDir, "wwn-0x60000970000120001263533030313434"),
			devicePath:      "/dev/sda",
			expectedSymlink: filepath.Join(tempDir, "wwn-0x60000970000120001263533030313434"),
			expectedDevice:  "/dev/sda",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Creating mock symlink
			require.NoError(t, os.MkdirAll(filepath.Dir(tt.symlinkPath), 0o755))
			require.NoError(t, os.Symlink(tt.devicePath, tt.symlinkPath))

			// Call the function with the test input
			symlink, device, err := fs.WWNToDevicePath(context.Background(), tt.wwn)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSymlink, symlink)
			assert.Equal(t, tt.expectedDevice, device)
		})
	}
}

func TestTargetIPLUNToDevicePath(t *testing.T) {
	tempDir := t.TempDir()
	bypathdir = tempDir // Use the temporary directory for testing
	require.NoError(t, os.MkdirAll(bypathdir, 0o755))

	fs := &FS{}

	tests := []struct {
		name       string
		targetIP   string
		lunID      int
		entries    map[string]string
		expected   map[string]string
		shouldFail bool
	}{
		{
			name:     "Single entry",
			targetIP: "1.1.1.1",
			lunID:    0,
			entries: map[string]string{
				"ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-0": "../../sdc",
			},
			expected: map[string]string{
				filepath.Join(bypathdir, "ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-0"): "/dev/sdc",
			},
		},
		{
			name:     "Multiple entries",
			targetIP: "1.1.1.1",
			lunID:    1,
			entries: map[string]string{
				"ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-1":                  "../../sdd",
				"ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-0x0001000000000000": "../../sde",
			},
			expected: map[string]string{
				filepath.Join(bypathdir, "ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-1"):                  "/dev/sdd",
				filepath.Join(bypathdir, "ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-0x0001000000000000"): "/dev/sde",
			},
		},
		{
			name:       "No matching entries",
			targetIP:   "2.2.2.2",
			lunID:      0,
			entries:    map[string]string{},
			expected:   map[string]string{},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock symlinks
			createdEntries := []string{}
			for entry, target := range tt.entries {
				symlinkPath := filepath.Join(bypathdir, entry)
				require.NoError(t, os.MkdirAll(filepath.Dir(symlinkPath), 0o755))
				require.NoError(t, os.Symlink(target, symlinkPath))
				createdEntries = append(createdEntries, symlinkPath)
			}

			// Call the function with the test input
			result, err := fs.targetIPLUNToDevicePath(context.Background(), tt.targetIP, tt.lunID)
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			// Cleanup created entries
			for _, entry := range createdEntries {
				require.NoError(t, os.Remove(entry), "failed to clean up test entry")
			}
		})
	}
}

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
			fs := FS{}
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
		expect   string
	}{
		{
			testname: "Invalid mount args",
			mntCmnd:  "invalid",
			source:   "/",
			target:   "",
			fstype:   "",
			opts:     []string{"a", "b"},
			expect:   "Path: / is invalid",
		},
		{
			testname: "Valid mount command",
			mntCmnd:  "mount",
			source:   "dev",
			target:   "usr",
			fstype:   "ext4",
			opts:     []string{"key=value", "variable"},
			expect:   "mount failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{}
			err := fs.doMount(tt.ctx, tt.mntCmnd, tt.source, tt.target, tt.fstype, tt.opts...)
			assert.Equal(t, true, strings.Contains(err.Error(), tt.expect))
		})
	}
}

func TestUnMount(t *testing.T) {
	tests := []struct {
		testname string
		ctx      context.Context
		target   string
		expect   string
	}{
		{
			testname: "Invalid path",
			target:   "/",
			expect:   "Path: / is invalid",
		},
		{
			testname: "Invalid arguments",
			target:   "/abc",
			expect:   "unmount failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{}
			err := fs.unmount(tt.ctx, tt.target)
			assert.Equal(t, true, strings.Contains(err.Error(), tt.expect))
		})
	}
}

func TestMultipathCommand(t *testing.T) {
	tests := []struct {
		testname       string
		ctx            context.Context
		timeoutSeconds time.Duration
		chroot         string
		arguments      []string
		expectErr      error
	}{
		{
			testname:       "Empty chroot",
			timeoutSeconds: time.Duration(10),
			chroot:         "",
			arguments:      []string{"A", "iR"},
			expectErr: &os.PathError{
				Op:   "fork/exec",
				Path: "/usr/sbin/multipath",
				Err:  syscall.ENOENT,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := FS{}
			_, err := fs.multipathCommand(tt.ctx, tt.timeoutSeconds, tt.chroot, tt.arguments...)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

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
			fs := FS{}
			_, err := fs.isBind(tt.ctx, tt.opts...)
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
			_, err := GetDevMounts(tt.ctx, tt.dev)
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
			_, err := ValidateDevice(tt.ctx, tt.source)
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
			testname: "Targets",
			targets:  []string{"iqn.2016-06.io.k8s", "iqn.2017-06.io.k8s", "0x500000"},
			lun:      "",
			expectErr: &os.PathError{
				Op:   "open",                     // Operation that caused the error
				Path: "/sys/class/iscsi_session", // Path where the error occurred
				Err:  syscall.ENOENT,             // Error code (e.g., 0x2 corresponds to ENOENT - "No such file or directory")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			err := RescanSCSIHost(tt.ctx, tt.targets, tt.lun)
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
			err := RemoveBlockDevice(tt.ctx, tt.blockDevicePath)
			assert.Equal(t, tt.expectErr, err)
		})
	}
}

func TestIssueLIPToAllFCHosts(t *testing.T) {
	tempDir := t.TempDir()
	fcHostsDir = tempDir
	require.NoError(t, os.MkdirAll(fcHostsDir, 0o755))

	fs := &FS{}

	tests := []struct {
		name       string
		hosts      map[string]string
		shouldFail bool
	}{
		{
			name: "Single host",
			hosts: map[string]string{
				"host1": "1",
			},
			shouldFail: false,
		},
		{
			name: "Multiple hosts",
			hosts: map[string]string{
				"host1": "1",
				"host2": "1",
			},
			shouldFail: false,
		},
		{
			name:       "No hosts",
			hosts:      map[string]string{},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock hosts and issue_lip files

			for host, lip := range tt.hosts {
				hostDir := filepath.Join(fcHostsDir, host)
				require.NoError(t, os.MkdirAll(hostDir, 0o755))
				lipFile := filepath.Join(hostDir, "issue_lip")
				require.NoError(t, os.WriteFile(lipFile, []byte(lip), 0o200))
			}

			// Call the function
			err := fs.issueLIPToAllFCHosts(context.Background())
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetFCHostPortWWNs(t *testing.T) {
	tempDir := t.TempDir()
	fcHostsDir = tempDir // Use the temporary directory for testing
	require.NoError(t, os.MkdirAll(fcHostsDir, 0o755))

	fs := &FS{}

	tests := []struct {
		name       string
		entries    map[string]string
		expected   []string
		shouldFail bool
	}{
		{
			name: "Single entry",
			entries: map[string]string{
				"host1/port_name": "0x500143802426baf7",
			},
			expected: []string{"0x500143802426baf7"},
		},
		{
			name: "Multiple entries",
			entries: map[string]string{
				"host1/port_name": "0x500143802426baf7",
				"host2/port_name": "0x500143802426baf8",
			},
			expected: []string{"0x500143802426baf7", "0x500143802426baf8"},
		},
		{
			name:       "No matching entries",
			entries:    map[string]string{},
			expected:   []string{},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock port_name files
			createdEntries := []string{}
			for entry, content := range tt.entries {
				filePath := filepath.Join(fcHostsDir, entry)
				require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
				createdEntries = append(createdEntries, filePath)
			}

			// Call the function with the test input
			result, err := fs.getFCHostPortWWNs(context.Background())
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, result)
			}

			// Cleanup created entries
			for _, entry := range createdEntries {
				require.NoError(t, os.Remove(entry), "failed to clean up test entry")
			}
		})
	}
}

func TestGetIscsiTargetHosts(t *testing.T) {
	tempDir := t.TempDir()
	sessionsdir = tempDir // Use the temporary directory for testing
	require.NoError(t, os.MkdirAll(sessionsdir, 0o755))

	tests := []struct {
		name       string
		targets    []string
		entries    map[string]string
		expected   []*targetdev
		shouldFail bool
	}{
		{
			name:    "Single target",
			targets: []string{"iqn.1992-04.com.emc:600009700bcbb70e3287017400000000"},
			entries: map[string]string{
				"session1/targetname":         "iqn.1992-04.com.emc:600009700bcbb70e3287017400000000",
				"session1/device/target0:0:0": "",
			},
			expected: []*targetdev{
				{host: "host0", channel: "0", target: "0"},
			},
		},
		{
			name:    "Multiple targets",
			targets: []string{"iqn.1992-04.com.emc:600009700bcbb70e3287017400000000", "iqn.1992-04.com.emc:600009700bcbb70e3287017400000001"},
			entries: map[string]string{
				"session1/targetname":         "iqn.1992-04.com.emc:600009700bcbb70e3287017400000000",
				"session1/device/target0:0:0": "",
				"session2/targetname":         "iqn.1992-04.com.emc:600009700bcbb70e3287017400000001",
				"session2/device/target1:0:0": "",
			},
			expected: []*targetdev{
				{host: "host0", channel: "0", target: "0"},
				{host: "host1", channel: "0", target: "0"},
			},
		},
		{
			name:       "No matching targets",
			targets:    []string{"iqn.1992-04.com.emc:600009700bcbb70e3287017400000002"},
			entries:    map[string]string{},
			expected:   []*targetdev{},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock targetname and device files
			for entry, content := range tt.entries {
				filePath := filepath.Join(sessionsdir, entry)
				require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
			}

			// Call the function with the test input
			result, err := getIscsiTargetHosts(tt.targets)
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetFCTargetHosts(t *testing.T) {
	tempDir := t.TempDir()
	fcRemotePortsDir = tempDir // Use the temporary directory for testing
	require.NoError(t, os.MkdirAll(fcRemotePortsDir, 0o755))

	tests := []struct {
		name       string
		targets    []string
		entries    map[string]string
		expected   []*targetdev
		shouldFail bool
	}{
		{
			name:    "Single target",
			targets: []string{"0x50060160c46036a0"},
			entries: map[string]string{
				"rport-0:0/port_name": "0x50060160c46036a0",
			},
			expected: []*targetdev{
				{host: "host0", channel: "-", target: "-"},
			},
		},
		{
			name:    "Multiple targets",
			targets: []string{"0x50060160c46036a0", "0x50060160c46036a1"},
			entries: map[string]string{
				"rport-0:0/port_name": "0x50060160c46036a0",
				"rport-1:0/port_name": "0x50060160c46036a1",
			},
			expected: []*targetdev{
				{host: "host0", channel: "-", target: "-"},
				{host: "host1", channel: "-", target: "-"},
			},
		},
		{
			name:       "No matching targets",
			targets:    []string{"0x50060160c46036a2"},
			entries:    map[string]string{},
			expected:   []*targetdev{},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock port_name files
			for entry, content := range tt.entries {
				filePath := filepath.Join(fcRemotePortsDir, entry)
				require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
			}

			// Call the function with the test input
			result, err := getFCTargetHosts(tt.targets)
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// func TestRemoveBlockDevice(t *testing.T) {
// 	tempDir := t.TempDir()
// 	fcRemotePortsDir = tempDir // Use the temporary directory for testing
// 	require.NoError(t, os.MkdirAll(fcRemotePortsDir, 0o755))
// 	fs := &FS{}

// 	tests := []struct {
// 		name            string
// 		blockDevicePath string
// 		stateContent    string
// 		expectedError   string
// 	}{
// 		{
// 			name:            "Successful removal",
// 			blockDevicePath: filepath.Join(tempDir, "dev/sda"),
// 			stateContent:    "running",
// 			expectedError:   "",
// 		},
// 		{
// 			name:            "Device in blocked state",
// 			blockDevicePath: filepath.Join(tempDir, "dev/sdb"),
// 			stateContent:    "blocked",
// 			expectedError:   "Device sdb is in blocked state",
// 		},
// 		{
// 			name:            "State file not found",
// 			blockDevicePath: filepath.Join(tempDir, "dev/sdc"),
// 			expectedError:   "Cannot read /sys/block/sdc/device/state",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			deviceName := filepath.Base(tt.blockDevicePath)
// 			sysBlockDir := filepath.Join(tempDir, "sys/block", deviceName, "device")
// 			require.NoError(t, os.MkdirAll(sysBlockDir, 0o755))

// 			if tt.stateContent != "" {
// 				statePath := filepath.Join(sysBlockDir, "state")
// 				require.NoError(t, os.WriteFile(statePath, []byte(tt.stateContent), 0o644))
// 			}

// 			deletePath := filepath.Join(sysBlockDir, "delete")
// 			require.NoError(t, os.WriteFile(deletePath, []byte{}, 0o644))

// 			err := fs.removeBlockDevice(context.Background(), tt.blockDevicePath)
// 			if tt.expectedError != "" {
// 				assert.Contains(t, err.Error(), tt.expectedError)
// 			} else {
// 				assert.NoError(t, err)
// 				content, err := os.ReadFile(deletePath)
// 				assert.NoError(t, err)
// 				assert.Equal(t, "1", string(content))
// 			}
// 		})
// 	}
// }
