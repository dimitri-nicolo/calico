// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package main

import (
	"fmt"
)

var VERSION, BUILD_DATE, GIT_DESCRIPTION, GIT_REVISION string

func Version() {
	fmt.Println("Version:     ", VERSION)
	fmt.Println("Build date:  ", BUILD_DATE)
	fmt.Println("Git tag ref: ", GIT_DESCRIPTION)
	fmt.Println("Git commit:  ", GIT_REVISION)
}
