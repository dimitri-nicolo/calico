// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package config

import (
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func isNflogSizeAvailable() bool {
	// bail out immediately for windows
	if runtime.GOOS == "windows" {
		return false
	}

	f, err := os.Open("/proc/version")
	if err != nil {
		log.WithError(err).Errorf("Unable to determine kernel version: %v", err)
		return false
	}
	procKernel, err := ioutil.ReadAll(f)
	if err != nil {
		log.WithError(err).Errorf("Failed to read /proc/kernel: %v", err)
		return false
	}
	// Expect proc version to be of form "Linux version 4.4.0-66-generic (buildd@lgw01-28)..."
	// Extracting the kernel major and minor version based on the core "4.4.0-66-generic" indexed at 2.
	version := strings.Split(strings.Split(string(procKernel), " ")[2], ".")
	majVer, err := strconv.Atoi(version[0])
	if err != nil {
		log.WithError(err).Errorf("Failed reading major version: %v", err)
		return false
	}
	minVer, err := strconv.Atoi(version[1])
	if err != nil {
		log.WithError(err).Errorf("Failed reading minor version: %v", err)
		return false
	}

	// nflog-size supported 4.8+
	if majVer >= 4 && minVer >= 8 {
		log.Infof("nflog-size enabled")
		return true
	}
	return false
}
