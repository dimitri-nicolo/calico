// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package elastic

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/net"

	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/lma/pkg/api"
)

const (
	FlowlogBuckets     = "flog_buckets"
	FlowDomainNone     = "-"
	FlowNameAggregated = "-"
	FlowNamespaceNone  = "-"
	FlowServiceNone    = "-"
	FlowIPNone         = "0.0.0.0"
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

// GetFlowEndpointDomain extracts the flow endpoint domains as a string from the composite
// aggregation key.
func GetFlowEndpointDomainsFromCompAggKey(k CompositeAggregationKey, idx int) string {
	if v := k[idx].Value; v == nil {
		return ""
	}
	if domains := k[idx].String(); domains != FlowDomainNone {
		return domains
	}
	return ""
}

// GetFlowEndpointPort extracts the flow endpoint port from the composite aggregation key.
func GetFlowEndpointPortFromCompAggKey(k CompositeAggregationKey, idx int) *uint16 {
	if v := k[idx].Float64(); v != 0 {
		u16 := uint16(v)
		return &u16
	}
	return nil
}

// GetFlowEndpointService extracts the flow endpoint service from the composite aggregation key.
func GetFlowEndpointServiceFromCompAggKey(k CompositeAggregationKey, idx int) string {
	if svc := k[idx].String(); svc != FlowServiceNone {
		return svc
	}
	return ""
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
			Type:        GetFlowEndpointTypeFromCompAggKey(k, compositeIdxs["dest_type"]),
			Name:        GetFlowEndpointNameFromCompAggKey(k, compositeIdxs["dest_name"], compositeIdxs["dest_name_aggr"]),
			Namespace:   GetFlowEndpointNamespaceFromCompAggKey(k, compositeIdxs["dest_namespace"]),
			Labels:      GetFlowEndpointLabelsFromCompAggKey(b.AggregatedTerms[termKeys["dest_labels"]]),
			IP:          GetFlowEndpointIPFromCompAggKey(k, compositeIdxs["dest_ip"]),
			Port:        GetFlowEndpointPortFromCompAggKey(k, compositeIdxs["dest_port"]),
			ServiceName: GetFlowEndpointServiceFromCompAggKey(k, compositeIdxs["dest_service_name"]),
			Domains:     GetFlowEndpointDomainsFromCompAggKey(k, compositeIdxs["dest_domains"]),
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

// EmptyToDash converts an empty string to a "-".
// Linseed returns fields such as namespaces as an empty string for global resources,
// whereas the UI expects a "-".
func EmptyToDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
