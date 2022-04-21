package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

type IndexSettings struct {
	Replicas int `json:"number_of_replicas"`
	Shards   int `json:"number_of_shards"`
}

func DefaultIndexSettings() IndexSettings {
	return IndexSettings{DefaultReplicas, DefaultShards}
}

func CreateOrUpdateIndex(ctx context.Context, esClient *elastic.Client, indexSettings IndexSettings, index, mapping string, ch chan struct{}) error {
	attempt := 0
	for {
		attempt++
		err := ensureIndexExists(ctx, esClient, indexSettings, index, mapping)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"index":       index,
				"attempt":     attempt,
				"retry_delay": CreateIndexFailureDelay,
			}).Errorf("Failed to create index")

			select {
			case <-ctx.Done():
				return err
			case <-time.After(CreateIndexFailureDelay):
				// noop
			}
		} else {
			close(ch)
			return nil
		}
	}
}

func ensureIndexExists(ctx context.Context, esClient *elastic.Client, indexSettings IndexSettings, idx, mapping string) error {
	exists, err := esClient.IndexExists(idx).Do(ctx)
	if err != nil {
		return err
	}
	if !exists {
		r, err := esClient.CreateIndex(idx).BodyJson(map[string]interface{}{
			"mappings": json.RawMessage(mapping),
			"settings": indexSettings,
		}).Do(ctx)
		if err != nil {
			return err
		}
		if !r.Acknowledged {
			return fmt.Errorf("not acknowledged index %s create", idx)
		}
	}
	return nil
}
