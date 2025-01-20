package gofsutil

import (
	"context"
	"testing"
)

func TestGetDiskFormat(t *testing.T) {
	ctx := context.Background()
	disk := "/dev/sda1" // Replace with a valid disk path for your system

	result, err := GetDiskFormat(ctx, disk)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Disk format: %s", result)
}

// func TestFormatAndMount(t *testing.T) {
// 	ctx := context.Background()
// 	source := "/dev/sda1" // Replace with a valid source path for your system
// 	target := "/mnt/test" // Replace with a valid target path for your system
// 	fsType := "ext4"
// 	opts := []string{"defaults"}

// 	err := FormatAndMount(ctx, source, target, fsType, opts...)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

// func TestMount(t *testing.T) {
// 	ctx := context.Background()
// 	source := "/dev/sda1" // Replace with a valid source path for your system
// 	target := "/mnt/test" // Replace with a valid target path for your system
// 	fsType := "ext4"
// 	opts := []string{"defaults"}

// 	err := Mount(ctx, source, target, fsType, opts...)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

// func TestBindMount(t *testing.T) {
// 	ctx := context.Background()
// 	source := "/dev/sda1" // Replace with a valid source path for your system
// 	target := "/mnt/test" // Replace with a valid target path for your system
// 	opts := []string{"defaults"}

// 	err := BindMount(ctx, source, target, opts...)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

// func TestUnmount(t *testing.T) {
// 	ctx := context.Background()
// 	target := "/mnt/test" // Replace with a valid target path for your system

// 	err := Unmount(ctx, target)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

func TestGetMountInfoFromDevice(t *testing.T) {
	ctx := context.Background()
	devID := "sda1" // Replace with a valid device ID for your system

	result, err := GetMountInfoFromDevice(ctx, devID)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Mount info: %+v", result)
}

func TestGetMpathNameFromDevice(t *testing.T) {
	ctx := context.Background()
	device := "sda1" // Replace with a valid device name for your system

	result, err := GetMpathNameFromDevice(ctx, device)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Mpath name: %s", result)
}

// func TestResizeFS(t *testing.T) {
// 	ctx := context.Background()
// 	volumePath := "/mnt/test" // Replace with a valid volume path for your system
// 	devicePath := "/dev/sda1" // Replace with a valid device path for your system
// 	ppathDevice := ""
// 	mpathDevice := ""
// 	fsType := "ext4"

// 	err := ResizeFS(ctx, volumePath, devicePath, ppathDevice, mpathDevice, fsType)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

// func TestResizeMultipath(t *testing.T) {
// 	ctx := context.Background()
// 	deviceName := "mpath0" // Replace with a valid device name for your system

// 	err := ResizeMultipath(ctx, deviceName)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

func TestFindFSType(t *testing.T) {
	ctx := context.Background()
	mountpoint := "/mnt/test" // Replace with a valid mount point for your system

	result, err := FindFSType(ctx, mountpoint)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Filesystem type: %s", result)
}

// func TestDeviceRescan(t *testing.T) {
// 	ctx := context.Background()
// 	devicePath := "/dev/sda1" // Replace with a valid device path for your system

// 	err := DeviceRescan(ctx, devicePath)
// 	if err != nil {
// 		t.Errorf("expected no error, got %v", err)
// 	}
// }

func TestGetMounts(t *testing.T) {
	ctx := context.Background()

	result, err := GetMounts(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Mounts: %+v", result)
}

func TestGetSysBlockDevicesForVolumeWWN(t *testing.T) {
	ctx := context.Background()
	volumeWWN := "60000970000120000549533030354435" // Replace with a valid volume WWN for your system

	result, err := GetSysBlockDevicesForVolumeWWN(ctx, volumeWWN)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("Sys block devices: %+v", result)
}
