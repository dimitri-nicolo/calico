// Copyright (c) 2017-2024 Tigera, Inc. All rights reserved.

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

package updateprocessors

import (
	"errors"
	"fmt"
	"strings"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/watchersyncer"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/names"
)

// Create a new SyncerUpdateProcessor to sync NetworkPolicy data in v1 format for
// consumption by Felix.
func NewNetworkPolicyUpdateProcessor() watchersyncer.SyncerUpdateProcessor {
	return NewSimpleUpdateProcessor(apiv3.KindNetworkPolicy, ConvertNetworkPolicyV3ToV1Key, ConvertNetworkPolicyV3ToV1Value)
}

func ConvertNetworkPolicyV3ToV1Key(v3key model.ResourceKey) (model.Key, error) {
	if v3key.Name == "" || v3key.Namespace == "" {
		return model.PolicyKey{}, errors.New("Missing Name or Namespace field to create a v1 NetworkPolicy Key")
	}
	tier, err := names.TierFromPolicyName(v3key.Name)
	if err != nil {
		return model.PolicyKey{}, err
	}
	return model.PolicyKey{
		Name: v3key.Namespace + "/" + v3key.Name,
		Tier: tier,
	}, nil
}

func ConvertNetworkPolicyV3ToV1Value(val interface{}) (interface{}, error) {
	v3res, ok := val.(*apiv3.NetworkPolicy)
	if !ok {
		return nil, errors.New("Value is not a valid NetworkPolicy resource value")
	}
	log := logrus.WithFields(logrus.Fields{"name": v3res.Name, "namespace": v3res.Namespace})

	// If this policy is namespaced, then add a namespace selector.
	spec := v3res.Spec
	selector := spec.Selector

	if v3res.Namespace != "" {
		nsSelector := fmt.Sprintf("%s == '%s'", apiv3.LabelNamespace, v3res.Namespace)
		if selector == "" {
			selector = nsSelector
		} else {
			selector = fmt.Sprintf("(%s) && %s", selector, nsSelector)
		}
	}

	// Determine if this NP is configured to match security groups or not.
	m, ok := v3res.Annotations["rules.networkpolicy.tigera.io/match-security-groups"]
	if ok {
		// The annotation is specified. Do some basic validation of the value and log a warning
		// if it's something weird.
		switch m {
		case "true", "false", "":
			// These are all normal / expected values.
		default:
			// The value is set but to something that isn't supported.
			log.Warnf("Unsupported value provided for match-security-groups annotation: %s", m)
		}
	}
	matchSGs := m == "true"
	selector = prefixAndAppendSelector(selector, spec.ServiceAccountSelector, conversion.ServiceAccountLabelPrefix)

	v1value := &model.Policy{
		Namespace:        v3res.Namespace,
		Order:            spec.Order,
		InboundRules:     RulesAPIV2ToBackend(spec.Ingress, v3res.Namespace, matchSGs),
		OutboundRules:    RulesAPIV2ToBackend(spec.Egress, v3res.Namespace, matchSGs),
		Selector:         selector,
		Types:            policyTypesAPIV2ToBackend(spec.Types),
		ApplyOnForward:   false,
		PerformanceHints: v3res.Spec.PerformanceHints,
	}

	return v1value, nil
}

// policyTypesAPIV2ToBackend converts the policy type field value from the API
// value to the equivalent backend value.
func policyTypesAPIV2ToBackend(ptypes []apiv3.PolicyType) []string {
	var v1ptypes []string
	for _, ptype := range ptypes {
		v1ptypes = append(v1ptypes, policyTypeAPIV2ToBackend(ptype))
	}
	return v1ptypes
}

func policyTypeAPIV2ToBackend(ptype apiv3.PolicyType) string {
	return strings.ToLower(string(ptype))
}
