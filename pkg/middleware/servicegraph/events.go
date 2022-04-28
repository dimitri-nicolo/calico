// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/k8s"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/tigera/es-proxy/pkg/elastic"
)

const (
	alertsQuerySize     = 1000
	K8sEventTypeWarning = "Warning"
)

var (
	replicaRegex = regexp.MustCompile("-[a-z0-9]{5}$")
)

// The extracted event information. This is simply an ID with a set of graph endpoints that it may correspond to. There
// is a bit of guesswork here - so the graphconstructor will use this as best effort to track down the appropriate
// node in the graph.
type Event struct {
	ID      v1.GraphEventID
	Details v1.GraphEventDetails

	// The set of flow endpoints may contain additional types than returned by L3 and L7 flows:
	// - Namespaces (so that we can alert on non-flow related resources)
	// - HostEndpoints (flows essentially return Hosts, but this may return HostEndpoints from resource changes which
	//                  will get mapped to Hosts in the cache processing).
	Endpoints []FlowEndpoint
}

// RawEvent used to unmarshaling the event.
type RawEvent struct {
	Time            int64           `json:"time"`
	Type            string          `json:"type"`
	Description     string          `json:"description"`
	Alert           string          `json:"alert"`
	Severity        int             `json:"severity"`
	SourceNamespace string          `json:"source_namespace"`
	SourceName      string          `json:"source_name"`
	DestNamespace   string          `json:"dest_namespace"`
	DestName        string          `json:"dest_name"`
	DestPort        int             `json:"dest_port"`
	Protocol        string          `json:"protocol"`
	Record          *RawEventRecord `json:"record,omitempty"`
}

type RawEventRecord struct {
	ResponseObjectKind string `json:"responseObject.kind"`
	ObjectRefResource  string `json:"objectRef.resource"`
	ObjectRefNamespace string `json:"objectRef.namespace"`
	ObjectRefName      string `json:"objectRef.name"`
	ClientNamespace    string `json:"client_namespace"`
	ClientName         string `json:"client_name"`
	ClientNameAggr     string `json:"client_name_aggr"`
	SourceType         string `json:"source_type"`
	SourceNamespace    string `json:"source_namespace"`
	SourceNameAggr     string `json:"source_name_aggr"`
	SourceName         string `json:"source_name"`
	DestType           string `json:"dest_type"`
	DestNamespace      string `json:"dest_namespace"`
	DestNameAggr       string `json:"dest_name_aggr"`
	DestName           string `json:"dest_name"`
	DestPort           int    `json:"dest_port"`
	Protocol           string `json:"proto"`
}

// GetEvents returns events and associated endpoints for each event for the specified time range. Note that Kubernetes
// events are only stored temporarily and so we query all Kubernetes events and filter by time.
//
// Since events contain info that may not always map to a service graph node we do what we can to filter in advance, but
// then let the graphconstructor add in the event information once the graph has been filtered down into the required
// view. If we have insufficient information in the event log to accurately pin the event to a service graph node then
// we will not include it in the node. The unfiltered alerts table will still provide the user with the opportunity to
// see all events.
func GetEvents(
	ctx context.Context, es lmaelastic.Client, csAppCluster k8s.ClientSet, cluster string, tr lmav1.TimeRange,
	cfg *Config,
) ([]Event, error) {
	/* Reinstate when we have k8s events too
	var tigeraEvents, kubernetesEvents []Event
	var tigeraErr, kubernetesErr error

	// Query the active Kubernetes events.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		kubernetesEvents, kubernetesErr = getKubernetesEvents(ctx, csAppCluster, tr)
	}()

	// Query the Tigera events.
	wg.Add(1)
	go func() {
		defer wg.Done()
		tigeraEvents, tigeraErr = getTigeraEvents(ctx, es, cluster, tr)
	}()

	wg.Wait()
	if tigeraErr != nil {
		return nil, tigeraErr
	}
	if kubernetesErr != nil {
		return nil, kubernetesErr
	}

	return append(tigeraEvents, kubernetesEvents...), nil
	*/

	return getTigeraEvents(ctx, es, cluster, tr, cfg)
}

func getTigeraEvents(
	ctx context.Context, es lmaelastic.Client, cluster string, tr lmav1.TimeRange,
	cfg *Config,
) (results []Event, err error) {
	// Trace progress.
	progress := newElasticProgress("events", tr)
	defer func() {
		progress.Complete(err)
	}()

	// Issue the query to Elasticsearch and send results out through the results channel. We terminate the search if:
	// - there are no more "buckets" returned by Elasticsearch or the equivalent no-or-empty "after_key" in the
	//   aggregated search results,
	// - we hit an error, or
	// - the context indicates "done"
	querySize := alertsQuerySize
	if cfg.ServiceGraphCacheMaxBucketsPerQuery > 0 {
		querySize = cfg.ServiceGraphCacheMaxBucketsPerQuery
	}
	query := lmaindex.Alerts().NewTimeRangeQuery(tr.From, tr.To)
	index := lmaindex.Alerts().GetIndex(elasticvariant.AddIndexInfix(cluster))
	var searchAfterKeys []interface{}
	for {
		log.Debugf("Issuing search query, start after %#v", searchAfterKeys)

		// Query the document index.
		search := es.Backend().Search(index).Query(query).Size(querySize).Sort("time", true)
		if searchAfterKeys != nil {
			search = search.SearchAfter(searchAfterKeys...)
		}

		searchResults, err := es.Do(ctx, search)
		if err != nil {
			// We hit an error, exit. This may be a context done error, but that's fine, pass the error on.
			log.WithError(err).Debugf("Error searching %s", index)
			return nil, err
		}

		// Exit if the search timed out. We return a very specific error type that can be recognized by the
		// consumer - this is useful in propagating the timeout up the stack when we are doing server side
		// aggregation.
		if searchResults.TimedOut {
			return nil, lmaelastic.TimedOutError(fmt.Sprintf("timed out querying %s", index))
		}

		// Loop through each of the items in the buckets and convert to a result bucket.
		for _, item := range searchResults.Hits.Hits {
			progress.IncRaw()
			searchAfterKeys = item.Sort
			if event := parseTigeraEvent(item.Id, item.Source); event != nil {
				results = append(results, *event)
				progress.IncAggregated()

				// Track the number of aggregated logs. Bail if we hit the absolute maximum number of aggregated events.
				if len(results) > cfg.ServiceGraphCacheMaxAggregatedRecords {
					return results, DataTruncatedError
				}
			}
		}

		if len(searchResults.Hits.Hits) < querySize {
			log.Debugf("Completed processing %s, found %d events", index, len(results))
			break
		}
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug("Tigera events:")
		for i := range results {
			log.Debugf(
				"-  Event %s: %s",
				results[i].ID,
				results[i].Details.Description,
			)
		}
	}

	return results, nil
}

/* TODO(rlb): Reinstate when we decide how to return k8s events to the user.
func getKubernetesEvents(ctx context.Context, cs k8s.ClientSet, tr lmav1.TimeRange) ([]Event, error) {
	var results []Event

	// Query the Kubernetes events
	k8sEvents, err := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		if kerrors.IsForbidden(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, ke := range k8sEvents.Items {
		if e := parseKubernetesEvent(ke, tr); e != nil {
			results = append(results, *e)
		}
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug("Kubernetes events:")
		for i := range results {
			log.Debugf("-  Event %s: %s", results[i].ID, results[i].Details.Description)
		}
	}

	return results, nil
}

func parseKubernetesEvent(rawEvent corev1.Event, tr lmav1.TimeRange) *Event {
	// We only care about warning events where the first/last event time falls within our time window.
	if rawEvent.Type != K8sEventTypeWarning {
		return nil
	}

	// If we can find a time associated with the event use it to determin whether we should include the event or not.
	var chosenTime *metav1.Time
	firstTime := rawEvent.FirstTimestamp
	lastTime := rawEvent.LastTimestamp
	eventTime := rawEvent.EventTime

	if !firstTime.IsZero() && !lastTime.IsZero() {
		if !tr.Overlaps(firstTime.Time, lastTime.Time) {
			log.Debugf("Skipping event, time range outside request range: %s/%s", rawEvent.Namespace, rawEvent.Name)
			return nil
		}
		chosenTime = &lastTime
	} else if !eventTime.IsZero() {
		if !tr.InRange(eventTime.Time) {
			log.Debugf("Skipping event, time outside request range: %s/%s", rawEvent.Namespace, rawEvent.Name)
			return nil
		}
		chosenTime = &metav1.Time{Time: eventTime.Time}
	}

	ee := getEventEndpointsFromObject(rawEvent.InvolvedObject.Kind, rawEvent.InvolvedObject.Namespace, rawEvent.InvolvedObject.Name)
	if len(ee) == 0 {
		log.Debugf("Skipping event, no involved object: %s/%s", rawEvent.Namespace, rawEvent.Name)
		return nil
	}

	event := &Event{
		ID: v1.GraphEventID{
			Type: v1.GraphEventTypeKubernetes,
			NamespacedName: v1.NamespacedName{
				Namespace: rawEvent.Namespace,
				Name:      rawEvent.Name,
			},
		},
		Details: v1.GraphEventDetails{
			Description: rawEvent.Message,
			Timestamp:   chosenTime,
		},
		Endpoints: ee,
	}
	return event
}
*/

// parseTigeraEvent parses the raw JSON event and converts it to an Event.  Returns nil if the format was not recognized or
// if the event could not be attributed a graph node.
func parseTigeraEvent(id string, item json.RawMessage) *Event {
	rawEvent := &RawEvent{}
	if err := json.Unmarshal(item, rawEvent); err != nil {
		log.WithError(err).Warning("Unable to parse event")
		return nil
	}

	log.Debugf("Processing event with ID %s: %#v", id, rawEvent)

	// Make sure fields that might contain a "-" meaning "no value" have no value (basically fields from flow logs
	// and DNS logs)
	rawEvent.SourceNamespace = singleDashToBlank(rawEvent.SourceNamespace)
	rawEvent.SourceName = singleDashToBlank(rawEvent.SourceName)
	rawEvent.DestNamespace = singleDashToBlank(rawEvent.DestNamespace)
	rawEvent.DestName = singleDashToBlank(rawEvent.DestName)
	if rawEvent.Record != nil {
		rawEvent.Record.SourceNamespace = singleDashToBlank(rawEvent.Record.SourceNamespace)
		rawEvent.Record.SourceName = singleDashToBlank(rawEvent.Record.SourceName)
		rawEvent.Record.DestNamespace = singleDashToBlank(rawEvent.Record.DestNamespace)
		rawEvent.Record.DestName = singleDashToBlank(rawEvent.Record.DestName)
	}

	sev := rawEvent.Severity
	event := &Event{
		ID: v1.GraphEventID{
			Type:           v1.GraphEventType(rawEvent.Type),
			ID:             id,
			NamespacedName: v1.NamespacedName{Name: rawEvent.Alert},
		},
		Details: v1.GraphEventDetails{
			Severity:    &sev,
			Description: rawEvent.Description,
			Timestamp:   &metav1.Time{Time: time.Unix(0, rawEvent.Time)},
		},
	}

	if eps := getEventEndpointsFromFlowEndpoint(
		"",
		rawEvent.SourceNamespace,
		rawEvent.SourceName,
		"",
		0, "",
	); len(eps) > 0 {
		event.Endpoints = append(event.Endpoints, eps...)
	}
	if eps := getEventEndpointsFromFlowEndpoint(
		"",
		rawEvent.DestNamespace,
		rawEvent.DestName,
		"",
		rawEvent.DestPort,
		rawEvent.Protocol,
	); len(eps) > 0 {
		event.Endpoints = append(event.Endpoints, eps...)
	}
	if rawEvent.Record != nil {
		log.Debugf("Parsing fields from Record: %#v", *rawEvent.Record)
		if eps := getEventEndpointsFromFlowEndpoint(
			rawEvent.Record.SourceType,
			rawEvent.Record.SourceNamespace,
			rawEvent.Record.SourceName,
			rawEvent.Record.SourceNameAggr,
			0, "",
		); len(eps) > 0 {
			event.Endpoints = append(event.Endpoints, eps...)
		}
		if eps := getEventEndpointsFromFlowEndpoint(
			rawEvent.Record.DestType,
			rawEvent.Record.DestNamespace,
			rawEvent.Record.DestName,
			rawEvent.Record.DestNameAggr,
			rawEvent.Record.DestPort,
			rawEvent.Record.Protocol,
		); len(eps) > 0 {
			event.Endpoints = append(event.Endpoints, eps...)
		}
		if eps := getEventEndpointsFromFlowEndpoint(
			"wep",
			rawEvent.Record.ClientNamespace,
			rawEvent.Record.ClientName,
			rawEvent.Record.ClientNameAggr,
			0, "",
		); len(eps) > 0 {
			event.Endpoints = append(event.Endpoints, eps...)
		}
		if eps := getEventEndpointsFromObject(
			nonEmptyString(rawEvent.Record.ObjectRefResource, rawEvent.Record.ResponseObjectKind),
			rawEvent.Record.ObjectRefNamespace,
			rawEvent.Record.ObjectRefName,
		); len(eps) > 0 {
			event.Endpoints = append(event.Endpoints, eps...)
		}
	}
	// Only return event IDs that we are able to correlate to a node.
	if len(event.Endpoints) == 0 {
		return nil
	}
	return event
}

func getEventEndpointsFromFlowEndpoint(epType, epNamespace, epName, epNameAggr string, epPort int, proto string) []FlowEndpoint {
	if epType == "" && epNamespace == "" && epName == "" && epNameAggr == "" {
		return nil
	}

	// If the name ends with "-*" then it is actually the aggregated name.
	if strings.HasSuffix(epName, "-*") {
		epNameAggr = epName
		epName = ""
	}

	// If no aggregated name, but we do have a full name - we may be able to extract an aggregated name from it.
	// We never use this "calculated" name to add new nodes to the graph, only to track down existing ones, so it feels
	// safe enough doing this until we sort out consistency in how event endpoints are output.
	if epName != "" && epNameAggr == "" {
		epNameAggr = getAggrNameFromName(epName)
	}

	// If we only have a namespace then return a namespace type since that is the best we can do.
	if epType == "" && epName == "" && epNameAggr == "" {
		return []FlowEndpoint{{
			Type: v1.GraphNodeTypeNamespace,
			Name: epNamespace,
		}}
	}

	// If we don't have an endpoint type then add an entry for all possible endpoint types based on whether we have
	// a namespace or not. Since we only use these endpoints to look up existing nodes in the graph (and never to create
	// new nodes), this feels safe enough.
	var epTypes []string

	if epType == "" {
		if epNamespace == "" {
			epTypes = []string{"hep", "ns"}
		} else {
			epTypes = []string{"wep", "ns"}
		}
	} else {
		epTypes = append(epTypes, epType)
	}

	eps := make([]FlowEndpoint, len(epTypes))
	for i, epType := range epTypes {
		eventEndpointType := mapRawTypeToGraphNodeType(epType, epName == "")
		eventEndpointName := epName
		eventEndpointNameAggr := epNameAggr

		eps[i] = FlowEndpoint{
			Type:      eventEndpointType,
			Namespace: epNamespace,
			Name:      eventEndpointName,
			NameAggr:  eventEndpointNameAggr,
			PortNum:   epPort,
			Protocol:  proto,
		}
	}
	return eps
}

// Return an endpoint from a resource.
func getEventEndpointsFromObject(objResource, objNamespace, objName string) []FlowEndpoint {
	switch objResource {
	case "pods", "Pod":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeWorkload,
			Namespace: objNamespace,
			Name:      objName,
			NameAggr:  getAggrNameFromName(objName),
		}}
	case "hostendpoints", "HostEndpoint":
		// For HostEndpoints we set the aggregated name to "*" in line with the processing in flowl3.go and flowl7.go.
		// Convert the HEP name to the appropriate host that it is configured on.
		return []FlowEndpoint{{
			Type:     v1.GraphNodeTypeHostEndpoint,
			Name:     objName,
			NameAggr: objName,
		}}
	case "networksets", "NetworkSet":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeNetworkSet,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "globalnetworksets", "GlobalNetworkSet":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeNetworkSet,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "replicasets", "ReplicaSet":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeReplicaSet,
			Namespace: objNamespace,
			Name:      objName + "-*",
		}}
	case "daemonsets", "DaemonSet":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeReplicaSet,
			Namespace: objNamespace,
			Name:      objName + "-*",
		}}
	case "endpoints", "Endpoints":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeService,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "services", "Service":
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeService,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "namespaces", "Namespace":
		return []FlowEndpoint{{
			Type: v1.GraphNodeTypeNamespace,
			Name: objName,
		}}
	case "nodes", "Node":
		return []FlowEndpoint{{
			Type:     v1.GraphNodeTypeHost,
			Name:     objName,
			NameAggr: objName,
		}}
	}
	if objNamespace != "" {
		return []FlowEndpoint{{
			Type:      v1.GraphNodeTypeNamespace,
			Namespace: objNamespace,
		}}
	}
	return nil
}

// getAggrNameFromName attempts to determine a pods aggregated name from its name.  This is used so that we can
// marry up a pod to an aggregated flow. We never use this value to add new nodes - just to work out which node to
// include the data in. If the node doesn't exist then we just include in the namespace, so this approximation is
// a reasonable approach.
//
//TODO(rlb): This doesn't work if the name of the replica set is long - in that case the pod name will not resemble
// the generate name. This also will not work if we decide to aggregate based on the conrolling resource (e.g.
// daemonset, deployment, cronjob). So we should fix this - but might involve us maintaining a DB of this information
// to cross reference.
func getAggrNameFromName(name string) string {
	if parts := replicaRegex.Split(name, 2); len(parts) == 2 {
		return parts[0] + "-*"
	}
	return name
}

// nonEmptyString returns the first non-empty string.
func nonEmptyString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
