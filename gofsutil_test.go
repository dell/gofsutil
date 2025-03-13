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
)

func TestGetDiskFormat(t *testing.T) {
	originalGetGofsutilGetDiskFormat := getGofsutilGetDiskFormat

	defer func() {
		getGofsutilGetDiskFormat = originalGetGofsutilGetDiskFormat
	}()

	getGofsutilGetDiskFormat = func(_ FSinterface, _ context.Context, _ string) (string, error) {
		return "string", nil
	}
	testCases := []struct {
		name    string
		disk    string
		wantErr bool
	}{
		{
			name:    "valid disk",
			disk:    "/dev/sda",
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			fs := &mockfs{}

			if _, err := fs.GetDiskFormat(context.Background(), tt.disk); (err != nil) != tt.wantErr {
				t.Errorf("GetDiskFormat() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestFormatAndMount(t *testing.T) {
	originalGetGofsutilFormatAndMount := getGofsutilFormatAndMount

	defer func() {
		getGofsutilFormatAndMount = originalGetGofsutilFormatAndMount
	}()

	getGofsutilFormatAndMount = func(_ FSinterface, _ context.Context, _ string, _ string, _ string, _ ...string) error {
		return nil
	}
	testCases := []struct {
		name    string
		source  string
		target  string
		fsType  string
		options []string
		wantErr bool
	}{
		{
			name:    "valid inputs",
			source:  "/dev/sda",
			target:  "/mnt",
			fsType:  "ext4",
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.FormatAndMount(context.Background(), tt.source, tt.target, tt.fsType, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatAndMount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	originalGetGofsutilFormat := getGofsutilFormat

	defer func() {
		getGofsutilFormat = originalGetGofsutilFormat
	}()

	getGofsutilFormat = func(_ FSinterface, _ context.Context, _ string, _ string, _ string, _ ...string) error {
		return nil
	}
	testCases := []struct {
		name    string
		source  string
		target  string
		fsType  string
		options []string
		wantErr bool
	}{
		{
			name:    "valid inputs",
			source:  "/dev/sda",
			target:  "/mnt",
			fsType:  "ext4",
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.Format(context.Background(), tt.source, tt.target, tt.fsType, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMount(t *testing.T) {
	originalGetGofsutilMount := getGofsutilMount

	defer func() {
		getGofsutilMount = originalGetGofsutilMount
	}()

	getGofsutilMount = func(_ FSinterface, _ context.Context, _ string, _ string, _ string, _ ...string) error {
		return nil
	}
	testCases := []struct {
		name    string
		source  string
		target  string
		fsType  string
		options []string
		wantErr bool
	}{
		{
			name:    "valid inputs",
			source:  "/dev/sda",
			target:  "/mnt",
			fsType:  "ext4",
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.Mount(context.Background(), tt.source, tt.target, tt.fsType, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBindMount(t *testing.T) {
	originalGetGofsutilBindMount := getGofsutilBindMount

	defer func() {
		getGofsutilBindMount = originalGetGofsutilBindMount
	}()

	getGofsutilBindMount = func(_ FSinterface, _ context.Context, _ string, _ string, _ ...string) error {
		return nil
	}

	testCases := []struct {
		name    string
		source  string
		target  string
		options []string
		wantErr bool
	}{
		{
			name:    "valid inputs",
			source:  "/dev/sda",
			target:  "/mnt",
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.BindMount(context.Background(), tt.source, tt.target, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
