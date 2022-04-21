// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"io"

	"github.com/tigera/intrusion-detection/controller/pkg/db"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

// Scroller is a subset of elastic.ScrollService
type Scroller interface {
	Do(context.Context) (*elastic.SearchResult, error)
	Clear(context.Context) error
}

type scrollerEntry struct {
	key      db.QueryKey
	scroller Scroller
	terms    []interface{}
}

type Iterator interface {
	Next() bool
	Value() (key db.QueryKey, hit *elastic.SearchHit)
	Err() error
}

type queryIterator struct {
	scrollers []scrollerEntry
	ctx       context.Context
	name      string
	hits      []*elastic.SearchHit
	key       db.QueryKey
	val       *elastic.SearchHit
	err       error
}

func (i *queryIterator) Next() bool {
	for len(i.scrollers) > 0 {
		if len(i.hits) == 0 {
			entry := i.scrollers[0]
			i.key = entry.key
			scroller := entry.scroller

			r, err := scroller.Do(i.ctx)
			if err == io.EOF {
				if err := scroller.Clear(i.ctx); err != nil {
					i.err = err
					return false
				}
				i.scrollers = i.scrollers[1:]
				continue
			}
			if err != nil {
				i.err = err
				return false
			}

			log.WithField("hits", r.TotalHits()).Debug("elastic query returned")
			i.hits = r.Hits.Hits
		}

		if len(i.hits) > 0 {
			i.val = i.hits[0]
			i.hits = i.hits[1:]
			return true
		}
	}

	return false
}

func (i *queryIterator) Value() (db.QueryKey, *elastic.SearchHit) {
	return i.key, i.val
}

func (i *queryIterator) Err() error {
	return i.err
}

func newQueryIterator(ctx context.Context, scrollers []scrollerEntry, name string) Iterator {
	return &queryIterator{ctx: ctx, scrollers: scrollers, name: name}
}
