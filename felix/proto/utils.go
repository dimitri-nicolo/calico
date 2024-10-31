// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package proto

import (
	"time"

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"
)

// ConvertTime converts a time.Time structure into gogo types Timestamp
func ConvertTime(time time.Time) *types.Timestamp {
	var val, err = types.TimestampProto(time)
	if err != nil {
		log.WithError(err).Panic("Failed to convert time to timestamp")
	}
	return val
}

// ConvertTimestamp converts a gogo types Timestamp structure into a time.Time
func ConvertTimestamp(timestamp *types.Timestamp) time.Time {
	var val, err = types.TimestampFromProto(timestamp)
	if err != nil {
		log.WithError(err).Panic("Failed to convert timestamp to time")
	}
	return val
}
