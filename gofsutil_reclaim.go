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
	"time"
)

// FstrimResult holds the result of an fstrim operation on a mounted filesystem.
type FstrimResult struct {
	// BytesTrimmed is the number of bytes reported as trimmed by fstrim -v.
	BytesTrimmed int64
	// Duration is the wall-clock time the fstrim command took.
	Duration time.Duration
}

// BlkdiscardResult holds the result of a blkdiscard operation on a block device.
type BlkdiscardResult struct {
	// BytesDiscarded is the total size of the device (all bytes are discarded).
	BytesDiscarded int64
	// Duration is the wall-clock time the blkdiscard command took.
	Duration time.Duration
}

// DiscardCapability describes whether a block device supports discard operations
// (SCSI UNMAP / NVMe Deallocate).
type DiscardCapability struct {
	// Supported is true when the device reports discard_max_bytes > 0.
	Supported bool
	// DiscardMaxBytes is the value read from /sys/block/<dev>/queue/discard_max_bytes.
	DiscardMaxBytes int64
	// Reason is a human-readable explanation when Supported is false.
	Reason string
}

// Fstrim runs fstrim on the specified mount point, returning the bytes trimmed.
// The context deadline/timeout controls the maximum execution time.
func Fstrim(ctx context.Context, mountPoint string) (*FstrimResult, error) {
	return fs.Fstrim(ctx, mountPoint)
}

// Blkdiscard runs blkdiscard on the specified block device path, returning the
// bytes discarded. The context deadline/timeout controls the maximum execution time.
func Blkdiscard(ctx context.Context, devicePath string) (*BlkdiscardResult, error) {
	return fs.Blkdiscard(ctx, devicePath)
}

// CheckDiscardSupport checks whether a block device supports discard operations
// (SCSI UNMAP / NVMe Deallocate) by reading sysfs attributes.
func CheckDiscardSupport(ctx context.Context, devicePath string) (*DiscardCapability, error) {
	return fs.CheckDiscardSupport(ctx, devicePath)
}

// Fstrim runs fstrim on the specified mount point, returning the bytes trimmed.
func (f *FS) Fstrim(ctx context.Context, mountPoint string) (*FstrimResult, error) {
	return f.fstrim(ctx, mountPoint)
}

// Blkdiscard runs blkdiscard on the specified block device, returning the bytes discarded.
func (f *FS) Blkdiscard(ctx context.Context, devicePath string) (*BlkdiscardResult, error) {
	return f.blkdiscard(ctx, devicePath)
}

// CheckDiscardSupport checks whether a block device supports discard (UNMAP/Deallocate).
func (f *FS) CheckDiscardSupport(ctx context.Context, devicePath string) (*DiscardCapability, error) {
	return f.checkDiscardSupport(ctx, devicePath)
}
