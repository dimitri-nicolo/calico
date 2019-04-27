package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/resources"
)

const (
	auditLogIndex = "tigera_secure_ee_audit_*"
	pageSize      = 100
)

func (c *client) GetAuditEvents(ctx context.Context, kind *metav1.TypeMeta, start, end *time.Time) <-chan *event.AuditEventResult {
	// create the channel that the retrieved events will fill into.
	var filter *v3.AuditEventsSelection

	// Create an audit event filter if needed
	if kind != nil {
		filter = &v3.AuditEventsSelection{
			Resources: []v3.ResourceID{v3.ResourceID{TypeMeta: metav1.TypeMeta{Kind: kind.Kind, APIVersion: kind.APIVersion}}}}
		for _, otherTM := range resources.GetResourceHelper(*kind).Deprecated() {
			filter.Resources = append(filter.Resources, v3.ResourceID{TypeMeta: otherTM})
		}
	}

	return c.SearchAuditEvents(ctx, filter, start, end)
}

// Query for audit events in a paginated fashion
func (c *client) SearchAuditEvents(ctx context.Context, filter *v3.AuditEventsSelection, start, end *time.Time) <-chan *event.AuditEventResult {
	ch := make(chan *event.AuditEventResult, pageSize)
	searchIndex := c.clusterIndex(auditLogIndex, "*")
	go func() {
		defer close(ch)
		// Make search query with scroll
		scroll := c.Scroll(searchIndex).
			Query(constructAuditEventsQuery(filter, start, end)).
			Sort("stageTimestamp", true).
			Size(pageSize)
		for {
			res, err := scroll.Do(context.Background())
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithError(err).Warn("failed to search for audit events")
				ch <- &event.AuditEventResult{Err: err}
				return
			}
			if res == nil {
				err = fmt.Errorf("Search expected results != nil; got nil")
			} else if res.Hits == nil {
				err = fmt.Errorf("Search expected results.Hits != nil; got nil")
			} else if len(res.Hits.Hits) == 0 {
				err = fmt.Errorf("Search expected results.Hits.Hits > 0; got %d", res.Hits.Hits)
			}
			if err != nil {
				log.WithError(err).Warn("Unexpected results from audit events search")
				ch <- &event.AuditEventResult{Err: err}
				return
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
		}

		if err := scroll.Clear(context.Background()); err != nil {
			log.WithError(err).Info("Failed to clear scroll context")
		}
	}()

	return ch
}

func constructAuditEventsQuery(filter *v3.AuditEventsSelection, start, end *time.Time) elastic.Query {
	queries := []elastic.Query{}

	// Query by filter if specified.
	if filter != nil {
		queries = append(queries, auditEventQueryFromAuditEventsSelection(filter))
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
	return elastic.NewBoolQuery().Must(queries...)
}

func auditEventQueryFromAuditEventsSelection(filter *v3.AuditEventsSelection) elastic.Query {
	if len(filter.Resources) == 0 {
		return nil
	}
	queries := []elastic.Query{}
	for _, resID := range filter.Resources {
		queries = append(queries, auditEventQueryFromResourceID(resID))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func auditEventQueryFromResourceID(resID v3.ResourceID) elastic.Query {
	queries := []elastic.Query{}
	if resID.Kind != "" {
		queries = append(queries, elastic.NewMatchQuery("responseObject.kind", resID.Kind))
	}
	if resID.APIVersion != "" {
		queries = append(queries, elastic.NewMatchQuery("responseObject.apiVersion", resID.APIVersion))
	}
	if resID.Name != "" {
		queries = append(queries, elastic.NewMatchQuery("responseObject.name", resID.Name))
	}
	if resID.Namespace != "" {
		queries = append(queries, elastic.NewMatchQuery("responseObject.namespace", resID.Namespace))
	}
	return elastic.NewBoolQuery().Must(queries...)
}
