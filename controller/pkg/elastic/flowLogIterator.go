package elastic

import (
	"context"
	"encoding/json"
	"io"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type Scroller interface {
	Do(context.Context) (*elastic.SearchResult, error)
}

type elasticFlowLogIterator struct {
	scroll Scroller
	ctx    context.Context
	hits   []*elastic.SearchHit
	val    db.FlowLog
	err    error
}

func (i *elasticFlowLogIterator) Next() bool {
	for {
		if len(i.hits) == 0 {
			r, err := i.scroll.Do(i.ctx)
			if err == io.EOF {
				return false
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

			var val db.FlowLog
			err := json.Unmarshal(*hit.Source, &val)
			if err != nil {
				log.WithError(err).WithField("raw", *hit.Source).Error("could not unmarshal")
			} else {
				i.val = val
				return true
			}
		}
	}
}

func (i *elasticFlowLogIterator) Value() db.FlowLog {
	return i.val
}

func (i *elasticFlowLogIterator) Err() error {
	return i.err
}
