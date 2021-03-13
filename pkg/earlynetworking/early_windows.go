// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package earlynetworking

import (
	"fmt"
	"os"
)

func Run() {
	fmt.Println("Early networking not supported on Windows.")
	os.Exit(1)
}
