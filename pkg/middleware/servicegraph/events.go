package servicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/common"
)

const (
	alertsQuerySize = 1000
)

var (
	replicaRegex = regexp.MustCompile("-[a-z0-9]{5}^")
)

// The extracted event information. This is simply an ID with a set of graph endpoints that it may correspond to. There
// is a bit of guesswork here - so the graphconstructor will use this as best effort to track down the appropriate
// node in the graph.
type EventID struct {
	ID string

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

// GetEventIDs returns events IDs and associated endpoints for each ID for the specified time range
////.
// Since events contain info that may not always map to a service graph node we do what we can to filter in advance, but
// then let the graphconstructor add in the event information once the graph has been filtered down into the required
// view. If we have insufficient information in the event log to accurately pin the event to a service graph node then
// we will not include it in the node. The unfiltered alerts table will still provide the user with the opportunity to
// see all events.
func GetEventIDs(ctx context.Context, client lmaelastic.Client, cluster string, t v1.TimeRange) ([]EventID, error) {
	query := common.GetTimeRangeQuery(t)
	index := common.GetEventsIndex(cluster)
	var results []EventID

	ctx, cancel := context.WithTimeout(ctx, flowTimeout)

	// Close the two channels when the go routine completes.
	defer func() {
		defer cancel()
	}()

	// Issue the query to Elasticsearch and send results out through the results channel. We terminate the search if:
	// - there are no more "buckets" returned by Elasticsearch or the equivalent no-or-empty "after_key" in the
	//   aggregated search results,
	// - we hit an error, or
	// - the context indicates "done"
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
			if event := parseEvent(item.Id, item.Source); event != nil {
				results = append(results, *event)
			}
		}

		if len(searchResults.Hits.Hits) < alertsQuerySize {
			log.Debugf("Completed processing %s, found %d events", index, len(results))
			break
		}
	}

	return results, nil
}

// parseEvent parses the raw JSON event and converts it to an EventID.  Returns nil if the format was not recognized or
// if the event could not be attributed a graph node.
func parseEvent(id string, item json.RawMessage) *EventID {
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

	eventID := &EventID{
		ID: id,
	}

	if eps := getEventEndpointsFromEndpoint(
		"",
		rawEvent.SourceNamespace,
		rawEvent.SourceName,
		"",
		0, "",
	); len(eps) > 0 {
		eventID.EventEndpoints = append(eventID.EventEndpoints, eps...)
	}
	if eps := getEventEndpointsFromEndpoint(
		"",
		rawEvent.DestNamespace,
		rawEvent.DestName,
		"",
		rawEvent.DestPort,
		rawEvent.Protocol,
	); len(eps) > 0 {
		eventID.EventEndpoints = append(eventID.EventEndpoints, eps...)
	}
	if rawEvent.Record != nil {
		log.Debugf("Parsing fields from Record: %#v", *rawEvent.Record)
		if eps := getEventEndpointsFromEndpoint(
			rawEvent.Record.SourceType,
			rawEvent.Record.SourceNamespace,
			rawEvent.Record.SourceName,
			rawEvent.Record.SourceNameAggr,
			0, "",
		); len(eps) > 0 {
			eventID.EventEndpoints = append(eventID.EventEndpoints, eps...)
		}
		if eps := getEventEndpointsFromEndpoint(
			rawEvent.Record.DestType,
			rawEvent.Record.DestNamespace,
			rawEvent.Record.DestName,
			rawEvent.Record.DestNameAggr,
			rawEvent.Record.DestPort,
			rawEvent.Record.Protocol,
		); len(eps) > 0 {
			eventID.EventEndpoints = append(eventID.EventEndpoints, eps...)
		}
		if eps := getEventEndpointsFromEndpoint(
			"wep",
			rawEvent.Record.ClientNamespace,
			rawEvent.Record.ClientName,
			rawEvent.Record.ClientNameAggr,
			0, "",
		); len(eps) > 0 {
			eventID.EventEndpoints = append(eventID.EventEndpoints, eps...)
		}
		if eps := getEventEndpointsFromObject(
			nonEmptyString(rawEvent.Record.ObjectRefResource, rawEvent.Record.ResponseObjectKind),
			rawEvent.Record.ObjectRefNamespace,
			rawEvent.Record.ObjectRefName,
		); len(eps) > 0 {
			eventID.EventEndpoints = append(eventID.EventEndpoints, eps...)
		}
	}

	// Only return event IDs that we are able to correlate to a node.
	if len(eventID.EventEndpoints) == 0 {
		return nil
	}
	return eventID
}

func getEventEndpointsFromEndpoint(epType, epNamespace, epName, epNameAggr string, epPort int, proto string) []EventEndpoint {
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
		eps[i] = EventEndpoint{
			Type:      mapRawTypeToGraphNodeType(epType, epName == ""),
			Namespace: epNamespace,
			Name:      epName,
			NameAggr:  epNameAggr,
			Port:      epPort,
			Proto:     proto,
		}
	}
	return eps
}

// Return an endpoint from a resource.
func getEventEndpointsFromObject(objResource, objNamespace, objName string) []EventEndpoint {
	switch objResource {
	case "pods", "Pod":
		return []EventEndpoint{{
			Type:      v1.GraphNodeTypeWorkload,
			Namespace: objNamespace,
			Name:      objName,
			NameAggr:  getAggrNameFromName(objName),
		}}
	case "hostendoints", "HostEndpoint":
		return []EventEndpoint{{
			Type:     v1.GraphNodeTypeHostEndpoint,
			NameAggr: objName,
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
