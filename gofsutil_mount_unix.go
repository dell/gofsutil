//go:build linux || darwin
// +build linux darwin

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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

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

// PowerMaxOUIPrefix - PowerMax format 6 OUI prefix
var PowerMaxOUIPrefix = "6000097"

// PowerStoreOUIPrefix - PowerStore format 6 OUI prefix
var PowerStoreOUIPrefix = "68ccf09"

var (
	isBindFunc = func(fs *FS, ctx context.Context, opts ...string) ([]string, bool) {
		return fs.isBind(ctx, opts...)
	}

	bindMountFunc = func(fs *FS, ctx context.Context, source, target string, opts ...string) error {
		return fs.bindMount(ctx, source, target, opts...)
	}

	doMountFunc = func(fs *FS, ctx context.Context, command, source, target, fsType string, opts ...string) error {
		return fs.doMount(ctx, command, source, target, fsType, opts...)
	}
)

func (fs *FS) mount(
	ctx context.Context,
	source, target, fsType string,
	opts ...string,
) error {
	// All Linux distributes should support bind mounts.
	if opts, ok := isBindFunc(fs, ctx, opts...); ok {
		return bindMountFunc(fs, ctx, source, target, opts...)
	}
	return doMountFunc(fs, ctx, "mount", source, target, fsType, opts...)
}

// validateMountArgs validates the arguments for mount operation.
func (fs *FS) validateMountArgs(source, target, fsType string, opts ...string) error {
	sourcePath := filepath.Clean(source)
	targetPath := filepath.Clean(target)

	if err := validatePath(sourcePath); err != nil {
		return err
	}

	if err := validatePath(targetPath); err != nil {
		return err
	}

	if fsType != "" {
		if err := validateFsType(fsType); err != nil {
			return err
		}
	}

	return validateMountOptions(opts...)
}

// doMount runs the mount command.
func (fs *FS) doMount(
	ctx context.Context,
	mntCmd, source, target, fsType string,
	opts ...string,
) error {
	if err := fs.validateMountArgs(source, target, fsType, opts...); err != nil {
		return err
	}

	mountArgs := MakeMountArgs(ctx, source, target, fsType, opts...)
	args := strings.Join(mountArgs, " ")

	f := log.Fields{
		"cmd":  mntCmd,
		"args": args,
	}
	log.WithFields(f).Info("mount command")
	/* #nosec G204 */
	buf, err := exec.Command(mntCmd, mountArgs...).CombinedOutput()
	if err != nil {
		out := string(buf)
		// check is explicitly placed for PowerScale driver only
		if !(strings.Contains(args, "/ifs") && (strings.Contains(strings.ToLower(out), "access denied by server while mounting") || strings.Contains(strings.ToLower(out), "no such file or directory"))) {
			log.WithFields(f).WithField("output", out).WithError(
				err).Error("mount Failed")
		}
		return fmt.Errorf(
			"mount failed: %v\nmounting arguments: %s\noutput: %s",
			err, args, out)
	}
	return nil
}

// unmount unmounts the target.
func (fs *FS) unmount(_ context.Context, target string) error {
	f := log.Fields{
		"path": target,
		"cmd":  "umount",
	}
	log.WithFields(f).Info("unmount syscall")
	path := filepath.Clean(target)
	if err := validatePath(path); err != nil {
		return err
	}

	err := syscall.Unmount(path, 0)
	if err != nil {
		log.WithFields(f).WithError(err).Error("unmount failed")
		return fmt.Errorf(
			"unmount failed: %v\nunmounting arguments: %s",
			err, target)
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
func (fs *FS) isBind(_ context.Context, opts ...string) ([]string, bool) {
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

var (
	lstatFunc = func(name string) (os.FileInfo, error) {
		return os.Lstat(name)
	}

	evalSymlinksFunc = func(ctx context.Context, path *string) error {
		return EvalSymlinks(ctx, path)
	}

	statFunc = func(name string) (os.FileInfo, error) {
		return os.Stat(name)
	}
)

func (fs *FS) validateDevice(
	ctx context.Context, source string,
) (string, error) {
	if _, err := lstatFunc(source); err != nil {
		return "", err
	}

	// Eval symlinks to ensure the specified path points to a real device.
	if err := evalSymlinksFunc(ctx, &source); err != nil {
		return "", err
	}

	st, err := statFunc(source)
	if err != nil {
		return "", err
	}

	if st.Mode()&os.ModeDevice == 0 {
		return "", fmt.Errorf("invalid device: %s", source)
	}

	return source, nil
}

// wwnToDevicePath looks up a volume WWN in /dev/disk/by-id
// and returns a) the symlink path in /dev/disk/by-id and
// b) the corresponding device entry in /dev.
func (fs *FS) wwnToDevicePath(
	_ context.Context, wwn string,
) (string, string, error) {
	// Look for multipath device.
	symlinkPath := fmt.Sprintf("%s%s", MultipathDevDiskByIDPrefix, wwn)
	devPath, err := os.Readlink(symlinkPath)

	// Look for nvme path device.
	if err != nil || devPath == "" {
		symlinkPath = filepath.Join(multipathDevDiskByID, fmt.Sprintf("nvme-eui.%s", wwn))
		devPath, err = os.Readlink(symlinkPath)
		if err != nil || devPath == "" {
			// Look for normal path device
			symlinkPath = filepath.Join(multipathDevDiskByID, fmt.Sprintf("wwn-0x%s", wwn))

			devPath, err = os.Readlink(symlinkPath)
			if err != nil {
				log.Printf("Check for disk path %s not found", symlinkPath)
				return "", "", err
			}
		}
	}
	components := strings.Split(devPath, "/")
	lastPart := components[len(components)-1]
	devPath = "/dev/" + lastPart
	log.Printf("Check for disk path %s found: %s", symlinkPath, devPath)
	return symlinkPath, devPath, err
}

// targetIPLUNToDevicePath returns all the /dev/disk/by-path entries for a give targetIP and lunID
func (fs *FS) targetIPLUNToDevicePath(_ context.Context, targetIP string, lunID int) (map[string]string, error) {
	result := make(map[string]string, 0)

	entries, err := os.ReadDir(bypathdir)
	if err != nil {
		log.Printf("%s not found: %s", bypathdir, err.Error())
		return result, err
	}
	// Loop through the entries
	for _, entry := range entries {
		name := entry.Name()
		// Looking for entries of these forms:
		// ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-0 -> ../../sdc
		// ip-1.1.1.1:3260-iscsi-iqn.1992-04.com.emc:600009700bcbb70e3287017400000000-lun-0x0101000000000000 -> ../../sdro
		if !strings.HasPrefix(name, "ip-"+targetIP+":") {
			continue
		}
		if !(strings.HasSuffix(name, fmt.Sprintf("-lun-%d", lunID)) ||
			strings.HasSuffix(name, fmt.Sprintf("-lun-0x%04x000000000000", lunID))) {
			continue
		}
		// Look up the symbolic link
		path := bypathdir + "/" + name
		devPath, err := os.Readlink(path)
		if err != nil {
			log.Printf("Check for disk path %s not found", path)
			return result, err
		}
		components := strings.Split(devPath, "/")
		lastPart := components[len(components)-1]
		devPath = "/dev/" + lastPart
		log.Printf("Check for disk path %s found: %s", path, devPath)
		result[path] = devPath
	}
	return result, nil
}

// targetdev for a rescan operation
type targetdev struct {
	host    string
	channel string
	target  string
}

func (td *targetdev) String() string {
	return fmt.Sprintf("%s:%s:%s", td.host, td.channel, td.target)
}

// rescanSCSIHost will rescan scsi hosts for a specified lun.
// If targets are specified, only hosts who are related to the specified
// iqn target(s) are rescanned.
// If lun is specified, then the rescan is for that particular volume.
func (fs *FS) rescanSCSIHost(_ context.Context, targets []string, lun string) error {
	var err error
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

	iscsiTargets, fcTargets := splitTargets(targets)
	targetDevices, err := getFCTargetHosts(fcTargets)
	if err != nil {
		return err
	}
	log.Printf("iscsiTargets: %s; fcTargets: %s", iscsiTargets, targetDevices)

	iscsiTargetDevices, err := getIscsiTargetHosts(iscsiTargets)
	if err != nil {
		return err
	}
	targetDevices = append(targetDevices, iscsiTargetDevices...)

	hostsdir := "/sys/class/scsi_host"
	if len(targetDevices) > 0 {
		for _, entry := range targetDevices {
			scanfile := fmt.Sprintf("%s/%s/scan", hostsdir, entry.host)
			scanstring := fmt.Sprintf("%s %s %s", entry.channel, entry.target, lun)
			log.Printf("rescanning %s with: "+scanstring, scanfile)
			f, err := os.OpenFile(filepath.Clean(scanfile), os.O_APPEND|os.O_WRONLY, 0o200)
			if err != nil {
				log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to open scanfile")
				continue
			}
			if _, err := f.WriteString(scanstring); err != nil {
				log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to write rescan file")
			}
			errs := f.Close()
			if errs != nil {
				return err
			}
		}
		return nil
	}

	// Fallback... we didn't find any target devices... so rescan all the hosts
	// Gather up the host devices.
	log.Printf("No targeted devices found... rescanning all the hosts")
	hosts, err := os.ReadDir(hostsdir)
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
		f, err := os.OpenFile(filepath.Clean(scanfile), os.O_APPEND|os.O_WRONLY, 0o200)
		if err != nil {
			log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to open scanfile")
			continue
		}
		if _, err := f.WriteString(scanstring); err != nil {
			log.WithFields(log.Fields{"file": scanfile, "error": err}).Error("Failed to write rescan file")
		}
		errs := f.Close()
		if errs != nil {
			return err
		}
	}
	return nil
}

// FCPortPrefix has the required port prefix for FCTargetHosts
const FCPortPrefix = "0x50"

// getFCTargetHosts adds the list of the fibre channel hosts in /sys/class/scsi_host to be rescanned,
// The targets are a list of array port WWNs in the port group used. They must start with 0x50 and
// be of the form 0x50000973b000b804 as an example.
// along with the channel and target, to the targetdev list.
func getFCTargetHosts(targets []string) ([]*targetdev, error) {
	targetDev := make([]*targetdev, 0)
	duplicates := make(map[string]bool)
	if len(targets) == 0 {
		return targetDev, nil
	}
	// Read the directory entries for fc_remote_ports
	remotePortEntries, err := os.ReadDir(fcRemotePortsDir)
	if err != nil {
		log.WithField("error", err).Error("Cannot read directory: " + fcRemotePortsDir)
	}

	// Look through
	for _, remotePort := range remotePortEntries {
		if !strings.HasPrefix(remotePort.Name(), "rport-") {
			continue
		}
		log.Debug("Processing fc_remote_port: " + remotePort.Name())

		if !strings.HasPrefix(remotePort.Name(), "rport-") {
			continue
		}

		arrayPortNameBytes, err := os.ReadFile(fcRemotePortsDir + "/" + remotePort.Name() + "/" + "port_name")
		if err != nil {
			continue
		}
		arrayPortName := strings.TrimSpace(string(arrayPortNameBytes))
		if !strings.HasPrefix(arrayPortName, FCPortPrefix) {
			continue
		}

		// Check that the arrayPortName matches one of our targets
		for _, tg := range targets {
			if tg == arrayPortName {
				split := strings.Split(remotePort.Name(), ":")
				if len(split) >= 2 {
					entry := new(targetdev)
					entry.host = strings.Replace(split[0], "rport-", "host", 1)
					entry.channel = "-"
					entry.target = "-"
					if !duplicates[entry.host] {
						targetDev = append(targetDev, entry)
						log.Debug(fmt.Sprintf("Adding target: %s", entry))
						duplicates[entry.host] = true
					}
				}
			}
		}
	}
	return targetDev, nil
}

// getIscsiTargetHosts adds the list of the scsi hosts in /sys/class/scsi_host to be rescanned,
// along with the channel and target, to the targetdev list.
func getIscsiTargetHosts(targets []string) ([]*targetdev, error) {
	targetDev := make([]*targetdev, 0)
	if len(targets) == 0 {
		return targetDev, nil
	}
	// Read the sessions.
	sessions, err := os.ReadDir(sessionsdir)
	if err != nil {
		log.WithField("error", err).Error("Cannot read directory: " + sessionsdir)
		return targetDev, err
	}
	// Look through the iscsi sessions
	for _, session := range sessions {
		if !strings.HasPrefix(session.Name(), "session") {
			continue
		}
		log.Debug("Processing iscsi_session: " + session.Name())
		if len(targets) > 0 {
			targetBytes, err := os.ReadFile(sessionsdir + "/" + session.Name() + "/" + "targetname")
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
		devices, err := os.ReadDir(devicedir)
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
					targetDev = append(targetDev, entry)
					log.Debug(fmt.Sprintf("Adding target: %s", entry))
				}
				break
			}
		}
	}
	return targetDev, nil
}

// Splits the targeets into those for iscsi or fibre channel
func splitTargets(targets []string) ([]string, []string) {
	iscsiTargets := make([]string, 0)
	fibreChannelTargets := make([]string, 0)
	for _, target := range targets {
		if strings.HasPrefix(target, "iqn.") {
			iscsiTargets = append(iscsiTargets, target)
		} else if strings.HasPrefix(target, FCPortPrefix) {
			fibreChannelTargets = append(fibreChannelTargets, target)
		} else {
			log.Error("unknown target: " + target)
		}
	}
	return iscsiTargets, fibreChannelTargets
}

// removeBlockDevice removes a block device by getting the device name
// from the last component of the blockDevicePath and then removing the
// device by writing '1' to /sys/block{deviceName}/device/delete
func (fs *FS) removeBlockDevice(_ context.Context, blockDevicePath string) error {
	// Here we want to remove /sys/block/{deviceName} by writing a 1 to
	// /sys/block{deviceName}/device/delete
	devicePathComponents := strings.Split(blockDevicePath, "/")
	if len(devicePathComponents) > 1 {
		deviceName := devicePathComponents[len(devicePathComponents)-1]
		statePath := filepath.Join(sysBlockDir, fmt.Sprintf("%s/device/state", deviceName))
		stateBytes, err := os.ReadFile(filepath.Clean(statePath))
		if err != nil {
			return fmt.Errorf("Cannot read %s: %s", statePath, err)
		}
		deviceState := strings.TrimSpace(string(stateBytes))
		if deviceState == "blocked" {
			return fmt.Errorf("Device %s is in blocked state", deviceName)
		}
		blockDeletePath := filepath.Join(sysBlockDir, fmt.Sprintf("%s/device/delete", deviceName))
		f, err := os.OpenFile(filepath.Clean(blockDeletePath), os.O_APPEND|os.O_WRONLY, 0o200)
		if err != nil {
			log.WithField("BlockDeletePath", blockDeletePath).Error("Could not open delete block device delete path")
			return err
		}
		log.WithField("BlockDeletePath", blockDeletePath).Info("Writing '1' to block device delete path")
		if _, err := f.WriteString("1"); err != nil {
			log.WithField("BlockDeletePath", blockDeletePath).Error("Could not write to block device delete path")
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// Execute the multipath command with a timeout and various arguments.
// Optionally a chroot directory can be specified for changing root directory.
// This only works in a container or another environment where it can chroot to /noderoot.
// When the -f <dev-name> option has been specified, the flush seems to happen but the
// command seems to hang. The reason is currently unknown.
func (fs *FS) multipathCommand(ctx context.Context, timeoutSeconds time.Duration, chroot string, arguments ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	args := make([]string, 0)

	if err := validateMultipathArgs(arguments...); err != nil {
		return nil, err
	}

	if chroot == "" {
		args = append(args, arguments...)
		log.Printf("/usr/sbin/multipath %v", args)
		/* #nosec G204 */
		cmd = exec.CommandContext(ctx, "/usr/sbin/multipath", args...)
	} else {
		args = append(args, chroot)
		args = append(args, "/usr/sbin/multipath")
		args = append(args, arguments...)
		log.Printf("/usr/sbin/chroot %v", args)
		/* #nosec G204 */
		cmd = exec.CommandContext(ctx, "/usr/sbin/chroot", args...)
	}
	textBytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("multipath command failed: " + err.Error())
	}
	if len(textBytes) > 0 {
		log.Debug(fmt.Printf("multipath output: %s", string(textBytes)))
	}
	return textBytes, err
}

// getFCHostPortWWNs returns the port WWN addresses of local FC adapters.
func (fs *FS) getFCHostPortWWNs(_ context.Context) ([]string, error) {
	portWWNs := make([]string, 0)
	// Read the directory entries for fc_remote_ports
	hostEntries, err := os.ReadDir(fcHostsDir)
	if err != nil {
		log.WithField("error", err).Error("Cannot read directory: " + fcHostsDir)
		return portWWNs, err
	}

	// Look through the hosts retrieving the port_name
	for _, host := range hostEntries {
		if !strings.HasPrefix(host.Name(), "host") {
			continue
		}

		hostPortNameBytes, err := os.ReadFile(fcHostsDir + "/" + host.Name() + "/" + "port_name")
		if err != nil {
			continue
		}
		hostPortName := strings.TrimSpace(string(hostPortNameBytes))
		portWWNs = append(portWWNs, hostPortName)
	}
	return portWWNs, nil
}

// issueLIPToAllFCHosts issues the LIP command to all FC hosts.
func (fs *FS) issueLIPToAllFCHosts(_ context.Context) error {
	var savedError error
	// Read the directory entries for fc_remote_ports
	fcHostEntries, err := os.ReadDir(fcHostsDir)
	if err != nil {
		log.WithField("error", err).Error("Cannot read directory: " + fcHostsDir)
	}

	// Look through the fc_hosts
	for _, hostEntry := range fcHostEntries {
		if !strings.HasPrefix(hostEntry.Name(), "host") {
			continue
		}

		lipFile := fmt.Sprintf("%s/%s/issue_lip", fcHostsDir, hostEntry.Name())
		lipString := fmt.Sprintf("%s", "1")
		log.Printf("issuing lip command %s to %s", lipString, lipFile)
		f, err := os.OpenFile(filepath.Clean(lipFile), os.O_APPEND|os.O_WRONLY, 0o200)
		if err != nil {
			log.Error("Could not open issue_lip file at: " + lipFile)
			continue
		}
		if _, err := f.WriteString(lipString); err != nil {
			log.Error(fmt.Sprintf("Error issuing lip at %s: %s", lipFile, err))
			savedError = err
		}
		errs := f.Close()
		if errs != nil {
			return err
		}
	}
	return savedError
}

// getSysBlockDevicesForVolumeWWN given a volumeWWN will return a list of devices in /sys/block for that WWN (e.g. sdx, sdaa)
func (fs *FS) getSysBlockDevicesForVolumeWWN(_ context.Context, volumeWWN string) ([]string, error) {
	start := time.Now()
	result := make([]string, 0)
	sysBlocks, err := os.ReadDir(sysBlockDir)
	if err != nil {
		return result, fmt.Errorf("Error reading %s: %s", sysBlockDir, err)
	}

	for _, sysBlock := range sysBlocks {
		name := sysBlock.Name()
		// Check for both "sd" and "nvme" prefixes
		if !strings.HasPrefix(name, "sd") && !strings.HasPrefix(name, "nvme") {
			continue
		}

		// Set the WWID path based on the device type
		var wwidPath string
		if strings.HasPrefix(name, "nvme") {
			wwidPath = sysBlockDir + "/" + name + "/wwid" // For NVMe devices
		} else {
			wwidPath = sysBlockDir + "/" + name + "/device/wwid" // For SCSI devices
		}

		bytes, err := os.ReadFile(filepath.Clean(wwidPath))
		if err != nil {
			continue
		}

		wwid := strings.TrimSpace(string(bytes))

		// Replace "eui." for NVMe devices and "naa." for others
		if strings.HasPrefix(name, "nvme") {
			wwid = strings.Replace(wwid, "eui.", "", 1)
			// Use wwnMatches for NVMe comparison
			if wwnMatches(wwid, volumeWWN) {
				result = append(result, name)
			}
		} else {
			wwid = strings.Replace(wwid, "naa.", "", 1)
			// Compare directly for SCSI devices
			if wwid == volumeWWN {
				result = append(result, name)
			}
		}
	}

	end := time.Now()
	dur := end.Sub(start)
	log.Printf("getSysBlockDevicesForVolumeWWN %d %f", len(sysBlocks), dur.Seconds())
	return result, nil
}

func wwnMatches(nguid, wwn string) bool {
	/*
			// PowerStore
			Sample wwn : naa.68ccf098001111a2222b3d4444a1b23c
			token1: 1111a2222b3d4444
			token2: a1b23c
			Sample nguid : 1111a2222b3d44448ccf096800a1b23c

			// PowerMax
			nguid: 12635330303134340000976000012000
			wwn:   60000970000120001263533030313434
		           11aaa111111111a11a111a1111aa1111
		           1a111a1111aa1111 1aaa11 1 1111111a1
			nguid: wwn[last16] 		+ wwn[1:6] 	+ wwn[0] + wwn[7:15]
				   1263533030313434 + 000097 	+ 6		 + 000012000
	*/
	if len(wwn) < 32 {
		return false
	}

	wwn = strings.ToLower(wwn)
	if strings.HasPrefix(wwn, "naa.") {
		wwn = wwn[4:]
	}

	var token1, token2 string
	if strings.HasPrefix(wwn, PowerStoreOUIPrefix) {
		token1 = wwn[13 : len(wwn)-7]
		token2 = wwn[len(wwn)-6 : len(wwn)-1]
		log.Infof("PowerStore: %s %s %s %t", token1, token2, nguid, strings.Contains(nguid, token2))
		if strings.Contains(nguid, token1) && strings.Contains(nguid, token2) {
			return true
		}
	} else if strings.HasPrefix(wwn, PowerMaxOUIPrefix) {
		token1 = wwn[16:]
		token2 = wwn[1:7]
		log.Infof("Powermax: %s %s %s %t", token1, token2, nguid, strings.HasPrefix(nguid, token1+token2))
		if strings.HasPrefix(nguid, token1+token2) {
			return true
		}
	}

	return false
}

// GetNVMeController retrieves the NVMe controller for a given NVMe device.
func (fs *FS) getNVMeController(device string) (string, error) {
	devicePath := filepath.Join(sysBlockDir, device)

	// Check if the device path exists
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return "", fmt.Errorf("device %s does not exist", device)
	}

	// Resolve the symlink to find the actual path in /sys/device e.g. /sys/devices/virtual/nvme-fabrics/ctl/nvme0/nvme0n1
	realPath, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return "", fmt.Errorf("error resolving symlink for %s: %v", device, err)
	}

	isNvmeController := false
	// Split the path and look for the controller in /sys/class/nvme
	pathParts := strings.Split(realPath, "/")
	for i, part := range pathParts {
		if strings.Contains(part, "ctl") {
			isNvmeController = true
		} else if isNvmeController && part == device {
			// The controller is the part right before the device name
			if i > 0 && strings.HasPrefix(pathParts[i-1], "nvme") {
				return pathParts[i-1], nil
			}
		}
	}

	if !isNvmeController {
		log.Infof("Not a valid nvme controller device: %s ", device)
		return "", nil
	}

	return "", fmt.Errorf("controller not found for device %s", device)
}
