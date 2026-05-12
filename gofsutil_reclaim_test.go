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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Test Helpers -----------------------------------------------------------

// resetGOFSMock resets all reclamation-related GOFSMock flags and custom result
// variables to zero values. Called via t.Cleanup() to avoid cross-test pollution.
func resetGOFSMock(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		GOFSMock.InduceFstrimError = false
		GOFSMock.InduceBlkdiscardError = false
		GOFSMock.InduceCheckDiscardSupportError = false
		GOFSMockFstrimResult = nil
		GOFSMockBlkdiscardResult = nil
		GOFSMockDiscardCapability = nil
	})
}

// withMockFS sets the package-level fs to a mockfs instance and restores
// the original on test cleanup. Returns the original fs for inspection if needed.
func withMockFS(t *testing.T) {
	t.Helper()
	origFS := fs
	UseMockFS()
	t.Cleanup(func() { fs = origFS })
}

// ---- Contract Tests ---------------------------------------------------------

// Test ID: C-001
// Package-level Fstrim() delegates to fs.Fstrim() which delegates to fs.fstrim().
// Verifies the full call chain through the FSinterface.
func TestFstrim_PackageLevelDelegatesToFS(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)

	ctx := context.Background()
	result, err := Fstrim(ctx, "/mnt/data")

	require.NoError(t, err)
	require.NotNil(t, result, "Fstrim should return a non-nil result via mock")
	assert.Equal(t, int64(1073741824), result.BytesTrimmed,
		"Expected default mock BytesTrimmed of 1 GiB (1073741824)")
}

// Test ID: C-002
// Package-level Blkdiscard() delegates to fs.Blkdiscard() which delegates to fs.blkdiscard().
// Verifies the full call chain through the FSinterface.
func TestBlkdiscard_PackageLevelDelegatesToFS(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)

	ctx := context.Background()
	result, err := Blkdiscard(ctx, "/dev/sda")

	require.NoError(t, err)
	require.NotNil(t, result, "Blkdiscard should return a non-nil result via mock")
	assert.Equal(t, int64(107374182400), result.BytesDiscarded,
		"Expected default mock BytesDiscarded of 100 GiB (107374182400)")
}

// Test ID: C-003
// Package-level CheckDiscardSupport() delegates through the full call chain.
// Verifies the full call chain through the FSinterface.
func TestCheckDiscardSupport_PackageLevelDelegatesToFS(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)

	ctx := context.Background()
	result, err := CheckDiscardSupport(ctx, "/dev/sda")

	require.NoError(t, err)
	require.NotNil(t, result, "CheckDiscardSupport should return a non-nil result via mock")
	assert.True(t, result.Supported, "Expected default mock Supported == true")
	assert.Equal(t, int64(4294967295), result.DiscardMaxBytes,
		"Expected default mock DiscardMaxBytes of 4294967295")
}

// Test ID: C-004
// GOFSMock.InduceFstrimError causes Fstrim() to return error through mock chain.
func TestMockFstrim_InduceError(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)
	GOFSMock.InduceFstrimError = true

	ctx := context.Background()
	result, err := Fstrim(ctx, "/mnt/data")

	require.Error(t, err, "Expected an error when InduceFstrimError is true")
	assert.Nil(t, result, "Expected nil result when error is induced")
	assert.Contains(t, err.Error(), "fstrim induced error",
		"Error should contain 'fstrim induced error'")
}

// Test ID: C-005
// GOFSMock.InduceBlkdiscardError causes Blkdiscard() to return error through mock chain.
func TestMockBlkdiscard_InduceError(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)
	GOFSMock.InduceBlkdiscardError = true

	ctx := context.Background()
	result, err := Blkdiscard(ctx, "/dev/sda")

	require.Error(t, err, "Expected an error when InduceBlkdiscardError is true")
	assert.Nil(t, result, "Expected nil result when error is induced")
	assert.Contains(t, err.Error(), "blkdiscard induced error",
		"Error should contain 'blkdiscard induced error'")
}

// Test ID: C-006
// GOFSMock.InduceCheckDiscardSupportError causes CheckDiscardSupport() to return error.
func TestMockCheckDiscardSupport_InduceError(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)
	GOFSMock.InduceCheckDiscardSupportError = true

	ctx := context.Background()
	result, err := CheckDiscardSupport(ctx, "/dev/sda")

	require.Error(t, err, "Expected an error when InduceCheckDiscardSupportError is true")
	assert.Nil(t, result, "Expected nil result when error is induced")
	assert.Contains(t, err.Error(), "checkDiscardSupport induced error",
		"Error should contain 'checkDiscardSupport induced error'")
}

// Test ID: C-007
// GOFSMockFstrimResult overrides the default mock return value.
func TestMockFstrim_CustomResult(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)
	GOFSMockFstrimResult = &FstrimResult{BytesTrimmed: 42}

	ctx := context.Background()
	result, err := Fstrim(ctx, "/mnt/data")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil custom result")
	assert.Equal(t, int64(42), result.BytesTrimmed,
		"Expected custom BytesTrimmed value of 42")
}

// Test ID: C-008
// GOFSMockBlkdiscardResult overrides the default mock return value.
func TestMockBlkdiscard_CustomResult(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)
	GOFSMockBlkdiscardResult = &BlkdiscardResult{BytesDiscarded: 999}

	ctx := context.Background()
	result, err := Blkdiscard(ctx, "/dev/sda")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil custom result")
	assert.Equal(t, int64(999), result.BytesDiscarded,
		"Expected custom BytesDiscarded value of 999")
}

// Test ID: C-009
// GOFSMockDiscardCapability overrides the default mock return value.
func TestMockCheckDiscardSupport_CustomCapability(t *testing.T) {
	withMockFS(t)
	resetGOFSMock(t)
	GOFSMockDiscardCapability = &DiscardCapability{Supported: false, Reason: "test"}

	ctx := context.Background()
	result, err := CheckDiscardSupport(ctx, "/dev/sda")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil custom capability")
	assert.False(t, result.Supported, "Expected custom Supported == false")
	assert.Equal(t, "test", result.Reason,
		"Expected custom Reason 'test'")
}

// ---- Unit Tests: Mock Method Defaults (U-030 to U-032) ---------------------

// Test ID: U-030
// Mock fstrim() returns default FstrimResult when no overrides are set.
func TestMockFstrim_DefaultResult(t *testing.T) {
	resetGOFSMock(t)
	m := &mockfs{}

	ctx := context.Background()
	result, err := m.fstrim(ctx, "/mnt/data")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil default mock result")
	assert.Equal(t, int64(1073741824), result.BytesTrimmed,
		"Expected default BytesTrimmed of 1 GiB")
	assert.Equal(t, 500*time.Millisecond, result.Duration,
		"Expected default Duration of 500ms")
}

// Test ID: U-031
// Mock blkdiscard() returns default BlkdiscardResult when no overrides are set.
func TestMockBlkdiscard_DefaultResult(t *testing.T) {
	resetGOFSMock(t)
	m := &mockfs{}

	ctx := context.Background()
	result, err := m.blkdiscard(ctx, "/dev/sda")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil default mock result")
	assert.Equal(t, int64(107374182400), result.BytesDiscarded,
		"Expected default BytesDiscarded of 100 GiB")
	assert.Equal(t, 2*time.Second, result.Duration,
		"Expected default Duration of 2s")
}

// Test ID: U-032
// Mock checkDiscardSupport() returns default DiscardCapability when no overrides are set.
func TestMockCheckDiscardSupport_DefaultResult(t *testing.T) {
	resetGOFSMock(t)
	m := &mockfs{}

	ctx := context.Background()
	result, err := m.checkDiscardSupport(ctx, "/dev/sda")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil default mock result")
	assert.True(t, result.Supported, "Expected default Supported == true")
	assert.Equal(t, int64(4294967295), result.DiscardMaxBytes,
		"Expected default DiscardMaxBytes of 4294967295")
	assert.Empty(t, result.Reason, "Expected empty Reason for supported device")
}

// ---- Unit Tests: Type Zero Values (U-036 to U-038) -------------------------

// Test ID: U-036
// FstrimResult zero value has BytesTrimmed == 0 and Duration == 0.
func TestFstrimResult_ZeroValue(t *testing.T) {
	result := FstrimResult{}
	assert.Equal(t, int64(0), result.BytesTrimmed,
		"Zero-value FstrimResult should have BytesTrimmed == 0")
	assert.Equal(t, time.Duration(0), result.Duration,
		"Zero-value FstrimResult should have Duration == 0")
}

// Test ID: U-037
// BlkdiscardResult zero value has BytesDiscarded == 0 and Duration == 0.
func TestBlkdiscardResult_ZeroValue(t *testing.T) {
	result := BlkdiscardResult{}
	assert.Equal(t, int64(0), result.BytesDiscarded,
		"Zero-value BlkdiscardResult should have BytesDiscarded == 0")
	assert.Equal(t, time.Duration(0), result.Duration,
		"Zero-value BlkdiscardResult should have Duration == 0")
}

// Test ID: U-038
// DiscardCapability zero value has Supported == false, DiscardMaxBytes == 0, Reason == "".
func TestDiscardCapability_ZeroValue(t *testing.T) {
	dc := DiscardCapability{}
	assert.False(t, dc.Supported,
		"Zero-value DiscardCapability should have Supported == false")
	assert.Equal(t, int64(0), dc.DiscardMaxBytes,
		"Zero-value DiscardCapability should have DiscardMaxBytes == 0")
	assert.Equal(t, "", dc.Reason,
		"Zero-value DiscardCapability should have Reason == empty string")
}
