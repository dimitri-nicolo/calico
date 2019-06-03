// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"encoding/json"
	"fmt"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

const AnomalyDetectionType = "anomaly_detection"

type XPackSecurityEvent struct {
	Description string             `json:"description"`
	Severity    int                `json:"severity"`
	Record      elastic.RecordSpec `json:"-"`
}

func (s XPackSecurityEvent) ID() string {
	jobID := s.Record.Id
	if jobID == "" {
		jobID = "unknown"
	}
	return fmt.Sprintf("%s-%d-%d-%s-%d-%q-%q-%q",
		AnomalyDetectionType,
		s.Record.Timestamp.Unix(),
		s.Record.BucketSpan,
		jobID,
		s.Record.DetectorIndex,
		s.Record.PartitionFieldValue,
		s.Record.OverFieldValue,
		s.Record.ByFieldValue,
	)
}

func (s XPackSecurityEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type        string `json:"type"`
		Time        int64  `json:"time"`
		Description string `json:"description"`
		Severity    int    `json:"severity"`
		elastic.RecordSpec
	}{
		AnomalyDetectionType,
		s.Record.Timestamp.Unix(),
		s.Description,
		s.Severity,
		s.Record,
	})
}

func (s *XPackSecurityEvent) UnmarshalJSON(data []byte) error {
	e := struct {
		Type        string `json:"type"`
		Description string `json:"description"`
		Severity    int    `json:"severity"`
	}{}

	err := json.Unmarshal(data, &e)
	if err != nil {
		return err
	}

	if e.Type != AnomalyDetectionType {
		return fmt.Errorf("Invalid type: %s", e.Type)
	}

	*s = XPackSecurityEvent{
		Description: e.Description,
		Severity:    e.Severity,
	}

	return json.Unmarshal(data, &s.Record)
}
