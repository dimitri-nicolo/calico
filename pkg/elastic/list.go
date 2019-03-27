package elastic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/errors"

	"github.com/tigera/compliance/pkg/list"
)

func (c *Client) RetrieveList(tm metav1.TypeMeta, from time.Time) (*list.TimestampedResourceList, error) {
	// Execute query.
	res, err := c.Search().
		Index(snapshotsIndex).
		//TODO(rlb): Shouldn't this include the api version too?
		Query(
			elastic.NewBoolQuery().Must(
				elastic.NewTermQuery("kind", tm.Kind),
				elastic.NewRangeQuery("timestamp").From(from))).
		Sort("timestamp", true).
		Size(1). // Only retrieve the first document found.
		Do(context.Background())
	if err != nil {
		log.WithError(err).Error("failed to execute query")
		return nil, err
	}
	log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

	// Should only return one document.
	switch len(res.Hits.Hits) {
	case 0:
		log.Error("no hits found")
		return nil, &errors.ErrorResourceDoesNotExist{}
	case 1:
		break
	default:
		log.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	l := new(list.TimestampedResourceList)
	if err = json.Unmarshal(*hit.Source, l); err != nil {
		log.WithError(err).Error("failed to extract list from result")
		return nil, err
	}

	return l, nil
}

//TODO(rlb): What is the ES Id all about?
func (c *Client) StoreList(_ metav1.TypeMeta, l *list.TimestampedResourceList) error {
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
