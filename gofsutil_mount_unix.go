// +build linux darwin

package gofsutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// mount mounts source to target as fsType with given options.
//
// The parameters 'source' and 'fsType' must be empty strings in case they
// are not required, e.g. for remount, or for an auto filesystem type where
// the kernel handles fsType automatically.
//
// The 'options' parameter is a list of options. Please see mount(8) for
// more information. If no options are required then please invoke Mount
// with an empty or nil argument.
func (fs *FS) mount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string) error {

	// All Linux distributes should support bind mounts.
	if opts, ok := fs.isBind(ctx, opts...); ok {
		return fs.bindMount(ctx, source, target, opts...)
	}
	return fs.doMount(ctx, "mount", source, target, fsType, opts...)
}

// doMount runs the mount command.
func (fs *FS) doMount(
	ctx context.Context,
	mntCmd, source, target, fsType string,
	opts ...string) error {

	mountArgs := MakeMountArgs(ctx, source, target, fsType, opts...)
	args := strings.Join(mountArgs, " ")

	f := log.Fields{
		"cmd":  mntCmd,
		"args": args,
	}
	log.WithFields(f).Info("mount command")

	buf, err := exec.Command(mntCmd, mountArgs...).CombinedOutput()
	if err != nil {
		out := string(buf)
		log.WithFields(f).WithField("output", out).WithError(
			err).Error("mount Failed")
		return fmt.Errorf(
			"mount failed: %v\nmounting arguments: %s\noutput: %s",
			err, args, out)
	}
	return nil
}

// unmount unmounts the target.
func (fs *FS) unmount(ctx context.Context, target string) error {
	f := log.Fields{
		"path": target,
		"cmd":  "umount",
	}
	log.WithFields(f).Info("unmount command")
	buf, err := exec.Command("umount", target).CombinedOutput()
	if err != nil {
		out := string(buf)
		f["output"] = out
		log.WithFields(f).WithError(err).Error("unmount failed")
		return fmt.Errorf(
			"unmount failed: %v\nunmounting arguments: %s\nOutput: %s",
			err, target, out)
	}
	return nil
}

// isBind detects whether a bind mount is being requested and determines
// which remount options are needed. A secondary mount operation is
// required for bind mounts as the initial operation does not apply the
// request mount options.
//
// The returned options will be "bind", "remount", and the provided
// list of options.
func (fs *FS) isBind(ctx context.Context, opts ...string) ([]string, bool) {
	bind := false
	remountOpts := append([]string(nil), bindRemountOpts...)

	for _, o := range opts {
		switch o {
		case "bind":
			bind = true
			break
		case "remount":
			break
		default:
			remountOpts = append(remountOpts, o)
		}
	}

	return remountOpts, bind
}

// getDevMounts returns a slice of all mounts for dev
func (fs *FS) getDevMounts(ctx context.Context, dev string) ([]Info, error) {

	allMnts, err := fs.getMounts(ctx)
	if err != nil {
		return nil, err
	}

	var mountInfos []Info
	for _, m := range allMnts {
		if m.Device == dev {
			mountInfos = append(mountInfos, m)
		}
	}

	return mountInfos, nil
}

func (fs *FS) validateDevice(
	ctx context.Context, source string) (string, error) {

	if _, err := os.Lstat(source); err != nil {
		return "", err
	}

	// Eval symlinks to ensure the specified path points to a real device.
	if err := EvalSymlinks(ctx, &source); err != nil {
		return "", err
	}

	st, err := os.Stat(source)
	if err != nil {
		return "", err
	}

	if st.Mode()&os.ModeDevice == 0 {
		return "", fmt.Errorf("invalid device: %s", source)
	}

	return source, nil
}

// wwnToDevicePath looks up a volume WWN in /dev/disk/by-id
// and returns the corresponding device entry in /dev.
func (fs *FS) wwnToDevicePath(
	ctx context.Context, wwn string) (string, error) {
	path := fmt.Sprintf("/dev/disk/by-id/wwn-0x%s", wwn)
	devPath, err := os.Readlink(path)
	if err != nil {
		log.Printf("Check for disk path %s not found", path)
		return "", err
	}
	components := strings.Split(devPath, "/")
	lastPart := components[len(components)-1]
	devPath = "/dev/" + lastPart
	log.Printf("Check for disk path %s found: %s", path, devPath)
	return devPath, err
}

// rescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// iqn target(s) are rescanned.
// If lun is specified, then the rescan is for that particular volume.
func (fs *FS) rescanSCSIHost(ctx context.Context, targets []string, lun string) error {
	// If no lun is specifed, the "-" character is a wildcard that will update all LUNs.
	if lun == "" {
		lun = "-"
	} else {
		// The lun string is speficied as a hex string by Powermax.
		// We need decimal for the scan, so do the conversion.
		val, err := strconv.ParseInt(lun, 16, 32)
		if err == nil {
			lun = strconv.Itoa(int(val))
		} else {
			lun = "-"
		}
	}

	type targetdev struct {
		host    string
		channel string
		target  string
	}
	var targetDevices []*targetdev

	// Read the sessions.
	sessionsdir := "/sys/class/iscsi_session"
	sessions, err := ioutil.ReadDir(sessionsdir)
	if err != nil {
		log.WithField("error", err).Error("Cannot read directory: " + sessionsdir)
		return err
	}
	// Look through the iscsi sessions
	for _, session := range sessions {
		if !strings.HasPrefix(session.Name(), "session") {
			continue
		}
		log.Debug("Processing session: " + session.Name())
		if len(targets) > 0 {
			targetBytes, err := ioutil.ReadFile(sessionsdir + "/" + session.Name() + "/" + "targetname")
			if err != nil {
				continue
			}
			target := strings.Trim(string(targetBytes), "\n\r\t ")
			var hasTarget bool
			for _, tg := range targets {
				if tg == target {
					hasTarget = true
				}
			}
			if !hasTarget {
				continue
			}
		}
		// Read device/target entry to get the data for rescan.
		devicedir := sessionsdir + "/" + session.Name() + "/" + "device"
		devices, err := ioutil.ReadDir(devicedir)
		if err != nil {
			log.WithField("error", err).Error("Cannot read directory: " + devicedir)
			continue
		}
		// Loop through the devices for the target* one
		for _, device := range devices {
			if strings.HasPrefix(device.Name(), "target") {
				name := device.Name()[6:]
				split := strings.Split(name, ":")
				if len(split) >= 3 {
					entry := new(targetdev)
					entry.host = "host" + split[0]
					entry.channel = split[1]
					entry.target = split[2]
					targetDevices = append(targetDevices, entry)
					log.Debug(fmt.Sprintf("Adding targetdev: %#v", entry))
				}
				break
			}
		}
	}

	hostsdir := "/sys/class/scsi_host"
	if len(targetDevices) > 0 {
		for _, entry := range targetDevices {
			scanfile := fmt.Sprintf("%s/%s/scan", hostsdir, entry.host)
			scanstring := fmt.Sprintf("%s %s %s", entry.channel, entry.target, lun)
			log.Printf("rescanning %s with: "+scanstring, scanfile)
			f, err := os.OpenFile(scanfile, os.O_APPEND|os.O_WRONLY, 0200)
			if err != nil {
				log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to open scanfile")
				continue
			}
			if _, err := f.WriteString(scanstring); err != nil {
				log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to write rescan file")
			}
			f.Close()
		}
		return nil
	}

	// Fallback... we didn't find any target devices... so rescan all the hosts
	// Gather up the host devices.
	hosts, err := ioutil.ReadDir(hostsdir)
	if err != nil {
		log.WithField("error", err).Error("Cannot read directory: " + hostsdir)
		return err
	}
	// For each of the matching hosts, perform a rescan.
	for _, host := range hosts {
		if !strings.HasPrefix(host.Name(), "host") {
			continue
		}
		scanfile := fmt.Sprintf("%s/%s/scan", hostsdir, host.Name())
		scanstring := fmt.Sprintf("- - %s", lun)
		log.Printf("rescanning %s with: "+scanstring, scanfile)
		f, err := os.OpenFile(scanfile, os.O_APPEND|os.O_WRONLY, 0200)
		if err != nil {
			log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to open scanfile")
			continue
		}
		if _, err := f.WriteString(scanstring); err != nil {
			log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to write rescan file")
		}
		f.Close()
	}
	return nil
}

// removeBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func (fs *FS) removeBlockDevice(ctx context.Context, blockDevicePath string) error {
	// Here we want to remove /sys/block/{deviceName} by writing a 1 to
	// /sys/block{deviceName}/device/delete
	devicePathComponents := strings.Split(blockDevicePath, "/")
	if len(devicePathComponents) > 1 {
		deviceName := devicePathComponents[len(devicePathComponents)-1]
		blockDeletePath := fmt.Sprintf("/sys/block/%s/device/delete", deviceName)
		f, err := os.OpenFile(blockDeletePath, os.O_APPEND|os.O_WRONLY, 0200)
		if err != nil {
			log.WithField("BlockDeletePath", blockDeletePath).Error("Could not open delete block device delete path")
			return err
		} else {
			log.WithField("BlockDeletePath", blockDeletePath).Info("Writing '1' to block device delete path")
			if _, err := f.WriteString("1"); err != nil {
				log.WithField("BlockDeletePath", blockDeletePath).Error("Could not write to block device delete path")
			}
			f.Close()
		}
	}
	return nil
}
