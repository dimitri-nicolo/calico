// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/libcalico-go/lib/errors"

	"github.com/tigera/compliance/pkg/list"
)

func (c *client) RetrieveList(kind schema.GroupVersionKind, from, to *time.Time, ascending bool) (*list.TimestampedResourceList, error) {
	clog := log.WithField("kind", kind)
	// Construct the range query based on received arguments.
	rangeQuery := elastic.NewRangeQuery("requestCompletedTimestamp")
	if from != nil {
		rangeQuery = rangeQuery.From(*from)
	}
	if to != nil {
		rangeQuery = rangeQuery.To(*to)
	}

	// Execute query.
	res, err := c.Search().
		Index(snapshotsIndex).
		Query(
			elastic.NewBoolQuery().Must(
				elastic.NewTermQuery("apiVersion", kind.GroupVersion().String()),
				elastic.NewTermQuery("kind", kind.Kind),
				rangeQuery,
			)).
		Sort("requestCompletedTimestamp", ascending).
		Size(1). // Only retrieve the first document found.
		Do(context.Background())
	if err != nil {
		clog.WithError(err).Error("failed to execute query")
		return nil, err
	}
	clog.WithField("latency (ms)", res.TookInMillis).Debug("query success")

	// Should only return one document.
	switch len(res.Hits.Hits) {
	case 0:
		clog.Error("no hits found")
		return nil, errors.ErrorResourceDoesNotExist{}
	case 1:
		break
	default:
		clog.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	l := new(list.TimestampedResourceList)
	if err = json.Unmarshal(*hit.Source, l); err != nil {
		clog.WithError(err).Error("failed to extract list from result")
		return nil, err
	}

	return l, nil
}

//TODO(rlb): What is the ES Id all about?
func (c *client) StoreList(_ schema.GroupVersionKind, l *list.TimestampedResourceList) error {
	res, err := c.Index().
		Index(snapshotsIndex).
		Type("_doc").
		Id(l.String()).
		BodyJson(l).
		Do(context.Background())
	if err != nil {
		log.WithError(err).Error("failed to store list")
		return err
	}
	log.WithFields(log.Fields{"id": res.Id, "index": res.Index, "type": res.Type}).
		Info("successfully stored list")
	return nil
}
