// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package capture

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	GetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// GetFreeDiskSize returns free bytes using GetDiskFreeSpaceEx system call for windows OS using
// total number of free bytes on a disk
// (https://docs.microsoft.com/en-ca/windows/win32/api/fileapi/nf-fileapi-getdiskfreespaceexa?redirectedfrom=MSDN)
// or an error otherwise
func GetFreeDiskSize(dir string) (uint64, error) {
	freeBytesAvailable := uint64(0)
	totalNumberOfBytes := uint64(0)
	totalNumberOfFreeBytes := uint64(0)

	_, _, err := GetDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(dir))),
		uintptr(unsafe.Pointer(&freeBytesAvailable)), uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)))

	return totalNumberOfFreeBytes, err
}
