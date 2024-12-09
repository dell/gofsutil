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
	"os"
	"strings"
	"testing"
	"errors"
	"syscall"
	"time"
	"github.com/stretchr/testify/assert"
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

// func TestGetFCHostPortWWNs(t *testing.T) {
// 	wwns, err := gofsutil.GetFCHostPortWWNs(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	for _, wwn := range wwns {
// 		fmt.Printf("local FC port wwn: %s\n", wwn)
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
	tests := []struct {
		src    string
		tgt    string
		wwn    string
		result string
	}{
		{
			src:    "/dev/disk/by-id/wwn-0x60570970000197900046533030394146",
			tgt:    "../../mydeva",
			wwn:    "60570970000197900046533030394146",
			result: "/dev/mydeva",
		},
		{
			src:    "/dev/disk/by-id/dm-uuid-mpath-360570970000197900046533030394146",
			tgt:    "../../mydevb",
			wwn:    "60570970000197900046533030394146",
			result: "/dev/mydevb",
		},
		{
			src:    "/dev/disk/by-id/nvme-eui.12635330303134340000976000012000",
			tgt:    "../../mydevb",
			wwn:    "12635330303134340000976000012000",
			result: "/dev/mydevb",
		},
	}
	for _, tt := range tests {
		t.Run("", func(_ *testing.T) {
			// Change directories
			workingDirectory, _ := os.Getwd()
			err := os.Chdir("/dev/disk/by-id")
			if err != nil {
				t.Errorf("Couldn't Chdir to /dev/disk/by/id: %s", err)
			}
			// Create a target
			file, err := os.Create(tt.result)
			if err != nil {
				t.Errorf("Couldn't Create %s: %s", tt.result, err)
			}
			file.Close()
			// Create a symlink
			err = os.Symlink(tt.tgt, tt.src)
			if err != nil {
				t.Errorf("Couldn't create Symlink %s: %s", tt.tgt, err)
			}
			// Get the entry
			a, b, err := WWNToDevicePathX(context.Background(), tt.wwn)
			if err != nil {
				t.Errorf("Couldn't find DevicePathX: %s", err)
			}
			if a != tt.src {
				t.Errorf("Expected %s got %s", tt.src, a)
			}
			if b != tt.result {
				t.Errorf("Expected %s got %s", tt.result, b)
			}
			// Get the entry
			c, err := WWNToDevicePath(context.Background(), tt.wwn)
			if err != nil {
				t.Errorf("Couldn't find DevicePathX: %s", err)
			}
			if c != tt.result {
				t.Errorf("Expected %s got %s", tt.result, c)
			}
			// Remove symlink
			err = os.Remove(tt.src)
			if err != nil {
				t.Errorf("Couldn't remove %s: %s", tt.src, err)
			}
			// Remove target
			err = os.Remove(tt.result)
			if err != nil {
				t.Errorf("Couldn't remove %s: %s", tt.result, err)
			}
			// Change directories
			err = os.Chdir(workingDirectory)
			if err != nil {
				t.Errorf("Couldn't Chdir to /dev/disk/by/id: %s", err)
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
			fs := FS{SysBlockDir: "string"}
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
			fs := FS{SysBlockDir: "string"}
			err := fs.unmount(tt.ctx, tt.target)
			assert.Equal(t, true, strings.Contains(err.Error(), tt.expect))
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
			fs := FS{SysBlockDir: "string"}
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
			fs := FS{SysBlockDir: "string"}
			_, err := fs.isBind(tt.ctx, tt.opts...)
			assert.Equal(t, tt.expect, err)
		})
	}
}

