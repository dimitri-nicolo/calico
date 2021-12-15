// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package testutils

import (
	"os"
	"path"
)

func TestDataFile(name string) string {
	dir, _ := os.Getwd()

	return path.Join(dir, "testdata", name)
}
