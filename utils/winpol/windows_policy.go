// Copyright (c) 2018 Tigera, Inc. All rights reserved.

// This package contains algorithmic support code for Windows.  I.e. code that is used on
// Windows but can be UTed on any platform.
package winpol

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
)

type PolicyMarshaller interface {
	MarshalPolicies() []json.RawMessage
}

// CalculateEndpointPolicies augments the hns.Netconf policies with NAT exceptions for our IPAM blocks.
func CalculateEndpointPolicies(
	n PolicyMarshaller,
	allIPAMPools []*net.IPNet,
	natOutgoing bool,
	logger *logrus.Entry,
) ([]json.RawMessage, error) {
	inputPols := n.MarshalPolicies()
	var outputPols []json.RawMessage
	found := false
	for _, inPol := range inputPols {
		// Decode the raw policy as a dict so we can inspect it without losing any fields.
		decoded := map[string]interface{}{}
		err := json.Unmarshal(inPol, &decoded)
		if err != nil {
			logger.WithError(err).Error("MarshalPolicies() returned bad JSON")
			return nil, err
		}

		// We're looking for an entry like this:
		//
		// {
		//   "Type":  "OutBoundNAT",
		//   "ExceptionList":  [
		//     "10.96.0.0/12"
		//   ]
		// }
		// We'll add the other IPAM pools to the list.
		outPol := inPol
		if strings.EqualFold(decoded["Type"].(string), "OutBoundNAT") {
			found = true
			if !natOutgoing {
				logger.Info("NAT-outgoing disabled for this IP pool, ignoring OutBoundNAT policy from NetConf.")
				continue
			}

			excList, _ := decoded["ExceptionList"].([]interface{})
			for _, poolCIDR := range allIPAMPools {
				excList = append(excList, poolCIDR.String())
			}
			decoded["ExceptionList"] = excList
			outPol, err = json.Marshal(decoded)
			if err != nil {
				logger.WithError(err).Error("Failed to add outbound NAT exclusion.")
				return nil, err
			}
			logger.WithField("policy", string(outPol)).Debug(
				"Updated OutBoundNAT policy to add Calico IP pools.")
		}
		outputPols = append(outputPols, outPol)
	}
	if !found {
		var exceptions []string
		for _, poolCIDR := range allIPAMPools {
			exceptions = append(exceptions, poolCIDR.String())
		}
		dict := map[string]interface{}{
			"Type":          "OutBoundNAT",
			"ExceptionList": exceptions,
		}
		encoded, err := json.Marshal(dict)
		if err != nil {
			logger.WithError(err).Error("Failed to add outbound NAT exclusion.")
			return nil, err
		}

		outputPols = append(outputPols, json.RawMessage(encoded))
	}
	return outputPols, nil
}
