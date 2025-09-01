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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	bindRemountOpts = []string{}
	mountRX         = regexp.MustCompile(`^(.+) on (.+) \((.+)\)$`)
)

// getDiskFormat uses 'lsblk' to see if the given disk is unformated
func (fs *FS) getDiskFormat(ctx context.Context, disk string) (string, error) {
	mps, err := fs.getMounts(ctx)
	if err != nil {
		return "", err
	}
	for _, i := range mps {
		if i.Device == disk {
			return i.Type, nil
		}
	}
	return "", fmt.Errorf("getDiskFormat: failed: %s", disk)
}

// formatAndMount uses unix utils to format and mount the given disk
func (fs *FS) formatAndMount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string,
) error {
	return ErrNotImplemented
}

// format uses unix utils to format the given disk
func (fs *FS) format(
	ctx context.Context,
	source, target, fsType string,
	opts ...string,
) error {
	return ErrNotImplemented
}

// getMounts returns a slice of all the mounted filesystems
func (fs *FS) getMounts(ctx context.Context) ([]Info, error) {
	out, err := exec.Command("mount").CombinedOutput()
	if err != nil {
		return nil, err
	}

	var mountInfos []Info
	scan := bufio.NewScanner(bytes.NewReader(out))

	for scan.Scan() {
		m := mountRX.FindStringSubmatch(scan.Text())
		if len(m) != 4 {
			continue
		}
		device := m[1]
		if !strings.HasPrefix(device, "/") {
			continue
		}
		var (
			path    = m[2]
			source  = device
			options = strings.Split(m[3], ",")
		)
		if len(options) == 0 {
			return nil, fmt.Errorf(
				"getMounts: invalid mount options: %s", device)
		}
		for i, v := range options {
			options[i] = strings.TrimSpace(v)
		}
		fsType := options[0]
		if len(options) > 1 {
			options = options[1:]
		} else {
			options = nil
		}
		mountInfos = append(mountInfos, Info{
			Device: device,
			Path:   path,
			Source: source,
			Type:   fsType,
			Opts:   options,
		})
	}
	return mountInfos, nil
}

// bindMount performs a bind mount
func (fs *FS) bindMount(
	ctx context.Context,
	source, target string, opts ...string,
) error {
	return fs.doMount(ctx, "bindfs", source, target, "", opts...)
}

// readProcMounts is not implemented for darwin but defined for testing purposes
func (fs *FS) readProcMounts(ctx context.Context,
	path string,
	info bool,
) ([]Info, uint32, error) {
	return nil, 0, errors.New("not implemented")
}
