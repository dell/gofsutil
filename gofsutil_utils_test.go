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
	"errors"
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path   string
		result error
	}{
		{
			path:   "/",
			result: errors.New("Path: / is invalid"),
		},
		{
			path:   "/dev/disk/by-id/wwn-0x60570970000197900046533030394146",
			result: nil,
		},
		{
			path:   "../../mydevb",
			result: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			err := validatePath(tt.path)
			if err != nil {
				if tt.result == nil {
					t.Errorf("Validation of path is incorrect, \n\tgot: %s \n\twant: %v",
						err, tt.result)
				} else {
					if err.Error() != tt.result.Error() {
						t.Errorf("Validation of path is incorrect, \n\tgot: %s \n\twant: %s",
							err, tt.result)
					}
				}
			}
		})
	}
}

func TestValidateFsType(t *testing.T) {
	tests := []struct {
		fsType string
		result error
	}{
		{
			fsType: "smtp",
			result: errors.New("FsType: smtp is invalid"),
		},
		{
			fsType: " ",
			result: errors.New("FsType:   is invalid"),
		},
		{
			fsType: "ext3",
			result: nil,
		},
		{
			fsType: "ext4",
			result: nil,
		},
		{
			fsType: "xfs",
			result: nil,
		},
		{
			fsType: "nfs",
			result: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			err := validateFsType(tt.fsType)
			if err != nil {
				if tt.result == nil {
					t.Errorf("Validation of fsType is incorrect, \n\tgot: %s \n\twant: %v",
						err, tt.result)
				} else {
					if err.Error() != tt.result.Error() {
						t.Errorf("Validation of fsType is incorrect, \n\tgot: %s \n\twant: %s",
							err, tt.result)
					}
				}
			}
		})
	}
}

func TestValidateMountOptions(t *testing.T) {
	tests := []struct {
		mountOptions []string
		result       error
	}{
		{
			mountOptions: []string{"*", "##", "()"},
			result:       errors.New("Mount option: * is invalid"),
		},
		{
			mountOptions: []string{""},
			result:       nil,
		},
		{
			mountOptions: []string{"", " ", ""},
			result:       nil,
		},
		{
			mountOptions: []string{"rw", "noatime"},
			result:       nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			optsStr := strings.Join(tt.mountOptions, " ")
			optsStr = strings.TrimSpace(optsStr)
			if len(optsStr) != 0 {
				err := validateMountOptions(tt.mountOptions...)
				if err != nil {
					if tt.result == nil {
						t.Errorf("Validation of mountOptions is incorrect, \n\tgot: %s \n\twant: %v",
							err, tt.result)
					} else {
						if err.Error() != tt.result.Error() {
							t.Errorf("Validation of mountOptions is incorrect, \n\tgot: %s \n\twant: %s",
								err, tt.result)
						}
					}
				}
			}
		})
	}
}

func TestValidateMultipathArgs(t *testing.T) {
	tests := []struct {
		pathArgs []string
		result   error
	}{
		{
			pathArgs: []string{"/data0", "-A", "-iR", "/tmp"},
			result:   nil,
		},
		{
			pathArgs: []string{"-/abc", "-h1", "/dev*"},
			result:   nil,
		},
		{
			pathArgs: []string{"/"},
			result:   errors.New("Multipath option: / is invalid"),
		},
		{
			pathArgs: []string{""},
			result:   nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			err := validateMultipathArgs(tt.pathArgs...)
			if err != nil {
				if tt.result == nil {
					t.Errorf("Validation of path args is incorrect, \n\tgot: %s \n\twant: %v",
						err, tt.result)
				} else {
					if err.Error() != tt.result.Error() {
						t.Errorf("Validation of path args is incorrect, \n\tgot: %s \n\twant: %s",
							err, tt.result)
					}
				}
			}
		})
	}
}

// TestValidateDeviceID tests the validateDeviceID function for OS command injection prevention
// This test covers CVE-2022-34374 style vulnerabilities (CWE-78)
func TestValidateDeviceID(t *testing.T) {
	tests := []struct {
		name      string
		devID     string
		wantError bool
		errorMsg  string
	}{
		// Valid device IDs
		{
			name:      "valid simple device",
			devID:     "sda",
			wantError: false,
		},
		{
			name:      "valid device with number",
			devID:     "sda1",
			wantError: false,
		},
		{
			name:      "valid nvme device",
			devID:     "nvme0n1",
			wantError: false,
		},
		{
			name:      "valid nvme partition",
			devID:     "nvme0n1p1",
			wantError: false,
		},
		{
			name:      "valid mpath device",
			devID:     "mpath0",
			wantError: false,
		},
		{
			name:      "valid mpath device with letters",
			devID:     "mpatha",
			wantError: false,
		},
		{
			name:      "valid emcpower device",
			devID:     "emcpowera",
			wantError: false,
		},
		{
			name:      "valid dm device",
			devID:     "dm-0",
			wantError: false,
		},
		{
			name:      "valid device with underscore",
			devID:     "vol_abc123",
			wantError: false,
		},
		{
			name:      "valid device with hyphen",
			devID:     "vol-abc123",
			wantError: false,
		},
		{
			name:      "valid device with dot",
			devID:     "vol.abc123",
			wantError: false,
		},
		{
			name:      "valid WWN-style ID",
			devID:     "3600601xxxxxxx",
			wantError: false,
		},
		{
			name:      "valid long alphanumeric",
			devID:     "60000970000120001263533030313434",
			wantError: false,
		},
		// Invalid device IDs - command injection attempts
		{
			name:      "empty device ID",
			devID:     "",
			wantError: true,
			errorMsg:  "device ID cannot be empty",
		},
		{
			name:      "single quote injection",
			devID:     "x'; id; echo '",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "semicolon injection",
			devID:     "sda; rm -rf /",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "backtick injection",
			devID:     "sda`id`",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "dollar sign injection",
			devID:     "sda$(id)",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "pipe injection",
			devID:     "sda|cat /etc/passwd",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "ampersand injection",
			devID:     "sda&id",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "newline injection",
			devID:     "sda\nid",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "space injection",
			devID:     "sda id",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "forward slash",
			devID:     "/dev/sda",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "backslash",
			devID:     "sda\\nid",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "double quote injection",
			devID:     `sda"; id; echo "`,
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "greater than redirect",
			devID:     "sda>/tmp/pwned",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "less than redirect",
			devID:     "sda</etc/passwd",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "hash comment",
			devID:     "sda#comment",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "exclamation mark",
			devID:     "sda!id",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "at sign",
			devID:     "sda@host",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "percent sign",
			devID:     "sda%s",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "caret",
			devID:     "sda^id",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "asterisk",
			devID:     "sda*",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "parentheses",
			devID:     "sda()",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "brackets",
			devID:     "sda[]",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "braces",
			devID:     "sda{}",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "equals sign",
			devID:     "sda=value",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "plus sign",
			devID:     "sda+1",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "colon",
			devID:     "sda:1",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "question mark",
			devID:     "sda?",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
		{
			name:      "tilde",
			devID:     "~sda",
			wantError: true,
			errorMsg:  "device ID contains invalid characters",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(st *testing.T) {
			st.Parallel()
			err := validateDeviceID(tt.devID)
			if tt.wantError {
				if err == nil {
					st.Errorf("validateDeviceID(%q) expected error but got nil", tt.devID)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					st.Errorf("validateDeviceID(%q) error = %q, want error containing %q", tt.devID, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					st.Errorf("validateDeviceID(%q) unexpected error: %v", tt.devID, err)
				}
			}
		})
	}
}
