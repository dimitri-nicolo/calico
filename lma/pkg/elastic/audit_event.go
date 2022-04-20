package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	api "github.com/projectcalico/calico/lma/pkg/api"
)

const (
	AuditLogIndex   = "tigera_secure_ee_audit_*"
	DefaultPageSize = 100
)

func (c *client) GetAuditEvents(ctx context.Context, start, end *time.Time) <-chan *api.AuditEventResult {
	return c.SearchAuditEvents(ctx, nil, start, end)
}

// Query for audit events in a paginated fashion
func (c *client) SearchAuditEvents(ctx context.Context, filter *v3.AuditEventsSelection, start, end *time.Time) <-chan *api.AuditEventResult {
	ch := make(chan *api.AuditEventResult, DefaultPageSize)
	searchIndex := c.ClusterIndex(AuditLogIndex, "*")
	go func() {
		defer close(ch)
		// Make search query with scroll
		scroll := c.Scroll(searchIndex).
			Query(constructAuditEventsQuery(filter, start, end)).
			Sort("stageTimestamp", true).
			Size(DefaultPageSize)
		for {
			res, err := scroll.Do(context.Background())
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithError(err).Warn("failed to search for audit events")
				ch <- &api.AuditEventResult{Err: err}
				return
			}
			if res == nil {
				err = fmt.Errorf("Search expected results != nil; got nil")
			} else if res.Hits == nil {
				err = fmt.Errorf("Search expected results.Hits != nil; got nil")
			} else if len(res.Hits.Hits) == 0 {
				err = fmt.Errorf("Search expected results.Hits.Hits > 0; got 0")
			}
			if err != nil {
				log.WithError(err).Warn("Unexpected results from audit events search")
				ch <- &api.AuditEventResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// define function that pushes the search results into the channel.
			for _, hit := range res.Hits.Hits {
				ev := new(auditv1.Event)
				if err := json.Unmarshal(hit.Source, ev); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
					continue
				}
				ch <- &api.AuditEventResult{Event: ev}
			}
		}

		if err := scroll.Clear(context.Background()); err != nil {
			log.WithError(err).Info("Failed to clear scroll context")
		}
	}()

	return ch
}

func constructAuditEventsQuery(filter *v3.AuditEventsSelection, start, end *time.Time) elastic.Query {
	// Limit query to include ResponseComplete stage only since that has that has the most information, and only
	// to the configuration event verb types.
	queries := []elastic.Query{
		elastic.NewMatchQuery("stage", auditv1.StageResponseComplete),
		auditConfigurationEventQuery(),
	}

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

func auditConfigurationEventQuery() elastic.Query {
	queries := []elastic.Query{}
	for _, verb := range api.EventConfigurationVerbs {
		queries = append(queries, elastic.NewMatchQuery("verb", verb))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func auditEventQueryFromAuditEventsSelection(filter *v3.AuditEventsSelection) elastic.Query {
	if len(filter.Resources) == 0 {
		return nil
	}
	queries := []elastic.Query{}
	for _, res := range filter.Resources {
		queries = append(queries, auditEventQueryFromAuditResource(res))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func auditEventQueryFromAuditResource(res v3.AuditResource) elastic.Query {
	queries := []elastic.Query{}
	if res.Resource != "" {
		queries = append(queries, elastic.NewMatchQuery("objectRef.resource", res.Resource))
	}
	if res.APIGroup != "" {
		queries = append(queries, elastic.NewMatchQuery("objectRef.apiGroup", res.APIGroup))
	}
	if res.APIVersion != "" {
		queries = append(queries, elastic.NewMatchQuery("objectRef.apiVersion", res.APIVersion))
	}
	if res.Name != "" {
		queries = append(queries, elastic.NewMatchQuery("objectRef.name", res.Name))
	}
	if res.Namespace != "" {
		queries = append(queries, elastic.NewMatchQuery("objectRef.namespace", res.Namespace))
	}
	return elastic.NewBoolQuery().Must(queries...)
}
