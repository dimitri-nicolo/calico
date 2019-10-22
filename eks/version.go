// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import "fmt"

var BUILD_DATE, GIT_COMMIT, GIT_TAG, VERSION string

func PrintVersion() {
	fmt.Printf("BuildDate\t: %s\n", BUILD_DATE)
	fmt.Printf("GitCommit\t: %s\n", GIT_COMMIT)
	fmt.Printf("GitTag\t\t: %s\n", GIT_TAG)
	fmt.Printf("Version\t\t: %s\n", VERSION)
}
