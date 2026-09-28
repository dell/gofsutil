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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Linux Test Helpers -----------------------------------------------------

// setupMockSysfs creates a temp directory mimicking /sys/block/<dev>/ with
// the specified files and their content. Returns the temp base path.
// The caller should override sysBlockDir to point to this path.
func setupMockSysfs(t *testing.T, devName string, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()
	devDir := filepath.Join(tmpDir, devName)
	for relPath, content := range files {
		fullPath := filepath.Join(devDir, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(t, err, "failed to create sysfs mock directory")
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err, "failed to write sysfs mock file")
	}
	return tmpDir
}

// setupReclaimMockExec overrides reclaimExecFn for testing and returns a
// cleanup function that restores the original. Follows the setMockExec
// pattern from gofsutil_fsck_test.go.
func setupReclaimMockExec(handler func(ctx context.Context, name string, args ...string) ([]byte, error)) func() {
	orig := reclaimExecFn
	reclaimExecFn = handler
	return func() { reclaimExecFn = orig }
}

// overrideSysBlockDir temporarily overrides the sysBlockDir package variable
// for tests that need mock sysfs access. Returns a cleanup function.
func overrideSysBlockDir(t *testing.T, newDir string) {
	t.Helper()
	orig := sysBlockDir
	sysBlockDir = newDir
	t.Cleanup(func() { sysBlockDir = orig })
}

// ---- Unit Tests: parseFstrimBytes (U-001 to U-008) -------------------------

// Test ID: U-001
// Test ID: U-002
// Test ID: U-003
// Test ID: U-004
// Test ID: U-005
// Test ID: U-006
// Test ID: U-007
// Test ID: U-008
func TestParseFstrimBytes(t *testing.T) {
	tests := []struct {
		testID   string
		name     string
		input    string
		expected int64
	}{
		{
			testID:   "U-001",
			name:     "standard_output",
			input:    "/mnt/data: 42949672960 bytes were trimmed",
			expected: 42949672960,
		},
		{
			testID:   "U-002",
			name:     "zero_bytes_trimmed",
			input:    "/mnt/data: 0 bytes were trimmed",
			expected: 0,
		},
		{
			testID:   "U-003",
			name:     "singular_byte",
			input:    "/mnt: 1 byte trimmed",
			expected: 1,
		},
		{
			testID:   "U-004",
			name:     "empty_string",
			input:    "",
			expected: 0,
		},
		{
			testID:   "U-005",
			name:     "no_match",
			input:    "fstrim: some error occurred",
			expected: 0,
		},
		{
			testID:   "U-006",
			name:     "large_value",
			input:    "/mnt: 9223372036854775807 bytes were trimmed",
			expected: 9223372036854775807, // math.MaxInt64
		},
		{
			testID:   "U-007",
			name:     "bytes_without_were",
			input:    "/mnt: 1024 bytes trimmed",
			expected: 1024,
		},
		{
			testID:   "U-008",
			name:     "multiline_output",
			input:    "\n/mnt: 512 bytes were trimmed\n",
			expected: 512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFstrimBytes(tt.input)
			assert.Equal(t, tt.expected, result,
				"[%s] parseFstrimBytes(%q) should return %d", tt.testID, tt.input, tt.expected)
		})
	}
}

// ---- Unit Tests: getBlockDeviceSize (U-009 to U-013) ------------------------

// Test ID: U-009
func TestGetBlockDeviceSize_ValidDevice(t *testing.T) {
	// Create temp sysfs with size file: 2097152 sectors = 1 GiB
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"size": "2097152\n",
	})
	overrideSysBlockDir(t, mockDir)

	size, err := getBlockDeviceSize("/dev/sda")

	require.NoError(t, err, "getBlockDeviceSize should not return error for valid device")
	assert.Equal(t, int64(1073741824), size,
		"Expected 2097152 sectors * 512 = 1073741824 bytes")
}

// Test ID: U-010
func TestGetBlockDeviceSize_MissingSysfsFile(t *testing.T) {
	// Point to a non-existent directory
	overrideSysBlockDir(t, "/tmp/nonexistent-sysfs-path-for-test")

	_, err := getBlockDeviceSize("/dev/nonexistent")

	require.Error(t, err, "getBlockDeviceSize should return error for missing sysfs file")
}

// Test ID: U-011
func TestGetBlockDeviceSize_InvalidContent(t *testing.T) {
	mockDir := setupMockSysfs(t, "sdb", map[string]string{
		"size": "not_a_number\n",
	})
	overrideSysBlockDir(t, mockDir)

	_, err := getBlockDeviceSize("/dev/sdb")

	require.Error(t, err, "getBlockDeviceSize should return error for non-numeric content")
	assert.Contains(t, err.Error(), "failed to parse size",
		"Error should mention 'failed to parse size'")
}

// Test ID: U-012
func TestGetBlockDeviceSize_ZeroSectors(t *testing.T) {
	mockDir := setupMockSysfs(t, "sdc", map[string]string{
		"size": "0\n",
	})
	overrideSysBlockDir(t, mockDir)

	size, err := getBlockDeviceSize("/dev/sdc")

	require.NoError(t, err, "getBlockDeviceSize should not return error for zero sectors")
	assert.Equal(t, int64(0), size,
		"Expected 0 sectors * 512 = 0 bytes")
}

// Test ID: U-013
func TestGetBlockDeviceSize_MapperDeviceSymlink(t *testing.T) {
	// Create temp sysfs with dm-0/size and a symlink from mapper/vol -> dm-0
	tmpDir := t.TempDir()

	// Create the dm-0 device in sysfs
	dm0Dir := filepath.Join(tmpDir, "dm-0")
	require.NoError(t, os.MkdirAll(dm0Dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dm0Dir, "size"), []byte("4194304\n"), 0o644))

	overrideSysBlockDir(t, tmpDir)

	// Create a symlink: /tmp/.../mapper/vol -> /tmp/.../devdir/dm-0
	mapperDir := filepath.Join(tmpDir, "mapper")
	require.NoError(t, os.MkdirAll(mapperDir, 0o755))
	devDir := filepath.Join(tmpDir, "devdir")
	require.NoError(t, os.MkdirAll(devDir, 0o755))
	// Create the actual dm-0 device file (symlink target)
	dmTarget := filepath.Join(devDir, "dm-0")
	require.NoError(t, os.WriteFile(dmTarget, []byte{}, 0o644))
	// Create symlink
	volLink := filepath.Join(mapperDir, "vol")
	require.NoError(t, os.Symlink(dmTarget, volLink))

	// Call with mapper path - stub won't resolve but test verifies compilation
	size, err := getBlockDeviceSize(volLink)

	// The stub returns 0, nil. In the real implementation, it would resolve the
	// symlink and read dm-0/size (4194304 sectors * 512 = 2147483648 bytes).
	require.NoError(t, err)
	assert.Equal(t, int64(2147483648), size,
		"Expected 4194304 sectors * 512 = 2147483648 bytes for mapper device")
}

// ---- Unit Tests: checkDiscardSupport (U-014 to U-021) -----------------------

// Test ID: U-014
func TestCheckDiscardSupport_SupportedDevice(t *testing.T) {
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"queue/discard_max_bytes": "4294967295\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.checkDiscardSupport(context.Background(), "/dev/sda")

	require.NoError(t, err, "checkDiscardSupport should not error for supported device")
	require.NotNil(t, result, "Expected non-nil DiscardCapability")
	assert.True(t, result.Supported, "Expected Supported == true")
	assert.Equal(t, int64(4294967295), result.DiscardMaxBytes,
		"Expected DiscardMaxBytes == 4294967295")
	assert.Empty(t, result.Reason, "Expected empty Reason for supported device")
}

// Test ID: U-015
func TestCheckDiscardSupport_UnsupportedZeroMaxBytes(t *testing.T) {
	mockDir := setupMockSysfs(t, "sdb", map[string]string{
		"queue/discard_max_bytes": "0\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.checkDiscardSupport(context.Background(), "/dev/sdb")

	require.NoError(t, err, "checkDiscardSupport should not error for unsupported device")
	require.NotNil(t, result, "Expected non-nil DiscardCapability")
	assert.False(t, result.Supported, "Expected Supported == false when discard_max_bytes=0")
	assert.Equal(t, int64(0), result.DiscardMaxBytes,
		"Expected DiscardMaxBytes == 0")
	assert.Contains(t, result.Reason, "discard_max_bytes=0",
		"Reason should mention discard_max_bytes=0")
}

// Test ID: U-016
func TestCheckDiscardSupport_MissingSysfsPath(t *testing.T) {
	overrideSysBlockDir(t, "/tmp/nonexistent-sysfs-path-for-test")

	fsObj := &FS{}
	_, err := fsObj.checkDiscardSupport(context.Background(), "/dev/nonexistent")

	require.Error(t, err, "checkDiscardSupport should error for missing sysfs path")
	assert.Contains(t, err.Error(), "failed to read",
		"Error should mention 'failed to read'")
}

// Test ID: U-017
func TestCheckDiscardSupport_InvalidDiscardMaxBytes(t *testing.T) {
	mockDir := setupMockSysfs(t, "sdc", map[string]string{
		"queue/discard_max_bytes": "abc\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	_, err := fsObj.checkDiscardSupport(context.Background(), "/dev/sdc")

	require.Error(t, err, "checkDiscardSupport should error for non-numeric discard_max_bytes")
	assert.Contains(t, err.Error(), "failed to parse discard_max_bytes",
		"Error should mention 'failed to parse discard_max_bytes'")
}

// Test ID: U-018
func TestCheckDiscardSupport_DmCryptWithoutAllowDiscards(t *testing.T) {
	mockDir := setupMockSysfs(t, "dm-0", map[string]string{
		"queue/discard_max_bytes":   "4294967295\n",
		"dm/uuid":                   "CRYPT-LUKS2-abc123\n",
		"queue/discard_granularity": "0\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.checkDiscardSupport(context.Background(), "/dev/dm-0")

	require.NoError(t, err, "checkDiscardSupport should not error")
	require.NotNil(t, result, "Expected non-nil DiscardCapability")
	assert.False(t, result.Supported,
		"Expected Supported == false for dm-crypt without allow-discards")
	assert.Contains(t, result.Reason, "dm-crypt",
		"Reason should mention dm-crypt")
	assert.Contains(t, result.Reason, "allow-discards",
		"Reason should mention allow-discards")
}

// Test ID: U-019
func TestCheckDiscardSupport_DmCryptWithAllowDiscards(t *testing.T) {
	mockDir := setupMockSysfs(t, "dm-1", map[string]string{
		"queue/discard_max_bytes":   "4294967295\n",
		"dm/uuid":                   "CRYPT-LUKS2-abc123\n",
		"queue/discard_granularity": "4096\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.checkDiscardSupport(context.Background(), "/dev/dm-1")

	require.NoError(t, err, "checkDiscardSupport should not error")
	require.NotNil(t, result, "Expected non-nil DiscardCapability")
	assert.True(t, result.Supported,
		"Expected Supported == true for dm-crypt with allow-discards")
	assert.Equal(t, int64(4294967295), result.DiscardMaxBytes,
		"Expected DiscardMaxBytes to match sysfs value")
}

// Test ID: U-020
func TestCheckDiscardSupport_NonCryptDmDevice(t *testing.T) {
	mockDir := setupMockSysfs(t, "dm-2", map[string]string{
		"queue/discard_max_bytes": "4294967295\n",
		"dm/uuid":                 "LVM-abc123\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.checkDiscardSupport(context.Background(), "/dev/dm-2")

	require.NoError(t, err, "checkDiscardSupport should not error")
	require.NotNil(t, result, "Expected non-nil DiscardCapability")
	assert.True(t, result.Supported,
		"Expected Supported == true for non-CRYPT dm device with valid discard")
}

// Test ID: U-021
func TestCheckDiscardSupport_NoDmUuidFile(t *testing.T) {
	mockDir := setupMockSysfs(t, "sdd", map[string]string{
		"queue/discard_max_bytes": "4294967295\n",
		// No dm/uuid file - not a DM device
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.checkDiscardSupport(context.Background(), "/dev/sdd")

	require.NoError(t, err, "checkDiscardSupport should not error")
	require.NotNil(t, result, "Expected non-nil DiscardCapability")
	assert.True(t, result.Supported,
		"Expected Supported == true for device without dm/uuid and discard > 0")
}

// ---- Unit Tests: fstrim command execution (U-022 to U-025) ------------------

// Test ID: U-022
func TestFstrim_Success(t *testing.T) {
	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "fstrim" {
			return []byte("/mnt/data: 1048576 bytes were trimmed"), nil
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	result, err := fsObj.fstrim(context.Background(), "/mnt/data")

	require.NoError(t, err, "fstrim should succeed")
	require.NotNil(t, result, "Expected non-nil FstrimResult")
	assert.Equal(t, int64(1048576), result.BytesTrimmed,
		"Expected BytesTrimmed == 1048576")
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0),
		"Expected Duration > 0")
}

// Test ID: U-023
func TestFstrim_CommandFailure(t *testing.T) {
	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "fstrim" {
			return []byte("fstrim: /mnt/data: the discard operation is not supported"), os.ErrPermission
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	_, err := fsObj.fstrim(context.Background(), "/mnt/data")

	require.Error(t, err, "fstrim should fail when command exits non-zero")
	assert.Contains(t, err.Error(), "fstrim failed on",
		"Error should contain 'fstrim failed on'")
}

// Test ID: U-024
func TestFstrim_ContextTimeout(t *testing.T) {
	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "fstrim" {
			// Simulate a killed process error
			return nil, &timeoutExecError{msg: "signal: killed"}
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	_, err := fsObj.fstrim(context.Background(), "/mnt/data")

	require.Error(t, err, "fstrim should fail on context timeout")
	assert.Contains(t, err.Error(), "fstrim timed out",
		"Error should contain 'fstrim timed out'")
}

// Test ID: U-025
func TestFstrim_ZeroBytesOutput(t *testing.T) {
	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "fstrim" {
			return []byte("/mnt: 0 bytes were trimmed"), nil
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	result, err := fsObj.fstrim(context.Background(), "/mnt")

	require.NoError(t, err, "fstrim should succeed with 0 bytes trimmed")
	require.NotNil(t, result, "Expected non-nil FstrimResult")
	assert.Equal(t, int64(0), result.BytesTrimmed,
		"Expected BytesTrimmed == 0")
}

// ---- Unit Tests: blkdiscard command execution (U-026 to U-029) --------------

// Test ID: U-026
func TestBlkdiscard_Success(t *testing.T) {
	// Mock sysfs for device size
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"size": "2097152\n", // 1 GiB in 512-byte sectors
	})
	overrideSysBlockDir(t, mockDir)

	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "blkdiscard" {
			return []byte(""), nil
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	result, err := fsObj.blkdiscard(context.Background(), "/dev/sda")

	require.NoError(t, err, "blkdiscard should succeed")
	require.NotNil(t, result, "Expected non-nil BlkdiscardResult")
	assert.Equal(t, int64(1073741824), result.BytesDiscarded,
		"Expected BytesDiscarded == device size (1073741824)")
}

// Test ID: U-027
func TestBlkdiscard_DeviceSizeReadFailure(t *testing.T) {
	// Point to non-existent sysfs
	overrideSysBlockDir(t, "/tmp/nonexistent-sysfs-path-for-test")

	fsObj := &FS{}
	_, err := fsObj.blkdiscard(context.Background(), "/dev/nonexistent")

	require.Error(t, err, "blkdiscard should fail when device size cannot be read")
	assert.Contains(t, err.Error(), "failed to get device size",
		"Error should contain 'failed to get device size'")
}

// Test ID: U-028
func TestBlkdiscard_CommandFailure(t *testing.T) {
	// Mock sysfs for device size (so we get past the size check)
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"size": "2097152\n",
	})
	overrideSysBlockDir(t, mockDir)

	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "blkdiscard" {
			return []byte("blkdiscard: /dev/sda: BLKDISCARD ioctl failed: Operation not permitted"), os.ErrPermission
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	_, err := fsObj.blkdiscard(context.Background(), "/dev/sda")

	require.Error(t, err, "blkdiscard should fail when command exits non-zero")
	assert.Contains(t, err.Error(), "blkdiscard failed on",
		"Error should contain 'blkdiscard failed on'")
}

// Test ID: U-029
func TestBlkdiscard_ContextTimeout(t *testing.T) {
	// Mock sysfs for device size
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"size": "2097152\n",
	})
	overrideSysBlockDir(t, mockDir)

	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "blkdiscard" {
			// Simulate a killed process error
			return nil, &timeoutExecError{msg: "signal: killed"}
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	_, err := fsObj.blkdiscard(context.Background(), "/dev/sda")

	require.Error(t, err, "blkdiscard should fail on context timeout")
	assert.Contains(t, err.Error(), "blkdiscard timed out",
		"Error should contain 'blkdiscard timed out'")
}

// ---- Unit Tests: Exported package-level wrappers (U-030 to U-032) -----------

// Test ID: U-030
func TestPackageFstrim_ViaUseMockFS(t *testing.T) {
	UseMockFS()
	defer func() { fs = &FS{ScanEntry: defaultEntryScanFunc} }()

	GOFSMock.InduceFstrimError = false
	result, err := Fstrim(context.Background(), "/mnt/data")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(1073741824), result.BytesTrimmed)

	GOFSMock.InduceFstrimError = true
	_, err = Fstrim(context.Background(), "/mnt/data")
	require.Error(t, err)
	GOFSMock.InduceFstrimError = false
}

// Test ID: U-031
func TestPackageBlkdiscard_ViaUseMockFS(t *testing.T) {
	UseMockFS()
	defer func() { fs = &FS{ScanEntry: defaultEntryScanFunc} }()

	GOFSMock.InduceBlkdiscardError = false
	result, err := Blkdiscard(context.Background(), "/dev/sda")
	require.NoError(t, err)
	require.NotNil(t, result)

	GOFSMock.InduceBlkdiscardError = true
	_, err = Blkdiscard(context.Background(), "/dev/sda")
	require.Error(t, err)
	GOFSMock.InduceBlkdiscardError = false
}

// Test ID: U-032
func TestPackageCheckDiscardSupport_ViaUseMockFS(t *testing.T) {
	UseMockFS()
	defer func() { fs = &FS{ScanEntry: defaultEntryScanFunc} }()

	GOFSMock.InduceCheckDiscardSupportError = false
	result, err := CheckDiscardSupport(context.Background(), "/dev/sda")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Supported)

	GOFSMock.InduceCheckDiscardSupportError = true
	_, err = CheckDiscardSupport(context.Background(), "/dev/sda")
	require.Error(t, err)
	GOFSMock.InduceCheckDiscardSupportError = false
}

// ---- Unit Tests: FS exported method wrappers (U-033 to U-035) ---------------

// Test ID: U-033
func TestFS_Fstrim(t *testing.T) {
	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "fstrim" {
			return []byte("/mnt: 512 bytes were trimmed"), nil
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	result, err := fsObj.Fstrim(context.Background(), "/mnt")
	require.NoError(t, err)
	assert.Equal(t, int64(512), result.BytesTrimmed)
}

// Test ID: U-034
func TestFS_Blkdiscard(t *testing.T) {
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"size": "2097152\n",
	})
	overrideSysBlockDir(t, mockDir)

	restore := setupReclaimMockExec(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "blkdiscard" {
			return []byte(""), nil
		}
		return nil, nil
	})
	defer restore()

	fsObj := &FS{}
	result, err := fsObj.Blkdiscard(context.Background(), "/dev/sda")
	require.NoError(t, err)
	assert.Equal(t, int64(1073741824), result.BytesDiscarded)
}

// Test ID: U-035
func TestFS_CheckDiscardSupport(t *testing.T) {
	mockDir := setupMockSysfs(t, "sda", map[string]string{
		"queue/discard_max_bytes": "4294967295\n",
	})
	overrideSysBlockDir(t, mockDir)

	fsObj := &FS{}
	result, err := fsObj.CheckDiscardSupport(context.Background(), "/dev/sda")
	require.NoError(t, err)
	assert.True(t, result.Supported)
}

// ---- Unit Tests: defaultReclaimExec (U-036) ---------------------------------

// Test ID: U-036
func TestDefaultReclaimExec_Success(t *testing.T) {
	out, err := defaultReclaimExec(context.Background(), "echo", "hello")
	require.NoError(t, err, "defaultReclaimExec should succeed with echo")
	assert.Contains(t, string(out), "hello",
		"Output should contain 'hello'")
}

// ---- Unit Tests: parseFstrimBytes overflow (U-037) --------------------------

// Test ID: U-037
func TestParseFstrimBytes_Overflow(t *testing.T) {
	// A number larger than int64 max triggers ParseInt error → returns 0
	result := parseFstrimBytes("99999999999999999999999 bytes trimmed")
	assert.Equal(t, int64(0), result,
		"parseFstrimBytes should return 0 for overflow value")
}

// ---- Test helpers for simulating exec errors --------------------------------

// timeoutExecError is a test helper that simulates a process-killed error
// for testing timeout/cancellation paths. It is NOT an exec.ExitError, but
// is used to represent the error that would be returned when a process is killed.
type timeoutExecError struct {
	msg string
}

func (e *timeoutExecError) Error() string {
	return e.msg
}
