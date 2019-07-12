// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"

	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

const (
	EndpointTypeHEP = "hep"
	EndpointTypeWEP = "wep"
	EndpointTypeNet = "net"
	EndpointTypeNS  = "ns"

	ActionAllow   = "allow"
	ActionDeny    = "deny"
	ActionUnknown = "unknown"
)

var (
	ipVersion4 = int(4)
	zeroIPv4   = net.ParseIP("0.0.0.0")
	zeroIPv6   = net.ParseIP("::")
)

type pipResponseHook struct {
	pip pip.PIP
}

func NewPIPResponseHook(p pip.PIP) ResponseHook {
	return &pipResponseHook{
		pip: p,
	}
}

// ModifyResponse alters the flows in the response by calling the
// CalculateFlowImpact method of the PIP object with the extracted flow data
func (rh *pipResponseHook) ModifyResponse(r *http.Response) error {

	// Extract the context from the request
	cxt := r.Request.Context()

	// Look for the policy impact request data in the context
	changes := cxt.Value(pip.PolicyImpactContextKey)

	// If there were no changes, no need to modify the response
	if changes == nil {
		return nil
	}

	log.Debug("Policy Impact ModifyResponse executing")

	// Assert that we have network policy changes
	res := changes.([]pip.ResourceChange)

	// Read the flows from the response body
	esResponseBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	// Unmarshal the data from the request
	var esResponse map[string]interface{}
	if err := json.Unmarshal(esResponseBytes, &esResponse); err != nil {
		return err
	}

	aggs, ok := esResponse["aggregations"].(map[string]interface{})
	if !ok {
		// If there were no flows, the aggregations will be empty, so this is a valid condition - just exit.
		return nil
	}

	flogBuckets, ok := aggs["flog_buckets"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to unpack flow buckets")
	}

	flows, ok := flogBuckets["buckets"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to extract a flow for bucket: %v", flogBuckets)
	}

	// Calculate the flow impact
	pc, err := rh.pip.GetPolicyCalculator(cxt, res)
	if err != nil {
		return err
	}

	for _, f := range flows {
		flowData, ok := f.(map[string]interface{})
		if !ok {
			log.Error("invalid input")
			continue
		}
		esKey, ok := flowData["key"].(map[string]interface{})
		if !ok {
			log.Error("invalid input")
			continue
		}

		pipFlow := flowFromEs(flowData, esKey)

		// Calculate the flow impact
		processed, before, after := pc.Action(&pipFlow)
		log.WithFields(log.Fields{
			"processed": processed,
			"before":    before,
			"after":     after,
		}).Debug("Processed flow")

		// Set before/after action fields.
		setActionField(esKey, "action", before)
		setActionField(esKey, "preview_action", after)
	}

	// Put the returned flows back into the response body and remarshal
	newBodyContent, err := json.Marshal(esResponse)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(newBodyContent))

	// Fix the content length as it might have changed
	r.ContentLength = int64(len(newBodyContent))

	return nil
}

func flowFromEs(flowData map[string]interface{}, esKey map[string]interface{}) policycalc.Flow {
	sourceIP := getIP(esKey, "source_ip")
	destIP := getIP(esKey, "dest_ip")

	// Assume IP version 4 unless we have IP addresses available. Also, if the IPs are zeros - set to nil.
	ipVersion := ipVersion4
	if sourceIP != nil {
		ipVersion = sourceIP.Version()
	} else if destIP != nil {
		ipVersion = destIP.Version()
	}
	if sourceIP == zeroIPv4 || sourceIP == zeroIPv6 {
		sourceIP = nil
	}
	if destIP == zeroIPv4 || destIP == zeroIPv6 {
		destIP = nil
	}

	return policycalc.Flow{
		Source: policycalc.FlowEndpointData{
			Type:      getEndpointType(esKey, "source_type"),
			Namespace: getStringField(esKey, "source_namespace"),
			Name:      getStringField(esKey, "source_name"),
			Port:      getPortField(esKey, "source_port"),
			Labels:    parseLabels(flowData["source_labels"]),
			IP:        sourceIP,
		},
		Destination: policycalc.FlowEndpointData{
			Type:      getEndpointType(esKey, "dest_type"),
			Namespace: getStringField(esKey, "dest_namespace"),
			Name:      getStringField(esKey, "dest_name"),
			Port:      getPortField(esKey, "dest_port"),
			Labels:    parseLabels(flowData["dest_labels"]),
			IP:        destIP,
		},
		Proto:     getProtoField(esKey, "proto"),
		Action:    getActionField(esKey, "action"),
		IPVersion: &ipVersion,
	}
}

// parseLabels converts endpoint labels stored in unknown typed esflow data into
// a more usable map[string]string type, e.g.
//
// {"by_kvpair": {"buckets": [{"key": "foo=bar"}, {"key":"boo=baz"}]}}
//
// becomes a map with values
//
// {"foo": "bar","boo": "baz"}
func parseLabels(labelData interface{}) map[string]string {
	var labels = map[string]string{}
	if labelData == nil {
		return labels
	}

	byKVPair, ok := labelData.(map[string]interface{})["by_kvpair"]
	if !ok {
		return labels
	}

	buckets, ok := byKVPair.(map[string]interface{})["buckets"]
	if !ok {
		return labels
	}

	bucketList, ok := buckets.([]interface{})
	if !ok {
		return labels
	}

	for _, bucket := range bucketList {
		rawKvPair, ok := bucket.(map[string]interface{})
		if !ok {
			log.Warn("skipping bad esflow data")
			continue
		}
		kvPairStr, ok := rawKvPair["key"].(string)
		if !ok {
			log.Warn("skipping bad esflow data")
			continue
		}

		// labels are stored in this field in elasticsearch as 'key=value',
		// so split them by '=' to convert to a map.
		keyVals := strings.Split(kvPairStr, "=")
		if len(keyVals) != 2 {
			log.Warn("skipping bad esflow label data")
			continue
		}
		labels[keyVals[0]] = keyVals[1]
	}

	return labels
}

// getStringField is a helper function for getting the value
// for a string or returning an empty string if it doesn't exist,
// effectively eliminating any chance of panic when reading the fields.
func getStringField(esKey map[string]interface{}, field string) string {
	s, _ := esKey[field].(string)
	return s
}

// getNamespaceField is a helper function for getting the value
// for a namespace field. It handles the fact that the ES data contains a "-" for no namespace.
func getNamespaceField(esKey map[string]interface{}, field string) string {
	s := getStringField(esKey, field)
	if s == "-" {
		return ""
	}
	return s
}

// getPortField extracts the port field and converts to a uint8 pointer.
func getPortField(esKey map[string]interface{}, field string) *uint16 {
	port, ok := esKey[field]
	if !ok {
		return nil
	}

	var portNum uint16
	switch v := port.(type) {
	case string:
		if v == "" {
			return nil
		}
		num, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return nil
		}
		portNum = uint16(num)
	case float64:
		portNum = uint16(v)
	}

	if portNum == 0 {
		return nil
	}
	return &portNum
}

// getProtoField extracts the protocol string field and converts to a uint8 pointer.
func getProtoField(esKey map[string]interface{}, field string) *uint8 {
	p, ok := esKey[field].(string)
	if ok {
		proto := numorstring.ProtocolFromString(p)
		return policycalc.GetProtocolNumber(&proto)
	}
	return nil
}

// getEndpointType extracts the endpoint type string and converts to the policycalc equivalent.
func getEndpointType(esKey map[string]interface{}, field string) policycalc.EndpointType {
	ept := getStringField(esKey, field)
	switch ept {
	case EndpointTypeHEP:
		return policycalc.EndpointTypeHep
	case EndpointTypeWEP:
		return policycalc.EndpointTypeWep
	case EndpointTypeNet:
		return policycalc.EndpointTypeNet
	case EndpointTypeNS:
		return policycalc.EndpointTypeNs
	default:
		return policycalc.EndpointTypeUnknown
	}
}

// getActionField extracts the action string and converts to the policycalc equivalent.
func getActionField(esKey map[string]interface{}, field string) policycalc.Action {
	a := getStringField(esKey, field)
	switch a {
	case ActionAllow:
		return policycalc.ActionAllow
	case ActionDeny:
		return policycalc.ActionDeny
	default:
		return policycalc.ActionUnknown
	}
}

// setActionField sets the action field (in flow log format) from the policycalc value.
func setActionField(esKey map[string]interface{}, field string, a policycalc.Action) {
	switch a {
	case policycalc.ActionAllow:
		esKey[field] = ActionAllow
	case policycalc.ActionDeny:
		esKey[field] = ActionDeny
	case policycalc.ActionIndeterminate:
		esKey[field] = ActionUnknown
	}
}

// getIP extracts the IP string and converts to a net.IP value required by the policycalc.
func getIP(esKey map[string]interface{}, field string) *net.IP {
	if ipStr := getStringField(esKey, field); ipStr != "" {
		return net.ParseIP(getStringField(esKey, field))
	}
	return nil
}
