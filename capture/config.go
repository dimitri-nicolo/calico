// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package capture

// Config represents PacketCapture configuration
type Config struct {
	Directory       string
	MaxSizeBytes    int
	RotationSeconds int
	MaxFiles        int
}
