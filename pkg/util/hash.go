// Copyright 2021 Tigera Inc. All rights reserved.

package util

import (
	"crypto/sha256"
	"fmt"
)

func ComputeSha256Hash(obj interface{}) string {
	encoder := sha256.New()
	encoder.Write([]byte(fmt.Sprintf("%v", obj)))
	return fmt.Sprintf("%x", encoder.Sum(nil))
}
