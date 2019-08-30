// Copyright 2019 Tigera Inc. All rights reserved.

package ut

// Code coverage barfs on modules that contain nothing but test code, so we
// we define a random function to satisfy it.

func Fake() bool {
	return true
}
