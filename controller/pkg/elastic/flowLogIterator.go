// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"io"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/events"
)

type Scroller interface {
	Do(context.Context) (*elastic.SearchResult, error)
}

type elasticFlowLogIterator struct {
	scrollers map[string]Scroller
	ctx       context.Context
	name      string
	hits      []*elastic.SearchHit
	key       string
	val       events.SecurityEvent
	err       error
}

func (i *elasticFlowLogIterator) Next() bool {
	for len(i.scrollers) > 0 {
		if len(i.hits) == 0 {
			var scroller Scroller

			// Get a random scroller for results
			for i.key, scroller = range i.scrollers {
				break
			}

			r, err := scroller.Do(i.ctx)
			if err == io.EOF {
				delete(i.scrollers, i.key)
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

func (i *elasticFlowLogIterator) Value() events.SecurityEvent {
	return i.val
}

func (i *elasticFlowLogIterator) Err() error {
	return i.err
}
