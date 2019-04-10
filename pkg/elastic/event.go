package elastic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/compliance/pkg/event"
)

const (
	auditLogIndex = "tigera_secure_ee_audit_*"
	pageSize      = 100
)

func (c *client) GetAuditEvents(ctx context.Context, kind *metav1.TypeMeta, start, end *time.Time) <-chan *event.AuditEventResult {
	// create the channel that the retrieved events will fill into.
	ch := make(chan *event.AuditEventResult, pageSize)
	var filter *v3.AuditEventsSelection

	// Create an audit event filter if needed
	if kind != nil {
		filter = &v3.AuditEventsSelection{
			Resources: []v3.ResourceID{v3.ResourceID{TypeMeta: metav1.TypeMeta{Kind: kind.Kind, APIVersion: kind.APIVersion}}}}
	}

	// retrieve the events on a goroutine.
	go func() {
		defer close(ch)

		c.searchAuditEvents(ctx, filter, start, end, ch)
	}()

	return ch
}

// addAuditEvents reads audit logs from storage, filters them based on the resources specified in
// `filter`. Blank fields in the filter ResourceIDs are regarded as wildcard matches for that
// parameter.  Fields within a ResourceID are ANDed, different ResourceIDs are ORed. For example:
// - an empty filter would include no audit events
// - a filter containing a blank ResourceID would contain all audit events
// - a filter containing two ResourceIDs, one with Kind set to "NetworkPolicy", the other with kind
//   set to "GlobalNetworkPolicy" would include all Kubernetes and Calico NetworkPolicy and
//   all Calico GlobalNetworkPolicy audit events.
func (c *client) AddAuditEvents(ctx context.Context, data *v3.ReportData, filter *v3.AuditEventsSelection, start, end time.Time) {
	ch := make(chan *event.AuditEventResult, pageSize)
	c.searchAuditEvents(ctx, filter, &start, &end, ch)
	for event := range ch {
		data.AuditEvents = append(data.AuditEvents, *event.Event)
	}
}

// Query for audit events in a paginated fashion
func (c *client) searchAuditEvents(ctx context.Context, filter *v3.AuditEventsSelection, start, end *time.Time, eventChan chan *event.AuditEventResult) {
	exit := false
	for i := 0; !exit; i += pageSize {
		// Make search query
		res, err := c.Search().
			Index(auditLogIndex).
			Query(constructAuditEventsQuery(filter, start, end)).
			Sort("stageTimestamp", true).
			From(i).Size(pageSize).
			Do(context.Background())
		if err != nil {
			eventChan <- &event.AuditEventResult{Err: err}
		}
		log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

		// define function that pushes the search results into the channel.
		for _, hit := range res.Hits.Hits {
			ev := new(auditv1.Event)
			if err := json.Unmarshal(*hit.Source, ev); err != nil {
				log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
				eventChan <- &event.AuditEventResult{nil, err}
			}
			eventChan <- &event.AuditEventResult{ev, nil}
		}

		exit = i+pageSize > int(res.Hits.TotalHits)
	}
}

func constructAuditEventsQuery(filter *v3.AuditEventsSelection, start, end *time.Time) elastic.Query {
	queries := []elastic.Query{}

	// Query by filter if specified.
	if filter != nil {
		queries = append(queries, queryFromAuditEventsSelection(filter))
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

func queryFromAuditEventsSelection(filter *v3.AuditEventsSelection) elastic.Query {
	if len(filter.Resources) == 0 {
		return nil
	}
	queries := []elastic.Query{}
	for _, resID := range filter.Resources {
		queries = append(queries, queryFromResourceID(resID))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func queryFromResourceID(resID v3.ResourceID) elastic.Query {
	queries := []elastic.Query{}
	if resID.Kind != "" {
		queries = append(queries, elastic.NewMatchQuery("responseObject.kind", resID.Kind)) //TODO is this capitalized singular or lowercase plural?
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
