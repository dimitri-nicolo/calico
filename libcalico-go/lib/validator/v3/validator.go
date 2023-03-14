// Copyright (c) 2016-2022 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v3

import (
	"fmt"
	"math/bits"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	calicoconversion "github.com/projectcalico/calico/libcalico-go/lib/backend/k8s/conversion"

	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	validator "gopkg.in/go-playground/validator.v9"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	"github.com/projectcalico/calico/libcalico-go/lib/compliance"

	wireguard "golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	libapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/errors"
	cnet "github.com/projectcalico/calico/libcalico-go/lib/net"
	"github.com/projectcalico/calico/libcalico-go/lib/selector"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var validate *validator.Validate

const (
	// Maximum size of annotations.
	totalAnnotationSizeLimitB int64 = 256 * (1 << 10) // 256 kB

	// linux can support route-table indices up to 0xFFFFFFFF
	// however, using 0xFFFFFFFF tables would require too much computation, so the total number of designated tables is capped at 0xFFFF
	routeTableMaxLinux       uint32 = 0xffffffff
	routeTableRangeMaxTables uint32 = 0xffff

	globalSelector = "global()"

	operatorApiVersion     = "operator.tigera.io/v1"
	intrusionDetectionKind = "IntrusionDetection"
)

const (
	// Constants when validating a CRON schedule. This is sneaking into the cron implementation a little, but we have
	// validation tests that should protect against any changes to the internals.
	lower60Bits             = (uint64(1) << 60) - 1
	maxCRONSchedulesPerHour = 12
	errorTooManySchedules   = "schedule is not valid: no more than 12 schedules are permitted per hour (av. /5mins)"
)

var (
	nameLabelFmt     = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	nameSubdomainFmt = nameLabelFmt + "(\\." + nameLabelFmt + ")*"

	// All resource names must follow the subdomain name format.  Some resources we impose
	// more restrictive naming requirements.
	nameRegex = regexp.MustCompile("^" + nameSubdomainFmt + "$")

	// Construct a regex for a graph node name. No point being too fussy about the format, but since these names are
	// not configurable and are derived from Kubernetes resources, no point in being too permissive either. Need to
	// support at least the following:
	//   <name>
	//   <name>/<name>(/<name>)...
	//   *
	//   <name>*
	//   <name>.*
	//   <name>-*
	graphNodeNameRegex = regexp.MustCompile("^[-/;.*a-z0-9]+$")

	// Tiers and UISettingsGroups must have simple names with no dots, since they appear as sub-components of other
	// names.
	tierNameRegex            = regexp.MustCompile("^" + nameLabelFmt + "$")
	uiSettingsGroupNameRegex = regexp.MustCompile("^" + nameLabelFmt + "$")

	containerIDFmt   = "[a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])?"
	containerIDRegex = regexp.MustCompile("^" + containerIDFmt + "$")

	// NetworkPolicy names must either be a simple DNS1123 label format (nameLabelFmt), or
	// nameLabelFmt.nameLabelFmt (with a single dot), or
	// must be the standard name format (nameRegex) prefixed with "knp.default" or "ossg.default".
	networkPolicyNameRegex = regexp.MustCompile("^((" + nameLabelFmt + ")(\\." + nameLabelFmt + ")?|((?:knp|ossg)\\.default\\.(" + nameSubdomainFmt + ")))$")

	// GlobalNetworkPolicy names must be a simple DNS1123 label format (nameLabelFmt) or
	// have a single dot.
	globalNetworkPolicyNameRegex = regexp.MustCompile("^(" + nameLabelFmt + "\\.)?" + nameLabelFmt + "$")

	// Hostname  have to be valid ipv4, ipv6 or strings up to 64 characters.
	prometheusHostRegexp = regexp.MustCompile(`^[a-zA-Z0-9:._+-]{1,64}$`)

	// global() cannot be used with other selectors.
	andOr               = `(&&|\|\|)`
	globalSelectorRegex = regexp.MustCompile(fmt.Sprintf(`%v global\(\)|global\(\) %v`, andOr, andOr))

	interfaceRegex        = regexp.MustCompile("^[a-zA-Z0-9_.-]{1,15}$")
	ignoredInterfaceRegex = regexp.MustCompile("^[a-zA-Z0-9_.*-]{1,15}$")
	ifaceFilterRegex      = regexp.MustCompile("^[a-zA-Z0-9:._+-]{1,15}$")
	actionRegex           = regexp.MustCompile("^(Allow|Deny|Log|Pass)$")
	protocolRegex         = regexp.MustCompile("^(TCP|UDP|ICMP|ICMPv6|SCTP|UDPLite)$")
	ipipModeRegex         = regexp.MustCompile("^(Always|CrossSubnet|Never)$")
	vxlanModeRegex        = regexp.MustCompile("^(Always|CrossSubnet|Never)$")
	logLevelRegex         = regexp.MustCompile("^(Debug|Info|Warning|Error|Fatal)$")

	IPSeclogLevelRegex = regexp.MustCompile("^(None|Notice|Info|Debug|Verbose)$")
	IPSecModeRegex     = regexp.MustCompile("^(PSK)$")

	bpfLogLevelRegex    = regexp.MustCompile("^(Debug|Info|Off)$")
	bpfServiceModeRegex = regexp.MustCompile("^(Tunnel|DSR)$")

	datastoreType         = regexp.MustCompile("^(etcdv3|kubernetes)$")
	routeSource           = regexp.MustCompile("^(WorkloadIPs|CalicoIPAM)$")
	dropAcceptReturnRegex = regexp.MustCompile("^(Drop|Accept|Return)$")
	acceptReturnRegex     = regexp.MustCompile("^(Accept|Return)$")
	ipTypeRegex           = regexp.MustCompile("^(CalicoNodeIP|InternalIP|ExternalIP)$")
	fileRegex             = regexp.MustCompile("^[^\x00]+$")
	endpointFmt           = "https?://[^:]+:\\d+"
	k8sEndpointRegex      = regexp.MustCompile("^" + endpointFmt + "$")
	etcdEndpointsRegex    = regexp.MustCompile("^" + endpointFmt + "(," + endpointFmt + ")*$")
	standardCommunity     = regexp.MustCompile(`^(\d+):(\d+)$`)
	largeCommunity        = regexp.MustCompile(`^(\d+):(\d+):(\d+)$`)
	number                = regexp.MustCompile(`(\d+)`)
	IPv4PortFormat        = regexp.MustCompile(`^(\d+).(\d+).(\d+).(\d+):(\d+)$`)
	IPv6PortFormat        = regexp.MustCompile(`^\[[0-9a-fA-F:.]+\]:(\d+)$`)
	reasonString          = "Reason: "
	poolUnstictCIDR       = "IP pool CIDR is not strictly masked"
	overlapsV4LinkLocal   = "IP pool range overlaps with IPv4 Link Local range 169.254.0.0/16"
	overlapsV6LinkLocal   = "IP pool range overlaps with IPv6 Link Local range fe80::/10"
	protocolPortsMsg      = "rules that specify ports must set protocol to TCP or UDP or SCTP"
	protocolIcmpMsg       = "rules that specify ICMP fields must set protocol to ICMP"
	protocolAndHTTPMsg    = "rules that specify HTTP fields must set protocol to TCP or empty"

	awsSubnetRE        = regexp.MustCompile(`^subnet-[0-9a-f]{8,17}$`)
	invalidAWSSubnetID = "AWS subnet ID is invalid; should be 'subnet-' followed by 8 or 17 lower-case hex digits"
	ipv6AWSSubnet      = "IPv6 is not supported for AWS-backed IP pools"

	globalSelectorEntRule = fmt.Sprintf("%v can only be used in an EntityRule namespaceSelector", globalSelector)
	globalSelectorOnly    = fmt.Sprintf("%v cannot be combined with other selectors", globalSelector)

	dropActionOverrideRegex = regexp.MustCompile("^(Drop|Accept|LogAndDrop|LogAndAccept)$")

	SourceAddressRegex        = regexp.MustCompile("^(UseNodeIP|None)$")
	FailureDetectionModeRegex = regexp.MustCompile("^(None|BFDIfDirectlyConnected)$")
	RestartModeRegex          = regexp.MustCompile("^(GracefulRestart|LongLivedGracefulRestart)$")
	BIRDGatewayModeRegex      = regexp.MustCompile("^(Recursive|DirectIfDirectlyConnected)$")

	GlobalAlertTypeRegex = regexp.MustCompile("^(RuleBased|AnomalyDetection)$")

	filterActionRegex  = regexp.MustCompile("^(Accept|Reject)$")
	matchOperatorRegex = regexp.MustCompile("^(Equal|In|NotEqual|NotIn)$")

	minAggregationKindValue    = 0
	maxAggregationKindValue    = 2
	minDNSAggregationKindValue = 0
	maxDNSAggregationKindValue = 1

	L7HTTPHeaderInfoAggregationRegex = regexp.MustCompile("^(IncludeL7HTTPHeaderInfo|ExcludeL7HTTPHeaderInfo)$")
	L7HTTPMethodAggregationRegex     = regexp.MustCompile("^(IncludeL7HTTPMethod|ExcludeL7HTTPMethod)$")
	L7ServiceAggregationRegex        = regexp.MustCompile("^(IncludeL7ServiceInfo|ExcludeL7ServiceInfo)$")
	L7DestinationAggregationRegex    = regexp.MustCompile("^(IncludeL7DestinationInfo|ExcludeL7DestinationInfo)$")
	L7SourceAggregationRegex         = regexp.MustCompile("^(IncludeL7SourceInfo|IncludeL7SourceInfoNoPort|ExcludeL7SourceInfo)$")
	L7ResponseCodeAggregationRegex   = regexp.MustCompile("^(IncludeL7ResponseCode|ExcludeL7ResponseCode)$")
	L7URLAggregationRegex            = regexp.MustCompile("^(IncludeL7FullURL|TrimURLQuery|TrimURLQueryAndPath|ExcludeL7URL)$")

	ipv4LinkLocalNet = net.IPNet{
		IP:   net.ParseIP("169.254.0.0"),
		Mask: net.CIDRMask(16, 32),
	}

	ipv6LinkLocalNet = net.IPNet{
		IP:   net.ParseIP("fe80::"),
		Mask: net.CIDRMask(10, 128),
	}

	// reserved linux kernel routing tables (cannot be targeted by routeTableRanges)
	routeTablesReservedLinux = []int{253, 254, 255}

	stagedActionRegex = regexp.MustCompile("^(" + string(api.StagedActionSet) + "|" + string(api.StagedActionDelete) + ")$")

	allDigits = regexp.MustCompile(`^\d+$`)
	portRange = regexp.MustCompile(`^(\d+):(\d+)$`)

	// Hostname have to be a strings up to 64 characters.
	uiSettingsRegexp = regexp.MustCompile(`[a-zA-Z0-9_]{1,64}$`)
)

// Validate is used to validate the supplied structure according to the
// registered field and structure validators.
func Validate(current interface{}) error {
	// Perform field-only validation first, that way the struct validators can assume
	// individual fields are valid format.
	if err := validate.Struct(current); err != nil {
		return convertError(err)
	}
	return nil
}

func convertError(err error) errors.ErrorValidation {
	verr := errors.ErrorValidation{}
	for _, f := range err.(validator.ValidationErrors) {
		verr.ErroredFields = append(verr.ErroredFields,
			errors.ErroredField{
				Name:   f.StructField(),
				Value:  f.Value(),
				Reason: extractReason(f),
			})
	}
	return verr
}

func init() {
	// Initialise static data.
	validate = validator.New()

	// Register field validators.
	registerFieldValidator("action", validateAction)
	registerFieldValidator("interface", validateInterface)
	registerFieldValidator("ignoredInterface", validateIgnoredInterface)
	registerFieldValidator("datastoreType", validateDatastoreType)
	registerFieldValidator("name", validateName)
	registerFieldValidator("wildname", ValidateWildName)
	registerFieldValidator("timestamp", validateTimestamp)
	registerFieldValidator("ipOrK8sService", validateIPOrK8sService)
	registerFieldValidator("containerID", validateContainerID)
	registerFieldValidator("selector", validateSelector)
	registerFieldValidator("labels", validateLabels)
	registerFieldValidator("ipVersion", validateIPVersion)
	registerFieldValidator("ipIpMode", validateIPIPMode)
	registerFieldValidator("stagedAction", validateStagedAction)
	registerFieldValidator("vxlanMode", validateVXLANMode)
	registerFieldValidator("policyType", validatePolicyType)
	registerFieldValidator("logLevel", validateLogLevel)
	registerFieldValidator("ipsecLogLevel", validateIPSecLogLevel)
	registerFieldValidator("ipsecMode", validateIPSecMode)
	registerFieldValidator("bpfLogLevel", validateBPFLogLevel)
	registerFieldValidator("bpfServiceMode", validateBPFServiceMode)
	registerFieldValidator("dropAcceptReturn", validateFelixEtoHAction)
	registerFieldValidator("acceptReturn", validateAcceptReturn)
	registerFieldValidator("portName", validatePortName)
	registerFieldValidator("dropActionOverride", validateDropActionOverride)
	registerFieldValidator("mustBeNil", validateMustBeNil)
	registerFieldValidator("mustBeFalse", validateMustBeFalse)
	registerFieldValidator("ifaceFilter", validateIfaceFilter)
	registerFieldValidator("dnsAggregationKind", validateDNSAggregationKind)
	registerFieldValidator("l7HTTPHeaderAggregation", validateL7HTTPHeaderAggregation)
	registerFieldValidator("l7HTTPMethodAggregation", validateL7HTTPMethodAggregation)
	registerFieldValidator("l7ServiceAggregation", validateL7ServiceAggregation)
	registerFieldValidator("l7DestinationAggregation", validateL7DestinationAggregation)
	registerFieldValidator("l7SourceAggregation", validateL7SourceAggregation)
	registerFieldValidator("l7ResponseCodeAggregation", validateL7ResponseCodeAggregation)
	registerFieldValidator("l7URLAggregation", validateL7URLAggregation)
	registerFieldValidator("flowLogAggregationKind", validateFlowLogAggregationKind)
	registerFieldValidator("mac", validateMAC)
	registerFieldValidator("iptablesBackend", validateIptablesBackend)
	registerFieldValidator("keyValueList", validateKeyValueList)
	registerFieldValidator("prometheusHost", validatePrometheusHost)
	registerFieldValidator("ipType", validateIPType)

	registerFieldValidator("sourceAddress", RegexValidator("SourceAddress", SourceAddressRegex))
	registerFieldValidator("failureDetectionMode", RegexValidator("FailureDetectionMode", FailureDetectionModeRegex))
	registerFieldValidator("restartMode", RegexValidator("RestartMode", RestartModeRegex))
	registerFieldValidator("birdGatewayMode", RegexValidator("BIRDGatewayMode", BIRDGatewayModeRegex))
	registerFieldValidator("regexp", validateRegexp)
	registerFieldValidator("routeSource", validateRouteSource)
	registerFieldValidator("wireguardPublicKey", validateWireguardPublicKey)
	registerFieldValidator("IP:port", validateIPPort)
	registerFieldValidator("reachableBy", validateReachableByField)

	registerFieldValidator("globalAlertType", RegexValidator("GlobalAlertType", GlobalAlertTypeRegex))

	// Register filter action and match operator validators (used in BGPFilter)
	registerFieldValidator("filterAction", RegexValidator("FilterAction", filterActionRegex))
	registerFieldValidator("matchOperator", RegexValidator("MatchOperator", matchOperatorRegex))

	// Register network validators (i.e. validating a correctly masked CIDR).  Also
	// accepts an IP address without a mask (assumes a full mask).
	registerFieldValidator("netv4", validateIPv4Network)
	registerFieldValidator("netv6", validateIPv6Network)
	registerFieldValidator("net", validateIPNetwork)

	// Override the default CIDR validator.  Validates an arbitrary CIDR (does not
	// need to be correctly masked).  Also accepts an IP address without a mask.
	registerFieldValidator("cidrv4", validateCIDRv4)
	registerFieldValidator("cidrv6", validateCIDRv6)
	registerFieldValidator("cidr", validateCIDR)
	registerFieldValidator("cidrs", validateCIDRs)

	// Register configuration validators
	registerFieldValidator("file", validateFile)
	registerFieldValidator("etcdEndpoints", validateEtcdEndpoints)
	registerFieldValidator("k8sEndpoint", validateK8sEndpoint)

	registerFieldValidator("clusterAccessSecret", validateClusterAccessSecret)

	// Register UI settings validators
	registerFieldValidator("uiDescription", validateUIDescription)
	registerFieldValidator("servicegraphId", validateServiceGraphId)
	registerFieldValidator("serviceGraphNodeName", validateServiceGraphNodeName)
	registerFieldValidator("servicegraphNodeType", validateServiceGraphNodeType)
	registerFieldValidator("icon", validateIcon)
	registerFieldValidator("color", validateColor)

	// Register struct validators.
	registerStructValidator(validate, validateProtocol, numorstring.Protocol{})
	registerStructValidator(validate, validateProtoPort, api.ProtoPort{})
	registerStructValidator(validate, validatePort, numorstring.Port{})
	registerStructValidator(validate, validateEndpointPort, api.EndpointPort{})
	registerStructValidator(validate, validateWorkloadEndpointPort, libapi.WorkloadEndpointPort{})
	registerStructValidator(validate, validateIPNAT, libapi.IPNAT{})
	registerStructValidator(validate, validateICMPFields, api.ICMPFields{})
	registerStructValidator(validate, validateIPPoolSpec, api.IPPoolSpec{})
	registerStructValidator(validate, validateNodeSpec, libapi.NodeSpec{})
	registerStructValidator(validate, validateIPAMConfigSpec, libapi.IPAMConfigSpec{})
	registerStructValidator(validate, validateObjectMeta, metav1.ObjectMeta{})
	registerStructValidator(validate, validateTier, api.Tier{})
	registerStructValidator(validate, validateUISettingsGroup, api.UISettingsGroup{})
	registerStructValidator(validate, validateUISettings, api.UISettings{})
	registerStructValidator(validate, validateHTTPRule, api.HTTPMatch{})
	registerStructValidator(validate, validateFelixConfigSpec, api.FelixConfigurationSpec{})
	registerStructValidator(validate, validateWorkloadEndpointSpec, libapi.WorkloadEndpointSpec{})
	registerStructValidator(validate, validateHostEndpointSpec, api.HostEndpointSpec{})
	registerStructValidator(validate, validateRule, api.Rule{})
	registerStructValidator(validate, validateEntityRule, api.EntityRule{})
	registerStructValidator(validate, validateRemoteClusterConfigSpec, api.RemoteClusterConfigurationSpec{})
	registerStructValidator(validate, validateBGPPeerSpec, api.BGPPeerSpec{})
	registerStructValidator(validate, validateNetworkPolicy, api.NetworkPolicy{})
	registerStructValidator(validate, validateGlobalNetworkPolicy, api.GlobalNetworkPolicy{})
	registerStructValidator(validate, validateStagedNetworkPolicy, api.StagedNetworkPolicy{})
	registerStructValidator(validate, validateStagedGlobalNetworkPolicy, api.StagedGlobalNetworkPolicy{})
	registerStructValidator(validate, validateStagedKubernetesNetworkPolicy, api.StagedKubernetesNetworkPolicy{})
	registerStructValidator(validate, validateGlobalNetworkSet, api.GlobalNetworkSet{})
	registerStructValidator(validate, validateNetworkSet, api.NetworkSet{})
	registerStructValidator(validate, validatePull, api.Pull{})
	registerStructValidator(validate, validateAlertExceptionSpec, api.AlertExceptionSpec{})
	registerStructValidator(validate, validateGlobalAlertTemplate, api.GlobalAlertTemplate{})
	registerStructValidator(validate, validateGlobalAlert, api.GlobalAlert{})
	registerStructValidator(validate, validateGlobalThreatFeedSpec, api.GlobalThreatFeed{})
	registerStructValidator(validate, validateFeedFormat, api.ThreatFeedFormat{})
	registerStructValidator(validate, validateFeedFormatJSON, api.ThreatFeedFormatJSON{})
	registerStructValidator(validate, validateFeedFormatCSV, api.ThreatFeedFormatCSV{})
	registerStructValidator(validate, validateHTTPHeader, api.HTTPHeader{})
	registerStructValidator(validate, validateConfigMapKeyRef, k8sv1.ConfigMapKeySelector{})
	registerStructValidator(validate, validateSecretKeyRef, k8sv1.SecretKeySelector{})
	registerStructValidator(validate, validateGlobalReportType, api.GlobalReportType{})
	registerStructValidator(validate, validateReportSpec, api.ReportSpec{})
	registerStructValidator(validate, validateReportTemplate, api.ReportTemplate{})
	registerStructValidator(validate, validateRuleMetadata, api.RuleMetadata{})
	registerStructValidator(validate, validateRouteTableIDRange, api.RouteTableIDRange{})
	registerStructValidator(validate, validateRouteTableRange, api.RouteTableRange{})
	registerStructValidator(validate, validateBGPConfigurationSpec, api.BGPConfigurationSpec{})
	registerStructValidator(validate, validateBlockAffinitySpec, libapi.BlockAffinitySpec{})
	registerStructValidator(validate, validateHealthTimeoutOverride, api.HealthTimeoutOverride{})
	registerStructValidator(validate, validatePacketCapture, api.PacketCapture{})
	registerStructValidator(validate, validatePacketCaptureSpec, api.PacketCaptureSpec{})
	registerStructValidator(validate, validatePacketCaptureRule, api.PacketCaptureRule{})
	registerStructValidator(validate, validateDeepPacketInspection, api.DeepPacketInspection{})
	registerStructValidator(validate, validateBlockAffinitySpec, libapi.BlockAffinitySpec{})
	registerStructValidator(validate, validateHealthTimeoutOverride, api.HealthTimeoutOverride{})
}

// reason returns the provided error reason prefixed with an identifier that
// allows the string to be used as the field tag in the validator and then
// re-extracted as the reason when the validator returns a field error.
func reason(r string) string {
	return reasonString + r
}

// extractReason extracts the error reason from the field tag in a validator
// field error (if there is one).
func extractReason(e validator.FieldError) string {
	if strings.HasPrefix(e.Tag(), reasonString) {
		return strings.TrimPrefix(e.Tag(), reasonString)
	}
	switch e.Tag() {
	case "wildname":
		return fmt.Sprintf("%s must be a domain name, optionally with one wildcard at the end (x.y.*), at the beginning (*.x.y), or in the middle (x.*.y)",
			e.Field(),
		)
	case "ipOrK8sService":
		return fmt.Sprintf("%s must be <ip>[:<port>] (indicating an explicit IP) or k8s-service:[<namespace>/]<name>[:port] (indicating a Kubernetes service); an IPv6 address with a port must use square brackets, for example \"[fd00:83a6::12]:5353\"",
			e.Field(),
		)
	}
	return fmt.Sprintf("%sfailed to validate Field: %s because of Tag: %s ",
		reasonString,
		e.Field(),
		e.Tag(),
	)
}

func registerFieldValidator(key string, fn validator.Func) {
	// We need to register the field validation funcs for all validators otherwise
	// the validator panics on an unknown validation type.
	validate.RegisterValidation(key, fn)
}

func registerStructValidator(validator *validator.Validate, fn validator.StructLevelFunc, t ...interface{}) {
	validator.RegisterStructValidation(fn, t...)
}

func validateAction(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate action: %s", s)
	return actionRegex.MatchString(s)
}

func validateInterface(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate interface: %s", s)
	return s == "*" || interfaceRegex.MatchString(s)
}

func validateIgnoredInterface(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate ignored interface name: %s", s)
	return s != "*" && ignoredInterfaceRegex.MatchString(s)
}

func validateIfaceFilter(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Interface Filter : %s", s)
	return ifaceFilterRegex.MatchString(s)
}

func validateDatastoreType(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Datastore Type: %s", s)
	return datastoreType.MatchString(s)
}

func validateRegexp(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate regexp: %s", s)
	_, err := regexp.Compile(s)
	return err == nil
}

func validateRouteSource(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate routeSource: %s", s)
	_, err := regexp.Compile(s)
	return err == nil
}

func validateWireguardPublicKey(fl validator.FieldLevel) bool {
	k := fl.Field().String()
	log.Debugf("Validate Wireguard public-key %s", k)
	_, err := wireguard.ParseKey(k)
	return err == nil
}

func validateName(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate name: %s", s)
	return nameRegex.MatchString(s)
}

// Public because also useful for the v1 validator.
func ValidateWildName(fl validator.FieldLevel) bool {
	s := strings.ToLower(fl.Field().String())
	log.Debugf("Validate wild name: %s", s)

	// Allow the name to contain one wildcard at the end (x.y.*) or at the beginning (*.x.y) or
	// in the middle (x.*.y).  The implementation in Felix currently supports more general cases
	// than this - e.g. g*.com and *.thing.* - but allowing those through would make it harder
	// to optimize the Felix code in future, if we needed to for performance.
	if strings.HasSuffix(s, ".*") {
		s = s[:len(s)-2] + ".example"
	} else if strings.HasPrefix(s, "*.") {
		s = "example." + s[2:]
	} else if p := strings.Index(s, ".*."); p >= 0 {
		s = s[:p] + ".example." + s[p+3:]
	}
	return nameRegex.MatchString(s)
}

func validateTimestamp(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate timestamp: %s", s)
	t := time.Time{}
	if err := t.UnmarshalText([]byte(s)); err != nil {
		log.Debugf("Invalid timestamp '%v': %v", s, err.Error())
		return false
	}
	return true
}

const k8sServicePrefix = "k8s-service:"

func validateIPOrK8sService(fl validator.FieldLevel) bool {
	s := strings.ToLower(fl.Field().String())
	log.Debugf("Validate IP or Kubernetes service: %s", s)

	checkPort := func(portStr string) bool {
		if port, err := strconv.Atoi(portStr); err != nil {
			log.Debugf("Invalid port '%v': %v", portStr, err.Error())
			return false
		} else if port < 0 || port > 65535 {
			log.Debugf("Port '%v' should be between 0 and 65535", port)
			return false
		}
		return true
	}

	if strings.HasPrefix(s, k8sServicePrefix) {
		s := s[len(k8sServicePrefix):]
		if slash := strings.Index(s, "/"); slash >= 0 {
			if ok := nameRegex.MatchString(s[:slash]); !ok {
				log.Debugf("Invalid chars in namespace '%v'", s[:slash])
				return false
			}
			s = s[slash+1:]
		}
		if colon := strings.Index(s, ":"); colon >= 0 {
			if !checkPort(s[colon+1:]) {
				return false
			}
			s = s[:colon]
		}
		if ok := nameRegex.MatchString(s); !ok {
			log.Debugf("Invalid chars in service name '%v'", s)
			return false
		}
		// Valid.
		return true
	}

	// 10.25.3.4
	// 10.25.3.4:536
	// [fd10:25::2]:536
	// fd10:25::2
	if colon := strings.Index(s, "]:"); colon >= 0 && strings.HasPrefix(s, "[") {
		// IPv6 address with port number.
		if !checkPort(s[colon+2:]) {
			return false
		}
		s = s[1:colon]
	} else if colon := strings.Index(s, ":"); colon >= 0 && strings.Count(s, ":") == 1 {
		// IPv4 address with port number.
		if !checkPort(s[colon+1:]) {
			return false
		}
		s = s[:colon]
	}
	if net.ParseIP(s) == nil {
		log.Debugf("Invalid IP address '%v'", s)
		return false
	}

	return true
}

func validateContainerID(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate containerID: %s", s)
	return containerIDRegex.MatchString(s)
}

func validatePrometheusHost(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate prometheusHost: %s", s)
	return prometheusHostRegexp.MatchString(s)
}

func validatePortName(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate port name: %s", s)
	return len(s) != 0 && len(k8svalidation.IsValidPortName(s)) == 0
}

func validateMustBeNil(fl validator.FieldLevel) bool {
	log.WithField("field", fl.Field().String()).Debugf("Validate field must be nil")
	return fl.Field().IsNil()
}

func validateMustBeFalse(fl validator.FieldLevel) bool {
	log.WithField("field", fl.Field().String()).Debugf("Validate field must be false")
	return !fl.Field().Bool()
}

func validateIPVersion(fl validator.FieldLevel) bool {
	ver := fl.Field().Int()
	log.Debugf("Validate ip version: %d", ver)
	return ver == 4 || ver == 6
}

func validateIPIPMode(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate IPIP Mode: %s", s)
	return ipipModeRegex.MatchString(s)
}

func validateIPType(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate IPType: %s", s)
	return ipTypeRegex.MatchString(s)
}

func validateStagedAction(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate StagedAction Mode: %s", s)
	return stagedActionRegex.MatchString(s)
}

func validateVXLANMode(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate VXLAN Mode: %s", s)
	return vxlanModeRegex.MatchString(s)
}

func RegexValidator(desc string, rx *regexp.Regexp) func(fl validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		log.Debugf("Validate %s: %s", desc, s)
		return rx.MatchString(s)
	}
}

func validateMAC(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate MAC Address: %s", s)

	if _, err := net.ParseMAC(s); err != nil {
		return false
	}
	return true
}

func validateIptablesBackend(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Iptables Backend: %s", s)
	return s == "" || s == api.IptablesBackendAuto || s == api.IptablesBackendNFTables || s == api.IptablesBackendLegacy
}

func validateLogLevel(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Felix log level: %s", s)
	return logLevelRegex.MatchString(s)
}

func validateIPSecLogLevel(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate IPSec log level: %s", s)
	return IPSeclogLevelRegex.MatchString(s)
}

func validateIPSecMode(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate IPSec mode: %s", s)
	return IPSecModeRegex.MatchString(s)
}

func validateBPFLogLevel(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Felix BPF log level: %s", s)
	return bpfLogLevelRegex.MatchString(s)
}

func validateBPFServiceMode(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Felix BPF service mode: %s", s)
	return bpfServiceModeRegex.MatchString(s)
}

func validateFelixEtoHAction(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Felix DefaultEndpointToHostAction: %s", s)
	return dropAcceptReturnRegex.MatchString(s)
}

func validateAcceptReturn(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Accept Return Action: %s", s)
	return acceptReturnRegex.MatchString(s)
}

func validateDropActionOverride(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate DropActionOverride: %s", s)
	return dropActionOverrideRegex.MatchString(s)
}

func validateDNSAggregationKind(fl validator.FieldLevel) bool {
	kind := int(fl.Field().Int())
	log.Debugf("Validate DNS logs aggregation kind: %d", kind)
	return kind >= minDNSAggregationKindValue && kind <= maxDNSAggregationKindValue
}

func validateL7HTTPHeaderAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs HTTP Header aggregation: %s", s)
	return L7HTTPHeaderInfoAggregationRegex.MatchString(s)
}

func validateL7HTTPMethodAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs HTTP Method aggregation: %s", s)
	return L7HTTPMethodAggregationRegex.MatchString(s)
}

func validateL7ServiceAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs service aggregation: %s", s)
	return L7ServiceAggregationRegex.MatchString(s)
}

func validateL7DestinationAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs destination aggregation: %s", s)
	return L7DestinationAggregationRegex.MatchString(s)
}

func validateL7SourceAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs source aggregation: %s", s)
	return L7SourceAggregationRegex.MatchString(s)
}

func validateL7ResponseCodeAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs response code aggregation: %s", s)
	return L7ResponseCodeAggregationRegex.MatchString(s)
}

func validateL7URLAggregation(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate L7 logs URL aggregation: %s", s)
	return L7URLAggregationRegex.MatchString(s)
}

func validateSelector(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate selector: %s", s)

	// We use the selector parser to validate a selector string.
	_, err := selector.Parse(s)
	if err != nil {
		log.Debugf("Selector %#v was invalid: %v", s, err)
		return false
	}
	return true
}

func validateFlowLogAggregationKind(fl validator.FieldLevel) bool {
	kind := int(fl.Field().Int())
	log.Debugf("Validate FlowLogs aggregation kind: %d", kind)
	return kind >= minAggregationKindValue && kind <= maxAggregationKindValue
}

func validateTag(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate tag: %s", s)
	return nameRegex.MatchString(s)
}

func validateLabels(fl validator.FieldLevel) bool {
	labels := fl.Field().Interface().(map[string]string)
	for k, v := range labels {
		if len(k8svalidation.IsQualifiedName(k)) != 0 {
			return false
		}
		if len(k8svalidation.IsValidLabelValue(v)) != 0 {
			return false
		}
	}
	return true
}

func validatePolicyType(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate policy type: %s", s)
	if s == string(api.PolicyTypeIngress) || s == string(api.PolicyTypeEgress) {
		return true
	}
	return false
}

func validateProtocol(structLevel validator.StructLevel) {
	p := structLevel.Current().Interface().(numorstring.Protocol)
	log.Debugf("Validate protocol: %v %s %d", p.Type, p.StrVal, p.NumVal)

	// The protocol field may be an integer 1-255 (i.e. not 0), or one of the valid protocol
	// names.
	if num, err := p.NumValue(); err == nil {
		if num == 0 {
			structLevel.ReportError(reflect.ValueOf(p.NumVal),
				"Protocol", "", reason("protocol number invalid"), "")
		}
	} else if !protocolRegex.MatchString(p.String()) {
		structLevel.ReportError(reflect.ValueOf(p.String()),
			"Protocol", "", reason("protocol name invalid"), "")
	}
}

// validateIPv4Network validates the field is a valid (strictly masked) IPv4 network.
// An IP address is valid, and assumed to be fully masked (i.e /32)
func validateIPv4Network(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate IPv4 network: %s", n)
	ipa, ipn, err := cnet.ParseCIDROrIP(n)
	if err != nil {
		return false
	}

	// Check for the correct version and that the CIDR is correctly masked (by comparing the
	// parsed IP against the IP in the parsed network).
	return ipa.Version() == 4 && ipn.IP.String() == ipa.String()
}

// validateIPv4Network validates the field is a valid (strictly masked) IPv6 network.
// An IP address is valid, and assumed to be fully masked (i.e /128)
func validateIPv6Network(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate IPv6 network: %s", n)
	ipa, ipn, err := cnet.ParseCIDROrIP(n)
	if err != nil {
		return false
	}

	// Check for the correct version and that the CIDR is correctly masked (by comparing the
	// parsed IP against the IP in the parsed network).
	return ipa.Version() == 6 && ipn.IP.String() == ipa.String()
}

// validateIPv4Network validates the field is a valid (strictly masked) IP network.
// An IP address is valid, and assumed to be fully masked (i.e /32 or /128)
func validateIPNetwork(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate IP network: %s", n)
	ipa, ipn, err := cnet.ParseCIDROrIP(n)
	if err != nil {
		return false
	}

	// Check  that the CIDR is correctly masked (by comparing the parsed IP against
	// the IP in the parsed network).
	return ipn.IP.String() == ipa.String()
}

// validateIPv4Network validates the field is a valid (not strictly masked) IPv4 network.
// An IP address is valid, and assumed to be fully masked (i.e /32)
func validateCIDRv4(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate IPv4 network: %s", n)
	ipa, _, err := cnet.ParseCIDROrIP(n)
	if err != nil {
		return false
	}

	return ipa.Version() == 4
}

// validateIPv4Network validates the field is a valid (not strictly masked) IPv6 network.
// An IP address is valid, and assumed to be fully masked (i.e /128)
func validateCIDRv6(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate IPv6 network: %s", n)
	ipa, _, err := cnet.ParseCIDROrIP(n)
	if err != nil {
		return false
	}

	return ipa.Version() == 6
}

// validateCIDR validates the field is a valid (not strictly masked) IP network.
// An IP address is valid, and assumed to be fully masked (i.e /32 or /128)
func validateCIDR(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate IP network: %s", n)
	_, _, err := cnet.ParseCIDROrIP(n)
	return err == nil
}

// validateCIDRs validates the field is a slice of valid (not strictly masked) IP networks.
// An IP address is valid, and assumed to be fully masked (i.e /32 or /128)
func validateCIDRs(fl validator.FieldLevel) bool {
	addrs := fl.Field().Interface().([]string)
	log.Debugf("Validate IP CIDRs: %s", addrs)
	for _, addr := range addrs {
		_, _, err := cnet.ParseCIDROrIP(addr)
		if err != nil {
			return false
		}
	}
	return true
}

// validateKeyValueList validates the field is a comma separated list of key=value pairs.
var kvRegex = regexp.MustCompile("^\\s*(\\w+)=(.*)$")

func validateKeyValueList(fl validator.FieldLevel) bool {
	n := fl.Field().String()
	log.Debugf("Validate KeyValueList: %s", n)

	if len(strings.TrimSpace(n)) == 0 {
		return true
	}

	for _, item := range strings.Split(n, ",") {
		if item == "" {
			// Accept empty items (e.g tailing ",")
			continue
		}
		kv := kvRegex.FindStringSubmatch(item)
		if kv == nil {
			return false
		}
	}

	return true
}

// validateIPPort validates the IP and Port given in either <IPv4>:<port> or [<IPv6>]:<port> or <IP> format
func validateIPPort(fl validator.FieldLevel) bool {
	ipPort := fl.Field().String()
	_, _, ok := processIPPort(ipPort)
	return ok
}

// processIPPort processes the IP and Port given in either <IPv4>:<port> or [<IPv6>]:<port> or <IP> format
// and return the IP, port and a bool if the format is as expected
func processIPPort(ipPort string) (string, int, bool) {
	if ipPort != "" {
		var ipStr, portStr string
		var err error
		var port uint64
		ipStr = ipPort
		// If PeerIP has both IP and port, validate both
		if IPv4PortFormat.MatchString(ipPort) || IPv6PortFormat.MatchString(ipPort) {
			ipStr, portStr, err = net.SplitHostPort(ipPort)
			if err != nil {
				log.Debugf("PeerIP value is invalid, it should either be \"<IP>\" or \"<IPv4>:<port>\" or \"[<IPv6>]:<port>\".")
				return "", 0, false
			}
			port, err = strconv.ParseUint(portStr, 10, 16)
			if err != nil {
				log.Debugf("PeerIP value has invalid port.")
				return "", 0, false
			}
			if port < 1 {
				log.Debugf("PeerIP value has invalid port.")
				return "", 0, false
			}
		}

		parsedIP := net.ParseIP(ipStr)
		if parsedIP == nil {
			log.Debugf("PeerIP value is invalid.")
			return "", 0, false
		}

		return ipStr, int(port), true
	}
	return "", 0, false
}

// validateHTTPMethods checks if the HTTP method match clauses are valid.
func validateHTTPMethods(methods []string) error {
	// check for duplicates
	s := set.FromArray(methods)
	if s.Len() != len(methods) {
		return fmt.Errorf("Invalid methods (duplicates): %v", methods)
	}
	return nil
}

// validateHTTPPaths checks if the HTTP path match clauses are valid.
func validateHTTPPaths(paths []api.HTTPPath) error {
	for _, path := range paths {
		if path.Exact != "" && path.Prefix != "" {
			return fmt.Errorf("Invalid path match. Both 'exact' and 'prefix' are set")
		}
		v := path.Exact
		if v == "" {
			v = path.Prefix
		}
		if v == "" {
			return fmt.Errorf("Invalid path match. Either 'exact' or 'prefix' must be set")
		}
		// Checks from https://tools.ietf.org/html/rfc3986#page-22
		if !strings.HasPrefix(v, "/") ||
			strings.ContainsAny(v, "? #") {
			return fmt.Errorf("Invalid path %s. (must start with `/` and not contain `?` or `#`", v)
		}
	}
	return nil
}

func validateHTTPRule(structLevel validator.StructLevel) {
	h := structLevel.Current().Interface().(api.HTTPMatch)
	log.Debugf("Validate HTTP Rule: %v", h)
	if err := validateHTTPMethods(h.Methods); err != nil {
		structLevel.ReportError(reflect.ValueOf(h.Methods), "Methods", "", reason(err.Error()), "")
	}
	if err := validateHTTPPaths(h.Paths); err != nil {
		structLevel.ReportError(reflect.ValueOf(h.Paths), "Paths", "", reason(err.Error()), "")
	}
}

func validatePort(structLevel validator.StructLevel) {
	p := structLevel.Current().Interface().(numorstring.Port)

	// Check that the port range is in the correct order.  The YAML parsing also checks this,
	// but this protects against misuse of the programmatic API.
	log.Debugf("Validate port: %v", p)
	if p.MinPort > p.MaxPort {
		structLevel.ReportError(reflect.ValueOf(p.MaxPort),
			"Port", "", reason("port range invalid"), "")
	}

	if p.PortName != "" {
		if p.MinPort != 0 || p.MaxPort != 0 {
			structLevel.ReportError(reflect.ValueOf(p.PortName),
				"Port", "", reason("named port invalid, if name is specified, min and max should be 0"), "")
		}
	} else if p.MinPort < 1 {
		structLevel.ReportError(reflect.ValueOf(p.MinPort),
			"Port", "", reason("port range invalid, port number must be between 1 and 65535"), "")
	} else if p.MaxPort < 1 {
		structLevel.ReportError(reflect.ValueOf(p.MaxPort),
			"Port", "", reason("port range invalid, port number must be between 1 and 65535"), "")
	}
}

func validateIPNAT(structLevel validator.StructLevel) {
	i := structLevel.Current().Interface().(libapi.IPNAT)
	log.Debugf("Internal IP: %s; External IP: %s", i.InternalIP, i.ExternalIP)

	iip, _, err := cnet.ParseCIDROrIP(i.InternalIP)
	if err != nil {
		structLevel.ReportError(reflect.ValueOf(i.ExternalIP),
			"InternalIP", "", reason("invalid IP address"), "")
	}

	eip, _, err := cnet.ParseCIDROrIP(i.ExternalIP)
	if err != nil {
		structLevel.ReportError(reflect.ValueOf(i.ExternalIP),
			"InternalIP", "", reason("invalid IP address"), "")
	}

	// An IPNAT must have both the internal and external IP versions the same.
	if iip.Version() != eip.Version() {
		structLevel.ReportError(reflect.ValueOf(i.ExternalIP),
			"ExternalIP", "", reason("mismatched IP versions"), "")
	}
}

func validateFile(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate file: %s", s)
	return fileRegex.MatchString(s)
}

func validateK8sEndpoint(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate K8s Endpoint: %s", s)
	return k8sEndpointRegex.MatchString(s)
}

func validateClusterAccessSecret(fl validator.FieldLevel) bool {
	or := fl.Field().Interface().(k8sv1.ObjectReference)
	s := k8sv1.Secret{}
	log.Debugf("Validate clusterAccessSecret")
	if or.Kind != s.GetObjectKind().GroupVersionKind().Kind {
		log.Debugf("Only Secret references are allowed for a ClusterAccessSecret")
		return false
	}
	return true
}

func validateEtcdEndpoints(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Etcd Endpoints: %s", s)
	return etcdEndpointsRegex.MatchString(s)
}

func validateUIDescription(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate UI description: %s", s)

	// Just check the length of the runes (limit to 64)
	runes := utf8.RuneCountInString(s)
	return runes > 0 && runes < 65
}

func validateServiceGraphId(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Service Graph ID: %s", s)
	// TODO(rlb): Move service graph API defs into API and then move ID management into libcalico-go. For now just
	//            allow.
	return true
}

func validateServiceGraphNodeName(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Service Graph Node Name: %s", s)

	// Normal name format is ok.
	return graphNodeNameRegex.MatchString(s)
}

func validateIcon(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Service Graph Icon Type: %s", s)
	// TODO(rlb): Decide what the UI team need and then we can implement - most likely just an SVG checker.
	//            For now always allow.
	return true
}

func validateColor(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Service Graph Color Type: %s", s)
	// TODO(rlb): Decide what the UI team need and then we can implement - probably any valid SVG color format.
	//            For now always allow.
	return true
}

func validateServiceGraphNodeType(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	log.Debugf("Validate Service Graph Node Type: %s", s)

	// TODO(rlb): Move service graph API defs into API and then move ID management into libcalico-go. For now just
	//            check for a sensible length.
	l := len(s)
	return l > 0 && l < 64
}

func validateFelixConfigSpec(structLevel validator.StructLevel) {
	c := structLevel.Current().Interface().(api.FelixConfigurationSpec)

	// Validate that the node port ranges list isn't too long and contains only numeric ports.
	// We set the limit at 7 because the iptables multiport match can accept at most 15 port
	// numbers, with each port range requiring 2 entries.
	if c.KubeNodePortRanges != nil {
		if len(*c.KubeNodePortRanges) > 7 {
			structLevel.ReportError(reflect.ValueOf(*c.KubeNodePortRanges),
				"KubeNodePortRanges", "",
				reason("node port ranges list is too long (max 7)"), "")
		}

		for _, p := range *c.KubeNodePortRanges {
			if p.PortName != "" {
				structLevel.ReportError(reflect.ValueOf(*c.KubeNodePortRanges),
					"KubeNodePortRanges", "",
					reason("node port ranges should not contain named ports"), "")
			}
		}
	}

	// Validate that the externalNodesCIDRList is composed of valid cidr's.
	if c.ExternalNodesCIDRList != nil {
		for _, cidr := range *c.ExternalNodesCIDRList {
			log.Debugf("Cidr is: %s", cidr)
			ip, _, err := cnet.ParseCIDROrIP(cidr)
			if err != nil {
				structLevel.ReportError(reflect.ValueOf(cidr),
					"ExternalNodesCIDRList", "", reason("has invalid CIDR(s)"), "")
			} else if ip.Version() != 4 {
				structLevel.ReportError(reflect.ValueOf(cidr),
					"ExternalNodesCIDRList", "", reason("has invalid IPv6 CIDR"), "")
			}
		}
	}

	// Validate that the OpenStack region is suitable for use in a namespace name.
	const regionNamespacePrefix = "openstack-region-"
	const maxRegionLength int = k8svalidation.DNS1123LabelMaxLength - len(regionNamespacePrefix)
	if len(c.OpenstackRegion) > maxRegionLength {
		structLevel.ReportError(reflect.ValueOf(c.OpenstackRegion),
			"OpenstackRegion", "", reason("is too long"), "")
	} else if len(c.OpenstackRegion) > 0 {
		problems := k8svalidation.IsDNS1123Label(c.OpenstackRegion)
		if len(problems) > 0 {
			structLevel.ReportError(reflect.ValueOf(c.OpenstackRegion),
				"OpenstackRegion", "", reason("must be a valid DNS label"), "")
		}
	}

	// Validate that the WindowsNetworkName is a valid regex.
	if c.WindowsNetworkName != nil {
		_, err := regexp.Compile(*c.WindowsNetworkName)
		if err != nil {
			structLevel.ReportError(reflect.ValueOf(*c.WindowsNetworkName),
				"WindowsNetworkName", "", reason("must be a valid regular expression"), "")
		}
	}

	if c.NATOutgoingAddress != "" {
		parsedAddress := cnet.ParseIP(c.NATOutgoingAddress)
		if parsedAddress == nil || parsedAddress.Version() != 4 {
			structLevel.ReportError(reflect.ValueOf(c.NATOutgoingAddress),
				"NATOutgoingAddress", "", reason("is not a valid IPv4 address"), "")
		}
	}

	if c.DeviceRouteSourceAddress != "" {
		parsedAddress := cnet.ParseIP(c.DeviceRouteSourceAddress)
		if parsedAddress == nil || parsedAddress.Version() != 4 {
			structLevel.ReportError(reflect.ValueOf(c.DeviceRouteSourceAddress),
				"DeviceRouteSourceAddress", "", reason("is not a valid IPv4 address"), "")
		}
	}

	if c.DeviceRouteSourceAddressIPv6 != "" {
		parsedAddress := cnet.ParseIP(c.DeviceRouteSourceAddressIPv6)
		if parsedAddress == nil || parsedAddress.Version() != 6 {
			structLevel.ReportError(reflect.ValueOf(c.DeviceRouteSourceAddressIPv6),
				"DeviceRouteSourceAddressIPv6", "", reason("is not a valid IPv6 address"), "")
		}
	}

	if c.RouteTableRange != nil && c.RouteTableRanges != nil {
		structLevel.ReportError(reflect.ValueOf(c.RouteTableRange),
			"RouteTableRange", "", reason("cannot be set when `RouteTableRanges` is also set"), "")
	}

	if c.RouteTableRanges != nil && c.RouteTableRanges.NumDesignatedTables() > int(routeTableRangeMaxTables) {
		structLevel.ReportError(reflect.ValueOf(c.RouteTableRanges),
			"RouteTableRanges", "", reason("targets too many tables"), "")
	}
}

func validateWorkloadEndpointSpec(structLevel validator.StructLevel) {
	w := structLevel.Current().Interface().(libapi.WorkloadEndpointSpec)

	// The configured networks only support /32 (for IPv4) and /128 (for IPv6) at present.
	for _, netw := range w.IPNetworks {
		_, nw, err := cnet.ParseCIDROrIP(netw)
		if err != nil {
			structLevel.ReportError(reflect.ValueOf(netw),
				"IPNetworks", "", reason("invalid CIDR"), "")
		}

		ones, bits := nw.Mask.Size()
		if bits != ones {
			structLevel.ReportError(reflect.ValueOf(w.IPNetworks),
				"IPNetworks", "", reason("IP network contains multiple addresses"), "")
		}
	}

	_, v4gw, err := cnet.ParseCIDROrIP(w.IPv4Gateway)
	if err != nil {
		structLevel.ReportError(reflect.ValueOf(w.IPv4Gateway),
			"IPv4Gateway", "", reason("invalid CIDR"), "")
	}

	_, v6gw, err := cnet.ParseCIDROrIP(w.IPv6Gateway)
	if err != nil {
		structLevel.ReportError(reflect.ValueOf(w.IPv6Gateway),
			"IPv6Gateway", "", reason("invalid CIDR"), "")
	}

	if v4gw.IP != nil && v4gw.Version() != 4 {
		structLevel.ReportError(reflect.ValueOf(w.IPv4Gateway),
			"IPv4Gateway", "", reason("invalid IPv4 gateway address specified"), "")
	}

	if v6gw.IP != nil && v6gw.Version() != 6 {
		structLevel.ReportError(reflect.ValueOf(w.IPv6Gateway),
			"IPv6Gateway", "", reason("invalid IPv6 gateway address specified"), "")
	}

	// If NATs have been specified, then they should each be within the configured networks of
	// the endpoint.
	if len(w.IPNATs) > 0 {
		valid := false
		for _, nat := range w.IPNATs {
			_, natCidr, err := cnet.ParseCIDROrIP(nat.InternalIP)
			if err != nil {
				structLevel.ReportError(reflect.ValueOf(nat.InternalIP),
					"IPNATs", "", reason("invalid InternalIP CIDR"), "")
			}
			// Check each NAT to ensure it is within the configured networks.  If any
			// are not then exit without further checks.
			valid = false
			for _, cidr := range w.IPNetworks {
				_, nw, err := cnet.ParseCIDROrIP(cidr)
				if err != nil {
					structLevel.ReportError(reflect.ValueOf(cidr),
						"IPNetworks", "", reason("invalid CIDR"), "")
				}

				if nw.Contains(natCidr.IP) {
					valid = true
					break
				}
			}
			if !valid {
				break
			}
		}

		if !valid {
			structLevel.ReportError(reflect.ValueOf(w.IPNATs),
				"IPNATs", "", reason("NAT is not in the endpoint networks"), "")
		}
	}
}

func validateHostEndpointSpec(structLevel validator.StructLevel) {
	h := structLevel.Current().Interface().(api.HostEndpointSpec)

	// A host endpoint must have an interface name and/or some expected IPs specified.
	if h.InterfaceName == "" && len(h.ExpectedIPs) == 0 {
		structLevel.ReportError(reflect.ValueOf(h.InterfaceName),
			"InterfaceName", "", reason("no interface or expected IPs have been specified"), "")
	}
	// A host endpoint must have a nodename specified.
	if h.Node == "" {
		structLevel.ReportError(reflect.ValueOf(h.Node),
			"InterfaceName", "", reason("no node has been specified"), "")
	}
}

func validateRemoteClusterConfigSpec(structLevel validator.StructLevel) {
	h := structLevel.Current().Interface().(api.RemoteClusterConfigurationSpec)

	if h.ClusterAccessSecret != nil {
		emptyRCC := api.RemoteClusterConfigurationSpec{}
		emptyRCC.ClusterAccessSecret = h.ClusterAccessSecret

		if !reflect.DeepEqual(h, emptyRCC) {
			structLevel.ReportError(reflect.ValueOf(h),
				"RemoteClusterConfigurationSpec", "", reason("When ClusterAccessSecret is set all other fields should not be set."), "")

		}
		return
	}

	if !datastoreType.MatchString(h.DatastoreType) {
		structLevel.ReportError(reflect.ValueOf(h.DatastoreType),
			"DatastoreType", "", reason("DatastoreType is invalid"), "")
	}

	// An etcdv3 remote cluster may also have kubernetes datastore configuration for the federation
	// controller.  However, a kubernetes datastore should not have etcdv3 configuration specified.
	if h.DatastoreType == "kubernetes" {
		if len(h.EtcdEndpoints) != 0 {
			structLevel.ReportError(reflect.ValueOf(h.EtcdEndpoints),
				"EtcdEndpoints", "", reason("EtcdEndpoints can't be specified if the datastore type is 'kubernetes'"), "")
		}
		if len(h.EtcdCACertFile) != 0 {
			structLevel.ReportError(reflect.ValueOf(h.EtcdCACertFile),
				"EtcdCACertFile", "", reason("EtcdCACertFile can't be specified if the datastore type is 'kubernetes'"), "")
		}
		if len(h.EtcdCertFile) != 0 {
			structLevel.ReportError(reflect.ValueOf(h.EtcdCertFile),
				"EtcdCertFile", "", reason("EtcdCertFile can't be specified if the datastore type is 'kubernetes'"), "")
		}
		if len(h.EtcdKeyFile) != 0 {
			structLevel.ReportError(reflect.ValueOf(h.EtcdKeyFile),
				"EtcdKeyFile", "", reason("EtcdKeyFile can't be specified if the datastore type is 'kubernetes'"), "")
		}
		if len(h.EtcdPassword) != 0 {
			structLevel.ReportError(reflect.ValueOf(h.EtcdPassword),
				"EtcdPassword", "", reason("EtcdPassword can't be specified if the datastore type is 'kubernetes'"), "")
		}
		if len(h.EtcdPassword) != 0 {
			structLevel.ReportError(reflect.ValueOf(h.EtcdPassword),
				"EtcdPassword", "", reason("EtcdPassword can't be specified if the datastore type is 'kubernetes'"), "")
		}
	}
}

func validateIPPoolSpec(structLevel validator.StructLevel) {
	pool := structLevel.Current().Interface().(api.IPPoolSpec)

	// Spec.CIDR field must not be empty.
	if pool.CIDR == "" {
		structLevel.ReportError(reflect.ValueOf(pool.CIDR),
			"IPpool.CIDR", "", reason("IPPool CIDR must be specified"), "")
	}

	// Make sure the CIDR is parsable.
	ipAddr, cidr, err := cnet.ParseCIDROrIP(pool.CIDR)
	if err != nil {
		structLevel.ReportError(reflect.ValueOf(pool.CIDR),
			"IPpool.CIDR", "", reason("IPPool CIDR must be a valid subnet"), "")
		return
	}

	// Normalize the CIDR before persisting.
	pool.CIDR = cidr.String()

	// IPIP cannot be enabled for IPv6.
	if cidr.Version() == 6 && pool.IPIPMode != api.IPIPModeNever {
		structLevel.ReportError(reflect.ValueOf(pool.IPIPMode),
			"IPpool.IPIPMode", "", reason("IPIPMode other than 'Never' is not supported on an IPv6 IP pool"), "")
	}

	// Cannot have both VXLAN and IPIP on the same IP pool.
	if ipipModeEnabled(pool.IPIPMode) && vxLanModeEnabled(pool.VXLANMode) {
		structLevel.ReportError(reflect.ValueOf(pool.IPIPMode),
			"IPpool.IPIPMode", "", reason("IPIPMode and VXLANMode cannot both be enabled on the same IP pool"), "")
	}

	if pool.AWSSubnetID != "" {
		// AWS-backed IP pools have additional restrictions...
		if !awsSubnetRE.MatchString(pool.AWSSubnetID) {
			structLevel.ReportError(reflect.ValueOf(pool.AWSSubnetID),
				"IPpool.AWSSubnetID", "", reason(invalidAWSSubnetID), "")
		}
		if ipAddr.Version() != 4 {
			structLevel.ReportError(reflect.ValueOf(pool.AWSSubnetID),
				"IPpool.CIDR", "", reason(ipv6AWSSubnet), "")
		}
		if pool.BlockSize == 0 {
			// AWS doesn't require aggregation into blocks and the main use case of AWS-backed pools is for
			// egress gateways, which often have fixed IPs.  We don't want to allocate large blocks and then
			// get stuck with an affinity that blocks us from moving the block if IP borrowing is turned off.
			pool.BlockSize = 32
		}
		if pool.BlockSize != 32 {
			structLevel.ReportError(reflect.ValueOf(pool.AWSSubnetID),
				"IPpool.BlockSize", "", reason("AWS-backed IP pools should use block size 32"), "")
		}
	}

	// Default the blockSize
	if pool.BlockSize == 0 {
		if ipAddr.Version() == 4 {
			pool.BlockSize = 26
		} else {
			pool.BlockSize = 122
		}
	}

	// The Calico IPAM places restrictions on the minimum IP pool size.  If
	// the ippool is enabled, check that the pool is at least the minimum size.
	if !pool.Disabled {
		ones, _ := cidr.Mask.Size()
		log.Debugf("Pool CIDR: %s, mask: %d, blockSize: %d", cidr.String(), ones, pool.BlockSize)
		if ones > pool.BlockSize {
			structLevel.ReportError(reflect.ValueOf(pool.CIDR),
				"IPpool.CIDR", "", reason("IP pool size is too small for use with Calico IPAM. It must be equal to or greater than the block size."), "")
		}
	}

	// The Calico CIDR should be strictly masked
	log.Debugf("IPPool CIDR: %s, Masked IP: %d", pool.CIDR, cidr.IP)
	if cidr.IP.String() != ipAddr.String() {
		structLevel.ReportError(reflect.ValueOf(pool.CIDR),
			"IPpool.CIDR", "", reason(poolUnstictCIDR), "")
	}

	// IPv4 link local subnet.
	ipv4LinkLocalNet := net.IPNet{
		IP:   net.ParseIP("169.254.0.0"),
		Mask: net.CIDRMask(16, 32),
	}
	// IPv6 link local subnet.
	ipv6LinkLocalNet := net.IPNet{
		IP:   net.ParseIP("fe80::"),
		Mask: net.CIDRMask(10, 128),
	}

	// IP Pool CIDR cannot overlap with IPv4 or IPv6 link local address range.
	if cidr.Version() == 4 && cidr.IsNetOverlap(ipv4LinkLocalNet) {
		structLevel.ReportError(reflect.ValueOf(pool.CIDR),
			"IPpool.CIDR", "", reason(overlapsV4LinkLocal), "")
	}

	if cidr.Version() == 6 && cidr.IsNetOverlap(ipv6LinkLocalNet) {
		structLevel.ReportError(reflect.ValueOf(pool.CIDR),
			"IPpool.CIDR", "", reason(overlapsV6LinkLocal), "")
	}

	// Allowed use must be one of the enums.
	for _, a := range pool.AllowedUses {
		switch a {
		case api.IPPoolAllowedUseWorkload, api.IPPoolAllowedUseTunnel, api.IPPoolAllowedUseHostSecondary:
			continue
		default:
			structLevel.ReportError(reflect.ValueOf(pool.AllowedUses),
				"IPpool.AllowedUses", "", reason("unknown use: "+string(a)), "")
		}
	}
}

func vxLanModeEnabled(mode api.VXLANMode) bool {
	return mode == api.VXLANModeAlways || mode == api.VXLANModeCrossSubnet
}

func ipipModeEnabled(mode api.IPIPMode) bool {
	return mode == api.IPIPModeAlways || mode == api.IPIPModeCrossSubnet
}

func validateICMPFields(structLevel validator.StructLevel) {
	icmp := structLevel.Current().Interface().(api.ICMPFields)

	// Due to Kernel limitations, ICMP code must always be specified with a type.
	if icmp.Code != nil && icmp.Type == nil {
		structLevel.ReportError(reflect.ValueOf(icmp.Code),
			"Code", "", reason("ICMP code specified without an ICMP type"), "")
	}
}

func validateRule(structLevel validator.StructLevel) {
	rule := structLevel.Current().Interface().(api.Rule)

	// If the protocol does not support ports check that the port values have not
	// been specified.
	if rule.Protocol == nil || !rule.Protocol.SupportsPorts() {
		if len(rule.Source.Ports) > 0 {
			structLevel.ReportError(reflect.ValueOf(rule.Source.Ports),
				"Source.Ports", "", reason(protocolPortsMsg), "")
		}
		if len(rule.Source.NotPorts) > 0 {
			structLevel.ReportError(reflect.ValueOf(rule.Source.NotPorts),
				"Source.NotPorts", "", reason(protocolPortsMsg), "")
		}

		if len(rule.Destination.Ports) > 0 {
			structLevel.ReportError(reflect.ValueOf(rule.Destination.Ports),
				"Destination.Ports", "", reason(protocolPortsMsg), "")
		}
		if len(rule.Destination.NotPorts) > 0 {
			structLevel.ReportError(reflect.ValueOf(rule.Destination.NotPorts),
				"Destination.NotPorts", "", reason(protocolPortsMsg), "")
		}
	}

	// Check that HTTP must not use non-TCP protocols
	if rule.HTTP != nil && rule.Protocol != nil {
		tcp := numorstring.ProtocolFromString("TCP")
		if *rule.Protocol != tcp {
			structLevel.ReportError(reflect.ValueOf(rule.Protocol), "Protocol", "", reason(protocolAndHTTPMsg), "")
		}
	}

	icmp := numorstring.ProtocolFromString("ICMP")
	icmpv6 := numorstring.ProtocolFromString("ICMPv6")
	if rule.ICMP != nil && (rule.Protocol == nil || (*rule.Protocol != icmp && *rule.Protocol != icmpv6)) {
		structLevel.ReportError(reflect.ValueOf(rule.ICMP), "ICMP", "", reason(protocolIcmpMsg), "")
	}

	// Check that the IPVersion of the protocol matches the IPVersion of the ICMP protocol.
	if (rule.Protocol != nil && *rule.Protocol == icmp) || (rule.NotProtocol != nil && *rule.NotProtocol == icmp) {
		if rule.IPVersion != nil && *rule.IPVersion != 4 {
			structLevel.ReportError(reflect.ValueOf(rule.ICMP), "IPVersion", "", reason("must set ipversion to '4' with protocol icmp"), "")
		}
	}
	if (rule.Protocol != nil && *rule.Protocol == icmpv6) || (rule.NotProtocol != nil && *rule.NotProtocol == icmpv6) {
		if rule.IPVersion != nil && *rule.IPVersion != 6 {
			structLevel.ReportError(reflect.ValueOf(rule.ICMP), "IPVersion", "", reason("must set ipversion to '6' with protocol icmpv6"), "")
		}
	}

	var seenV4, seenV6 bool

	scanNets := func(nets []string, fieldName string) {
		var v4, v6 bool
		for _, n := range nets {
			_, cidr, err := cnet.ParseCIDR(n)
			if err != nil {
				structLevel.ReportError(reflect.ValueOf(n), fieldName,
					"", reason("invalid CIDR"), "")
			} else {
				v4 = v4 || cidr.Version() == 4
				v6 = v6 || cidr.Version() == 6
			}
		}
		if rule.IPVersion != nil && ((v4 && *rule.IPVersion != 4) || (v6 && *rule.IPVersion != 6)) {
			structLevel.ReportError(reflect.ValueOf(rule.Source.Nets), fieldName,
				"", reason("rule IP version doesn't match CIDR version"), "")
		}
		if v4 && seenV6 || v6 && seenV4 || v4 && v6 {
			// This field makes the rule inconsistent.
			structLevel.ReportError(reflect.ValueOf(nets), fieldName,
				"", reason("rule contains both IPv4 and IPv6 CIDRs"), "")
		}
		seenV4 = seenV4 || v4
		seenV6 = seenV6 || v6
	}

	scanNets(rule.Source.Nets, "Source.Nets")
	scanNets(rule.Source.NotNets, "Source.NotNets")
	scanNets(rule.Destination.Nets, "Destination.Nets")
	scanNets(rule.Destination.NotNets, "Destination.NotNets")

	usesALP, alpValue, alpField := ruleUsesAppLayerPolicy(&rule)
	if rule.Action != api.Allow && usesALP {
		structLevel.ReportError(alpValue, alpField,
			"", reason("only valid for Allow rules"), "")
	}

	if len(rule.Source.Domains) != 0 {
		structLevel.ReportError(reflect.ValueOf(rule.Source.Domains), "Source.Domains",
			"", reason("Domains can only be specified in the destination of an egress Allow rule"), "")
	}

	if len(rule.Destination.Domains) != 0 {
		if rule.Action != api.Allow {
			structLevel.ReportError(reflect.ValueOf(rule.Destination.Domains), "Destination.Domains",
				"", reason("only valid for Allow rules"), "")
		}
		if len(rule.Destination.Nets) != 0 {
			structLevel.ReportError(reflect.ValueOf(rule.Destination.Nets), "Destination.Nets",
				"", reason("must be left empty when Destination.Domains is specified"), "")
		}
		if rule.Destination.Selector != "" {
			structLevel.ReportError(reflect.ValueOf(rule.Destination.Selector), "Destination.Selector",
				"", reason("must be left empty when Destination.Domains is specified"), "")
		}
	}

	// Check that destination service rules do not use ports.
	// Destination service rules use ports specified on the endpoints.
	if rule.Destination.Services != nil && len(rule.Destination.Ports) != 0 {
		structLevel.ReportError(reflect.ValueOf(rule.Destination.Ports),
			"Destination.Ports", "", reason("cannot specify ports with a service selector"), "")
	}
	if rule.Destination.Services != nil && len(rule.Destination.NotPorts) != 0 {
		structLevel.ReportError(reflect.ValueOf(rule.Destination.NotPorts),
			"Destination.NotPorts", "", reason("cannot specify notports with a service selector"), "")
	}
}

func validateEntityRule(structLevel validator.StructLevel) {
	rule := structLevel.Current().Interface().(api.EntityRule)
	if strings.Contains(rule.Selector, globalSelector) {
		structLevel.ReportError(reflect.ValueOf(rule.Selector),
			"Selector field", "", reason(globalSelectorEntRule), "")
	}

	// Get the parsed and canonicalised string of the namespaceSelector
	// so we can make assertions against it.
	// Note: err can be ignored; the field is validated separately and before
	// this point.
	n, _ := selector.Parse(rule.NamespaceSelector)
	namespaceSelector := n.String()

	// If the namespaceSelector contains global(), then it should be the only selector.
	if globalSelectorRegex.MatchString(namespaceSelector) {
		structLevel.ReportError(reflect.ValueOf(rule.NamespaceSelector),
			"NamespaceSelector field", "", reason(globalSelectorOnly), "")
	}

	if rule.Services != nil {
		// Make sure it's not empty.
		if rule.Services.Name == "" {
			structLevel.ReportError(reflect.ValueOf(rule.Services),
				"Services field", "", reason("must specify a service name"), "")
		}

		// Make sure the rest of the entity rule is consistent.
		if rule.NamespaceSelector != "" {
			structLevel.ReportError(reflect.ValueOf(rule.Services),
				"Services field", "", reason("cannot specify NamespaceSelector and Services on the same rule"), "")
		}
		if rule.Selector != "" || rule.NotSelector != "" {
			structLevel.ReportError(reflect.ValueOf(rule.Services),
				"Services field", "", reason("cannot specify Selector/NotSelector and Services on the same rule"), "")
		}
		if rule.ServiceAccounts != nil {
			structLevel.ReportError(reflect.ValueOf(rule.Services),
				"Services field", "", reason("cannot specify ServiceAccounts and Services on the same rule"), "")
		}
		if len(rule.Nets) != 0 || len(rule.NotNets) != 0 {
			// Service rules use IPs specified on the endpoints.
			structLevel.ReportError(reflect.ValueOf(rule.Services),
				"Services field", "", reason("cannot specify Nets/NotNets and Services on the same rule"), "")
		}
	}
}

func validateIPAMConfigSpec(structLevel validator.StructLevel) {
	ics := structLevel.Current().Interface().(libapi.IPAMConfigSpec)

	if ics.MaxBlocksPerHost < 0 {
		structLevel.ReportError(reflect.ValueOf(ics.MaxBlocksPerHost), "MaxBlocksPerHost", "",
			reason("must be greater than or equal to 0"), "")
	}
}

func validateNodeSpec(structLevel validator.StructLevel) {
	ns := structLevel.Current().Interface().(libapi.NodeSpec)

	if ns.BGP != nil {
		if reflect.DeepEqual(*ns.BGP, libapi.NodeBGPSpec{}) {
			structLevel.ReportError(reflect.ValueOf(ns.BGP), "BGP", "",
				reason("Spec.BGP should not be empty"), "")
		}
	}
}

func validateBGPPeerSpec(structLevel validator.StructLevel) {
	ps := structLevel.Current().Interface().(api.BGPPeerSpec)

	if ps.Node != "" && ps.NodeSelector != "" {
		structLevel.ReportError(reflect.ValueOf(ps.Node), "Node", "",
			reason("Node field must be empty when NodeSelector is specified"), "")
	}
	if ps.PeerIP != "" && ps.PeerSelector != "" {
		structLevel.ReportError(reflect.ValueOf(ps.PeerIP), "PeerIP", "",
			reason("PeerIP field must be empty when PeerSelector is specified"), "")
	}
	if uint32(ps.ASNumber) != 0 && ps.PeerSelector != "" {
		structLevel.ReportError(reflect.ValueOf(ps.ASNumber), "ASNumber", "",
			reason("ASNumber field must be empty when PeerSelector is specified"), "")
	}
	ok, msg := validateReachableBy(ps.ReachableBy, ps.PeerIP)
	if !ok {
		structLevel.ReportError(reflect.ValueOf(ps.ReachableBy), "ReachableBy", "",
			reason(msg), "")
	}
}

func validateReachableBy(reachableBy, peerIP string) (bool, string) {
	if reachableBy == "" {
		return true, ""
	}
	if reachableBy != "" && peerIP == "" {
		return false, "ReachablyBy field must be empty when PeerIP is empty"
	}
	reachableByAddr := cnet.ParseIP(reachableBy)
	if reachableByAddr == nil {
		return false, "ReachableBy is invalid address"
	}
	peerAddrStr, _, ok := processIPPort(peerIP)
	if !ok {
		return false, "PeerIP is invalid address"
	}
	peerAddr := cnet.ParseIP(peerAddrStr)
	if peerAddr == nil {
		return false, "PeerIP is invalid IP address"
	}
	if reachableByAddr.Version() != peerAddr.Version() {
		return false, "ReachableBy and PeerIP address family mismatched"
	}
	return true, ""
}

// validateReachableByField validates that reachableBy value, the address of the
// gateway the BGP peer is connected to, is a correct address
func validateReachableByField(fl validator.FieldLevel) bool {
	reachableBy := fl.Field().String()

	if reachableBy != "" {
		reachableByAddr := cnet.ParseIP(reachableBy)
		if reachableByAddr == nil {
			log.Debugf("ReachableBy value is invalid address")
			return false
		}
	}
	return true
}

func validateEndpointPort(structLevel validator.StructLevel) {
	port := structLevel.Current().Interface().(api.EndpointPort)

	if !port.Protocol.SupportsPorts() {
		structLevel.ReportError(
			reflect.ValueOf(port.Protocol),
			"EndpointPort.Protocol",
			"",
			reason("EndpointPort protocol does not support ports."),
			"",
		)
	}
}

func validateWorkloadEndpointPort(structLevel validator.StructLevel) {
	port := structLevel.Current().Interface().(libapi.WorkloadEndpointPort)

	if !port.Protocol.SupportsPorts() {
		structLevel.ReportError(
			reflect.ValueOf(port.Protocol),
			"WorkloadEndpointPort.Protocol",
			"",
			reason("WorkloadEndpointPort protocol does not support ports."),
			"",
		)
	}

	if port.Name == "" && port.HostPort == 0 {
		structLevel.ReportError(
			reflect.ValueOf(port.Name),
			"WorkloadEndpointPort.Name",
			"",
			reason("WorkloadEndpointPort name must not be empty if no HostPort is specified"),
			"",
		)
	}
}

func validateProtoPort(structLevel validator.StructLevel) {
	m := structLevel.Current().Interface().(api.ProtoPort)

	if m.Protocol != "TCP" && m.Protocol != "UDP" && m.Protocol != "SCTP" {
		structLevel.ReportError(
			reflect.ValueOf(m.Protocol),
			"ProtoPort.Protocol",
			"",
			reason("protocol must be 'TCP' or 'UDP' or 'SCTP'."),
			"",
		)
	}
}

func validateObjectMeta(structLevel validator.StructLevel) {
	om := structLevel.Current().Interface().(metav1.ObjectMeta)

	// Check the name is within the max length.
	if len(om.Name) > k8svalidation.DNS1123SubdomainMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(om.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(om.Name)-k8svalidation.DNS1123SubdomainMaxLength)),
			"",
		)
	}

	// Uses the k8s DN1123 subdomain format for most resource names.
	matched := nameRegex.MatchString(om.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(om.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters, '-' or '.' (regex: "+nameSubdomainFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, om.Annotations)
	validateObjectMetaLabels(structLevel, om.Labels)
}

func validateTier(structLevel validator.StructLevel) {
	tier := structLevel.Current().Interface().(api.Tier)

	// Check the name is within the max length.
	// Tier names are dependent on the label max length since policy lookup by tier in KDD requires the name to fit in a label.
	if len(tier.Name) > k8svalidation.DNS1123LabelMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(tier.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(tier.Name)-k8svalidation.DNS1123LabelMaxLength)),
			"",
		)
	}

	// Tiers must have simple (no dot) names, since they appear as sub-components of other names.
	matched := tierNameRegex.MatchString(tier.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(tier.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, tier.Annotations)
	validateObjectMetaLabels(structLevel, tier.Labels)
}

func validateUISettingsGroup(structLevel validator.StructLevel) {
	group := structLevel.Current().Interface().(api.UISettingsGroup)

	// Check the name is within the max length.
	// UISettingsGroup names are dependent on the label max length since settings lookup by group in KDD requires the
	// name to fit in a label.
	if len(group.Name) > k8svalidation.DNS1123LabelMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(group.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(group.Name)-k8svalidation.DNS1123LabelMaxLength)),
			"",
		)
	}

	// UISettingsGroups must have simple (no dot) names, since they appear as sub-components of UISettings.
	matched := uiSettingsGroupNameRegex.MatchString(group.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(group.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, group.Annotations)
	validateObjectMetaLabels(structLevel, group.Labels)
}

func validateUISettings(structLevel validator.StructLevel) {
	uisettings := structLevel.Current().Interface().(api.UISettings)

	group := uisettings.Spec.Group

	// Groups must have simple (no dot) names, since they appear as sub-components of the UISettings.
	matched := uiSettingsGroupNameRegex.MatchString(group)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(group),
			"Spec.Group",
			"",
			reason("name must only consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	if !strings.HasPrefix(uisettings.Name, group+".") {
		log.Infof("UISettings name %v is not prefixed by the UISettingsGroup name %v", uisettings.Name, group)
		structLevel.ReportError(
			reflect.ValueOf(group),
			"Spec.Group",
			"",
			reason("UISettings name is not prefixed by the UISettings group name"),
			"",
		)
	}

	numSettings := 0
	var settingsField string
	if uisettings.Spec.View != nil {
		numSettings++
		settingsField = "Spec.View"
	}
	if uisettings.Spec.Layer != nil {
		numSettings++
		settingsField = "Spec.Layer"
	}
	if uisettings.Spec.Dashboard != nil {
		numSettings++
		settingsField = "Spec.Dashboard"
	}

	if numSettings != 1 {
		structLevel.ReportError(
			reflect.ValueOf(uisettings.Name),
			settingsField,
			"",
			reason("UISettings should consists of exactly one View, Layer or Dashboard"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, uisettings.Annotations)
	validateObjectMetaLabels(structLevel, uisettings.Labels)
}

func validateNetworkPolicySpec(spec *api.NetworkPolicySpec, structLevel validator.StructLevel) {
	// Check (and disallow) any repeats in Types field.
	mp := map[api.PolicyType]bool{}
	for _, t := range spec.Types {
		if _, exists := mp[t]; exists {
			structLevel.ReportError(reflect.ValueOf(spec.Types),
				"NetworkPolicySpec.Types", "", reason("'"+string(t)+"' type specified more than once"), "")
		} else {
			mp[t] = true
		}
	}

	for _, r := range spec.Egress {
		// Services are only allowed in the destination on Egress rules.
		if r.Source.Services != nil {
			structLevel.ReportError(
				reflect.ValueOf(r.Source.Services), "Services", "",
				reason("not allowed in egress rule source"), "",
			)
		}

		// Check (and disallow) rules with application layer policy for egress rules.
		useALP, v, f := ruleUsesAppLayerPolicy(&r)
		if useALP {
			structLevel.ReportError(v, f, "", reason("not allowed in egress rule"), "")
		}
	}

	// Services are only allowed in the source on Ingress rules.
	for _, r := range spec.Ingress {
		if r.Destination.Services != nil {
			structLevel.ReportError(
				reflect.ValueOf(r.Destination.Services), "Services", "",
				reason("not allowed in ingress rule destination"), "",
			)
		}
	}

	// Check that the selector doesn't have the global() selector which is only
	// valid as an EntityRule namespaceSelector.
	if strings.Contains(spec.Selector, globalSelector) {
		structLevel.ReportError(
			reflect.ValueOf(spec.Selector),
			"NetworkPolicySpec.Selector",
			"",
			reason(globalSelectorEntRule),
			"")
	}

	if strings.Contains(spec.ServiceAccountSelector, globalSelector) {
		structLevel.ReportError(
			reflect.ValueOf(spec.ServiceAccountSelector),
			"NetworkPolicySpec.ServiceAccountSelector",
			"",
			reason(globalSelectorEntRule),
			"")
	}
}

func validateNetworkPolicy(structLevel validator.StructLevel) {
	np := structLevel.Current().Interface().(api.NetworkPolicy)
	spec := np.Spec

	// Check the name is within the max length.
	if len(np.Name) > k8svalidation.DNS1123SubdomainMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(np.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(np.Name)-k8svalidation.DNS1123SubdomainMaxLength)),
			"",
		)
	}

	// Uses the k8s DN1123 label format for policy names (plus knp.default prefixed k8s policies).
	matched := networkPolicyNameRegex.MatchString(np.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(np.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, np.Annotations)
	validateObjectMetaLabels(structLevel, np.Labels)

	validateNetworkPolicySpec(&spec, structLevel)
}

func validateStagedNetworkPolicy(structLevel validator.StructLevel) {
	staged := structLevel.Current().Interface().(api.StagedNetworkPolicy)

	// Check the name is within the max length.
	if len(staged.Name) > k8svalidation.DNS1123SubdomainMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(staged.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(staged.Name)-k8svalidation.DNS1123SubdomainMaxLength)),
			"",
		)
	}

	// Uses the k8s DN1123 label format for policy names (plus knp.default prefixed k8s policies).
	matched := networkPolicyNameRegex.MatchString(staged.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(staged.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, staged.Annotations)
	validateObjectMetaLabels(structLevel, staged.Labels)

	_, enforced := api.ConvertStagedPolicyToEnforced(&staged)

	if staged.Spec.StagedAction == api.StagedActionDelete {
		empty := api.NetworkPolicySpec{}
		empty.Tier = enforced.Spec.Tier
		if !reflect.DeepEqual(empty, enforced.Spec) {
			structLevel.ReportError(reflect.ValueOf(staged.Spec),
				"StagedNetworkPolicySpec", "", reason("Spec fields, except Tier, should all be zero-value if stagedAction is Delete"), "")
		}
	} else {
		validateNetworkPolicySpec(&enforced.Spec, structLevel)
	}
}

func validateNetworkSet(structLevel validator.StructLevel) {
	ns := structLevel.Current().Interface().(api.NetworkSet)
	for k := range ns.GetLabels() {
		if k == "projectcalico.org/namespace" {
			// The namespace label should only be used when mapping the real namespace through
			// to the v1 datamodel.  It shouldn't appear in the v3 datamodel.
			structLevel.ReportError(
				reflect.ValueOf(k),
				"Metadata.Labels (label)",
				"",
				reason("projectcalico.org/namespace is not a valid label name"),
				"",
			)
		}
	}
}

func validatePacketCaptureSpec(structLevel validator.StructLevel) {
	spec := structLevel.Current().Interface().(api.PacketCaptureSpec)

	if spec.StartTime != nil && spec.EndTime != nil {
		var start = spec.StartTime.Time
		var end = spec.EndTime.Time
		// endTime < startTime
		if start.After(end) {
			structLevel.ReportError(reflect.ValueOf(end),
				"EndTime", "", reason("must be set after startTime"), "")

		}
		// endTime == startTime
		if start.Equal(end) {
			structLevel.ReportError(reflect.ValueOf(end),
				"EndTime", "", reason("must have a different value than startTime"), "")

		}
	}
}

func validatePacketCaptureRule(structLevel validator.StructLevel) {
	rule := structLevel.Current().Interface().(api.PacketCaptureRule)

	// If the protocol does not support ports check that the port values have not
	// been specified.
	if rule.Protocol != nil && !rule.Protocol.SupportsPorts() {
		if len(rule.Ports) > 0 {
			structLevel.ReportError(reflect.ValueOf(rule.Protocol),
				"Protocol", "", reason("protocol does not accept ports"), "")
		}
	}

	if len(rule.Ports) != 0 {
		for _, port := range rule.Ports {
			if !allDigits.MatchString(port.String()) && !portRange.MatchString(port.String()) {
				structLevel.ReportError(reflect.ValueOf(rule.Ports),
					"Ports", "", reason("accepts only numerical values"), "")
			}
		}
	}
}

func validatePacketCapture(structLevel validator.StructLevel) {
	pc := structLevel.Current().Interface().(api.PacketCapture)
	for k := range pc.GetLabels() {
		if k == "projectcalico.org/namespace" {
			// The namespace label should only be used when mapping the real namespace through
			// to the v1 datamodel.  It shouldn't appear in the v3 datamodel.
			structLevel.ReportError(
				reflect.ValueOf(k),
				"Metadata.LabelPacketCaptures (label)",
				"",
				reason("projectcalico.org/namespace is not a valid label name"),
				"",
			)
		}
	}
}

func validateDeepPacketInspection(structLevel validator.StructLevel) {
	dpi := structLevel.Current().Interface().(api.DeepPacketInspection)
	for k := range dpi.GetLabels() {
		if k == "projectcalico.org/namespace" {
			// The namespace label should only be used when mapping the real namespace through
			// to the v1 datamodel.  It shouldn't appear in the v3 datamodel.
			structLevel.ReportError(
				reflect.ValueOf(k),
				"Metadata.LabelDeepPacketInspections (label)",
				"",
				reason("projectcalico.org/namespace is not a valid label name"),
				"",
			)
		}
	}
}

func validateGlobalNetworkSet(structLevel validator.StructLevel) {
	gns := structLevel.Current().Interface().(api.GlobalNetworkSet)
	for k := range gns.GetLabels() {
		if k == "projectcalico.org/namespace" {
			// The namespace label should only be used when mapping the real namespace through
			// to the v1 datamodel.  It shouldn't appear in the v3 datamodel.
			structLevel.ReportError(
				reflect.ValueOf(k),
				"Metadata.Labels (label)",
				"",
				reason("projectcalico.org/namespace is not a valid label name"),
				"",
			)
		}
	}
}

func validateGlobalNetworkPolicySpec(spec *api.GlobalNetworkPolicySpec, structLevel validator.StructLevel) {
	if spec.DoNotTrack && spec.PreDNAT {
		structLevel.ReportError(reflect.ValueOf(spec.PreDNAT),
			"PolicySpec.PreDNAT", "", reason("PreDNAT and DoNotTrack cannot both be true, for a given PolicySpec"), "")
	}

	if spec.PreDNAT && len(spec.Egress) > 0 {
		structLevel.ReportError(reflect.ValueOf(spec.Egress),
			"PolicySpec.Egress", "", reason("PreDNAT PolicySpec cannot have any Egress rules"), "")
	}

	if spec.PreDNAT && len(spec.Types) > 0 {
		for _, t := range spec.Types {
			if t == api.PolicyTypeEgress {
				structLevel.ReportError(reflect.ValueOf(spec.Types),
					"PolicySpec.Types", "", reason("PreDNAT PolicySpec cannot have 'egress' Type"), "")
			}
		}
	}

	if !spec.ApplyOnForward && (spec.DoNotTrack || spec.PreDNAT) {
		structLevel.ReportError(reflect.ValueOf(spec.ApplyOnForward),
			"PolicySpec.ApplyOnForward", "", reason("ApplyOnForward must be true if either PreDNAT or DoNotTrack is true, for a given PolicySpec"), "")
	}

	// Check (and disallow) any repeats in Types field.
	mp := map[api.PolicyType]bool{}
	for _, t := range spec.Types {
		if _, exists := mp[t]; exists {
			structLevel.ReportError(reflect.ValueOf(spec.Types),
				"GlobalNetworkPolicySpec.Types", "", reason("'"+string(t)+"' type specified more than once"), "")
		} else {
			mp[t] = true
		}
	}

	for _, r := range spec.Egress {
		// Services are only allowed as a destination on Egress rules.
		if r.Source.Services != nil {
			structLevel.ReportError(
				reflect.ValueOf(r.Source.Services), "Services", "",
				reason("not allowed in egress rule source"), "",
			)
		}

		// Check (and disallow) rules with application layer policy for egress rules.
		useALP, v, f := ruleUsesAppLayerPolicy(&r)
		if useALP {
			structLevel.ReportError(v, f, "", reason("not allowed in egress rules"), "")
		}
	}

	// Services are only allowed as a source on Ingress rules.
	for _, r := range spec.Ingress {
		if r.Destination.Services != nil {
			structLevel.ReportError(
				reflect.ValueOf(r.Destination.Services), "Services", "",
				reason("not allowed in ingress rule destination"), "",
			)
		}
	}

	// If a ServiceSelector is specified by name, we also need a namespace. At a global scope,
	// service names are not fully qualified and so need a namespace.
	for _, r := range spec.Egress {
		if r.Destination.Services != nil && r.Destination.Services.Namespace == "" {
			structLevel.ReportError(
				reflect.ValueOf(r.Destination.Services.Namespace), "Namespace", "",
				reason("must specify a namespace"), "",
			)
		}
	}
	for _, r := range spec.Ingress {
		if r.Source.Services != nil && r.Source.Services.Namespace == "" {
			structLevel.ReportError(
				reflect.ValueOf(r.Source.Services.Namespace), "Namespace", "",
				reason("must specify a namespace"), "",
			)
		}
	}

	// Check that the selector doesn't have the global() selector which is only
	// valid as an EntityRule namespaceSelector.
	if strings.Contains(spec.Selector, globalSelector) {
		structLevel.ReportError(
			reflect.ValueOf(spec.Selector),
			"GlobalNetworkPolicySpec.Selector",
			"",
			reason(globalSelectorEntRule),
			"")
	}

	if strings.Contains(spec.ServiceAccountSelector, globalSelector) {
		structLevel.ReportError(
			reflect.ValueOf(spec.Selector),
			"GlobalNetworkPolicySpec.ServiceAccountSelector",
			"",
			reason(globalSelectorEntRule),
			"")
	}

	if strings.Contains(spec.NamespaceSelector, globalSelector) {
		structLevel.ReportError(
			reflect.ValueOf(spec.Selector),
			"GlobalNetworkPolicySpec.NamespaceSelector",
			"",
			reason(globalSelectorEntRule),
			"")
	}
}

func validateGlobalNetworkPolicy(structLevel validator.StructLevel) {
	gnp := structLevel.Current().Interface().(api.GlobalNetworkPolicy)
	spec := gnp.Spec

	// Check the name is within the max length.
	if len(gnp.Name) > k8svalidation.DNS1123SubdomainMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(gnp.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(gnp.Name)-k8svalidation.DNS1123SubdomainMaxLength)),
			"",
		)
	}

	// Uses the k8s DN1123 label format for policy names.
	matched := globalNetworkPolicyNameRegex.MatchString(gnp.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(gnp.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, gnp.Annotations)
	validateObjectMetaLabels(structLevel, gnp.Labels)
	validateGlobalNetworkPolicySpec(&spec, structLevel)
}

func validateStagedGlobalNetworkPolicy(structLevel validator.StructLevel) {
	staged := structLevel.Current().Interface().(api.StagedGlobalNetworkPolicy)

	// Check the name is within the max length.
	if len(staged.Name) > k8svalidation.DNS1123SubdomainMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(staged.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(staged.Name)-k8svalidation.DNS1123SubdomainMaxLength)),
			"",
		)
	}

	// Uses the k8s DN1123 label format for policy names.
	matched := globalNetworkPolicyNameRegex.MatchString(staged.Name)
	if !matched {
		structLevel.ReportError(
			reflect.ValueOf(staged.Name),
			"Metadata.Name",
			"",
			reason("name must consist of lower case alphanumeric characters or '-' (regex: "+nameLabelFmt+")"),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, staged.Annotations)
	validateObjectMetaLabels(structLevel, staged.Labels)

	_, enforced := api.ConvertStagedGlobalPolicyToEnforced(&staged)

	if staged.Spec.StagedAction == api.StagedActionDelete {
		// the network policy fields should all "zero-value" when the update type is "delete"
		empty := api.GlobalNetworkPolicySpec{}
		empty.Tier = enforced.Spec.Tier
		if !reflect.DeepEqual(empty, enforced.Spec) {
			structLevel.ReportError(reflect.ValueOf(staged.Spec),
				"StagedGlobalNetworkPolicySpec", "", reason("Spec fields, except Tier, should all be zero-value if stagedAction is Delete"), "")
		}
	} else {
		validateGlobalNetworkPolicySpec(&enforced.Spec, structLevel)
	}
}

func validateStagedKubernetesNetworkPolicy(structLevel validator.StructLevel) {
	staged := structLevel.Current().Interface().(api.StagedKubernetesNetworkPolicy)

	// Check the name is within the max length.
	if len(staged.Name) > k8svalidation.DNS1123SubdomainMaxLength {
		structLevel.ReportError(
			reflect.ValueOf(staged.Name),
			"Metadata.Name",
			"",
			reason(fmt.Sprintf("name is too long by %d bytes", len(staged.Name)-k8svalidation.DNS1123SubdomainMaxLength)),
			"",
		)
	}

	validateObjectMetaAnnotations(structLevel, staged.Annotations)
	validateObjectMetaLabels(structLevel, staged.Labels)

	if staged.Spec.StagedAction == api.StagedActionDelete {
		// the network policy fields should all "zero-value" when the update type is "delete"
		empty := api.NewStagedKubernetesNetworkPolicy()
		empty.Spec.StagedAction = api.StagedActionDelete
		if !reflect.DeepEqual(empty.Spec, staged.Spec) {
			structLevel.ReportError(reflect.ValueOf(staged.Spec),
				"StagedKubernetesNetworkPolicySpec", "", reason("Spec fields should all be zero-value if stagedAction is Delete"), "")
		}
	} else {
		c := calicoconversion.NewConverter()
		_, v1np := api.ConvertStagedKubernetesPolicyToK8SEnforced(&staged)
		npKVPair, err := c.K8sNetworkPolicyToCalico(v1np)
		if err != nil {
			structLevel.ReportError(
				reflect.ValueOf(staged.Spec),
				"PolicySpec",
				"",
				reason(fmt.Sprintf("conversion to stagednetworkpolicy failed %v", err)),
				"",
			)
		}

		v3np := npKVPair.Value.(*api.NetworkPolicy)
		validateNetworkPolicySpec(&v3np.Spec, structLevel)
	}
}

func validatePull(structLevel validator.StructLevel) {
	p := structLevel.Current().Interface().(api.Pull)
	if p.Period == "" {
		// Allow empty, which means default.
		return
	}
	d, err := time.ParseDuration(p.Period)
	if err != nil {
		structLevel.ReportError(
			reflect.ValueOf(p.Period),
			"Period",
			"",
			reason("invalid duration string"),
			"")
		return
	}
	if d < api.MinPullPeriod {
		structLevel.ReportError(
			reflect.ValueOf(p.Period),
			"Period",
			"",
			reason("Period cannot be shorter than 5m"),
			"")
	}
	return
}

func validateReportTemplate(structLevel validator.StructLevel) {
	rt := structLevel.Current().Interface().(api.ReportTemplate)
	tmpl := rt.Template

	if tmpl != "" {
		// Validate template is ok using sensible data.
		_, err := compliance.RenderTemplate(tmpl, &compliance.ReportDataSample)
		if err != nil {
			structLevel.ReportError(
				reflect.ValueOf(rt.Template),
				"Template",
				"template",
				reason("Invalid template defined in: "+rt.Name+": "+err.Error()),
				"",
			)

			// No point in doing additional checks if the template doesn't validate with sensible data.
			return
		}

		// Run past nil pointer data to see if the template is valid.
		for i := range compliance.ReportDataNilEntries {
			_, err = compliance.RenderTemplate(tmpl, &compliance.ReportDataNilEntries[i])
			if err != nil {
				structLevel.ReportError(
					reflect.ValueOf(rt.Name),
					"Template",
					"template",
					reason("Template does not handle nil pointer in reportData field: "+
						compliance.ReportDataNilEntries[i].ReportName+": "+err.Error()),
					"",
				)
			}
		}
	}
}

func validateGlobalReportType(structLevel validator.StructLevel) {
	grt := structLevel.Current().Interface().(api.GlobalReportType)
	spec := grt.Spec

	// Validate unique name across templates.
	tmplNames := map[string]bool{spec.UISummaryTemplate.Name: true}
	for i, t := range spec.DownloadTemplates {
		if _, exists := tmplNames[t.Name]; exists {
			structLevel.ReportError(reflect.ValueOf(t.Name),
				fmt.Sprintf("Spec.DownloadTemplates[%d].Name", i), "", reason("template name '"+t.Name+"' is already in use."), "")
		}
	}
}

func validateReportSpec(structLevel validator.StructLevel) {
	spec := structLevel.Current().Interface().(api.ReportSpec)

	if spec.Schedule == "" {
		return
	}

	// Check that the cron tab parses ok.
	s, err := cron.ParseStandard(spec.Schedule)
	if err != nil {
		structLevel.ReportError(reflect.ValueOf(spec.Schedule),
			"Spec.Schedule", "", reason(fmt.Sprintf("schedule is not valid: %v", err)), "")
		return
	}

	// Check that there are at most 2 schedules per hour.
	if ss, ok := s.(*cron.SpecSchedule); ok {
		if bits.OnesCount64(ss.Minute&lower60Bits) > maxCRONSchedulesPerHour {
			structLevel.ReportError(reflect.ValueOf(spec.Schedule),
				"Spec.Schedule", "", reason(errorTooManySchedules), "")
		}
	}
}

func validateObjectMetaAnnotations(structLevel validator.StructLevel, annotations map[string]string) {
	var totalSize int64
	for k, v := range annotations {
		for _, errStr := range k8svalidation.IsQualifiedName(strings.ToLower(k)) {
			structLevel.ReportError(
				reflect.ValueOf(k),
				"Metadata.Annotations (key)",
				"",
				reason(errStr),
				"",
			)
		}
		totalSize += (int64)(len(k)) + (int64)(len(v))
	}

	if totalSize > (int64)(totalAnnotationSizeLimitB) {
		structLevel.ReportError(
			reflect.ValueOf(annotations),
			"Metadata.Annotations (key)",
			"",
			reason(fmt.Sprintf("total size of annotations is too large by %d bytes", totalSize-totalAnnotationSizeLimitB)),
			"",
		)
	}
}

func validateObjectMetaLabels(structLevel validator.StructLevel, labels map[string]string) {
	for k, v := range labels {
		for _, errStr := range k8svalidation.IsQualifiedName(k) {
			structLevel.ReportError(
				reflect.ValueOf(k),
				"Metadata.Labels (label)",
				"",
				reason(errStr),
				"",
			)
		}
		for _, errStr := range k8svalidation.IsValidLabelValue(v) {
			structLevel.ReportError(
				reflect.ValueOf(v),
				"Metadata.Labels (value)",
				"",
				reason(errStr),
				"",
			)
		}
	}
}

func validateRuleMetadata(structLevel validator.StructLevel) {
	ruleMeta := structLevel.Current().Interface().(api.RuleMetadata)
	validateObjectMetaAnnotations(structLevel, ruleMeta.Annotations)
}

func validateRouteTableRange(structLevel validator.StructLevel) {
	r := structLevel.Current().Interface().(api.RouteTableRange)
	if r.Min >= 1 && r.Max >= r.Min && r.Max <= 250 {
		log.Debugf("RouteTableRange is valid: %v", r)
	} else {
		log.Warningf("RouteTableRange is invalid: %v", r)
		structLevel.ReportError(
			reflect.ValueOf(r),
			"RouteTableRange",
			"",
			reason("must be a range of route table indices within 1..250"),
			"",
		)
	}
}

func validateRouteTableIDRange(structLevel validator.StructLevel) {
	r := structLevel.Current().Interface().(api.RouteTableIDRange)
	if r.Min > r.Max {
		log.Warningf("RouteTableRange is invalid: %v", r)
		structLevel.ReportError(
			reflect.ValueOf(r),
			"RouteTableRange",
			"",
			reason("min value cannot be greater than max value"),
			"",
		)
	}

	if r.Min <= 0 {
		log.Warningf("RouteTableRange is invalid: %v", r)
		structLevel.ReportError(
			reflect.ValueOf(r),
			"RouteTableRange",
			"",
			reason("cannot target indices < 1"),
			"",
		)
	}

	if int64(r.Max) > int64(routeTableMaxLinux) {
		log.Warningf("RouteTableRange is invalid: %v", r)
		structLevel.ReportError(
			reflect.ValueOf(r),
			"RouteTableRange",
			"",
			reason("max index too high"),
			"",
		)
	}

	// check if ranges collide with reserved linux tables
	includesReserved := false
	for _, rsrv := range routeTablesReservedLinux {
		if r.Min <= rsrv && r.Max >= rsrv {
			includesReserved = false
		}
	}
	if includesReserved {
		log.Infof("Felix route-table range includes reserved Linux tables, values 253-255 will be ignored.")
	}
}

func validateBGPConfigurationSpec(structLevel validator.StructLevel) {
	spec := structLevel.Current().Interface().(api.BGPConfigurationSpec)

	// check if Spec.Communities[] are valid
	communities := spec.Communities
	for _, community := range communities {
		isValid := isValidCommunity(community.Value, "Spec.Communities[].Value", structLevel)
		if !isValid {
			log.Warningf("community value is invalid: %v", community.Value)
			structLevel.ReportError(reflect.ValueOf(community.Value), "Spec.Communities[].Value", "",
				reason("invalid community value or format used."), "")
		}
	}

	if (len(spec.PrefixAdvertisements) == 0) && (len(communities) != 0) {
		structLevel.ReportError(reflect.ValueOf(communities), "Spec.Communities[]", "",
			reason("communities are defined but not used in Spec.PrefixAdvertisement[]."), "")
	}

	// check if Spec.PrefixAdvertisement.Communities are valid
	for _, pa := range spec.PrefixAdvertisements {
		_, _, err := cnet.ParseCIDROrIP(pa.CIDR)
		if err != nil {
			log.Warningf("CIDR value is invalid: %v", pa.CIDR)
			structLevel.ReportError(reflect.ValueOf(pa.CIDR), "Spec.PrefixAdvertisement[].CIDR", "",
				reason("invalid CIDR value."), "")
		}

		for _, v := range pa.Communities {
			isValid := isValidCommunity(v, "Spec.PrefixAdvertisement[].Communities[]", structLevel)
			if !isValid {
				if !isCommunityDefined(v, communities) {
					structLevel.ReportError(reflect.ValueOf(v), "Spec.PrefixAdvertisement[].Communities[]", "",
						reason("community used is invalid or not defined."), "")
				}
			}
		}
	}

	// Check that node mesh password cannot be set if node to node mesh is disabled.
	if spec.NodeMeshPassword != nil && spec.NodeToNodeMeshEnabled != nil && !*spec.NodeToNodeMeshEnabled {
		structLevel.ReportError(reflect.ValueOf(spec), "Spec.NodeMeshPassword", "", reason("spec.NodeMeshPassword cannot be set if spec.NodeToNodeMesh is disabled"), "")
	}

	// Check that node mesh max restart time cannot be set if node to node mesh is disabled.
	if spec.NodeMeshMaxRestartTime != nil && spec.NodeToNodeMeshEnabled != nil && !*spec.NodeToNodeMeshEnabled {
		structLevel.ReportError(reflect.ValueOf(spec), "Spec.NodeMeshMaxRestartTime", "", reason("spec.NodeMeshMaxRestartTime cannot be set if spec.NodeToNodeMesh is disabled"), "")
	}
}

func validateBlockAffinitySpec(structLevel validator.StructLevel) {
	spec := structLevel.Current().Interface().(libapi.BlockAffinitySpec)
	if spec.Deleted == fmt.Sprintf("%t", true) {
		structLevel.ReportError(reflect.ValueOf(spec), "Spec.Deleted", "", reason("spec.Deleted cannot be set to \"true\""), "")
	}
}

var htoNameRegex = regexp.MustCompile("^[a-zA-Z0-9_ -]+$")

func validateHealthTimeoutOverride(structLevel validator.StructLevel) {
	hto := structLevel.Current().Interface().(api.HealthTimeoutOverride)
	if !htoNameRegex.MatchString(hto.Name) {
		structLevel.ReportError(reflect.ValueOf(hto), "HealthTimeoutOverride.Name", "", reason("name should match regex "+htoNameRegex.String()), "")
	}
	if hto.Timeout.Duration < 0 {
		structLevel.ReportError(reflect.ValueOf(hto), "HealthTimeoutOverride.Timeout", "", reason("Timeout should not be negative"), "")
	}
}

func isCommunityDefined(community string, communityKVPairs []api.Community) bool {
	for _, val := range communityKVPairs {
		if val.Name == community {
			return true
		}
	}
	return false
}

func isValidCommunity(communityValue string, fieldName string, structLevel validator.StructLevel) bool {
	if standardCommunity.MatchString(communityValue) {
		validateCommunityValue(communityValue, fieldName, structLevel, false)
	} else if largeCommunity.MatchString(communityValue) {
		validateCommunityValue(communityValue, fieldName, structLevel, true)
	} else {
		return false
	}
	return true
}

// Validate that if standard community is used, community value must follow `aa:nn` format, where `aa` and `nn` are 16 bit integers,
// and if large community is used, value must follow `aa:nn:mm` format, where all `aa`, `nn` and `mm` are 32 bit integers.
func validateCommunityValue(val string, fieldName string, structLevel validator.StructLevel, isLargeCommunity bool) {
	splitValue := number.FindAllString(val, -1)
	bitSize := 16

	if isLargeCommunity {
		bitSize = 32
	}

	for _, v := range splitValue {
		_, err := strconv.ParseUint(v, 10, bitSize)
		if err != nil {
			structLevel.ReportError(reflect.ValueOf(val), fieldName, "",
				reason(fmt.Sprintf("invalid community value, expected %d bit value", bitSize)), "")
		}
	}
}

// ruleUsesAppLayerPolicy checks if a rule uses application layer policy, and
// if it does, returns true and the type of application layer clause. If it does
// not it returns false and the empty string.
func ruleUsesAppLayerPolicy(rule *api.Rule) (bool, reflect.Value, string) {
	if rule.HTTP != nil {
		return true, reflect.ValueOf(rule.HTTP), "HTTP"
	}
	return false, reflect.Value{}, ""
}
