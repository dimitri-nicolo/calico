package features

const (
	All                   = "all"
	DropActionOverride    = "drop-action-override"
	PrometheusMetrics     = "prometheus-metrics"
	AWSCloudwatchFlowLogs = "aws-cloudwatch-flow-logs"
	AWSCloudwatchMetrics  = "aws-cloudwatch-metrics"
	AWSSecurityGroups     = "aws-security-groups"
	IPSec                 = "ipsec"
	FederatedServices     = "federated-services"
	FileOutputFlowLogs    = "file-output-flow-logs"
	FileOutputL7Logs      = "file-output-l7-logs"
	ManagementPortal      = "management-portal"
	PolicyRecommendation  = "policy-recommendation"
	PolicyPreview         = "policy-preview"
	PolicyManagement      = "policy-management"
	Tiers                 = "tiers"
	EgressAccessControl   = "egress-access-control"
	ExportLogs            = "export-logs"
	AlertManagement       = "alert-management"
	ApplicationTelementry = "application-telemetry"
	TopologicalGraph      = "topological-graph"
	KibanaDashboard       = "kibana-dashboard"
	DualNIC               = "dual-nic"
	ComplianceReports     = "compliance-reports"
	ThreatDefense         = "threat-defense"
)

type set map[string]bool

func merge(a set, b set) set {
	var new set = make(set)

	if a != nil {
		for k, v := range a {
			new[k] = v
		}
	}

	if b != nil {
		for k, v := range b {
			new[k] = v
		}
	}

	return new
}

const (
	// Constants to define a license package for Calico Cloud
	CloudCommunity = "cloud|community"
	CloudStarter   = "cloud|starter"
	CloudPro       = "cloud|pro"
	// Constant to define a license package for Calico Enterprise using a self - hosted environment
	Enterprise = "cnx|all"
)

// PackageNames defines the name of the license packages available
var PackageNames = set{CloudCommunity: true, CloudStarter: true, CloudPro: true, Enterprise: true}

// IsValidPackageName return true if a package name matches the one defined in PackageNames
func IsValidPackageName(value string) bool {
	return PackageNames[value]
}

// CloudCommunity package is defined by features such as: Management Portal UI, Policy Management and Policy Troubleshooting
var CloudCommunityFeatures = set{ManagementPortal: true, PolicyRecommendation: true, PolicyPreview: true, PolicyManagement: true, FileOutputFlowLogs: true, PrometheusMetrics: true}

// CloudStarter package has in addition to CloudCommuniy EgressAccessControl and Tiers
var CloudStarterFeatures = merge(CloudCommunityFeatures, set{EgressAccessControl: true, Tiers: true})

// CloudPro package contains all available features except Compliance and Threat Defense features
var CloudProFeatures = merge(CloudStarterFeatures, set{FederatedServices: true, ExportLogs: true, AlertManagement: true, ApplicationTelementry: true, TopologicalGraph: true, KibanaDashboard: true, FileOutputL7Logs: true})
