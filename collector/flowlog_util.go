// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

const (
	flowLogBufferSize = 1000

	flowLogNamespaceGlobal  = "-"
	flowLogFieldNotIncluded = "-"

	FlowLogActionAllow FlowLogAction = "allow"
	FlowLogActionDeny  FlowLogAction = "deny"

	FlowLogReporterSrc FlowLogReporter = "src"
	FlowLogReporterDst FlowLogReporter = "dst"

	FlowLogEndpointTypeWep FlowLogEndpointType = "wep"
	FlowLogEndpointTypeHep FlowLogEndpointType = "hep"
	FlowLogEndpointTypeNs  FlowLogEndpointType = "ns"
	FlowLogEndpointTypeNet FlowLogEndpointType = "net"

	PrivateNet       FlowLogSubnetType = "pvt"
	AWSMetaServerNet FlowLogSubnetType = "aws"
	PublicNet        FlowLogSubnetType = "pub"
)

// extractPartsFromAggregatedTuple converts and returns each field of a tuple to a string.
// If a field is missing a "-" is used in it's place. This can happen if:
// - This field has been aggregated over.
// - This is a ICMP flow in which case it is a "3-tuple" where only source ip,
//   destination IP and protocol makes sense.
func extractPartsFromAggregatedTuple(t Tuple) (srcIP, dstIP, proto, l4Src, l4Dst string) {
	// Try to extract source and destination IP address.
	// This field is aggregated over when using PrefixName aggregation.
	sip := net.IP(t.src[:16])
	if sip.IsUnspecified() {
		srcIP = flowLogFieldNotIncluded
	} else {
		srcIP = sip.String()
	}
	dip := net.IP(t.dst[:16])
	if dip.IsUnspecified() {
		dstIP = flowLogFieldNotIncluded
	} else {
		dstIP = dip.String()
	}

	proto = fmt.Sprintf("%d", t.proto)

	if t.proto != 1 {
		// Check if SourcePort has been aggregated over.
		if t.l4Src == unsetIntField {
			l4Src = flowLogFieldNotIncluded
		} else {
			l4Src = fmt.Sprintf("%d", t.l4Src)
		}
		l4Dst = fmt.Sprintf("%d", t.l4Dst)
	} else {
		// ICMP has no l4 fields.
		l4Src = flowLogFieldNotIncluded
		l4Dst = flowLogFieldNotIncluded
	}
	return
}

func deconstructNamespaceAndNameFromWepName(wepName string) (string, string, error) {
	parts := strings.Split(wepName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("Could not parse name %v", wepName)
}

func getEndpointNamePrefix(ed *calc.EndpointData) (prefix string) {
	switch ed.Key.(type) {
	case model.WorkloadEndpointKey:
		v := ed.Endpoint.(*model.WorkloadEndpoint)
		prefix = v.GenerateName
	case model.HostEndpointKey:
		v := ed.Endpoint.(*model.HostEndpoint)
		prefix = v.Name
	}
	return
}

func getFlowLogEndpointMetadata(ed *calc.EndpointData) (EndpointMetadata, error) {
	var em EndpointMetadata

	switch k := ed.Key.(type) {
	case model.WorkloadEndpointKey:
		ns, name, err := deconstructNamespaceAndNameFromWepName(k.WorkloadID)
		if err != nil {
			return EndpointMetadata{}, err
		}
		em = EndpointMetadata{
			Type:      FlowLogEndpointTypeWep,
			Name:      name,
			Namespace: ns,
		}
	case model.HostEndpointKey:
		em = EndpointMetadata{
			Type:      FlowLogEndpointTypeHep,
			Name:      k.EndpointID,
			Namespace: flowLogNamespaceGlobal,
		}
	case model.NetworkSetKey:
		// No Endpoint was found so instead, a NetworkSet was returned.
		v := ed.Networkset.(*model.NetworkSet)
		labels, err := json.Marshal(v.Labels)
		if err != nil {
			return EndpointMetadata{}, err
		}
		em = EndpointMetadata{
			Type:      FlowLogEndpointTypeNs,
			Namespace: flowLogFieldNotIncluded,
			Name:      k.Name,
			Labels:    string(labels),
		}
	default:
		return EndpointMetadata{}, fmt.Errorf("Unknown key %#v of type %v", ed.Key, reflect.TypeOf(ed.Key))
	}

	return em, nil
}

func getFlowLogEndpointLabels(ed *calc.EndpointData) map[string]string {
	labels := map[string]string{}
	if ed == nil {
		return labels
	}
	switch ed.Key.(type) {
	case model.WorkloadEndpointKey:
		labels = ed.Endpoint.(*model.WorkloadEndpoint).Labels
	case model.HostEndpointKey:
		labels = ed.Endpoint.(*model.HostEndpoint).Labels
	}

	return labels
}

// getFlowLogActionAndReporterFromRuleID converts the action to a string value.
func getFlowLogActionAndReporterFromRuleID(r *calc.RuleID) (fla FlowLogAction, flr FlowLogReporter) {
	switch r.Action {
	case rules.RuleActionDeny:
		fla = FlowLogActionDeny
	case rules.RuleActionAllow:
		fla = FlowLogActionAllow
	}
	switch r.Direction {
	case rules.RuleDirIngress:
		flr = FlowLogReporterDst
	case rules.RuleDirEgress:
		flr = FlowLogReporterSrc
	}
	return
}

func ipStrTo16Byte(ipStr string) [16]byte {
	addr := net.ParseIP(ipStr)
	var addrB [16]byte
	copy(addrB[:], addr.To16()[:16])
	return addrB
}

func getSubnetType(addrBytes [16]byte) FlowLogSubnetType {
	IP := net.IP(addrBytes[:16])

	// Currently checking for only private blocks
	_, private24BitBlock, _ := net.ParseCIDR("10.0.0.0/8")
	_, private20BitBlock, _ := net.ParseCIDR("172.16.0.0/12")
	_, private16BitBlock, _ := net.ParseCIDR("192.168.0.0/16")
	isPrivateIP := private24BitBlock.Contains(IP) || private20BitBlock.Contains(IP) || private16BitBlock.Contains(IP)
	if isPrivateIP {
		return PrivateNet
	}

	// Based on https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
	_, awsMetaBlock, _ := net.ParseCIDR("169.254.169.254/32")
	isAWSMeta := awsMetaBlock.Contains(IP)
	if isAWSMeta {
		return AWSMetaServerNet
	}

	return PublicNet
}

func labelsToString(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	respStr, _ := json.Marshal(labels)
	return string(respStr)
}
