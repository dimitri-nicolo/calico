// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"

	"github.com/projectcalico/calico/lma/pkg/list"
)

func (c *client) RetrieveList(kind metav1.TypeMeta, from, to *time.Time, ascending bool) (*list.TimestampedResourceList, error) {
	clog := log.WithField("kind", kind)

	// Construct the range query based on received arguments.
	rangeQuery := elastic.NewRangeQuery("requestCompletedTimestamp")
	if from != nil {
		rangeQuery = rangeQuery.From(*from)
	}
	if to != nil {
		rangeQuery = rangeQuery.To(*to)
	}

	searchIndex := c.ClusterIndex(SnapshotsIndex, "*")
	// Execute query.
	res, err := c.Search().
		Index(searchIndex).
		Query(
			elastic.NewBoolQuery().Must(
				elastic.NewTermQuery("apiVersion", kind.APIVersion),
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
		clog.Info("no hits found")
		return nil, cerrors.ErrorResourceDoesNotExist{
			Err:        errors.New("insufficient archived data in Elastic"),
			Identifier: kind.Kind + "/" + kind.APIVersion,
		}
	case 1:
		break
	default:
		clog.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	l := new(list.TimestampedResourceList)
	if err = json.Unmarshal(hit.Source, l); err != nil {
		clog.WithError(err).Error("failed to extract list from result")
		return nil, err
	}

	return l, nil
}

func (c *client) StoreList(_ metav1.TypeMeta, l *list.TimestampedResourceList) error {
	index := c.ClusterAlias(SnapshotsIndex)
	snapshotsTemplate, err := c.IndexTemplate(index, SnapshotsIndex, snapshotsMapping, true)
	if err != nil {
		log.WithError(err).Error("failed to build index template")
		return err
	}

	if err := c.ensureIndexExistsWithRetry(SnapshotsIndex, snapshotsTemplate, true); err != nil {
		return err
	}
	res, err := c.Index().
		Index(index).
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
