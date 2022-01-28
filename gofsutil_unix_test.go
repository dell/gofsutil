//go:build linux || darwin
// +build linux darwin

package gofsutil_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/dell/gofsutil"
)

func TestFCRescanSCSIHost(t *testing.T) {
	var targets []string
	// Scan the remote ports to find the array port WWNs
	fcRemotePortsDir := "/sys/class/fc_remote_ports"
	remotePortEntries, err := ioutil.ReadDir(fcRemotePortsDir)
	if err != nil {
		t.Errorf("error reading %s: %s", fcRemotePortsDir, err)
	}
	for _, remotePort := range remotePortEntries {
		if !strings.HasPrefix(remotePort.Name(), "rport-") {
			continue
		}

		if !strings.HasPrefix(remotePort.Name(), "rport-") {
			continue
		}

		arrayPortNameBytes, err := ioutil.ReadFile(fcRemotePortsDir + "/" + remotePort.Name() + "/" + "port_name")
		if err != nil {
			continue
		}
		arrayPortName := strings.TrimSpace(string(arrayPortNameBytes))
		if !strings.HasPrefix(arrayPortName, gofsutil.FCPortPrefix) {
			continue
		}
		targets = append(targets, arrayPortName)

	}

	if len(targets) > 0 {
		err := gofsutil.RescanSCSIHost(context.Background(), targets, "1")
		if err != nil {
			t.Errorf("RescanSCSIHost failed: %s", err)
		}
	}
}

func TestGetFCHostPortWWNs(t *testing.T) {
	wwns, err := gofsutil.GetFCHostPortWWNs(context.Background())
	if err != nil {
		t.Error(err)
	}
	for _, wwn := range wwns {
		fmt.Printf("local FC port wwn: %s\n", wwn)
	}
}

func TestMountArgs(t *testing.T) {
	tests := []struct {
		src    string
		tgt    string
		fst    string
		opts   []string
		result string
	}{
		{
			src:    "localhost:/data",
			tgt:    "/mnt",
			fst:    "nfs",
			result: "-t nfs localhost:/data /mnt",
		},
		{
			src:    "localhost:/data",
			tgt:    "/mnt",
			result: "localhost:/data /mnt",
		},
		{
			src:    "localhost:/data",
			tgt:    "/mnt",
			fst:    "nfs",
			opts:   []string{"tcp", "vers=4"},
			result: "-t nfs -o tcp,vers=4 localhost:/data /mnt",
		},
		{
			src:    "/dev/disk/mydisk",
			tgt:    "/mnt/mydisk",
			fst:    "xfs",
			opts:   []string{"ro", "noatime", "ro"},
			result: "-t xfs -o ro,noatime /dev/disk/mydisk /mnt/mydisk",
		},
		{
			src:    "/dev/sdc",
			tgt:    "/mnt",
			opts:   []string{"rw", "", "noatime"},
			result: "-o rw,noatime /dev/sdc /mnt",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			opts := gofsutil.MakeMountArgs(
				context.TODO(), tt.src, tt.tgt, tt.fst, tt.opts...)
			optsStr := strings.Join(opts, " ")
			if optsStr != tt.result {
				t.Errorf("Formatting of mount args incorrect, got: %s want: %s",
					optsStr, tt.result)
			}
		})
	}
}

func TestWWNToDevicePath(t *testing.T) {
	tests := []struct {
		src    string
		tgt    string
		wwn    string
		result string
	}{
		{
			src:    "/dev/disk/by-id/wwn-0x60570970000197900046533030394146",
			tgt:    "../../mydeva",
			wwn:    "60570970000197900046533030394146",
			result: "/dev/mydeva",
		},
		{
			src:    "/dev/disk/by-id/dm-uuid-mpath-360570970000197900046533030394146",
			tgt:    "../../mydevb",
			wwn:    "60570970000197900046533030394146",
			result: "/dev/mydevb",
		},
	}
	for _, tt := range tests {
		t.Run("", func(st *testing.T) {
			// Change directories
			workingDirectory, _ := os.Getwd()
			err := os.Chdir("/dev/disk/by-id")
			if err != nil {
				t.Errorf("Couldn't Chdir to /dev/disk/by/id: %s", err)
			}
			// Create a target
			file, err := os.Create(tt.result)
			if err != nil {
				t.Errorf("Couldn't Create %s: %s", tt.result, err)
			}
			file.Close()
			// Create a symlink
			err = os.Symlink(tt.tgt, tt.src)
			if err != nil {
				t.Errorf("Couldn't create Symlink %s: %s", tt.tgt, err)
			}
			// Get the entry
			a, b, err := gofsutil.WWNToDevicePathX(context.Background(), tt.wwn)
			if err != nil {
				t.Errorf("Couldn't find DevicePathX: %s", err)
			}
			if a != tt.src {
				t.Errorf("Expected %s got %s", tt.src, a)
			}
			if b != tt.result {
				t.Errorf("Expected %s got %s", tt.result, b)
			}
			// Get the entry
			c, err := gofsutil.WWNToDevicePath(context.Background(), tt.wwn)
			if err != nil {
				t.Errorf("Couldn't find DevicePathX: %s", err)
			}
			if c != tt.result {
				t.Errorf("Expected %s got %s", tt.result, c)
			}
			// Remove symlink
			err = os.Remove(tt.src)
			if err != nil {
				t.Errorf("Couldn't remove %s: %s", tt.src, err)
			}
			// Remove target
			err = os.Remove(tt.result)
			if err != nil {
				t.Errorf("Couldn't remove %s: %s", tt.result, err)
			}
			// Change directories
			err = os.Chdir(workingDirectory)
			if err != nil {
				t.Errorf("Couldn't Chdir to /dev/disk/by/id: %s", err)
			}
		})
	}
}
