package gofsutil

import (
	"errors"
	"path/filepath"
	"regexp"
)

func validatePath(path string) error {
	if path == "/" {
		return errors.New("Path: " + path + " is invalid")
	}

	return nil
}

func validateFsType(fsType string) error {
	if fsType != "ext4" && fsType != "ext3" &&
		fsType != "xfs" && fsType != "nfs" {
		return errors.New("FsType: " + fsType + " is invalid")
	}

	return nil
}

func validateMountOptions(mountOptions ...string) error {
	for _, opt := range mountOptions {
		matched, err := regexp.Match(`[\w]+[=]*[\w]*`, []byte(opt))
		if !matched || err != nil {
			return errors.New("Mount option: " + opt + " is invalid")
		}
	}
	return nil
}

func validateMultipathArgs(options ...string) error {
	for _, opt := range options {
		// check for options
		matched, err := regexp.Match(`[[-][AaBbCcdFfhilpqrTtUuWw0-9]+]*[0-9]*`, []byte(opt))
		if matched && err == nil {
			continue
		}

		// check for file or device path
		if err := validatePath(filepath.Clean(opt)); err != nil {
			return errors.New("Multipath option: " + opt + " is invalid")
		}
	}

	return nil
}
