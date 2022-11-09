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
