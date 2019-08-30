package pip

import (
	"context"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"

	"github.com/tigera/es-proxy/pkg/pip/elastic"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var (
	// This is the set of composite sources requested by the UI.
	//TODO(rlb): Extract this from the UI query.
	UICompositeSources = []elastic.AggCompositeSourceInfo{
		{"source_type", "source_type"},
		{"source_namespace", "source_namespace"},
		{"source_name", "source_name_aggr"},
		{"dest_type", "dest_type"},
		{"dest_namespace", "dest_namespace"},
		{"dest_name", "dest_name_aggr"},
		{"reporter", "reporter"},
		{"action", "action"},
	}

	// This is the full set of composite sources required by PIP.
	// The order of this is important since ES orders its responses based on this source order - and PIP utilizes this
	// to simplify the aggregation processing allowing us to pipeline the conversion.
	PIPCompositeSources = []elastic.AggCompositeSourceInfo{
		// This first set of fields matches the set requested by the UI and will never be modified by the policy
		// calculation. These are non-aggregated and non-cached in the pipeline converter.
		{"source_type", "source_type"},
		{"source_namespace", "source_namespace"},
		{"source_name", "source_name_aggr"},
		{"dest_type", "dest_type"},
		{"dest_namespace", "dest_namespace"},
		{"dest_name", "dest_name_aggr"},

		// This first set of fields matches the set requested by the UI and may be modified by the policy calculation.
		// These are not aggregated within the pipeine, but we do need to keep a cache of these values because the
		// converter may result in adjusted ordering. These values are explicitly requested *after* the non-cached
		// values - this allows us to close out the set of cached aggregations when the non-aggregated sources clock.
		{"reporter", "reporter"},
		{"action", "action"},

		// These are additional fields that we require to do the policy calculation, but we aggregate out
		// in the pipeline processing.
		{"proto", "proto"},
		{"source_ip", "source_ip"},
		{"source_name_full", "source_name"},
		{"source_port", "source_port"},
		{"dest_ip", "dest_ip"},
		{"dest_name_full", "dest_name"},
		{"dest_port", "dest_port"},
	}

	// ^^^ A note on the above regarding source/dest names ^^^
	// The UI queries for "source_name" and "dest_name" which are extracted from the backing fields "source_name_aggr"
	// and "dest_name_aggr". So we use "source_name_full" and "dest_name_full" for the backing "source_name" and
	// "dest_name" fields.
	// As it happens we actually use index values to access these fields in the parse composite sources when
	// processing the flow, so we shouldn't need any magic code to handle the cross-over naming scheme. When we actually
	// base the query on the parsed UI request then we'll need to be more clever.

	// Indexes into the raw flow data.
	PIPCompositeSourcesRawIdxSourceType      = 0
	PIPCompositeSourcesRawIdxSourceNamespace = 1
	PIPCompositeSourcesRawIdxSourceNameAggr  = 2
	PIPCompositeSourcesRawIdxDestType        = 3
	PIPCompositeSourcesRawIdxDestNamespace   = 4
	PIPCompositeSourcesRawIdxDestNameAggr    = 5
	PIPCompositeSourcesRawIdxReporter        = 6
	PIPCompositeSourcesRawIdxAction          = 7
	PIPCompositeSourcesRawIdxProto           = 8
	PIPCompositeSourcesRawIdxSourceIP        = 9
	PIPCompositeSourcesRawIdxSourceName      = 10
	PIPCompositeSourcesRawIdxSourcePort      = 11
	PIPCompositeSourcesRawIdxDestIP          = 12
	PIPCompositeSourcesRawIdxDestName        = 13
	PIPCompositeSourcesRawIdxDestPort        = 14

	// The number of non-aggregated, non-cached entries that we need to check to determine if we have "clocked" to the
	// next set of aggregated results. See comments for PIPCompositeSources above.
	PIPCompositeSourcesNumNonAggregatedNonCached = 6

	// The composite source indexes that we reference directly, and the total number of sources for a PIP response.
	PIPCompositeSourcesIdxReporter     = 6
	PIPCompositeSourcesIdxAction       = 7
	PIPCompositeSourcesIdxSourceAction = 8
	PIPCompositeSourcesIdxImpacted     = 9
	PIPCompositeSourcesNum             = 10

	// The required aggregated terms that we need to run through PIP.
	//TODO(rlb): Calculate/combine with the actual UI query.
	AggregatedTerms = []elastic.AggNestedTermInfo{
		{"policies", "policies", "by_tiered_policy", "policies.all_policies"},
		{"dest_labels", "dest_labels", "by_kvpair", "dest_labels.labels"},
		{"source_labels", "source_labels", "by_kvpair", "source_labels.labels"},
	}

	PIPAggregatedTermsNameSourceLabels = "source_labels"
	PIPAggregatedTermsNameDestLabels   = "dest_labels"
	PIPAggregatedTermsNamePolicies     = "policies"

	// This should be parsed from the query.
	//TODO(rlb): Extract this from the UI query.
	UIAggregationSums = []elastic.AggSumInfo{
		{"sum_num_flows_started", "num_flows_started"},
		{"sum_num_flows_completed", "num_flows_completed"},
		{"sum_packets_in", "packets_in"},
		{"sum_bytes_in", "bytes_in"},
		{"sum_packets_out", "packets_out"},
		{"sum_bytes_out", "bytes_out"},
		{"sum_http_requests_allowed_in", "http_requests_allowed_in"},
		{"sum_http_requests_denied_in", "http_requests_denied_in"},
	}

	// The number of flows to return to the UI.
	//TODO(rlb): Extract this from the UI query.
	UINumAggregatedFlows = 1000
)

const (
	FlowIndex          = "tigera_secure_ee_flows"
	FlowlogBuckets     = "flog_buckets"
	FlowNameAggregated = "-"
	FlowNamespaceNone  = "-"
	FlowIPNone         = "0.0.0.0"
)

// ProcessedFlows contains a set of related aggregated flows returned from the ProcessFlowLogs pipeline processor.
type ProcessedFlows struct {
	Before []*elastic.CompositeAggregationBucket
	After  []*elastic.CompositeAggregationBucket
}

// SearchAndProcessFlowLogs provides a pipeline to search elastic flow logs, translate the results based on PIP and
// stream aggregated results through the returned channel.
//
// This will exit cleanly if the context is cancelled.
func (p *pip) SearchAndProcessFlowLogs(
	ctx context.Context,
	query *elastic.CompositeAggregationQuery,
	startAfterKey elastic.CompositeAggregationKey,
	calc policycalc.PolicyCalculator,
) (<-chan ProcessedFlows, <-chan error) {
	results := make(chan ProcessedFlows, UINumAggregatedFlows)
	errs := make(chan error, 1)

	// Modify the original query to include all of the required data.
	//TODO(rlb): Should really calculate this based on the original query from the UI, but at the moment we don't
	//           parse that.
	modifiedQuery := &elastic.CompositeAggregationQuery{
		DocumentIndex:           query.DocumentIndex,
		Query:                   query.Query,
		Name:                    query.Name,
		AggCompositeSourceInfos: PIPCompositeSources,
		AggNestedTermInfos:      AggregatedTerms,
		AggSumInfos:             UIAggregationSums,
	}

	// Create a cancellable context so we can exit cleanly when we hit our target number of aggregated results.
	ctx, cancel := context.WithCancel(ctx)

	// Search for the raw data in ES.
	rcvdBuckets, rcvdErrs := p.esClient.SearchCompositeAggregations(ctx, modifiedQuery, nil)
	var sent int

	go func() {
		defer func() {
			cancel()
			close(results)
			close(errs)
		}()

		// Initialize the last known raw key to simplify our processing.
		lastRawKey := make(elastic.CompositeAggregationKey, PIPCompositeSourcesNumNonAggregatedNonCached)

		// Initialize the before/after caches of aggregations with common non-aggregated, non-cached indices.
		cacheBefore := make(sortedCache, 0)
		cacheAfter := make(sortedCache, 0)
		cacheImpacted := false

		// Handler function to send the result.
		sendResult := func() (exit bool) {
			// Check that we have data to send, if not exit.
			if len(cacheBefore) == 0 && len(cacheAfter) == 0 {
				return false
			}

			// The the cache has been impacted by the resource update then we need to update all flows in the cache
			// accordingly.
			if cacheImpacted {
				for i := range cacheBefore {
					cacheBefore[i].CompositeAggregationKey[PIPCompositeSourcesIdxImpacted].Value = true
				}
				for i := range cacheAfter {
					cacheAfter[i].CompositeAggregationKey[PIPCompositeSourcesIdxImpacted].Value = true
				}
			}

			// Sort the before and after caches and copy into a results struct.
			log.Debug("Packaging up data and sending")
			result := ProcessedFlows{
				Before: cacheBefore.SortAndCopy(),
				After:  cacheAfter.SortAndCopy(),
			}
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return true
			case results <- result:
				// Increment the number sent by the number of flows in the "before" set. We use this for consistency
				// with the non-PIP case.
				sent += len(cacheBefore)
			}

			if sent >= UINumAggregatedFlows {
				// We reached or exceeded the maximum number of aggregated flows.
				return true
			}

			// Reset the before/after sets of buckets. We can re-use the slice.
			cacheBefore = cacheBefore[:0]
			cacheAfter = cacheAfter[:0]
			cacheImpacted = false

			return false
		}

		// Iterate through all the raw buckets from ES until the channel is closed. Buckets are ordered in the natural
		// order of the composite sources, thus we can enumerate, process and aggregate related buckets, forwarding the
		// aggregated bucket when the new raw bucket belongs in a different aggregation group.
		for rawBucket := range rcvdBuckets {
			// Check the last raw key to see if we have clocked, if so send any aggregated results and reset the
			// aggregations. Composite key values are returned in strict order.
			if !lastRawKey.SameBucket(rawBucket.CompositeAggregationKey) {
				log.Debug("Clocked to next set of aggregations")

				// Handle the aggregated results by sending over the results channel. If this indicates we should
				// exit (either due to error or we've hit our results limit) then exit.
				if exit := sendResult(); exit {
					return
				}

				// Update the last key. We only track the indices that are common to the new set of aggregations.
				lastRawKey = rawBucket.CompositeAggregationKey[:PIPCompositeSourcesNumNonAggregatedNonCached]
			}

			// Convert the bucket to a policycalc.Flow, run through the policy calculator, and aggregate the results.
			if flow := convertFlow(rawBucket); flow != nil {
				// Calculate the before and after behavior of the flow.
				processed, before, after := calc.Calculate(flow)

				// The flow is impacted if flow was processed and either source or dest action changed. Once a flow is
				// impacted, all of the flows in the local cache (i.e. all combinations of reporter, action and
				// source_action) will be flagged as impacted. We could do better than this ut it's complicated and
				// probably not worth the hassle since the "impacted" identification is really used to improve
				// visualization.
				cacheImpacted = cacheImpacted || (processed &&
					(before.Source.Action != after.Source.Action ||
						before.Destination.Action != after.Destination.Action))

				// Aggregate the before/after buckets for this flow. Aggregate the before buckets first though because
				// we modify the rawBucket in the aggregateRawFlowBucket call to update the policies - we want the
				// before information to remain as originally queried when we aggregate if the policies
				// were not re-calculated.
				aggregateRawFlowBucket(lastRawKey, rawBucket, before, &cacheBefore)
				aggregateRawFlowBucket(lastRawKey, rawBucket, after, &cacheAfter)
			}
		}

		// We reached the end of the enumeration. We might have data to send still, so send it now.
		if exit := sendResult(); exit {
			return
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

// aggregateRawFlowBucket aggregates the raw aggregation bucket into the further aggregated sets of related buckets in
// the supplied cache. The cache is updated as a result of the aggregation.
// -  The rawKey is the composite aggregation key common to all entries in the cache.
// -  The rawFlow contains additional sources in its key, but these are mostly aggregated out, with the exception of the
//    reporter and action fields which are modified by the policy calculator.
func aggregateRawFlowBucket(
	rawKey []elastic.CompositeAggregationSourceValue,
	rawFlow *elastic.CompositeAggregationBucket,
	resp *policycalc.Response,
	cachePtr *sortedCache,
) {
	if resp.Source.Include {
		b := getOrCreateAggregatedBucketFromRawFlowBucket(
			string(policycalc.ReporterTypeSource), string(resp.Source.Action), string(resp.Source.Action),
			rawKey, cachePtr,
		)

		// Modify the policies to those calculated if policies were calculated.
		if resp.Source.Policies != nil {
			rawFlow.SetAggregatedTermsFromStringSlice(PIPAggregatedTermsNamePolicies, resp.Source.Policies)
		}

		// Aggregate the raw flow data into the cached aggregated flow.
		b.Aggregate(rawFlow)
	}
	if resp.Destination.Include {
		b := getOrCreateAggregatedBucketFromRawFlowBucket(
			string(policycalc.ReporterTypeDestination), string(resp.Source.Action), string(resp.Destination.Action),
			rawKey, cachePtr,
		)

		// Modify the policies to those calculated if policies were calculated.
		if resp.Destination.Policies != nil {
			rawFlow.SetAggregatedTermsFromStringSlice(PIPAggregatedTermsNamePolicies, resp.Destination.Policies)
		}

		// Aggregate the raw flow data into the cached aggregated flow.
		b.Aggregate(rawFlow)
	}
}

// getOrCreateAggregatedBucketFromRawFlowBucket returns the currently cached bucket for the specified combination of
// reporter, source action and action. If the cache does not contain an entry, a new empty bucket is created and the
// cache updated.
func getOrCreateAggregatedBucketFromRawFlowBucket(
	reporter, sourceAction, action string,
	rawKey []elastic.CompositeAggregationSourceValue,
	cachePtr *sortedCache,
) *elastic.CompositeAggregationBucket {
	// Scan the cache to find the required entry. In general we don't expect many entries in the cache, so this
	// is probably better that using a map.
	cache := *cachePtr
	for i := range cache {
		if cache[i].CompositeAggregationKey[PIPCompositeSourcesIdxReporter].Value == reporter &&
			cache[i].CompositeAggregationKey[PIPCompositeSourcesIdxAction].Value == action &&
			cache[i].CompositeAggregationKey[PIPCompositeSourcesIdxSourceAction].Value == sourceAction {
			return cache[i]
		}
	}

	// Cached entry does not exist for the UI-aggregated set of data, create the bucket for this aggregation.
	key := make([]elastic.CompositeAggregationSourceValue, PIPCompositeSourcesNum)
	copy(key, rawKey)
	key[PIPCompositeSourcesIdxReporter] = elastic.CompositeAggregationSourceValue{
		Name:  "reporter",
		Value: reporter,
	}
	key[PIPCompositeSourcesIdxAction] = elastic.CompositeAggregationSourceValue{
		Name:  "action",
		Value: action,
	}
	key[PIPCompositeSourcesIdxSourceAction] = elastic.CompositeAggregationSourceValue{
		Name:  "source_action",
		Value: sourceAction,
	}
	// TODO(rlb): This is not a real key since for the other key values there can only be one value of this, but including as a
	//            key is a lot easier. Another alternative would be to include it as an aggregation so that we can see how much
	//            of this flow has changed.
	//            I've agreed this API with AV, so let's run with this for now, but in future we may want to revisit this.
	//            If we do revisit this then we should consider how this is weighted.  packets, flows etc - or perhaps we
	//            have a set of changed packets/flows/bytes etc.
	key[PIPCompositeSourcesIdxImpacted] = elastic.CompositeAggregationSourceValue{
		Name:  "flow_impacted",
		Value: false,
	}

	entry := elastic.NewCompositeAggregationBucket(0)
	entry.CompositeAggregationKey = key
	*cachePtr = append(cache, entry)

	return entry
}

// sortedCache is a sortable cache of CompositeAggregationBucket. Sorting is based solely on the Reporter, Action and
// SourceAction fields.
type sortedCache []*elastic.CompositeAggregationBucket

// Len implements the Sort interface.
func (s sortedCache) Len() int {
	return len(s)
}

// Less implements the Sort interface.
func (s sortedCache) Less(i, j int) bool {
	si := s[i].CompositeAggregationKey[PIPCompositeSourcesIdxReporter].String()
	sj := s[j].CompositeAggregationKey[PIPCompositeSourcesIdxReporter].String()
	if si < sj {
		return true
	} else if si > sj {
		return false
	}

	// Reporter index is equal, check action.
	si = s[i].CompositeAggregationKey[PIPCompositeSourcesIdxAction].String()
	sj = s[j].CompositeAggregationKey[PIPCompositeSourcesIdxAction].String()
	if si < sj {
		return true
	} else if si > sj {
		return false
	}

	// Action index is equal, check source action.
	si = s[i].CompositeAggregationKey[PIPCompositeSourcesIdxSourceAction].String()
	sj = s[j].CompositeAggregationKey[PIPCompositeSourcesIdxSourceAction].String()
	if si < sj {
		return true
	}

	return false
}

// Swap implements the Sort interface.
func (s sortedCache) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// SortAndCopy sorts and copies the cache.
func (s sortedCache) SortAndCopy() []*elastic.CompositeAggregationBucket {
	sort.Sort(s)
	d := make([]*elastic.CompositeAggregationBucket, len(s))
	copy(d, s)
	return d
}

// convertFlow converts the raw aggregation bucket into the required policy calculator flow.
func convertFlow(b *elastic.CompositeAggregationBucket) *policycalc.Flow {
	k := b.CompositeAggregationKey
	flow := &policycalc.Flow{
		Reporter: policycalc.ReporterType(k[PIPCompositeSourcesRawIdxReporter].String()),
		Source: policycalc.FlowEndpointData{
			Type:      getFlowEndpointType(k, PIPCompositeSourcesRawIdxSourceType),
			Name:      getFlowEndpointName(k, PIPCompositeSourcesRawIdxSourceName, PIPCompositeSourcesRawIdxSourceNameAggr),
			Namespace: getFlowEndpointNamespace(k, PIPCompositeSourcesRawIdxSourceNamespace),
			Labels:    getFlowEndpointLabels(b.AggregatedTerms[PIPAggregatedTermsNameSourceLabels]),
			IP:        getFlowEndpointIP(k, PIPCompositeSourcesRawIdxSourceIP),
			Port:      getFlowEndpointPort(k, PIPCompositeSourcesRawIdxSourcePort),
		},
		Destination: policycalc.FlowEndpointData{
			Type:      getFlowEndpointType(k, PIPCompositeSourcesRawIdxDestType),
			Name:      getFlowEndpointName(k, PIPCompositeSourcesRawIdxDestName, PIPCompositeSourcesRawIdxDestNameAggr),
			Namespace: getFlowEndpointNamespace(k, PIPCompositeSourcesRawIdxDestNamespace),
			Labels:    getFlowEndpointLabels(b.AggregatedTerms[PIPAggregatedTermsNameDestLabels]),
			IP:        getFlowEndpointIP(k, PIPCompositeSourcesRawIdxDestIP),
			Port:      getFlowEndpointPort(k, PIPCompositeSourcesRawIdxDestPort),
		},
		Action:   getFlowAction(k, PIPCompositeSourcesRawIdxAction),
		Proto:    getFlowProto(k, PIPCompositeSourcesRawIdxProto),
		Policies: getFlowPolicies(b.AggregatedTerms[PIPAggregatedTermsNamePolicies]),
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

// ---- Helper methods to convert the raw flow data into the policycalc.Flow data. ----

// getFlowAction extracts the flow action from the composite aggregation key.
func getFlowAction(k elastic.CompositeAggregationKey, idx int) policycalc.Action {
	return policycalc.Action(k[idx].String())
}

// getFlowProto extracts the flow protocol from the composite aggregation key.
func getFlowProto(k elastic.CompositeAggregationKey, idx int) *uint8 {
	if s := k[idx].String(); s != "" {
		proto := numorstring.ProtocolFromString(s)
		return policycalc.GetProtocolNumber(&proto)
	} else if f := k[idx].Float64(); f != 0 {
		proto := uint8(f)
		return &proto
	}
	return nil
}

// getFlowEndpointType extracts the flow endpoint type from the composite aggregation key.
func getFlowEndpointType(k elastic.CompositeAggregationKey, idx int) policycalc.EndpointType {
	return policycalc.EndpointType(k[idx].String())
}

// getFlowEndpointName extracts the flow endpoint name from the composite aggregation key.
func getFlowEndpointName(k elastic.CompositeAggregationKey, nameIdx, nameAggrIdx int) string {
	if name := k[nameIdx].String(); name != FlowNameAggregated {
		return name
	}
	return k[nameAggrIdx].String()
}

// getFlowEndpointLabels extracts the flow endpoint labels from the composite aggregation key.
func getFlowEndpointLabels(t *elastic.AggregatedTerm) map[string]string {
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

// getFlowPolicies extracts the flow policies that were applied reporter-side from the composite aggregation key.
func getFlowPolicies(t *elastic.AggregatedTerm) []string {
	if t == nil {
		return nil
	}
	var l []string
	for k := range t.Buckets {
		if s, ok := k.(string); ok {
			l = append(l, s)
		}
	}
	return l
}

// getFlowEndpointNamespace extracts the flow endpoint namespace from the composite aggregation key.
func getFlowEndpointNamespace(k elastic.CompositeAggregationKey, idx int) string {
	if ns := k[idx].String(); ns != FlowNamespaceNone {
		return ns
	}
	return ""
}

// getFlowEndpointIP extracts the flow endpoint IP from the composite aggregation key.
func getFlowEndpointIP(k elastic.CompositeAggregationKey, idx int) *net.IP {
	if s := k[idx].String(); s != "" && s != FlowIPNone {
		return net.ParseIP(s)
	}
	return nil
}

// getFlowEndpointPort extracts the flow endpoint port from the composite aggregation key.
func getFlowEndpointPort(k elastic.CompositeAggregationKey, idx int) *uint16 {
	if v := k[idx].Float64(); v != 0 {
		u16 := uint16(v)
		return &u16
	}
	return nil
}
