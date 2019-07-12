// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/net"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
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

	//extract the context from the request
	cxt := r.Request.Context()

	//look for the policy impact request data in the context
	changes := cxt.Value(pip.PolicyImpactContextKey)

	//if there were no changes, no need to modify the response
	if changes == nil {
		return nil
	}

	log.Debug("Policy Impact ModifyResponse executing")

	//assert that we have network policy changes
	res := changes.([]pip.ResourceChange)

	//read the flows from the response body
	esResponseBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to initialize pip: %v", err)
	}

	// unmarshal the data from the request
	var esResponse map[string]interface{}
	if err := json.Unmarshal(esResponseBytes, &esResponse); err != nil {
		return err
	}

	//calculate the flow impact
	pc, err := rh.pip.GetPolicyCalculator(cxt, res)
	if err != nil {
		return err
	}

	aggs, ok := esResponse["aggregations"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to unpack flow buckets")
	}

	flogBuckets, ok := aggs["flog_buckets"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to unpack flow buckets")
	}

	flows, ok := flogBuckets["buckets"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to extract a flow for bucket: %v", flogBuckets)
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

		//calculate the flow impact
		processed, before, after := pc.Action(&pipFlow)
		log.WithFields(log.Fields{
			"processed": processed,
			"before":    before,
			"after":     after,
		}).Debug("Processed flow")

		// esKey["action"] = string(before)
		esKey["preview_action"] = string(after)
	}

	//put the returned flows back into the response body and remarshal
	newBodyContent, err := json.Marshal(esResponse)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(newBodyContent))

	// fix the content length as it might have changed
	r.ContentLength = int64(len(newBodyContent))

	return nil
}

func flowFromEs(flowData map[string]interface{}, esKey map[string]interface{}) policycalc.Flow {
	return policycalc.Flow{
		Source: policycalc.FlowEndpointData{
			Type:      getEndpointType(esKey, "source_type"),
			Namespace: getStringField(esKey, "source_namespace"),
			Name:      getStringField(esKey, "source_name"),
			Port:      getPortField(esKey, "source_port"),
			Labels:    parseLabels(flowData["source_labels"]),
			IP:        parseIP(esKey, "source_ip"),
		},
		Destination: policycalc.FlowEndpointData{
			Type:      getEndpointType(esKey, "dest_type"),
			Namespace: getStringField(esKey, "dest_namespace"),
			Name:      getStringField(esKey, "dest_name"),
			Port:      getPortField(esKey, "dest_port"),
			Labels:    parseLabels(flowData["dest_labels"]),
			IP:        parseIP(esKey, "dest_ip"),
		},
		Proto:  getProtoField(esKey, "proto"),
		Action: getActionField(esKey, "action"),
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

func getPortField(esKey map[string]interface{}, field string) *uint16 {
	if port, ok := esKey[field].(uint16); ok {
		return &port
	}
	return nil
}

func getProtoField(esKey map[string]interface{}, field string) *uint8 {
	proto, ok := esKey[field].(uint8)
	if ok {
		return &proto
	}
	return nil
}

func getEndpointType(esKey map[string]interface{}, field string) policycalc.EndpointType {
	ept := getStringField(esKey, field)
	switch ept {
	case "hep":
		return policycalc.EndpointTypeHep
	case "wep":
		return policycalc.EndpointTypeWep
	case "net":
		return policycalc.EndpointTypeNet
	case "ns":
		return policycalc.EndpointTypeNs
	default:
		return policycalc.EndpointType("")
	}
}

func getActionField(esKey map[string]interface{}, field string) policycalc.Action {
	a := getStringField(esKey, field)
	switch a {
	case "allow":
		return policycalc.ActionAllow
	case "deny":
		return policycalc.ActionDeny
	default:
		return policycalc.Action("")
	}
}

func parseIP(esKey map[string]interface{}, field string) *net.IP {
	return net.ParseIP(getStringField(esKey, field))
}
