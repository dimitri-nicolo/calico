// Copyright (c) 2016-2024 Tigera, Inc. All rights reserved.
//
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

package rules

import (
	"net"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/felix/config"
	"github.com/projectcalico/calico/felix/ipsets"
	"github.com/projectcalico/calico/felix/iptables"
	"github.com/projectcalico/calico/felix/proto"
)

const (
	// ChainNamePrefix is a prefix used for all our iptables chain names.  We include a '-' at
	// the end to reduce clashes with other apps.  Our OpenStack DHCP agent uses prefix
	// 'calico-dhcp-', for example.
	ChainNamePrefix = "cali-"
	// IPSetNamePrefix: similarly for IP sets, we use the following prefix; the IP sets layer
	// adds its own "-" so it isn't included here.
	IPSetNamePrefix = ipsets.IPSetNamePrefix

	ChainFilterInput   = ChainNamePrefix + "INPUT"
	ChainFilterForward = ChainNamePrefix + "FORWARD"
	ChainFilterOutput  = ChainNamePrefix + "OUTPUT"

	ChainRawPrerouting         = ChainNamePrefix + "PREROUTING"
	ChainRawOutput             = ChainNamePrefix + "OUTPUT"
	ChainRawUntrackedFlows     = ChainNamePrefix + "untracked-flows"
	ChainRawBPFUntrackedPolicy = ChainNamePrefix + "untracked-policy"

	ChainFailsafeIn  = ChainNamePrefix + "failsafe-in"
	ChainFailsafeOut = ChainNamePrefix + "failsafe-out"

	ChainNATPrerouting       = ChainNamePrefix + "PREROUTING"
	ChainNATPreroutingEgress = ChainNamePrefix + "egress"
	ChainNATPostrouting      = ChainNamePrefix + "POSTROUTING"
	ChainNATOutput           = ChainNamePrefix + "OUTPUT"
	ChainNATOutgoing         = ChainNamePrefix + "nat-outgoing"

	ChainManglePrerouting              = ChainNamePrefix + "PREROUTING"
	ChainManglePostrouting             = ChainNamePrefix + "POSTROUTING"
	ChainManglePreroutingEgress        = ChainNamePrefix + "pre-egress"
	ChainManglePreroutingEgressInbound = ChainNamePrefix + "pre-egress-in"
	ChainManglePostroutingEgress       = ChainNamePrefix + "post-egress"
	ChainManglePreroutingTProxySvc     = ChainNamePrefix + "pre-tproxy-svc"
	ChainManglePreroutingTProxyNP      = ChainNamePrefix + "pre-tproxy-np"
	ChainManglePreroutingTProxyEstabl  = ChainNamePrefix + "pre-tproxy-establ"
	ChainManglePreroutingTProxySelect  = ChainNamePrefix + "pre-tproxy-selec"
	ChainMangleOutput                  = ChainNamePrefix + "OUTPUT"
	ChainMangleOutputTProxy            = ChainNamePrefix + "out-mangle-tproxy"
	ChainMangleOutputTProxyHostNet     = ChainNamePrefix + "out-mangle-tproxy-host"

	IPSetIDNATOutgoingAllPools  = "all-ipam-pools"
	IPSetIDNATOutgoingMasqPools = "masq-ipam-pools"

	IPSetIDAllHostNets        = "all-hosts-net"
	IPSetIDAllVXLANSourceNets = "all-vxlan-net"
	IPSetIDThisHostIPs        = "this-host"
	IPSetIDAllTunnelNets      = "all-tunnel-net"
	IPSetIDAllEGWHealthPorts  = "egw-health-ports"

	ChainFIPDnat = ChainNamePrefix + "fip-dnat"
	ChainFIPSnat = ChainNamePrefix + "fip-snat"

	ChainCIDRBlock = ChainNamePrefix + "cidr-block"

	PolicyInboundPfx   PolicyChainNamePrefix  = ChainNamePrefix + "pi-"
	PolicyOutboundPfx  PolicyChainNamePrefix  = ChainNamePrefix + "po-"
	ProfileInboundPfx  ProfileChainNamePrefix = ChainNamePrefix + "pri-"
	ProfileOutboundPfx ProfileChainNamePrefix = ChainNamePrefix + "pro-"

	PolicyGroupInboundPrefix  string = ChainNamePrefix + "gi-"
	PolicyGroupOutboundPrefix string = ChainNamePrefix + "go-"

	ChainWorkloadToHost       = ChainNamePrefix + "wl-to-host"
	ChainFromWorkloadDispatch = ChainNamePrefix + "from-wl-dispatch"
	ChainToWorkloadDispatch   = ChainNamePrefix + "to-wl-dispatch"

	ChainDispatchToHostEndpoint          = ChainNamePrefix + "to-host-endpoint"
	ChainDispatchFromHostEndpoint        = ChainNamePrefix + "from-host-endpoint"
	ChainDispatchToHostEndpointForward   = ChainNamePrefix + "to-hep-forward"
	ChainDispatchFromHostEndPointForward = ChainNamePrefix + "from-hep-forward"
	ChainDispatchSetEndPointMark         = ChainNamePrefix + "set-endpoint-mark"
	ChainDispatchFromEndPointMark        = ChainNamePrefix + "from-endpoint-mark"

	ChainForwardCheck        = ChainNamePrefix + "forward-check"
	ChainForwardEndpointMark = ChainNamePrefix + "forward-endpoint-mark"

	ChainSetWireguardIncomingMark = ChainNamePrefix + "wireguard-incoming-mark"

	ChainRpfSkip = ChainNamePrefix + "rpf-skip"

	ChainDNSLog = ChainNamePrefix + "log-dns"

	WorkloadToEndpointPfx   = ChainNamePrefix + "tw-"
	WorkloadPfxSpecialAllow = "ALLOW"
	WorkloadFromEndpointPfx = ChainNamePrefix + "fw-"

	SetEndPointMarkPfx = ChainNamePrefix + "sm-"

	HostToEndpointPfx          = ChainNamePrefix + "th-"
	HostFromEndpointPfx        = ChainNamePrefix + "fh-"
	HostToEndpointForwardPfx   = ChainNamePrefix + "thfw-"
	HostFromEndpointForwardPfx = ChainNamePrefix + "fhfw-"

	RPFChain = ChainNamePrefix + "rpf"

	RuleHashPrefix = "cali:"

	// NFLOGPrefixMaxLength is NFLOG max prefix length which is 64 characters.
	// Ref: http://ipset.netfilter.org/iptables-extensions.man.html#lbDI
	NFLOGPrefixMaxLength = 64

	// NFLOG groups. 1 for inbound and 2 for outbound.  3 for
	// snooping DNS response for domain information.
	NFLOGInboundGroup  uint16 = 1
	NFLOGOutboundGroup uint16 = 2
	NFLOGDomainGroup   uint16 = 3

	// Windows Hns rule delimeter between prefix string, rule name and sequence number.
	WindowsHnsRuleNameDelimeter = "---"

	// HistoricNATRuleInsertRegex is a regex pattern to match to match
	// special-case rules inserted by old versions of felix.  Specifically,
	// Python felix used to insert a masquerade rule directly into the
	// POSTROUTING chain.
	//
	// Note: this regex depends on the output format of iptables-save so,
	// where possible, it's best to match only on part of the rule that
	// we're sure can't change (such as the ipset name in the masquerade
	// rule).
	HistoricInsertedNATRuleRegex = `-A POSTROUTING .* felix-masq-ipam-pools .*|` +
		`-A POSTROUTING -o tunl0 -m addrtype ! --src-type LOCAL --limit-iface-out -m addrtype --src-type LOCAL -j MASQUERADE`

	KubeProxyInsertRuleRegex = `-j KUBE-[a-zA-Z0-9-]*SERVICES|-j KUBE-FORWARD`

	ChainFilterInputTProxy  = ChainNamePrefix + "input-filter-tproxy"
	ChainFilterOutputTProxy = ChainNamePrefix + "output-filter-tproxy"
)

type PolicyDirection string

const (
	PolicyDirectionInbound  PolicyDirection = "inbound"  // AKA ingress
	PolicyDirectionOutbound PolicyDirection = "outbound" // AKA egress
)

type RuleAction byte

const (
	// We define these with specific byte values as we write this value directly into the NFLOG
	// prefix.
	RuleActionAllow RuleAction = 'A'
	RuleActionDeny  RuleAction = 'D'
	// Pass onto the next tier
	RuleActionPass RuleAction = 'P'
)

func (r RuleAction) String() string {
	switch r {
	case RuleActionAllow:
		return "Allow"
	case RuleActionDeny:
		return "Deny"
	case RuleActionPass:
		return "Pass"
	}
	return ""
}

type RuleDir byte

const (
	// We define these with specific byte values as we write this value directly into the NFLOG
	// prefix.
	RuleDirIngress RuleDir = 'I'
	RuleDirEgress  RuleDir = 'E'
)

func (r RuleDir) String() string {
	switch r {
	case RuleDirIngress:
		return "Ingress"
	case RuleDirEgress:
		return "Egress"
	}
	return ""
}

type RuleOwnerType byte

const (
	// We define these with specific byte values as we write this value directly into the NFLOG
	// prefix.
	RuleOwnerTypePolicy  RuleOwnerType = 'P'
	RuleOwnerTypeProfile RuleOwnerType = 'R'
)

func (r RuleOwnerType) String() string {
	switch r {
	case RuleOwnerTypePolicy:
		return "Policy"
	case RuleOwnerTypeProfile:
		return "Profile"
	}
	return ""
}

// Typedefs to prevent accidentally passing the wrong prefix to the Policy/ProfileChainName()
type (
	PolicyChainNamePrefix  string
	ProfileChainNamePrefix string
)

var (
	// AllHistoricChainNamePrefixes lists all the prefixes that we've used for chains.  Keeping
	// track of the old names lets us clean them up.
	AllHistoricChainNamePrefixes = []string{
		// Current.
		"cali-",

		// Early RCs of Felix 2.1 used "cali" as the prefix for some chains rather than
		// "cali-".  This led to name clashes with the DHCP agent, which uses "calico-" as
		// its prefix.  We need to explicitly list these exceptions.
		"califw-",
		"calitw-",
		"califh-",
		"calith-",
		"calipi-",
		"calipo-",

		// Pre Felix v2.1.
		"felix-",
	}
	// AllHistoricIPSetNamePrefixes, similarly contains all the prefixes we've ever used for IP
	// sets.
	AllHistoricIPSetNamePrefixes = []string{"felix-", "cali"}
	// LegacyV4IPSetNames contains some extra IP set names that were used in older versions of
	// Felix and don't fit our versioned pattern.
	LegacyV4IPSetNames = []string{"felix-masq-ipam-pools", "felix-all-ipam-pools"}

	// Rule previxes used by kube-proxy.  Note: we exclude the so-called utility chains KUBE-MARK-MASQ and co because
	// they are jointly owned by kube-proxy and kubelet.
	KubeProxyChainPrefixes = []string{
		"KUBE-FORWARD",
		"KUBE-SERVICES",
		"KUBE-EXTERNAL-SERVICES",
		"KUBE-NODEPORTS",
		"KUBE-SVC-",
		"KUBE-SEP-",
		"KUBE-FW-",
		"KUBE-XLB-",
	}
)

type RuleRenderer interface {
	StaticFilterTableChains(ipVersion uint8) []*iptables.Chain
	StaticNATTableChains(ipVersion uint8) []*iptables.Chain
	StaticNATPostroutingChains(ipVersion uint8) []*iptables.Chain
	StaticRawTableChains(ipVersion uint8) []*iptables.Chain
	StaticBPFModeRawChains(ipVersion uint8, wgEncryptHost, disableConntrack bool) []*iptables.Chain
	StaticMangleTableChains(ipVersion uint8) []*iptables.Chain
	StaticRawOutputChain(tcBypassMark uint32, nodelocaldnsBroadcastedIPs []config.ServerPort) *iptables.Chain
	StaticRawPreroutingChain(ipVersion uint8, nodelocaldnsBroadcastedIPs []config.ServerPort) *iptables.Chain
	StaticFilterForwardAppendRules() []iptables.Rule

	WorkloadDispatchChains(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint) []*iptables.Chain
	WorkloadRPFDispatchChains(ipVersion uint8, gatewayInterfaceNames []string) []*iptables.Chain
	WorkloadEndpointToIptablesChains(
		ifaceName string,
		epMarkMapper EndpointMarkMapper,
		adminUp bool,
		tiers []TierPolicyGroups,
		profileIDs []string,
		isEgressGateway bool,
		egwHealthPort uint16,
		ipVersion uint8,
	) []*iptables.Chain

	WorkloadInterfaceAllowChains(endpoints map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint) []*iptables.Chain

	EndpointMarkDispatchChains(
		epMarkMapper EndpointMarkMapper,
		wlEndpoints map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint,
		hepEndpoints map[string]proto.HostEndpointID,
	) []*iptables.Chain

	HostDispatchChains(map[string]proto.HostEndpointID, string, bool) []*iptables.Chain
	FromHostDispatchChains(map[string]proto.HostEndpointID, string) []*iptables.Chain
	ToHostDispatchChains(map[string]proto.HostEndpointID, string) []*iptables.Chain
	HostEndpointToFilterChains(
		ifaceName string,
		tiers []TierPolicyGroups,
		forwardTiers []TierPolicyGroups,
		epMarkMapper EndpointMarkMapper,
		profileIDs []string,
	) []*iptables.Chain
	HostEndpointToMangleEgressChains(
		ifaceName string,
		tiers []TierPolicyGroups,
		profileIDs []string,
	) []*iptables.Chain
	HostEndpointToRawEgressChain(
		ifaceName string,
		untrackedTiers []TierPolicyGroups,
	) *iptables.Chain
	HostEndpointToRawChains(
		ifaceName string,
		untrackedTiers []TierPolicyGroups,
	) []*iptables.Chain
	HostEndpointToMangleIngressChains(
		ifaceName string,
		preDNATTiers []TierPolicyGroups,
	) []*iptables.Chain

	PolicyToIptablesChains(policyID *proto.PolicyID, policy *proto.Policy, ipVersion uint8) []*iptables.Chain
	ProfileToIptablesChains(profileID *proto.ProfileID, policy *proto.Profile, ipVersion uint8) (inbound, outbound *iptables.Chain)
	ProtoRuleToIptablesRules(pRule *proto.Rule, ipVersion uint8, owner RuleOwnerType, dir RuleDir, idx int, name string, untracked, staged bool) []iptables.Rule
	PolicyGroupToIptablesChains(group *PolicyGroup) []*iptables.Chain

	MakeNatOutgoingRule(protocol string, action iptables.Action, ipVersion uint8) iptables.Rule
	NATOutgoingChain(active bool, ipVersion uint8) *iptables.Chain

	DNATsToIptablesChains(dnats map[string]string) []*iptables.Chain
	SNATsToIptablesChains(snats map[string]string) []*iptables.Chain
	BlockedCIDRsToIptablesChains(cidrs []string, ipVersion uint8) []*iptables.Chain

	RPFilter(ipVersion uint8, mark, mask uint32, openStackSpecialCasesEnabled, acceptLocal bool) []iptables.Rule

	WireguardIncomingMarkChain() *iptables.Chain

	IptablesFilterDenyAction() iptables.Action
}

type DefaultRuleRenderer struct {
	Config

	dropRules                    []iptables.Rule
	inputAcceptActions           []iptables.Action
	filterAllowAction            iptables.Action
	mangleAllowAction            iptables.Action
	blockCIDRAction              iptables.Action
	iptablesFilterDenyAction     iptables.Action
	nfqueueRuleDelayDeniedPacket *iptables.Rule
}

func (r *DefaultRuleRenderer) IptablesFilterDenyAction() iptables.Action {
	return r.iptablesFilterDenyAction
}

func (r *DefaultRuleRenderer) ipSetConfig(ipVersion uint8) *ipsets.IPVersionConfig {
	if ipVersion == 4 {
		return r.IPSetConfigV4
	} else if ipVersion == 6 {
		return r.IPSetConfigV6
	} else {
		log.WithField("version", ipVersion).Panic("Unknown IP version")
		return nil
	}
}

type Config struct {
	IPSetConfigV4 *ipsets.IPVersionConfig
	IPSetConfigV6 *ipsets.IPVersionConfig

	WorkloadIfacePrefixes []string

	DNSPolicyMode       apiv3.DNSPolicyMode
	DNSPolicyNfqueueID  int64
	DNSPacketsNfqueueID int64

	IptablesMarkAccept   uint32
	IptablesMarkPass     uint32
	IptablesMarkDrop     uint32
	IptablesMarkIPsec    uint32
	IptablesMarkEgress   uint32
	IptablesMarkScratch0 uint32
	IptablesMarkScratch1 uint32
	IptablesMarkEndpoint uint32
	// IptablesMarkNonCaliEndpoint is an endpoint mark which is reserved
	// to mark non-calico (workload or host) endpoint.
	IptablesMarkNonCaliEndpoint uint32

	// IptablesMarkProxy marks packets that are to/from proxy.
	IptablesMarkProxy uint32

	// IptablesMarkDNSPolicy marks packets that have been evaluated by a DNS policy rule.
	IptablesMarkDNSPolicy uint32

	// IptablesMarkSkipDNSPolicyNfqueue marks a packet that should not be added to nfqueue again.
	IptablesMarkSkipDNSPolicyNfqueue uint32

	KubeNodePortRanges     []numorstring.Port
	KubeIPVSSupportEnabled bool
	KubeMasqueradeMark     uint32
	KubernetesProvider     config.Provider

	OpenStackMetadataIP          net.IP
	OpenStackMetadataPort        uint16
	OpenStackSpecialCasesEnabled bool

	VXLANEnabled   bool
	VXLANEnabledV6 bool
	VXLANPort      int
	VXLANVNI       int

	IPIPEnabled            bool
	FelixConfigIPIPEnabled *bool
	// IPIPTunnelAddress is an address chosen from an IPAM pool, used as a source address
	// by the host when sending traffic to a workload over IPIP.
	IPIPTunnelAddress net.IP
	// Same for VXLAN.
	VXLANTunnelAddress   net.IP
	VXLANTunnelAddressV6 net.IP

	AllowVXLANPacketsFromWorkloads bool
	AllowIPIPPacketsFromWorkloads  bool

	WireguardEnabled            bool
	WireguardEnabledV6          bool
	WireguardInterfaceName      string
	WireguardInterfaceNameV6    string
	WireguardIptablesMark       uint32
	WireguardListeningPort      int
	WireguardListeningPortV6    int
	WireguardEncryptHostTraffic bool
	RouteSource                 string

	IptablesLogPrefix         string
	IncludeDropActionInPrefix bool
	EndpointToHostAction      string
	ActionOnDrop              string
	IptablesFilterAllowAction string
	IptablesMangleAllowAction string
	IptablesFilterDenyAction  string

	FailsafeInboundHostPorts  []config.ProtoPort
	FailsafeOutboundHostPorts []config.ProtoPort

	DisableConntrackInvalid bool

	NATPortRange                       numorstring.Port
	IptablesNATOutgoingInterfaceFilter string

	NATOutgoingAddress             net.IP
	BPFEnabled                     bool
	BPFForceTrackPacketsFromIfaces []string

	ServiceLoopPrevention string

	EnableNflogSize bool
	IPSecEnabled    bool

	EgressIPEnabled   bool
	EgressIPVXLANPort int
	EgressIPVXLANVNI  int
	EgressIPInterface string

	EKSPrimaryENI string

	DNSTrustedServers []config.ServerPort

	TPROXYMode             string
	TPROXYPort             int
	TPROXYUpstreamConnMark uint32
}

var unusedBitsInBPFMode = map[string]bool{
	"IptablesMarkPass":                 true,
	"IptablesMarkScratch1":             true,
	"IptablesMarkEndpoint":             true,
	"IptablesMarkNonCaliEndpoint":      true,
	"IptablesMarkDNSPolicy":            true,
	"IptablesMarkSkipDNSPolicyNfqueue": true,
}

func (c *Config) validate() {
	// Scan for unset iptables mark bits.  We use reflection so that we have a hope of catching
	// newly-added fields.
	myValue := reflect.ValueOf(c).Elem()
	myType := myValue.Type()
	found := 0
	usedBits := uint32(0)
	for i := 0; i < myValue.NumField(); i++ {
		fieldName := myType.Field(i).Name
		if fieldName == "IptablesMarkNonCaliEndpoint" ||
			fieldName == "IptablesMarkIPsec" ||
			fieldName == "IptablesMarkEgress" {
			// These mark bits are only used when needed (by IPVS, IPsec and Egress IP support, respectively) so we allow them to
			// be zero.
			continue
		}
		// Not set if Proxy is not enabled
		if fieldName == "IptablesMarkProxy" && !c.TPROXYModeEnabled() {
			continue
		}
		if strings.HasPrefix(fieldName, "IptablesMark") {
			if c.BPFEnabled && unusedBitsInBPFMode[fieldName] {
				log.WithField("field", fieldName).Debug("Ignoring unused field in BPF mode.")
				continue
			}
			bits := myValue.Field(i).Interface().(uint32)
			if bits == 0 {
				log.WithField("field", fieldName).Panic(
					"IptablesMarkXXX field not set.")
			}
			if usedBits&bits > 0 {
				log.WithField("field", fieldName).Panic(
					"IptablesMarkXXX field overlapped with another's bits.")
			}
			usedBits |= bits
			found++
		}
	}
	if found == 0 {
		// Check the reflection found something we were expecting.
		log.Panic("Didn't find any IptablesMarkXXX fields.")
	}

	if c.IptablesMarkDNSPolicy != 0x0 && c.DNSPolicyNfqueueID == 0 {
		log.WithField("field", "DNSPolicyNfqueueID").Panic("Must not be 0 when the IptablesMarkDNSPolicy is set.")
	}
}

func NewRenderer(config Config) RuleRenderer {
	log.WithField("config", config).Info("Creating rule renderer.")
	config.validate()

	// First, what should we do when packets are not accepted.
	var iptablesFilterDenyAction iptables.Action
	switch config.IptablesFilterDenyAction {
	case "REJECT":
		log.Info("packets that are not passed by any policy or profile will be rejected.")
		iptablesFilterDenyAction = iptables.RejectAction{}
	default:
		log.Info("packets that are not passed by any policy or profile will be dropped.")
		iptablesFilterDenyAction = iptables.DropAction{}
	}

	// Convert configured actions to rule slices.
	// First, what should we actually do when we'd normally drop a packet?  For
	// sandbox mode, we support allowing the packet instead, or logging it.
	var dropRules []iptables.Rule
	var nfqueueRuleDelayDeniedPacket *iptables.Rule
	if strings.HasPrefix(config.ActionOnDrop, "LOG") {
		log.Warnf("Action on drop includes LOG.  All dropped packets will be logged. %s", config.IptablesLogPrefix)
		logPrefix := "calico-drop"
		if config.IncludeDropActionInPrefix {
			logPrefix = logPrefix + " " + config.ActionOnDrop
		}

		dropRules = append(dropRules, iptables.Rule{
			Match:  iptables.Match(),
			Action: iptables.LogAction{Prefix: logPrefix},
		})
	}

	if strings.HasSuffix(config.ActionOnDrop, "ACCEPT") {
		log.Warn("Action on drop set to ACCEPT.  Calico security is disabled!")
		dropRules = append(dropRules, iptables.Rule{
			Match:  iptables.Match(),
			Action: iptables.AcceptAction{},
		})
	} else {
		if config.DNSPolicyMode == apiv3.DNSPolicyModeDelayDeniedPacket && config.IptablesMarkDNSPolicy != 0x0 {
			nfqueueRuleDelayDeniedPacket = &iptables.Rule{
				Match: iptables.Match().
					MarkSingleBitSet(config.IptablesMarkDNSPolicy).
					NotMarkMatchesWithMask(config.IptablesMarkSkipDNSPolicyNfqueue, config.IptablesMarkSkipDNSPolicyNfqueue),
				Action: iptables.NfqueueAction{QueueNum: config.DNSPolicyNfqueueID},
			}
		}
		dropRules = append(dropRules, iptables.Rule{
			Match:  iptables.Match(),
			Action: iptablesFilterDenyAction,
		})
	}

	// Second, what should we do with packets that come from workloads to the host itself.
	var inputAcceptActions []iptables.Action
	switch config.EndpointToHostAction {
	case "DROP":
		log.Info("Workload to host packets will be dropped.")
		for _, rule := range dropRules {
			inputAcceptActions = append(inputAcceptActions, rule.Action)
		}
	case "REJECT":
		log.Info("Workload to host packets will be rejected.")
		inputAcceptActions = []iptables.Action{iptables.RejectAction{}}
	case "ACCEPT":
		log.Info("Workload to host packets will be accepted.")
		inputAcceptActions = []iptables.Action{iptables.AcceptAction{}}
	default:
		log.Info("Workload to host packets will be returned to INPUT chain.")
		inputAcceptActions = []iptables.Action{iptables.ReturnAction{}}
	}

	// What should we do with packets that are accepted in the forwarding chain
	var filterAllowAction, mangleAllowAction iptables.Action
	switch config.IptablesFilterAllowAction {
	case "RETURN":
		log.Info("filter table allowed packets will be returned to FORWARD chain.")
		filterAllowAction = iptables.ReturnAction{}
	default:
		log.Info("filter table allowed packets will be accepted immediately.")
		filterAllowAction = iptables.AcceptAction{}
	}
	switch config.IptablesMangleAllowAction {
	case "RETURN":
		log.Info("mangle table allowed packets will be returned to PREROUTING chain.")
		mangleAllowAction = iptables.ReturnAction{}
	default:
		log.Info("mangle table allowed packets will be accepted immediately.")
		mangleAllowAction = iptables.AcceptAction{}
	}

	// How should we block CIDRs for loop prevention?
	var blockCIDRAction iptables.Action
	switch config.ServiceLoopPrevention {
	case "Drop":
		log.Info("Packets to unknown service IPs will be dropped")
		blockCIDRAction = iptables.DropAction{}
	case "Reject":
		log.Info("Packets to unknown service IPs will be rejected")
		blockCIDRAction = iptables.RejectAction{}
	default:
		log.Info("Packets to unknown service IPs will be allowed to loop")
	}

	return &DefaultRuleRenderer{
		Config:                       config,
		dropRules:                    dropRules,
		nfqueueRuleDelayDeniedPacket: nfqueueRuleDelayDeniedPacket,
		inputAcceptActions:           inputAcceptActions,
		filterAllowAction:            filterAllowAction,
		mangleAllowAction:            mangleAllowAction,
		blockCIDRAction:              blockCIDRAction,
		iptablesFilterDenyAction:     iptablesFilterDenyAction,
	}
}

func (c *Config) TPROXYModeEnabled() bool {
	return c.TPROXYMode == "Enabled" || c.TPROXYMode == "EnabledAllServices"
}
