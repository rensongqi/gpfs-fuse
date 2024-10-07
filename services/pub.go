// Package services is public func
package services

import (
	logger "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"os"
	"strconv"
	"strings"
)

// Determines whether a file has a property
func hasSpecialAttribute(path string) bool {
	linkPath, err := os.Readlink(path)
	if err != nil {
		logger.Errorf("linkPath: %s, err: %v", linkPath, err)
		return false
	}
	if strings.HasPrefix(linkPath, "oss") {
		return true
	}
	return false
}

func getLinkFileSize(path string) uint64 {
	linkPath, err := os.Readlink(path)
	if err != nil {
		logger.Errorf("linkPath: %s, err: %v", linkPath, err)
		return 0
	}
	linkPathSplit := strings.Split(linkPath, "/")
	if len(linkPathSplit) >= 2 {
		size, err := strconv.ParseUint(linkPathSplit[1], 10, 64)
		if err != nil {
			logger.Errorf("linkPath: %s, err: %v", linkPath, err)
			return 0
		}
		return size
	}
	return 0
}

// readlink 
func readlink(fullPath string) (string, error) {
	buf := make([]byte, 1024)
	n, err := unix.Readlink(fullPath, buf)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	return string(buf[:n]), err
}
