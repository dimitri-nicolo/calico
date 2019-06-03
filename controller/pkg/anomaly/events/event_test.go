// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestXPackSecurityEvent_ID(t *testing.T) {
	g := NewWithT(t)

	e := XPackSecurityEvent{
		Description: "test anomaly detection",
		Record: elastic.RecordSpec{
			Id:                  "test_job",
			PartitionFieldValue: "pf",
			OverFieldValue:      "of",
			ByFieldValue:        "bf",
			ResultType:          "record",
			DetectorIndex:       1,
			BucketSpan:          600,
			Timestamp: elastic.Time{
				Time: time.Unix(100, 0),
			},
		},
	}

	expected := fmt.Sprintf("%s-%d-%d-%s-%d-%q-%q-%q",
		AnomalyDetectionType,
		e.Record.Timestamp.Time.Unix(),
		e.Record.BucketSpan,
		e.Record.Id,
		e.Record.DetectorIndex,
		e.Record.PartitionFieldValue,
		e.Record.OverFieldValue,
		e.Record.ByFieldValue)
	g.Expect(e.ID()).Should(Equal(expected))
}

func TestXPackSecurityEvent_MarshalJSON(t *testing.T) {
	g := NewWithT(t)

	e := XPackSecurityEvent{
		Description: "test anomaly detection",
		Record: elastic.RecordSpec{
			ResultType: "record",
			Timestamp: elastic.Time{
				Time: time.Now(),
			},
		},
	}

	b, err := json.Marshal(&e)
	g.Expect(err).ShouldNot(HaveOccurred())

	actual := make(map[string]interface{})
	err = json.Unmarshal(b, &actual)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(actual["type"]).Should(Equal(AnomalyDetectionType))
	g.Expect(actual["description"]).Should(Equal(e.Description))
	g.Expect(actual["time"]).Should(BeNumerically("==", e.Record.Timestamp.Unix()))
	g.Expect(actual["result_type"]).Should(Equal(e.Record.ResultType))
}

func TestXPackSecurityEvent_UnmarshalJSON(t *testing.T) {
	g := NewWithT(t)

	e := map[string]interface{}{
		"type":        AnomalyDetectionType,
		"severity":    100,
		"description": "test detector",
		"result_type": "record",
	}

	b, err := json.Marshal(&e)
	g.Expect(err).ShouldNot(HaveOccurred())

	var actual XPackSecurityEvent
	err = json.Unmarshal(b, &actual)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(actual.Severity).Should(BeNumerically("==", e["severity"]))
	g.Expect(actual.Description).Should(Equal(e["description"]))
	g.Expect(actual.Record.ResultType).Should(Equal(e["result_type"]))
}

func TestXPackSecurityEvent_UnmarshalJSON_WrongType(t *testing.T) {
	g := NewWithT(t)

	e := map[string]interface{}{
		"type":        "something_else",
		"severity":    100,
		"description": "test detector",
		"result_type": "record",
	}

	b, err := json.Marshal(&e)
	g.Expect(err).ShouldNot(HaveOccurred())

	var actual XPackSecurityEvent
	err = json.Unmarshal(b, &actual)
	g.Expect(err).Should(HaveOccurred())
}

func TestXPackSecurityEvent_UnmarshalJSON_MissingType(t *testing.T) {
	g := NewWithT(t)

	e := map[string]interface{}{
		"severity":    100,
		"description": "test detector",
		"result_type": "record",
	}

	b, err := json.Marshal(&e)
	g.Expect(err).ShouldNot(HaveOccurred())

	var actual XPackSecurityEvent
	err = json.Unmarshal(b, &actual)
	g.Expect(err).Should(HaveOccurred())
}
