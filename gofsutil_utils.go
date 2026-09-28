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
	"path/filepath"
	"regexp"
)

func validatePath(path string) error {
	if path == "/" {
		return errors.New("Path: " + path + " is invalid")
	}

	return nil
}

func validateFsType(fsType string) error {
	if fsType != "ext4" && fsType != "ext3" &&
		fsType != "xfs" && fsType != "nfs" {
		return errors.New("FsType: " + fsType + " is invalid")
	}

	return nil
}

var mountOptionRegex = regexp.MustCompile(`[\w]+[=]*[\w]*`)

func validateMountOptions(mountOptions ...string) error {
	for _, opt := range mountOptions {
		// regex e.g: "rw", "noatime", "", " "
		matched := mountOptionRegex.MatchString(opt)
		if !matched {
			return errors.New("Mount option: " + opt + " is invalid")
		}
	}
	return nil
}

var multipathArgRegex = regexp.MustCompile(`[[-][AaBbCcdFfhilpqrTtUuWw0-9]+]*[0-9]*`)

func validateMultipathArgs(options ...string) error {
	for _, opt := range options {
		// check for options
		// regex e.g: "-A", "-iR", "-h1", "-/data0", "", " "
		matched := multipathArgRegex.MatchString(opt)
		if matched {
			continue
		}

		// check for file or device path
		// regex e.g: "/tmp", "/data0", "", " "
		if err := validatePath(filepath.Clean(opt)); err != nil {
			return errors.New("Multipath option: " + opt + " is invalid")
		}
	}

	return nil
}

// validateDeviceID validates device identifiers to prevent OS command injection.
// Device IDs should only contain alphanumeric characters, underscores, hyphens, and dots.
// This follows the same pattern used in goiscsi for CVE-2022-34374 (DSA-2022-202).
func validateDeviceID(devID string) error {
	if devID == "" {
		return errors.New("device ID cannot be empty")
	}
	// Allow only alphanumeric characters, underscores, hyphens, and dots
	// This covers valid device names like: sda, sda1, nvme0n1, mpath0, emcpowera,
	// dm-0, 3600601xxxxxxx, vol-abc123, etc.
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, devID)
	if err != nil {
		return errors.New("failed to validate device ID: " + err.Error())
	}
	if !matched {
		return errors.New("device ID contains invalid characters: " + devID)
	}
	return nil
}
