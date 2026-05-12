// Copyright © 2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// reclaimExecFn is the mockable function variable used to execute system commands
// for reclaim operations (fstrim, blkdiscard). It can be overridden in tests.
var reclaimExecFn = defaultReclaimExec

func defaultReclaimExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204
	// Start the child process in a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}
	return cmd.CombinedOutput()
}

// fstrim runs the fstrim command on the specified mount point.
func (f *FS) fstrim(ctx context.Context, mountPoint string) (*FstrimResult, error) {
	start := time.Now()
	output, err := reclaimExecFn(ctx, "fstrim", "-v", mountPoint)
	duration := time.Since(start)

	if err != nil {
		if strings.Contains(err.Error(), "killed") {
			return nil, fmt.Errorf("fstrim timed out on %s: %w", mountPoint, err)
		}
		return nil, fmt.Errorf("fstrim failed on %s: %w", mountPoint, err)
	}

	bytesTrimmed := parseFstrimBytes(string(output))
	return &FstrimResult{
		BytesTrimmed: bytesTrimmed,
		Duration:     duration,
	}, nil
}

// blkdiscard runs the blkdiscard command on the specified block device.
func (f *FS) blkdiscard(ctx context.Context, devicePath string) (*BlkdiscardResult, error) {
	deviceSize, err := getBlockDeviceSize(devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get device size for %s: %w", devicePath, err)
	}

	start := time.Now()
	_, err = reclaimExecFn(ctx, "blkdiscard", devicePath)
	duration := time.Since(start)

	if err != nil {
		if strings.Contains(err.Error(), "killed") {
			return nil, fmt.Errorf("blkdiscard timed out on %s: %w", devicePath, err)
		}
		return nil, fmt.Errorf("blkdiscard failed on %s: %w", devicePath, err)
	}

	return &BlkdiscardResult{
		BytesDiscarded: deviceSize,
		Duration:       duration,
	}, nil
}

// checkDiscardSupport checks sysfs to determine discard capability.
func (f *FS) checkDiscardSupport(_ context.Context, devicePath string) (*DiscardCapability, error) {
	// Resolve symlinks to get the real device name
	resolved, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		resolved = devicePath
	}
	devName := filepath.Base(resolved)

	// Read discard_max_bytes from sysfs
	discardMaxPath := filepath.Join(sysBlockDir, devName, "queue", "discard_max_bytes")
	data, err := os.ReadFile(discardMaxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", discardMaxPath, err)
	}

	maxBytesStr := strings.TrimSpace(string(data))
	maxBytes, err := strconv.ParseInt(maxBytesStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse discard_max_bytes from %s: %w", discardMaxPath, err)
	}

	// If discard_max_bytes is 0, device does not support discard
	if maxBytes == 0 {
		return &DiscardCapability{
			Supported:       false,
			DiscardMaxBytes: 0,
			Reason:          "discard_max_bytes=0",
		}, nil
	}

	// Check for dm devices device by reading dm/uuid
	if strings.HasPrefix(devName, "dm-") {
		dmUUIDPath := filepath.Join(sysBlockDir, devName, "dm", "uuid")
		dmUUID, err := os.ReadFile(dmUUIDPath)
		if err == nil {
			uuid := strings.TrimSpace(string(dmUUID))
			if strings.HasPrefix(strings.ToUpper(uuid), "CRYPT-") {
				// dm-crypt device: check discard_granularity for allow-discards
				granularityPath := filepath.Join(sysBlockDir, devName, "queue", "discard_granularity")
				granData, gErr := os.ReadFile(granularityPath)
				if gErr != nil || strings.TrimSpace(string(granData)) == "0" {
					return &DiscardCapability{
						Supported:       false,
						DiscardMaxBytes: maxBytes,
						Reason:          "dm-crypt device without allow-discards",
					}, nil
				}
			}
		}
	}

	return &DiscardCapability{
		Supported:       true,
		DiscardMaxBytes: maxBytes,
	}, nil
}

// fstrimBytesRegex matches the byte count in fstrim -v output.
var fstrimBytesRegex = regexp.MustCompile(`(\d+)\s+bytes?`)

// parseFstrimBytes extracts the bytes value from fstrim -v output.
// Example output: "/mnt/data: 42949672960 bytes were trimmed"
func parseFstrimBytes(output string) int64 {
	matches := fstrimBytesRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0
	}
	val, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// getBlockDeviceSize returns the size in bytes of a block device by reading
// /sys/block/<dev>/size (which reports the size in 512-byte sectors).
// Symlinks are resolved via filepath.EvalSymlinks to handle device mapper paths.
func getBlockDeviceSize(devicePath string) (int64, error) {
	// Resolve symlinks to get the real device name
	resolved, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		resolved = devicePath
	}
	devName := filepath.Base(resolved)

	sizePath := filepath.Join(sysBlockDir, devName, "size")
	data, err := os.ReadFile(sizePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s: %w", sizePath, err)
	}

	sizeStr := strings.TrimSpace(string(data))
	sectors, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size from %s: %w", sizePath, err)
	}

	return sectors * 512, nil
}
