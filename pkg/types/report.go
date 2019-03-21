package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


type ReportSchedule struct {
	Frequency ReportFrequency

	// The start index provides additional control over the report start time.
	// For Yearly schedule, it represents month: 0 (Jan) - 11 (Dec)
	// For Weekly schedule it represents day: 0 (Sun) - 6 (Sat)
	// For Daily schedule it represents hour: 0 - 23
	StartIndex int
}

type ReportFrequency string
const (
	Never ReportFrequency  = "Never"
	Hourly                 = "Hourly"
	Daily                  = "Daily"
	Weekly                 = "Weekly"
	Monthly                = "Monthly"
)

type ReportInterval string
const (
	Snapshot ReportFrequency = "Snapshot"
	Minute                   = "Minute"
	Hour                     = "Hour"
	Day                      = "Day"
	Week                     = "Week"
	Month                    = "Month"
)

type ResourceID struct {
	metav1.TypeMeta
	Name       string
	Namespace  string
	UUID       types.UID
}

// GlobalReportTemplateBundle contains the configuration used to render a global report.
type GlobalReportTemplateBundle struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the GlobalReportTemplate.
	Spec GlobalReportTemplateBundleSpec `json:"spec,omitempty"`
}

// GlobalReportTemplateBundle contains the templates used to render a global report.
type GlobalReportTemplateBundleSpec struct {
	// A set of summary values to store for this report. This is defined as a set of go-templates with associated
	// summary names. The enumerated summary template values will be stored in the report and are available when
	// listing the archived reports. The UI will display this as a set of columns for each of the available reports.
	SummaryTemplates []ReportTemplate

	// The templates used to render the report. Multiple templates may be provided to handle either multiple separate
	// "files" containing subsets of the data (e.g. multiple pages of csv output), or alternative formats for the same
	// data (e.g. csv, json, xml, html).
	ReportTemplates []ReportTemplate
}

type ReportTemplate struct {
	// A name used to identify this template when downloading a template-rendered ReportFile. This should be unique
	// within a given GlobalReport.
	Name string

	// The template for the contents of a global  report. This is a go-template that may access any data from
	// the enumerated ReportData structure.
	Template string
}

// GlobalReport contains the configuration for a global report.
type GlobalReport struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the GlobalReport.
	Spec GlobalReportSpec `json:"spec,omitempty"`
}

// GlobalReport contains the values for a global report.
type GlobalReportSpec struct {
	// EndpointsSelection is used to specify which endpoints are in-scope. If not specified endpoint data will
	// not be collected in the ReportData.
	EndpointsSelection *EndpointsSelection

	// ConfigurationEventsSelection is used to specify which configuration event logs will be gathered. If not
	// specified configuration events will not be collected in the ReportData.
	ConfigurationEventsSelection *ConfigurationEventsSelection

	// The suggested set of templates for this report. When querying which reports are available for download,
	// the returned set is the expansion of the generated ReportData and the set of templates in the template bundle.
	// When downloading a file through the UI, the report name is constructed as follows:
	//   <Report Name>.<Start Time>.<Report Interval>.<Template Name>
	ReportTemplateBundle string

	// Schedule specifies the frequency and start time for each report generation. A scheduled report is generated
	// from the stored ReportData, and so a report may only be generated within the retention period of the stored data.
	Schedule ReportSchedule

	// The report interval. If this is omitted, the report interval is the same as the schedule interval.
	Interval *ReportInterval

	// The node selector used to specify which nodes the report job may be scheduled on.
	NodeSelector map[string]string
}

// EndpointsSelectors is a set of selectors used to select the endpoints that are considered to be in-scope for the
// report. An empty selector is equivalent to all(). All three selectors are ANDed together.
type EndpointsSelection struct {
	// Endpoints selector, selecting endpoints by endpoint labels.
	EndpointSelector string

	// Namespace selector, selecting endpoints by namespace labels.
	NamespaceSelector string

	// ServiceAccount selector, selecting endpoints by service account labels.
	ServiceAccountSelector string

	// Exclude flow log data. If this is true then the gathered endpoints data will not include flow log data for the
	// in-scope endpoints. If you do not need to include actual traffic data in the reports set this to true to reduce
	// storage footprint.
	ExcludeFlowLogData bool
}

// The GlobalReportSummary is used to enumerate the stored reports and list the available renderings that are
// available for download.
type GlobalReportSummary struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the GlobalReportFile.
	Spec GlobalReportSummarySpec
}

type GlobalReportSummarySpec struct {
	// A set of summary names and values. The UI will display these values in the corresponding columns.
	// These values are calculated when the ReportData is generated and stored and may therefore not reflect the
	// current configuration of the SummaryTemplates.
	Summary map[string]string

	// The frequency and start time for each report generation.
	Schedule ReportSchedule

	// The report interval.
	Interval ReportInterval

	// The start time used for the ReportData. Events from this time are aggregated into the report.
	StartTime    metav1.Time

	// The end time used for the ReportData. Events up to (but not including) this time are aggregated into the report.
	EndTime      metav1.Time

	// The report template bundle name.
	TemplateBundleName string

	// The report scope name.
	ReportScopeName string

	// The template names contained within the bundle.
	TemplateNames []string
}

// Resource type used to return a rendered report. This resource type only supports GET operations.
type GlobalReportRenderedData struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// The rendered report. This is only returned on an explicit get.
	RenderedReport string `json:"data,omitempty"`
}

// ReportData contains the aggregated data available for rendering in report templates. The data available is dependent
// on the selector/report configuration.
type ReportData struct {
	StartTime              metav1.Time
	EndTime                metav1.Time
	ScheduleName           string
	EndpointsData          *EndpointsData
	ConfigurationEventData *ConfigurationEventsData
}

// EndpointsReportData is not part of the resource API, but is available as the data supplied to the endpoints report
// template.
type EndpointsData struct {
	EndpointsSelection EndpointsSelection
	Endpoints          EndpointsReportEndpoints
	Namespaces         EndpointsReportNamespace
	Services           EndpointsReportService
}

type EndpointsReportEndpoints struct {
	// The total number of in-scope endpoints.
	//
	// Source: Calculated from pod/wep, hep, namespace and service account labels.
	NumTotal int

	// The number of in-scope endpoints that were ingress protected during the reporting interval (see below for defn of
	// ingress-protected).
	NumIngressProtected int

	// The number of in-scope endpoints that were egress protected during the reporting interval (see below for defn of
	// egress-protected).
	NumEgressProtected int

	// The number of inscope endpoints whose policy would allow ingress traffic from the internet for *any* period within the
	// reporting interval.
	// See below for how this is calculated for an endpoint.
	NumIngressFromInternet int

	// The number of inscope endpoints whose policy would allow egress traffic to the internet for *any* period within the
	// reporting interval.
	// See below for how this is calculated for an endpoint.
	NumEgressToInternet int

	// The number of inscope endpoints whose policy would allow ingress traffic from a different namespace for *any* period
	// within the reporting interval.
	// See below for how this is calculated for an endpoint.
	NumIngressFromOtherNamespace int

	// The number of inscope endpoints whose policy would allow ingress traffic from a different namespace for *any* period
	// within the reporting interval.
	// See below for how this is calculated for an endpoint.
	NumEgressToOtherNamespace int

	// The number of in-scope endpoints that were envoy-enabled within the reporting interval (see below for defn of
	// envoy-enabled)
	NumEnvoyEnabled int

	// The set of in-scope endpoints.
	Items []EndpointsReportEndpoint
}

type EndpointsReportEndpoint struct {
	ID ResourceID

	// Whether ingress traffic to this endpoint was always protected during the reporting interval.
	//
	// Ingress protection is defined as denying ingress traffic unless explicitly whitelisted. This is translated as
	// the endpoint having some explicit ingress policy applied to it.
	//
	// Source: Calculated from the set of ingress policies that apply to each endpoint.
	//
	// Set to:
	// - false if there are no ingress policies applied to the endpoint at any point during the reporting interval.
	// - true otherwise.
	//
	// Note: Policy is not inspected for protection bypass: for example match-all-and-allow rules which would effectively
	//       short-circuit the default tier-drop behavior, in this case the match-all-and-allow would be considered to be
	//       an explicit whitelist of all traffic. We could include simplistic all-match rules and check that they
	//       don't result in an allow. To check for more circuitous match-all allows is much trickier (e.g. you have one
	//       rule that allows for src!=1.2.3.0/24 and another rule that allows for src==1.2.3.0/24, which combined
	//       is essentially an allow-all).
	IngressProtected  bool

	// Whether egress traffic to this endpoint was always protected during the reporting interval.
	//
	// Egress protection is defined as denying egress traffic unless explicitly whitelisted. This is translated as
	// the endpoint having some explicit egress policy applied to it.
	//
	// Source: Calculated from the set of egress policies that apply to each endpoint.
	//
	// Set to:
	// - false if there are no egress policies applied to the endpoint at any point during the reporting interval.
	// - true otherwise.
	//
	// Note: Policy is not inspected for protection bypass: for example match-all-and-allow rules which would effectively
	//       short-circuit the default tier-drop behavior, in this case the match-all-and-allow would be considered to be
	//       an explicit whitelist of all traffic. We could include simplistic all-match rules and check that they
	//       don't result in an allow. To check for more circuitous match-all allows is much trickier (e.g. you have one
	//       rule that allows for src!=1.2.3.0/24 and another rule that allows for src==1.2.3.0/24, which combined
	//       is essentially an allow-all). Similarly, policy that only contains pass rules would still count as being
	//       protected.
	EgressProtected  bool

	// Whether the matching policy has any ingress allow rules from a public IP address (as defined by the complement of
	// the private addresses; private addresses default to those defined in RFC 1918, but may also be configured separately).
	//
	// Source: Calculated from the policies applied to the endpoint. The ingress allow rules in each policy are checked
	//         to determine if any CIDR specified in the rule, either directly or through a matching network set, is an
	//         internet address. Endpoint addresses are not included - therefore ingress from a pod that has a public
	//         IP address will not be considered as “from internet”.
	//
	// Note: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//       hit one of these allow rules (e.g. a previous rule may be a match-all-deny).
	IngressFromInternet  bool

	// Whether the matching policy has any egress allow rules to a public IP address (as defined by the complement of
	// the private addresses; private addresses default to those defined in RFC 1918, but may also be configured separately).
	//
	// Source: Calculated from the policies applied to the endpoint. The egress allow rules in each policy are checked
	//         to determine if any CIDR specified in the rule, either directly or through a matching network set, is an
	//         internet address. Endpoint addresses are not included - therefore egress to a pod that has a public
	//         IP address will not be considered as “to internet”.
	//
	// Note 1: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//         hit one of these allow rules (e.g. a previous rule may be a match-all-deny).
	EgressToInternet  bool

	// Whether the matching policy has any ingress allow rules from another namespace.
	//
	// Source: Calculated from the policies applied to the endpoint.
	//
	// Set to true if:
	// - this is a pod (i.e. namespaced) with an applied GlobalNetworkPolicy with an ingress allow rule with no CIDR match.
	// - this is a pod with an applied NetworkPolicy with an ingress allow rule with a non-empty NamespaceSelector.
	//
	// Note: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//       hit one of these allow rules (e.g. a previous rule may be a match-all-deny, or endpoint selector may not
	//       match any endpoints within the namespace).
	IngressFromOtherNamespace  bool

	// Whether the matching policy has any egress allow rules to another namespace.
	//
	// Source: Calculated from the policies applied to the endpoint.
	//
	// Set to true if:
	// - this is a pod endpoint (i.e. namespaced) matches a GlobalNetworkPolicy with an egress allow rule with no CIDR match.
	// - this is a pod endpoint which matches a NetworkPolicy with an egress allow rule with a non-empty NamespaceSelector.
	//
	// Note: This is a simplification since it does not examine the policies to determine if it's actually possible to
	//       hit one of these allow rules (e.g. a previous rule may be a match-all-deny, or endpoint selector may not
	//       match any endpoints within the namespace).
	EgressToOtherNamespace  bool

	// Whether this pod is envoy-enabled. This is simply an indicator of whether an Envoy container is running within the pod.
	// Provided Istio is configured appropriately, this can provide a simplistic determination of whether the pod is mTLS
	// enabled.
	//
	// Source: Pod spec.
	//
	// Set to:
	// - true if envoy is running within the pod
	// - false if envoy is not running within the pod
	EnvoyEnabled  bool

	// The total set of policies that may be applied to this endpoint. The set of policies that apply to an endpoint may
	// change within the reporting interval, this
	AppliedPolicies []ResourceID

	// The list of services that exposed this endpoint at any moment during the reporting interval.
	//
	// Source: Determined from the Kubernetes endpoints resource associated with the service.
	Services []ResourceID

	// The list of all endpoints that have been generating traffic to this endpoint. This list includes endpoints that are
	// not necessarily in-scope.
	//
	// Source: Measured from flow flogs.
	EndpointsGeneratingTrafficToThisEndpoint []EndpointsReportEndpointFlow

	// The list of endpoints that have been receiving traffic from this endpoint.  This list includes endpoints that are
	// not necessarily in-scope.
	//
	// Source: Measured from flow flogs.
	EndpointsReceivingTrafficFromThisEndpoint []EndpointsReportEndpointFlow
}

type EndpointsReportEndpointFlow struct {
	Endpoint ResourceID
	Allowed  EndpointFlowData
	Denied   EndpointFlowData
}

type EndpointFlowData struct {
	Bytes               int
	Packets             int
	HTTPRequestsAllowed int
	HTTPRequestsDenied  int
}

type EndpointsReportNamespaceData struct {
	// The total number of namespaces containing in-scope endpoints.
	//
	// Source: Calculated from pod/wep, hep, namespace and service account labels.
	NumTotal int

	// The number of namespaces whose in-scope endpoints were ingress protected during the reporting interval.
	NumIngressProtected int

	// The number of namespaces whose in-scope endpoints were egress protected during the reporting interval.
	NumEgressProtected int

	// The number of namespaces that contained in-scope endpoints that would allow ingress traffic from the internet for
	// *any* period within the reporting interval.
 	NumIngressFromInternet int

	// The number of namespaces that contained in-scope endpoints that would allow egress traffic to the internet for
	// *any* period within the reporting interval.
	NumEgressToInternet int

	// The number of namespaces that contained in-scope endpoints that would allow ingress traffic from another
	// namespace for *any* period within the reporting interval.
	NumIngressFromOtherNamespace int

	// The number of namespaces that contained in-scope endpoints that would allow egress traffic to another
	// namespace for *any* period within the reporting interval.
	NumEgressToOtherNamespace int

	// The number of namespaces whose in-scope endpoints were always Envoy-enabled
	NumEnvoyEnabled int

	// The set of namespaces containing in-scope endpoints.
	Items []EndpointsReportNamespace
}

type EndpointsReportNamespace struct {
	Namespace ResourceID

	// Whether ingress traffic was protected for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	IngressProtected bool

	// Whether egress traffic was protected for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	EgressProtected bool

	// Whether ingress traffic was allowed from the internet for any endpoint within this namespace within the reporting
	// interval.
	IngressFromInternet bool

	// Whether ingress traffic was allowed from the internet for any endpoint within this namespace within the reporting
	// interval.
	EgressToInternet bool

	// Whether ingress traffic was allowed from another namespace for any endpoint within this namespace within the
	// reporting interval.
	IngressFromOtherNamespace bool

	// Whether ingress traffic was allowed from another namespace for any endpoint within this namespace within the
	// reporting interval.
	EgressToOtherNamespace bool

	// Whether envoy was enabled for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	EnvoyEnabled bool
}

type EndpointsReportServiceData struct {
	// The total number of services containing in-scope endpoints.
	//
	// Source: Calculated from pod/wep, hep, service and service account labels.
	NumTotal int

	// The number of services whose in-scope endpoints were ingress protected during the reporting interval.
	NumIngressProtected int

	// The number of services that contained in-scope endpoints that would allow ingress traffic from the internet for
	// *any* period within the reporting interval.
	NumIngressFromInternet int

	// The number of services that contained in-scope endpoints that would allow ingress traffic from another
	// namespace for *any* period within the reporting interval.
	NumIngressFromOtherNamespace int

	// The number of services whose in-scope endpoints were always Envoy-enabled
	NumEnvoyEnabled int

	// The set of services containing in-scope endpoints.
	Items []EndpointsReportService
}

type EndpointsReportService struct {
	Service ResourceID

	// Whether ingress traffic was protected for all endpoints within this namespace within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	IngressProtected bool

	// Whether ingress traffic was allowed from the internet for any endpoint exposed by this service within the reporting
	// interval.
	IngressFromInternet bool

	// Whether ingress traffic was allowed from another namespace for any endpoint exposed by this service within the
	// reporting interval.
	IngressFromOtherNamespace bool

	// Whether envoy was enabled for all endpoints that were exposed by this service within the reporting interval.
	// This is a summary of information contained in the endpoints data.
	EnvoyEnabled bool
}

// Audit is not part of the resource API, but is available as the data supplied to the endpoints report
// template.
type ConfigurationEventsData struct {
	ConfigurationEventsSelection ConfigurationEventsSelection

	// The total number of in-scope audit logs.
	NumTotal int

	// The number of in-scope audit log create events.
	NumCreate int

	// The number of in-scope audit log patch or replace events.
	NumModified int

	// The number of in-scope audit log delete events.
	NumDelete int

	// The time-ordered set of in-scope audit events that occurred within the reporting interval.
	Items []audit.Event
}

type ConfigurationEventsSelection struct {
	Resources []ResourceID
}

