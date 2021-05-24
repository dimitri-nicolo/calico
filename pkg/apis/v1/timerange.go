// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/timeutils"
)

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type timeRangeInternal struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// UnmarshalJSON implements the unmarshalling interface for JSON.
func (t *TimeRange) UnmarshalJSON(b []byte) error {
	var err error

	// Just extract the timestamp and kind fields from the blob.
	s := new(timeRangeInternal)
	if err = json.Unmarshal(b, s); err != nil {
		log.WithError(err).Debug("Unable to unmarshal time")
		return err
	}

	now := time.Now().UTC()
	if from, _, err := timeutils.ParseElasticsearchTime(now, &s.From); err != nil {
		log.WithError(err).Debug("Unable to parse 'from' time")
		return err
	} else if to, _, err := timeutils.ParseElasticsearchTime(now, &s.To); err != nil {
		log.WithError(err).Debug("Unable to parse 'to' time")
		return err
	} else {
		t.From = *from
		t.To = *to
	}
	if t.From.After(t.To) {
		log.Debug("From is after To")
		return fmt.Errorf("incorrect time range specified: from (%s) is after to (%s)", s.From, s.To)
	}
	return nil
}

// MarshalJSON implements the marshalling interface for JSON. We need to implement this explicitly because the default
// implementation doesn't honor the "inline" directive when the parameter is an interface type.
func (t TimeRange) MarshalJSON() ([]byte, error) {
	// Just extract the timestamp and kind fields from the blob.
	s := timeRangeInternal{
		From: t.From.UTC().Format(time.RFC3339),
		To:   t.To.UTC().Format(time.RFC3339),
	}
	return json.Marshal(s)
}

func (t TimeRange) Duration() time.Duration {
	return t.To.Sub(t.From)
}

func (t TimeRange) InRange(t1 time.Time) bool {
	return !(t1.Before(t.From) || t1.After(t.To))
}

func (t TimeRange) Overlaps(from, to time.Time) bool {
	return !(to.Before(t.From) || from.After(t.To))
}
