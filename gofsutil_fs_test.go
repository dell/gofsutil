package gofsutil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFsInfo(t *testing.T) {
	tests := []struct {
		testname  string
		ctx       context.Context
		path      string
		induceErr bool
		expected  struct {
			available  int64
			capacity   int64
			usage      int64
			inodes     int64
			inodesFree int64
			inodesUsed int64
			err        error
		}
	}{
		{
			testname:  "Normal operation",
			path:      "/path",
			induceErr: false,
			expected: struct {
				available  int64
				capacity   int64
				usage      int64
				inodes     int64
				inodesFree int64
				inodesUsed int64
				err        error
			}{
				available:  1000,
				capacity:   2000,
				usage:      1000,
				inodes:     4,
				inodesFree: 2,
				inodesUsed: 2,
				err:        nil,
			},
		},
		{
			testname:  "Induced error",
			path:      "/path",
			induceErr: true,
			expected: struct {
				available  int64
				capacity   int64
				usage      int64
				inodes     int64
				inodesFree int64
				inodesUsed int64
				err        error
			}{
				available:  0,
				capacity:   0,
				usage:      0,
				inodes:     0,
				inodesFree: 0,
				inodesUsed: 0,
				err:        errors.New("filesystemInfo induced error: Failed to get filesystem stats"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testname, func(t *testing.T) {
			fs := &mockfs{}
			GOFSMock.InduceFilesystemInfoError = tt.induceErr
			available, capacity, usage, inodes, inodesFree, inodesUsed, err := fs.fsInfo(tt.ctx, tt.path)

			assert.Equal(t, tt.expected.available, available)
			assert.Equal(t, tt.expected.capacity, capacity)
			assert.Equal(t, tt.expected.usage, usage)
			assert.Equal(t, tt.expected.inodes, inodes)
			assert.Equal(t, tt.expected.inodesFree, inodesFree)
			assert.Equal(t, tt.expected.inodesUsed, inodesUsed)
			assert.Equal(t, tt.expected.err, err)
		})
	}
}
