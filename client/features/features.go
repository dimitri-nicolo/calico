package features

import (
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

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

// CloudCommunityAPIs maps cloud community package APIs
var CloudCommunityAPIs = set{
	api.NewAuthenticationReview().GetObjectKind().GroupVersionKind().String():              true,
	api.NewAuthenticationReviewList().GetObjectKind().GroupVersionKind().String():          true,
	api.NewAuthorizationReview().GetObjectKind().GroupVersionKind().String():               true,
	api.NewAuthorizationReviewList().GetObjectKind().GroupVersionKind().String():           true,
	api.NewBGPConfiguration().GetObjectKind().GroupVersionKind().String():                  true,
	api.NewBGPConfigurationList().GetObjectKind().GroupVersionKind().String():              true,
	api.NewBGPPeer().GetObjectKind().GroupVersionKind().String():                           true,
	api.NewBGPPeerList().GetObjectKind().GroupVersionKind().String():                       true,
	api.NewClusterInformation().GetObjectKind().GroupVersionKind().String():                true,
	api.NewClusterInformationList().GetObjectKind().GroupVersionKind().String():            true,
	api.NewFelixConfiguration().GetObjectKind().GroupVersionKind().String():                true,
	api.NewFelixConfigurationList().GetObjectKind().GroupVersionKind().String():            true,
	api.NewGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String():               true,
	api.NewGlobalNetworkPolicyList().GetObjectKind().GroupVersionKind().String():           true,
	api.NewGlobalNetworkSet().GetObjectKind().GroupVersionKind().String():                  true,
	api.NewGlobalNetworkSetList().GetObjectKind().GroupVersionKind().String():              true,
	api.NewHostEndpoint().GetObjectKind().GroupVersionKind().String():                      true,
	api.NewHostEndpointList().GetObjectKind().GroupVersionKind().String():                  true,
	api.NewIPPool().GetObjectKind().GroupVersionKind().String():                            true,
	api.NewIPPoolList().GetObjectKind().GroupVersionKind().String():                        true,
	api.NewKubeControllersConfiguration().GetObjectKind().GroupVersionKind().String():      true,
	api.NewKubeControllersConfigurationList().GetObjectKind().GroupVersionKind().String():  true,
	api.NewLicenseKey().GetObjectKind().GroupVersionKind().String():                        true,
	api.NewLicenseKeyList().GetObjectKind().GroupVersionKind().String():                    true,
	api.NewManagedCluster().GetObjectKind().GroupVersionKind().String():                    true,
	api.NewManagedClusterList().GetObjectKind().GroupVersionKind().String():                true,
	api.NewNetworkPolicy().GetObjectKind().GroupVersionKind().String():                     true,
	api.NewNetworkPolicyList().GetObjectKind().GroupVersionKind().String():                 true,
	api.NewNetworkSet().GetObjectKind().GroupVersionKind().String():                        true,
	api.NewNetworkSetList().GetObjectKind().GroupVersionKind().String():                    true,
	api.NewNode().GetObjectKind().GroupVersionKind().String():                              true,
	api.NewNodeList().GetObjectKind().GroupVersionKind().String():                          true,
	api.NewProfile().GetObjectKind().GroupVersionKind().String():                           true,
	api.NewProfileList().GetObjectKind().GroupVersionKind().String():                       true,
	api.NewStagedGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String():         true,
	api.NewStagedGlobalNetworkPolicyList().GetObjectKind().GroupVersionKind().String():     true,
	api.NewStagedKubernetesNetworkPolicy().GetObjectKind().GroupVersionKind().String():     true,
	api.NewStagedKubernetesNetworkPolicyList().GetObjectKind().GroupVersionKind().String(): true,
	api.NewStagedNetworkPolicy().GetObjectKind().GroupVersionKind().String():               true,
	api.NewStagedNetworkPolicyList().GetObjectKind().GroupVersionKind().String():           true,
	api.NewWorkloadEndpoint().GetObjectKind().GroupVersionKind().String():                  true,
	api.NewWorkloadEndpointList().GetObjectKind().GroupVersionKind().String():              true,
}

// CloudStarter package has in addition to CloudCommuniy EgressAccessControl and Tiers
var CloudStarterFeatures = merge(CloudCommunityFeatures, set{EgressAccessControl: true, Tiers: true})

// CloudStarterAPIs maps cloud starter package APIs
var CloudStarterAPIs = merge(CloudCommunityAPIs, set{api.NewTier().GetObjectKind().GroupVersionKind().String(): true,
	api.NewTierList().GetObjectKind().GroupVersionKind().String(): true})

// CloudPro package contains all available features except Compliance and Threat Defense features
var CloudProFeatures = merge(CloudStarterFeatures, set{FederatedServices: true, ExportLogs: true, AlertManagement: true, ApplicationTelementry: true, TopologicalGraph: true, KibanaDashboard: true, FileOutputL7Logs: true})

// CloudProAPIs maps cloud pro package APIs
var CloudProAPIs = merge(CloudStarterAPIs, set{api.NewGlobalAlert().GetObjectKind().GroupVersionKind().String(): true,
	api.NewGlobalAlertList().GetObjectKind().GroupVersionKind().String():                true,
	api.NewGlobalAlertTemplate().GetObjectKind().GroupVersionKind().String():            true,
	api.NewGlobalAlertTemplateList().GetObjectKind().GroupVersionKind().String():        true,
	api.NewPacketCapture().GetObjectKind().GroupVersionKind().String():                  true,
	api.NewPacketCaptureList().GetObjectKind().GroupVersionKind().String():              true,
	api.NewRemoteClusterConfiguration().GetObjectKind().GroupVersionKind().String():     true,
	api.NewRemoteClusterConfigurationList().GetObjectKind().GroupVersionKind().String(): true})

// EnterpriseAPIs maps enterprise package to all APIs
var EnterpriseAPIs = merge(CloudProAPIs, set{
	api.NewGlobalReport().GetObjectKind().GroupVersionKind().String():         true,
	api.NewGlobalReportList().GetObjectKind().GroupVersionKind().String():     true,
	api.NewGlobalReportType().GetObjectKind().GroupVersionKind().String():     true,
	api.NewGlobalReportTypeList().GetObjectKind().GroupVersionKind().String(): true,
	api.NewGlobalThreatFeed().GetObjectKind().GroupVersionKind().String():     true,
	api.NewGlobalThreatFeedList().GetObjectKind().GroupVersionKind().String(): true,
})

// EnterpriseFeatures package contains all available features
var EnterpriseFeatures = merge(CloudProFeatures, set{ComplianceReports: true, ThreatDefense: true})
