// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/rules"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

const (
	flowLogNamespaceGlobal                  = "-"
	flowLogFieldNotIncluded                 = "-"
	flowLogFieldNotIncludedForNumericFields = 0
	flowLogFieldAggregated                  = "*"

	FlowLogActionAllow FlowLogAction = "allow"
	FlowLogActionDeny  FlowLogAction = "deny"

	FlowLogReporterSrc FlowLogReporter = "src"
	FlowLogReporterDst FlowLogReporter = "dst"

	FlowLogEndpointTypeWep FlowLogEndpointType = "wep"
	FlowLogEndpointTypeHep FlowLogEndpointType = "hep"
	FlowLogEndpointTypeNs  FlowLogEndpointType = "ns"
	FlowLogEndpointTypeNet FlowLogEndpointType = "net"

	PrivateNet FlowLogSubnetType = "pvt"
	PublicNet  FlowLogSubnetType = "pub"
)

// extractPartsFromAggregatedTuple converts and returns each field of a tuple to a string.
// If a field is missing a "-" is used in it's place. This can happen if:
// - This field has been aggregated over.
// - This is a ICMP flow in which case it is a "3-tuple" where only source ip,
//   destination IP and protocol makes sense.
func extractPartsFromAggregatedTuple(t Tuple) (srcIP, dstIP, proto, l4Src, l4Dst string) {
	// Try to extract source and destination IP address.
	// This field is aggregated over when using FlowPrefixName aggregation.
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
		// Check if FlowSourcePort has been aggregated over.
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

func getFlowLogEndpointMetadata(ed *calc.EndpointData, ip [16]byte) (EndpointMetadata, error) {
	var em EndpointMetadata
	if ed == nil {
		return EndpointMetadata{
			Type:           FlowLogEndpointTypeNet,
			Namespace:      flowLogFieldNotIncluded,
			Name:           flowLogFieldNotIncluded,
			AggregatedName: string(getSubnetType(ip)),
		}, nil
	}

	switch k := ed.Key.(type) {
	case model.WorkloadEndpointKey:
		ns, name, err := deconstructNamespaceAndNameFromWepName(k.WorkloadID)
		if err != nil {
			return EndpointMetadata{}, err
		}
		v := ed.Endpoint.(*model.WorkloadEndpoint)
		var aggName string
		if v.GenerateName != "" {
			aggName = fmt.Sprintf("%s*", v.GenerateName)
		} else {
			aggName = name
		}
		em = EndpointMetadata{
			Type:           FlowLogEndpointTypeWep,
			Name:           name,
			AggregatedName: aggName,
			Namespace:      ns,
		}
	case model.HostEndpointKey:
		em = EndpointMetadata{
			Type:           FlowLogEndpointTypeHep,
			Name:           k.EndpointID,
			AggregatedName: k.Hostname,
			Namespace:      flowLogNamespaceGlobal,
		}
	case model.NetworkSetKey:
		namespace, name := extractNamespaceFromNetworkSet(k.Name)
		// No Endpoint was found so instead, a NetworkSet was returned.
		em = EndpointMetadata{
			Type:           FlowLogEndpointTypeNs,
			Namespace:      namespace,
			AggregatedName: name,
			Name:           name,
		}
	default:
		return EndpointMetadata{}, fmt.Errorf("Unknown key %#v of type %v", ed.Key, reflect.TypeOf(ed.Key))
	}

	return em, nil
}

func getFlowLogService(svc MetricServiceInfo) FlowService {
	if svc.Name == "" {
		return FlowService{
			Namespace: flowLogFieldNotIncluded,
			Name:      flowLogFieldNotIncluded,
			PortName:  flowLogFieldNotIncluded,
			PortNum:   flowLogFieldNotIncludedForNumericFields,
		}
	} else if svc.Port == "" { // proxy.ServicePortName.Port refers to the PortName
		// A single port for a service may not have a name.
		return FlowService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
			PortName:  flowLogFieldNotIncluded,
			PortNum:   svc.PortNum,
		}
	}
	return FlowService{
		Namespace: svc.Namespace,
		Name:      svc.Name,
		PortName:  svc.Port,
		PortNum:   svc.PortNum,
	}
}

func getFlowLogEndpointLabels(ed *calc.EndpointData) map[string]string {
	labels := map[string]string{}
	if ed == nil {
		return labels
	}

	var v map[string]string
	switch ed.Key.(type) {
	case model.WorkloadEndpointKey:
		v = ed.Endpoint.(*model.WorkloadEndpoint).Labels
	case model.HostEndpointKey:
		v = ed.Endpoint.(*model.HostEndpoint).Labels
	case model.NetworkSetKey:
		v = ed.Networkset.(*model.NetworkSet).Labels
	}

	if v != nil {
		labels = v
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
	return ipTo16Byte(addr)
}

func ipTo16Byte(addr net.IP) [16]byte {
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

	return PublicNet
}

func flattenLabels(labels map[string]string) []string {
	respSlice := []string{}
	for k, v := range labels {
		l := fmt.Sprintf("%v=%v", k, v)
		respSlice = append(respSlice, l)
	}

	return respSlice
}

func unflattenLabels(labelSlice []string) map[string]string {
	resp := map[string]string{}
	for _, label := range labelSlice {
		labelKV := strings.Split(label, "=")
		if len(labelKV) != 2 {
			continue
		}
		resp[labelKV[0]] = labelKV[1]
	}

	return resp
}

func labelsToString(labels map[string]string) string {
	if labels == nil {
		return "-"
	}

	return fmt.Sprintf("[%v]", strings.Join(flattenLabels(labels), ","))
}

func stringToLabels(labelStr string) map[string]string {
	if labelStr == "-" {
		return nil
	}

	labels := strings.Split(labelStr[1:len(labelStr)-1], ",")
	return unflattenLabels(labels)
}

func flattenIPSlice(ips []net.IP) []string {
	ipSlice := []string{}
	for _, ip := range ips {
		ipSlice = append(ipSlice, ip.String())
	}

	return ipSlice
}

func unflattenIPSlice(ipSlice []string) []net.IP {
	ips := []net.IP{}
	for _, ipStr := range ipSlice {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		ips = append(ips, ip)
	}

	return ips
}

func flowExtrasToString(extras FlowExtras) string {
	if len(extras.OriginalSourceIPs) == 0 {
		return "- 0"
	}
	s := fmt.Sprintf("[%v] %v", strings.Join(flattenIPSlice(extras.OriginalSourceIPs), ","), extras.NumOriginalSourceIPs)
	return s
}

func stringToFlowExtras(origIPsStr string, numOrigIPStr string) FlowExtras {
	if origIPsStr == "-" {
		return FlowExtras{}
	}

	ips := strings.Split(origIPsStr[1:len(origIPsStr)-1], ",")
	var (
		numOrigIP int
		err       error
	)
	numOrigIP, err = strconv.Atoi(numOrigIPStr)
	if err != nil {
		log.WithError(err).Warn("Could not convert NumOriginalSourceIPs to integer")
		numOrigIP = 0
	}
	return FlowExtras{
		OriginalSourceIPs:    unflattenIPSlice(ips),
		NumOriginalSourceIPs: numOrigIP,
	}
}

// There is support for both global and namespaced networkset. In case of
// namespaced networkset, aggregatedName is namespace/name format. Extract
// namespace and name from it.
func extractNamespaceFromNetworkSet(aggregatedName string) (string, string) {
	res := strings.Split(aggregatedName, "/")
	if (len(res)) > 1 {
		return res[0], res[1]
	}

	return flowLogFieldNotIncluded, aggregatedName
}
