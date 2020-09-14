// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package capture

// PacketCapture configuration
type Config struct {
	Directory       string
	MaxSizeBytes    int
	RotationSeconds int
	MaxFiles        int
}
