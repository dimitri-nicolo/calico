package elastic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1beta1"

	"github.com/tigera/compliance/pkg/event"
)

const (
	auditLogIndex = "tigera_secure_ee_audit_*"
	pageSize      = 100
)

func (c *client) GetAuditEvents(ctx context.Context, kind *metav1.TypeMeta, start, end *time.Time) <-chan *event.AuditEventResult {
	// create the channel that the retrieved events will fill into.
	ch := make(chan *event.AuditEventResult, pageSize)

	// retrieve the events on a goroutine.
	go func() {
		defer close(ch)

		// Query for audit events in a paginated fashion
		exit := false
		for i := 0; !exit; i += pageSize {
			// Construct query.
			queries := []elastic.Query{}

			// Query by TypeMeta if specified.
			if kind != nil {
				queries = append(queries, elastic.NewMatchQuery("responseObject.kind", kind.Kind))
				queries = append(queries, elastic.NewMatchQuery("responseObject.apiVersion", kind.APIVersion))
			}

			// Query by from/to if specified.
			if start != nil || end != nil {
				rangeQuery := elastic.NewRangeQuery("stageTimestamp")
				if start != nil {
					rangeQuery = rangeQuery.From(*start)
				}
				if end != nil {
					rangeQuery = rangeQuery.To(*end)
				}
				queries = append(queries, rangeQuery)
			}

			// Make search query
			res, err := c.Search().
				Index(auditLogIndex).
				Query(elastic.NewBoolQuery().Must(queries...)).
				Sort("stageTimestamp", true).
				From(i).Size(pageSize).
				Do(context.Background())
			if err != nil {
				ch <- &event.AuditEventResult{Err: err}
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// define function that pushes the search results into the channel.
			for _, hit := range res.Hits.Hits {
				ev := new(auditv1.Event)
				if err := json.Unmarshal(*hit.Source, ev); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
					ch <- &event.AuditEventResult{nil, err}
				}
				ch <- &event.AuditEventResult{ev, nil}
			}

			exit = i+pageSize > int(res.Hits.TotalHits)
		}
	}()

	return ch
}

type auditEventResult struct {
	*auditv1.Event
	Err error
}
