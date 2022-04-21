// Copyright 2021-2022 Tigera Inc. All rights reserved.

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

func ComputeSha256HashWithLimit(obj interface{}, limit int) string {
	hash := ComputeSha256Hash(obj)
	if len(hash) > limit {
		hash = hash[:limit]
	}

	return hash
}
