// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package capture

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MinTime represents minimal value a Time value can be assigned
// 0001-01-01 00:00:00 +0000 UTC
var MinTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

// MaxTime represents the maximum value a Time value can be assigned
// 9999-12-31 23:59:00 +0000 UTC
var MaxTime = time.Date(9999, 12, 31, 23, 59, 0, 0, time.UTC)

// RenderStartTime will render the start time for a PacketCapture as time.Time structure
// to be used internally. If no value is assigned, MinTime will be returned
func RenderStartTime(input *metav1.Time) time.Time {
	if input == nil {
		return MinTime
	}

	return input.Time
}

// RenderEndTime will render the end time for a PacketCapture as time.Time structure
// to be used internally. If no value is assigned, MaxTime will be returned
func RenderEndTime(input *metav1.Time) time.Time {
	if input == nil {
		return MaxTime
	}

	return input.Time
}
