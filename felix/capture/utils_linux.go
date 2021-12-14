// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package capture

import "syscall"

// GetFreeDiskSize returns free bytes using Statfs system call for linux OS using
// Bavail (Free blocks available to unprivileged user) multiplied by Frsize (Fragment size)
// or an error otherwise
func GetFreeDiskSize(dir string) (uint64, error) {
	var stat syscall.Statfs_t
	var err = syscall.Statfs(dir, &stat)
	if err != nil {
		return 0, err
	}

	return stat.Bavail * uint64(stat.Frsize), nil
}
