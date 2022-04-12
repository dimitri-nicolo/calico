// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"strings"
	"time"

	elastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/net"

	"github.com/tigera/api/pkg/lib/numorstring"
	"github.com/projectcalico/calico/lma/pkg/api"
)

const (
	FlowlogBuckets     = "flog_buckets"
	FlowNameAggregated = "-"
	FlowNamespaceNone  = "-"
	FlowIPNone         = "0.0.0.0"
)

var (
	// This is the set of composite sources requested by the UI.
	FlowCompositeSources = []AggCompositeSourceInfo{
		{Name: "source_type", Field: "source_type"},
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "source_name", Field: "source_name_aggr"},
		{Name: "dest_type", Field: "dest_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name", Field: "dest_name_aggr"},
		{Name: "action", Field: "action"},
		{Name: "reporter", Field: "reporter"},
	}
)

const (
	// Indexes into the API flow data.
	FlowCompositeSourcesIdxSourceType      = 0
	FlowCompositeSourcesIdxSourceNamespace = 1
	FlowCompositeSourcesIdxSourceNameAggr  = 2
	FlowCompositeSourcesIdxDestType        = 3
	FlowCompositeSourcesIdxDestNamespace   = 4
	FlowCompositeSourcesIdxDestNameAggr    = 5
	FlowCompositeSourcesIdxReporter        = 6
	FlowCompositeSourcesIdxAction          = 7
	FlowCompositeSourcesIdxSourceAction    = 8
	FlowCompositeSourcesIdxImpacted        = 9 // This is a PIP specific parameter, but part of the API, so defined here.
	FlowCompositeSourcesNum                = 10
)

const (
	FlowAggregatedTermsNameSourceLabels = "source_labels"
	FlowAggregatedTermsNameDestLabels   = "dest_labels"
	FlowAggregatedTermsNamePolicies     = "policies"
)

var (
	FlowAggregatedTerms = []AggNestedTermInfo{
		{"policies", "policies", "by_tiered_policy", "policies.all_policies"},
		{"dest_labels", "dest_labels", "by_kvpair", "dest_labels.labels"},
		{"source_labels", "source_labels", "by_kvpair", "source_labels.labels"},
	}

	FlowAggregationSums = []AggSumInfo{
		{"sum_num_flows_started", "num_flows_started"},
		{"sum_num_flows_completed", "num_flows_completed"},
		{"sum_packets_in", "packets_in"},
		{"sum_bytes_in", "bytes_in"},
		{"sum_packets_out", "packets_out"},
		{"sum_bytes_out", "bytes_out"},
		{"sum_http_requests_allowed_in", "http_requests_allowed_in"},
		{"sum_http_requests_denied_in", "http_requests_denied_in"},
	}
)

// GetCompositeAggrFlows returns the set of filtered flows based on the request parameters. The response is JSON serializable.
func GetCompositeAggrFlows(
	ctxIn context.Context, timeout time.Duration, client Client,
	query elastic.Query, docIndex string, filter FlowFilter, limit int32,
) (*CompositeAggregationResults, error) {
	// Create a context with timeout to ensure we don't block for too long with this query.
	ctxWithTimeout, cancel := context.WithTimeout(ctxIn, timeout)
	defer cancel() // Releases timer resources if the operation completes before the timeout.

	// Construct the query.
	compositeQuery := &CompositeAggregationQuery{
		Name:                    FlowlogBuckets,
		DocumentIndex:           docIndex,
		Query:                   query,
		AggCompositeSourceInfos: FlowCompositeSources,
		AggSumInfos:             FlowAggregationSums,
	}

	// Enumerate the aggregation buckets until we have all we need. The channel will be automatically closed.
	var results []*CompositeAggregationBucket
	startTime := time.Now()
	buckets, errs := SearchAndFilterCompositeAggrFlows(ctxWithTimeout, client, compositeQuery, filter, limit)
	for bucket := range buckets {
		results = append(results, bucket)
	}
	took := int64(time.Since(startTime) / time.Millisecond)

	// Check for errors.
	// We can use the blocking version of the channel operator since the error channel will have been closed (it
	// is closed alongside the results channel).
	err := <-errs

	// If there was an error, check for a timeout. If it timed out just flag this in the response, but return whatever
	// data we already have. Otherwise return the error.
	// For timeouts we have a couple of mechanisms for hitting this:
	// -  The elastic search query returns a timeout.
	// -  We exceed the context deadline.
	var timedOut bool
	if err != nil {
		if _, ok := err.(TimedOutError); ok { //nolint:golint,gosimple
			// Response from ES indicates a handled timeout.
			log.Info("Response from ES indicates time out - flag results as timedout")
			timedOut = true
		} else if ctxIn.Err() == nil && ctxWithTimeout.Err() == context.DeadlineExceeded {
			// The context passed to us has no error, but our context with timeout is indicating it has timed out.
			// We need to check the context error rather than checking the returned error since elastic wraps the
			// original context error.
			log.Info("Context deadline exceeded - flag results as timedout")
			timedOut = true
		} else {
			// Just pass the received error up the stack.
			log.WithError(err).Warning("Error response from elasticsearch query")
			return nil, err
		}
	}

	return &CompositeAggregationResults{
		TimedOut:     timedOut,
		Took:         took,
		Aggregations: CompositeAggregationBucketsToMap(results, compositeQuery),
	}, nil
}

// SearchAndFilterCompositeAggrFlows provides a pipeline to search elastic flow logs and filter the results.
//
// This will exit cleanly if the context is cancelled.
func SearchAndFilterCompositeAggrFlows(
	ctx context.Context,
	client Client,
	query *CompositeAggregationQuery,
	filter FlowFilter,
	limit int32,
) (<-chan *CompositeAggregationBucket, <-chan error) {
	results := make(chan *CompositeAggregationBucket, 1000)
	errs := make(chan error, 1)

	// Create a cancellable context so we can exit cleanly when we hit our target number of aggregated results.
	ctx, cancel := context.WithCancel(ctx)

	// Search for the raw data in ES.
	rcvdBuckets, rcvdErrs := client.SearchCompositeAggregations(ctx, query, nil)
	var sent int

	go func() {
		defer func() {
			cancel()
			close(results)
			close(errs)
		}()

		// Iterate through all the raw buckets from ES until the channel is closed. Buckets are ordered in the natural
		// order of the composite sources, thus we can enumerate, process and aggregate related buckets, forwarding the
		// aggregated bucket when the new raw bucket belongs in a different aggregation group.
		for bucket := range rcvdBuckets {
			if include, err := filter.IncludeFlow(bucket); err != nil {
				// Unable to check RBAC permissions.
				log.WithError(err).Info("Error determining if flow should be included")
				errs <- err
				return
			} else if !include {
				// The filter indicates that the flow should not be included.
				log.Debug("Flow should not be included")
				continue
			}

			// Send the flow, or exit if done.
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			case results <- bucket:
				// Increment the number sent.
				sent += 1
			}

			// Exit if we reach the limit of flows.
			if sent >= int(limit) {
				log.Debug("Reached or exceeded our limit of flows to return")
				return
			}
		}

		// If there was an error, send that. All data that we gathered has been sent now.
		// We can use the blocking version of the channel operator since the error channel will have been closed (it
		// is closed alongside the results channel).
		if err, ok := <-rcvdErrs; ok {
			log.WithError(err).Warning("Hit error processing flow logs")
			errs <- err
		}
	}()

	return results, errs
}

// ---- Helper methods to convert the raw flow data into the flows.Flow data. ----

// GetFlowAction extracts the flow action from the composite aggregation key.
func GetFlowActionFromCompAggKey(k CompositeAggregationKey, idx int) api.ActionFlag {
	return api.ActionFlagFromString(k[idx].String())
}

// GetFlowProto extracts the flow protocol from the composite aggregation key.
func GetFlowProtoFromCompAggKey(k CompositeAggregationKey, idx int) *uint8 {
	if s := k[idx].String(); s != "" {
		proto := numorstring.ProtocolFromString(s)
		return api.GetProtocolNumber(&proto)
	} else if f := k[idx].Float64(); f != 0 {
		proto := uint8(f)
		return &proto
	}
	return nil
}

// GetFlowEndpointType extracts the flow endpoint type from the composite aggregation key.
func GetFlowEndpointTypeFromCompAggKey(k CompositeAggregationKey, idx int) api.EndpointType {
	return api.EndpointType(k[idx].String())
}

// GetFlowEndpointName extracts the flow endpoint name from the composite aggregation key.
func GetFlowEndpointNameFromCompAggKey(k CompositeAggregationKey, nameIdx, nameAggrIdx int) string {
	if name := k[nameIdx].String(); name != FlowNameAggregated {
		return name
	}
	return k[nameAggrIdx].String()
}

// GetFlowEndpointLabels extracts the flow endpoint labels from the composite aggregation key.
func GetFlowEndpointLabelsFromCompAggKey(t *AggregatedTerm) map[string]string {
	if t == nil {
		return nil
	}
	l := make(map[string]string)
	ranks := make(map[string]int64)
	for k, rank := range t.Buckets {
		s, _ := k.(string)
		// Label bucket keys are of the format "<label-key>=<label-value>".
		if parts := strings.SplitN(s, "=", 2); len(parts) == 2 {
			// Extract the label key and label value parts of the bucket key.
			key := parts[0]
			value := parts[1]

			// If the ranking of this key value is higher then store it off.
			if rank > ranks[key] {
				l[key] = value
				ranks[key] = rank
			}
		}
	}
	return l
}

// GetFlowPolicies extracts the flow policies that were applied reporter-side from the composite aggregation key.
func GetFlowPoliciesFromAggTerm(t *AggregatedTerm) []api.PolicyHit {
	if t == nil {
		return nil
	}
	// Extract the policies from the raw data, protecting against multiple occurrences of the same policy with different
	// actions.
	var p []api.PolicyHit
	for k, v := range t.Buckets {
		if s, ok := k.(string); !ok {
			log.Errorf("aggregated term policy log is not a string: %#v", s)
			continue
		} else if h, err := api.PolicyHitFromFlowLogPolicyString(s, v); err == nil {
			p = append(p, h)
		} else {
			log.WithError(err).Errorf("failed to parse policy log '%s' as PolicyHit", s)
		}
	}
	return p
}

// Deprecated This function strips the signal that says if an endpoint type is a global endpoint type. The replacement for
// this will either a) assume anything getting a namespace from a flow knows the "-" means it's a global endpoint type
// and will know how to handle it or b) create a structure around a flow to keep the information that the flow is for
// a global resource type and users of that structure will understand how to use that information.
//
// GetFlowEndpointNamespace extracts the flow endpoint namespace from the composite aggregation key.
func GetFlowEndpointNamespaceFromCompAggKey(k CompositeAggregationKey, idx int) string {
	if ns := k[idx].String(); ns != FlowNamespaceNone {
		return ns
	}
	return ""
}

// GetFlowEndpointIP extracts the flow endpoint IP from the composite aggregation key.
func GetFlowEndpointIPFromCompAggKey(k CompositeAggregationKey, idx int) *net.IP {
	if s := k[idx].String(); s != "" && s != FlowIPNone {
		return net.ParseIP(s)
	}
	return nil
}

// GetFlowEndpointPort extracts the flow endpoint port from the composite aggregation key.
func GetFlowEndpointPortFromCompAggKey(k CompositeAggregationKey, idx int) *uint16 {
	if v := k[idx].Float64(); v != 0 {
		u16 := uint16(v)
		return &u16
	}
	return nil
}

// ------------------------- TODO: Similar logic in PIP.

// ConvertFlow converts the raw aggregation bucket into the required policy calculator flow.
// ConvertFlow takes in a maps that key the field name in elasticsearch to the index in the
// CompositeSources and AggregatedTerms that were used to create the query. These field names
// are always going to be the same for queries on flow logs since the field names should be
// the same across flow logs.
func ConvertFlow(b *CompositeAggregationBucket, compositeIdxs map[string]int, termKeys map[string]string) *api.Flow {
	k := b.CompositeAggregationKey
	flow := &api.Flow{
		Reporter: api.ReporterType(k[compositeIdxs["reporter"]].String()),
		Source: api.FlowEndpointData{
			Type:      GetFlowEndpointTypeFromCompAggKey(k, compositeIdxs["source_type"]),
			Name:      GetFlowEndpointNameFromCompAggKey(k, compositeIdxs["source_name"], compositeIdxs["source_name_aggr"]),
			Namespace: GetFlowEndpointNamespaceFromCompAggKey(k, compositeIdxs["source_namespace"]),
			Labels:    GetFlowEndpointLabelsFromCompAggKey(b.AggregatedTerms[termKeys["source_labels"]]),
			IP:        GetFlowEndpointIPFromCompAggKey(k, compositeIdxs["source_ip"]),
			Port:      GetFlowEndpointPortFromCompAggKey(k, compositeIdxs["source_port"]),
		},
		Destination: api.FlowEndpointData{
			Type:      GetFlowEndpointTypeFromCompAggKey(k, compositeIdxs["dest_type"]),
			Name:      GetFlowEndpointNameFromCompAggKey(k, compositeIdxs["dest_name"], compositeIdxs["dest_name_aggr"]),
			Namespace: GetFlowEndpointNamespaceFromCompAggKey(k, compositeIdxs["dest_namespace"]),
			Labels:    GetFlowEndpointLabelsFromCompAggKey(b.AggregatedTerms[termKeys["dest_labels"]]),
			IP:        GetFlowEndpointIPFromCompAggKey(k, compositeIdxs["dest_ip"]),
			Port:      GetFlowEndpointPortFromCompAggKey(k, compositeIdxs["dest_port"]),
		},
		ActionFlag: GetFlowActionFromCompAggKey(k, compositeIdxs["action"]),
		Proto:      GetFlowProtoFromCompAggKey(k, compositeIdxs["proto"]),
		Policies:   GetFlowPoliciesFromAggTerm(b.AggregatedTerms[termKeys["policies"]]),
	}

	// Assume IP version is 4 unless otherwise determined from actual IPs in the flow.
	ipVersion := 4
	if flow.Source.IP != nil {
		ipVersion = flow.Source.IP.Version()
	} else if flow.Destination.IP != nil {
		ipVersion = flow.Source.IP.Version()
	}
	flow.IPVersion = &ipVersion

	return flow
}
