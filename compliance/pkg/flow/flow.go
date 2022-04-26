package flow

import (
	"fmt"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	api "github.com/tigera/lma/pkg/api"
)

// Container type to hold the EndpointsReportFlow and/or an error.
type FlowLogResult struct {
	*apiv3.EndpointsReportFlow
	Err error
}

// Contains the namespace, endpoint and flow log aggregate endpoint names that
// should be included in the flow log query results.
type FlowLogFilter struct {
	Namespaces    map[string]bool
	Endpoints     map[string]bool
	AggrEndpoints map[string]bool
}

// Initialize and return a FlowLogFilter. Automatically tracks the "global"
// namespace.
func NewFlowLogFilter() *FlowLogFilter {
	namespaces := map[string]bool{
		api.FlowLogGlobalNamespace: true,
	}
	return &FlowLogFilter{
		Namespaces:    namespaces,
		Endpoints:     make(map[string]bool),
		AggrEndpoints: make(map[string]bool),
	}
}

// Adds namespace, endpoint name and aggregated endpoint name to the filter.
func (f *FlowLogFilter) TrackNamespaceAndEndpoint(namespace, endpointName, aggrEndpointName string) {
	if namespace == "" {
		namespace = api.FlowLogGlobalNamespace
	}
	f.Namespaces[namespace] = true
	if endpointName != "" {
		f.Endpoints[getNamespacedName(namespace, endpointName)] = true
	}
	if aggrEndpointName != "" {
		f.AggrEndpoints[getNamespacedName(namespace, aggrEndpointName)] = true
	}
}

// Filter only the endpoints in scope and hence required to be included in the  ReportData.
// We check if an endpoint is present by checking if the aggregated or non-aggregated name
// is present in either the source or destination flow log.
func (f *FlowLogFilter) FilterInFlow(erf *apiv3.EndpointsReportFlow) bool {
	return (erf.Source.NameIsAggregationPrefix && f.FilterInAggregateName(erf.Source.Namespace, erf.Source.Name)) ||
		(!erf.Source.NameIsAggregationPrefix && f.FilterInEndpoint(erf.Source.Namespace, erf.Source.Name)) ||
		(erf.Destination.NameIsAggregationPrefix && f.FilterInAggregateName(erf.Destination.Namespace, erf.Destination.Name)) ||
		(!erf.Destination.NameIsAggregationPrefix && f.FilterInEndpoint(erf.Destination.Namespace, erf.Destination.Name))

}

// Check whether the endpoint (specified by namespace and name) should be filtered in.
func (f *FlowLogFilter) FilterInEndpoint(namespace, name string) bool {
	return f.Endpoints[getNamespacedName(namespace, name)]
}

// Check whether the aggregated endpoint name (specified by namespace and aggregated name) should be filtered in.
func (f *FlowLogFilter) FilterInAggregateName(namespace, name string) bool {
	return f.AggrEndpoints[getNamespacedName(namespace, name)]
}

func getNamespacedName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
