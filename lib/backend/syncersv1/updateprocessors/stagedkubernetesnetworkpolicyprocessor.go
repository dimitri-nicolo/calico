// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"

	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/names"
)

// NewStagedKubernetesNetworkPolicyUpdateProcessor create a new SyncerUpdateProcessor to sync StagedKubernetesNetworkPolicy data in v1 format for
// consumption by Felix.
func NewStagedKubernetesNetworkPolicyUpdateProcessor() watchersyncer.SyncerUpdateProcessor {
	return NewSimpleUpdateProcessor(apiv3.KindStagedKubernetesNetworkPolicy, ConvertStagedKubernetesNetworkPolicyV3ToV1Key, ConvertStagedKubernetesNetworkPolicyV3ToV1Value)
}

func ConvertStagedKubernetesNetworkPolicyV3ToV1Key(v3key model.ResourceKey) (model.Key, error) {
	if v3key.Name == "" || v3key.Namespace == "" {
		return model.PolicyKey{}, errors.New("Missing Name or Namespace field to create a v1 StagedKubernetesNetworkPolicy Key")
	}

	c := conversion.Converter{}
	name := c.StagedKubernetesNetworkPolicyToStagedName(v3key.Name)

	tier, err := names.TierFromPolicyName(name)
	if err != nil {
		return model.PolicyKey{}, err
	}
	return model.PolicyKey{
		Name: model.PolicyNamePrefixStaged + v3key.Namespace + "/" + name,
		Tier: tier,
	}, nil
}

func ConvertStagedKubernetesNetworkPolicyV3ToV1Value(val interface{}) (interface{}, error) {
	staged, ok := val.(*apiv3.StagedKubernetesNetworkPolicy)
	if !ok {
		return nil, errors.New("Value is not a valid StagedKubernetesNetworkPolicy resource value")
	}

	//If the update type is delete return nil so the resource is not sent to felix.
	if staged.Spec.StagedAction == apiv3.StagedActionDelete {
		return nil, nil
	}

	c := conversion.Converter{}
	snpKVPair, err := c.StagedKubernetesNetworkPolicyToStaged(staged)
	if err != nil {
		return nil, err
	}

	v3snp := snpKVPair.Value.(*apiv3.StagedNetworkPolicy)
	return ConvertStagedNetworkPolicyV3ToV1Value(v3snp)
}
