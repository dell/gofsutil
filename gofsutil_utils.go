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

func validateMountOptions(mountOptions ...string) error {
	for _, opt := range mountOptions {
		// regex e.g: "rw", "noatime", "", " "
		matched, err := regexp.Match(`[\w]+[=]*[\w]*`, []byte(opt))
		if !matched || err != nil {
			return errors.New("Mount option: " + opt + " is invalid")
		}
	}
	return nil
}

func validateMultipathArgs(options ...string) error {
	for _, opt := range options {
		// check for options
		// regex e.g: "-A", "-iR", "-h1", "-/data0", "", " "
		matched, err := regexp.Match(`[[-][AaBbCcdFfhilpqrTtUuWw0-9]+]*[0-9]*`, []byte(opt))
		if matched && err == nil {
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
