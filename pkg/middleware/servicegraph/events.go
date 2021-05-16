// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	log "github.com/sirupsen/logrus"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/common"
	"github.com/tigera/es-proxy/pkg/middleware/k8s"
)

const (
	alertsQuerySize     = 1000
	K8sEventTypeWarning = "Warning"
)

var (
	replicaRegex = regexp.MustCompile("-[a-z0-9]{5}^")
)

// The extracted event information. This is simply an ID with a set of graph endpoints that it may correspond to. There
// is a bit of guesswork here - so the graphconstructor will use this as best effort to track down the appropriate
// node in the graph.
type Event struct {
	GraphEventID   v1.GraphEventID
	GraphEvent     v1.GraphEvent
	EventEndpoints []EventEndpoint
}

// The endpoint associated with an event.
// This looks identical to a FlowEndpoint, but this is kept separate to avoid confusion. The Type of this endpoint
// may include namespaces and services, and so covers a broader range of endpoints than a flow.
type EventEndpoint struct {
	Type      v1.GraphNodeType
	Namespace string
	Name      string
	NameAggr  string
	Port      int
	Proto     string
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
func GetEvents(ctx context.Context, client lmaelastic.Client, cluster string, t v1.TimeRange) ([]Event, error) {
	ctx, cancel := context.WithTimeout(ctx, flowTimeout)

	// Close the two channels when the go routine completes.
	defer func() {
		defer cancel()
	}()

	// Extract the cluster specific k8s client.
	k8sClient := k8s.GetClientSetApplicationFromContext(ctx)

	// Get the HostEndpoints to determine a HostEndpoint -> Host name mapping.
	hostEndpointsToHostname := make(map[string]string)
	if hostEndpoints, err := k8sClient.HostEndpoints().List(ctx, metav1.ListOptions{}); err != nil {
		return nil, err
	} else {
		for _, hep := range hostEndpoints.Items {
			hostEndpointsToHostname[hep.Name] = hep.Spec.Node
		}
	}

	var tigeraEvents, kubernetesEvents []Event
	var tigeraErr, kubernetesErr error

	// Query the active Kubernetes events.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		kubernetesEvents, kubernetesErr = getKubernetesEvents(ctx, k8sClient, t, hostEndpointsToHostname)
	}()

	// Query the Tigera events.
	wg.Add(1)
	go func() {
		defer wg.Done()
		tigeraEvents, tigeraErr = getTigeraEvents(ctx, client, cluster, t, hostEndpointsToHostname)
	}()

	wg.Wait()
	if tigeraErr != nil {
		return nil, tigeraErr
	}
	if kubernetesErr != nil {
		return nil, kubernetesErr
	}

	return append(tigeraEvents, kubernetesEvents...), nil
}

func getTigeraEvents(
	ctx context.Context, client lmaelastic.Client, cluster string, t v1.TimeRange, hostEndpointsToHostname map[string]string,
) ([]Event, error) {
	// Issue the query to Elasticsearch and send results out through the results channel. We terminate the search if:
	// - there are no more "buckets" returned by Elasticsearch or the equivalent no-or-empty "after_key" in the
	//   aggregated search results,
	// - we hit an error, or
	// - the context indicates "done"
	var results []Event
	query := common.GetTimeRangeQuery(t)
	index := common.GetEventsIndex(cluster)
	var searchAfterKeys []interface{}
	for {
		log.Debugf("Issuing search query, start after %#v", searchAfterKeys)

		// Query the document index.
		search := client.Backend().Search(index).Query(query).Size(alertsQuerySize).Sort("time", true)
		if searchAfterKeys != nil {
			search = search.SearchAfter(searchAfterKeys...)
		}

		searchResults, err := client.Do(ctx, search)
		if err != nil {
			// We hit an error, exit. This may be a context done error, but that's fine, pass the error on.
			log.WithError(err).Debugf("Error searching %s", index)
			return nil, err
		}

		// Exit if the search timed out. We return a very specific error type that can be recognized by the
		// consumer - this is useful in propagating the timeout up the stack when we are doing server side
		// aggregation.
		if searchResults.TimedOut {
			log.Errorf("Elastic query timed out: %s", index)
			return nil, lmaelastic.TimedOutError(fmt.Sprintf("timed out querying %s", index))
		}

		// Loop through each of the items in the buckets and convert to a result bucket.
		for _, item := range searchResults.Hits.Hits {
			searchAfterKeys = item.Sort
			if event := parseTigeraEvent(item.Id, item.Source, hostEndpointsToHostname); event != nil {
				results = append(results, *event)
			}
		}

		if len(searchResults.Hits.Hits) < alertsQuerySize {
			log.Debugf("Completed processing %s, found %d events", index, len(results))
			break
		}
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug("Tigera events:")
		for i := range results {
			log.Debugf(
				"-  Event %s: %s",
				results[i].GraphEventID.TigeraEventID,
				results[i].GraphEvent.Description,
			)
		}
	}

	return results, nil
}

func getKubernetesEvents(
	ctx context.Context, k8sClient k8s.ClientSet, t v1.TimeRange, hostEndpointsToHostname map[string]string,
) ([]Event, error) {
	var results []Event

	// Query the Kubernetes events
	k8sEvents, err := k8sClient.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ke := range k8sEvents.Items {
		if e := parseKubernetesEvent(ke, t, hostEndpointsToHostname); e != nil {
			results = append(results, *e)
		}
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug("Kubernetes events:")
		for i := range results {
			log.Debugf(
				"-  Event %s/%s: %s",
				results[i].GraphEventID.KubernetesEventID.Namespace,
				results[i].GraphEventID.KubernetesEventID.Name,
				results[i].GraphEvent.Description,
			)
		}
	}

	return results, nil
}

func parseKubernetesEvent(rawEvent corev1.Event, t v1.TimeRange, hostEndpointsToHostname map[string]string) *Event {
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
		if !firstTime.Time.Before(t.To) || !lastTime.Time.After(t.From) {
			log.Debugf("Skipping event, time range outside request range: %s/%s", rawEvent.Namespace, rawEvent.Name)
			return nil
		}
		chosenTime = &firstTime
	} else if !eventTime.IsZero() {
		if !firstTime.Time.Before(t.To) || !lastTime.Time.After(t.From) {
			log.Debugf("Skipping event, time outside request range: %s/%s", rawEvent.Namespace, rawEvent.Name)
			return nil
		}
		chosenTime = &metav1.Time{Time: eventTime.Time}
	}

	ee := getEventEndpointsFromObject(
		rawEvent.InvolvedObject.Kind, rawEvent.InvolvedObject.Namespace, rawEvent.InvolvedObject.Name, hostEndpointsToHostname,
	)
	if len(ee) == 0 {
		log.Debugf("Skipping event, no involved object: %s/%s", rawEvent.Namespace, rawEvent.Name)
		return nil
	}

	event := &Event{
		GraphEventID: v1.GraphEventID{
			KubernetesEventID: types.NamespacedName{
				Namespace: rawEvent.Namespace,
				Name:      rawEvent.Name,
			},
		},
		GraphEvent: v1.GraphEvent{
			Description: rawEvent.Message,
			Timestamp:   chosenTime,
		},
		EventEndpoints: ee,
	}
	return event
}

// parseTigeraEvent parses the raw JSON event and converts it to an Event.  Returns nil if the format was not recognized or
// if the event could not be attributed a graph node.
func parseTigeraEvent(id string, item json.RawMessage, hostEndpointsToHostname map[string]string) *Event {
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

	event := &Event{
		GraphEventID: v1.GraphEventID{
			TigeraEventID: id,
		},
		GraphEvent: v1.GraphEvent{
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
		hostEndpointsToHostname,
	); len(eps) > 0 {
		event.EventEndpoints = append(event.EventEndpoints, eps...)
	}
	if eps := getEventEndpointsFromFlowEndpoint(
		"",
		rawEvent.DestNamespace,
		rawEvent.DestName,
		"",
		rawEvent.DestPort,
		rawEvent.Protocol,
		hostEndpointsToHostname,
	); len(eps) > 0 {
		event.EventEndpoints = append(event.EventEndpoints, eps...)
	}
	if rawEvent.Record != nil {
		log.Debugf("Parsing fields from Record: %#v", *rawEvent.Record)
		if eps := getEventEndpointsFromFlowEndpoint(
			rawEvent.Record.SourceType,
			rawEvent.Record.SourceNamespace,
			rawEvent.Record.SourceName,
			rawEvent.Record.SourceNameAggr,
			0, "",
			hostEndpointsToHostname,
		); len(eps) > 0 {
			event.EventEndpoints = append(event.EventEndpoints, eps...)
		}
		if eps := getEventEndpointsFromFlowEndpoint(
			rawEvent.Record.DestType,
			rawEvent.Record.DestNamespace,
			rawEvent.Record.DestName,
			rawEvent.Record.DestNameAggr,
			rawEvent.Record.DestPort,
			rawEvent.Record.Protocol,
			hostEndpointsToHostname,
		); len(eps) > 0 {
			event.EventEndpoints = append(event.EventEndpoints, eps...)
		}
		if eps := getEventEndpointsFromFlowEndpoint(
			"wep",
			rawEvent.Record.ClientNamespace,
			rawEvent.Record.ClientName,
			rawEvent.Record.ClientNameAggr,
			0, "",
			hostEndpointsToHostname,
		); len(eps) > 0 {
			event.EventEndpoints = append(event.EventEndpoints, eps...)
		}
		if eps := getEventEndpointsFromObject(
			nonEmptyString(rawEvent.Record.ObjectRefResource, rawEvent.Record.ResponseObjectKind),
			rawEvent.Record.ObjectRefNamespace,
			rawEvent.Record.ObjectRefName,
			hostEndpointsToHostname,
		); len(eps) > 0 {
			event.EventEndpoints = append(event.EventEndpoints, eps...)
		}
	}

	// Only return event IDs that we are able to correlate to a node.
	if len(event.EventEndpoints) == 0 {
		return nil
	}
	return event
}

func getEventEndpointsFromFlowEndpoint(
	epType, epNamespace, epName, epNameAggr string, epPort int, proto string, hostEndpointsToHostname map[string]string,
) []EventEndpoint {
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
		return []EventEndpoint{{
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
	}

	eps := make([]EventEndpoint, len(epTypes))
	for i, epType := range epTypes {
		eventEndpointType := mapRawTypeToGraphNodeType(epType, epName == "")
		eventEndpointName := epName
		eventEndpointNameAggr := epNameAggr
		if eventEndpointType == v1.GraphNodeTypeHostEndpoint {
			// Tweak the host endpoint name to have an aggregated name of "*" - we do this because we by default
			// aggregate host endpoints together under a common "*" aggregated host endpoint.
			// Similar handling exists in flowsl3.go and flowsl7.go.
			if eventEndpointName == "" && eventEndpointNameAggr != "*" {
				eventEndpointName = hostEndpointsToHostname[eventEndpointNameAggr]
				eventEndpointNameAggr = "*"
			}
		}

		eps[i] = EventEndpoint{
			Type:      eventEndpointType,
			Namespace: epNamespace,
			Name:      eventEndpointName,
			NameAggr:  eventEndpointNameAggr,
			Port:      epPort,
			Proto:     proto,
		}
	}
	return eps
}

// Return an endpoint from a resource.
func getEventEndpointsFromObject(
	objResource, objNamespace, objName string, hostEndpointsToHostname map[string]string,
) []EventEndpoint {
	switch objResource {
	case "pods", "Pod":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeWorkload,
			Namespace: objNamespace,
			Name:      objName,
			NameAggr:  getAggrNameFromName(objName),
		}}
	case "hostendpoints", "HostEndpoint":
		// For HostEndpoints we set the aggregated name to "*" in line with the processing in flowl3.go and flowl7.go.
		// Convert the HEP name to the appropriate host that it is configured on.
		return []EventEndpoint{{
			Type:     v1.GraphNodeTypeHostEndpoint,
			Name:     hostEndpointsToHostname[objName],
			NameAggr: "*",
		}}
	case "networksets", "NetworkSet":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeNetworkSet,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "globalnetworksets", "GlobalNetworkSet":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeNetworkSet,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "replicasets", "ReplicaSet":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeReplicaSet,
			Namespace: objNamespace,
			Name:      objName + "-*",
		}}
	case "daemonsets", "DaemonSet":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeReplicaSet,
			Namespace: objNamespace,
			Name:      objName + "-*",
		}}
	case "endpoints", "Endpoints":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeService,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "services", "Service":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeService,
			Namespace: objNamespace,
			NameAggr:  objName,
		}}
	case "namespaces", "Namespace":
		return []EventEndpoint{{
			Type: v1.GraphNodeTypeNamespace,
			Name: objName,
		}}
	case "nodes", "Node":
		return []EventEndpoint{{
			Type:     v1.GraphNodeTypeHostEndpoint,
			Name:     objName,
			NameAggr: "*",
		}}
	}
	if objNamespace != "" {
		return []EventEndpoint{{
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
