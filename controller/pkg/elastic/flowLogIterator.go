// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"io"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
)

type Scroller interface {
	Do(context.Context) (*elastic.SearchResult, error)
}

type scrollerEntry struct {
	name     string
	scroller Scroller
	terms    []interface{}
}

type flowLogIterator struct {
	scrollers []scrollerEntry
	ctx       context.Context
	name      string
	hits      []*elastic.SearchHit
	key       string
	val       events.SuspiciousIPSecurityEvent
	err       error
}

func (i *flowLogIterator) Next() bool {
	for len(i.scrollers) > 0 {
		if len(i.hits) == 0 {
			entry := i.scrollers[0]
			i.key = entry.name
			scroller := entry.scroller

			r, err := scroller.Do(i.ctx)
			if err == io.EOF {
				i.scrollers = i.scrollers[1:]
				continue
			}
			if err != nil {
				i.err = err
				return false
			}

			log.WithField("hits", r.TotalHits()).Info("elastic query returned")
			i.hits = r.Hits.Hits
		}

		for len(i.hits) > 0 {
			hit := i.hits[0]
			i.hits = i.hits[1:]

			var flowLog events.FlowLogJSONOutput
			err := json.Unmarshal(*hit.Source, &flowLog)
			if err != nil {
				log.WithError(err).WithField("raw", *hit.Source).Error("could not unmarshal")
				continue
			}

			i.val = events.ConvertFlowLog(flowLog, i.key, hit, i.name)

			return true
		}
	}

	return false
}

func (i *flowLogIterator) Value() db.SecurityEventInterface {
	return i.val
}

func (i *flowLogIterator) Err() error {
	return i.err
}
