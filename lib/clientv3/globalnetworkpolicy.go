// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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

package clientv3

import (
	"context"
	"strings"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/names"
	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
	log "github.com/sirupsen/logrus"
)

// GlobalNetworkPolicyInterface has methods to work with GlobalNetworkPolicy resources.
type GlobalNetworkPolicyInterface interface {
	Create(ctx context.Context, res *apiv3.GlobalNetworkPolicy, opts options.SetOptions) (*apiv3.GlobalNetworkPolicy, error)
	Update(ctx context.Context, res *apiv3.GlobalNetworkPolicy, opts options.SetOptions) (*apiv3.GlobalNetworkPolicy, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalNetworkPolicy, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalNetworkPolicy, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalNetworkPolicyList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// globalNetworkPolicies implements GlobalNetworkPolicyInterface
type globalNetworkPolicies struct {
	client client
}

// Create takes the representation of a GlobalNetworkPolicy and creates it.  Returns the stored
// representation of the GlobalNetworkPolicy, and an error, if there is any.
func (r globalNetworkPolicies) Create(ctx context.Context, res *apiv3.GlobalNetworkPolicy, opts options.SetOptions) (*apiv3.GlobalNetworkPolicy, error) {
	// Before creating the policy, check that the tier exists.
	tier := names.TierOrDefault(res.Spec.Tier)
	if _, err := r.client.resources.Get(ctx, options.GetOptions{}, apiv3.KindTier, noNamespace, tier); err != nil {
		log.WithError(err).Infof("Tier %v does not exist", tier)
		return nil, err
	}
	defaultPolicyTypesField(res.Spec.Ingress, res.Spec.Egress, &res.Spec.Types)

	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	// Properly prefix the name
	backendPolicyName, err := names.BackendTieredPolicyName(res.GetObjectMeta().GetName(), res.Spec.Tier)
	if err != nil {
		return nil, err
	}
	res.GetObjectMeta().SetName(backendPolicyName)
	out, err := r.client.resources.Create(ctx, opts, apiv3.KindGlobalNetworkPolicy, res)
	if out != nil {
		retName, retErr := names.ClientTieredPolicyName(out.GetObjectMeta().GetName())
		if retErr != nil {
			return nil, retErr
		}
		// Remove the prefix out of the returned policy name.
		out.GetObjectMeta().SetName(retName)
		return out.(*apiv3.GlobalNetworkPolicy), err
	}

	// Remove the prefix out of the returned policy name.
	retName, retErr := names.ClientTieredPolicyName(res.GetObjectMeta().GetName())
	if retErr != nil {
		return nil, retErr
	}
	res.GetObjectMeta().SetName(retName)
	return nil, err
}

// Update takes the representation of a GlobalNetworkPolicy and updates it. Returns the stored
// representation of the GlobalNetworkPolicy, and an error, if there is any.
func (r globalNetworkPolicies) Update(ctx context.Context, res *apiv3.GlobalNetworkPolicy, opts options.SetOptions) (*apiv3.GlobalNetworkPolicy, error) {
	defaultPolicyTypesField(res.Spec.Ingress, res.Spec.Egress, &res.Spec.Types)

	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	// Properly prefix the name
	backendPolicyName, err := names.BackendTieredPolicyName(res.GetObjectMeta().GetName(), res.Spec.Tier)
	if err != nil {
		return nil, err
	}
	res.GetObjectMeta().SetName(backendPolicyName)
	out, err := r.client.resources.Update(ctx, opts, apiv3.KindGlobalNetworkPolicy, res)
	if out != nil {
		// Remove the prefix out of the returned policy name.
		retName, retErr := names.ClientTieredPolicyName(out.GetObjectMeta().GetName())
		if retErr != nil {
			return nil, retErr
		}
		out.GetObjectMeta().SetName(retName)
		return out.(*apiv3.GlobalNetworkPolicy), err
	}

	// Remove the prefix out of the returned policy name.
	retName, retErr := names.ClientTieredPolicyName(res.GetObjectMeta().GetName())
	if retErr != nil {
		return nil, retErr
	}
	res.GetObjectMeta().SetName(retName)
	return nil, err
}

// Delete takes name of the GlobalNetworkPolicy and deletes it. Returns an error if one occurs.
func (r globalNetworkPolicies) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalNetworkPolicy, error) {
	backendPolicyName := names.TieredPolicyName(name)
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindGlobalNetworkPolicy, noNamespace, backendPolicyName)
	if out != nil {
		retName, retErr := names.ClientTieredPolicyName(out.GetObjectMeta().GetName())
		if retErr != nil {
			return nil, retErr
		}
		// Remove the prefix out of the returned policy name.
		out.GetObjectMeta().SetName(retName)
		return out.(*apiv3.GlobalNetworkPolicy), err
	}
	return nil, err
}

// Get takes name of the GlobalNetworkPolicy, and returns the corresponding GlobalNetworkPolicy object,
// and an error if there is any.
func (r globalNetworkPolicies) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalNetworkPolicy, error) {
	backendPolicyName := names.TieredPolicyName(name)
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindGlobalNetworkPolicy, noNamespace, backendPolicyName)
	if out != nil {
		// Remove the prefix out of the returned policy name.
		retName, retErr := names.ClientTieredPolicyName(out.GetObjectMeta().GetName())
		if retErr != nil {
			return nil, retErr
		}
		out.GetObjectMeta().SetName(retName)
		return out.(*apiv3.GlobalNetworkPolicy), err
	}
	return nil, err
}

// List returns the list of GlobalNetworkPolicy objects that match the supplied options.
func (r globalNetworkPolicies) List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalNetworkPolicyList, error) {
	res := &apiv3.GlobalNetworkPolicyList{}
	// Add the name prefix if name is provided
	if opts.Name != "" {
		opts.Name = names.TieredPolicyName(opts.Name)
	}

	if err := r.client.resources.List(ctx, opts, apiv3.KindGlobalNetworkPolicy, apiv3.KindGlobalNetworkPolicyList, res); err != nil {
		return nil, err
	}

	// Remove the prefix off of each policy name
	for i, _ := range res.Items {
		name := res.Items[i].GetObjectMeta().GetName()
		retName, err := names.ClientTieredPolicyName(name)
		if err != nil {
			log.WithError(err).Infof("Skipping incorrectly formatted policy name %v", name)
			continue
		}
		res.Items[i].GetObjectMeta().SetName(retName)
	}

	return res, nil
}

// Watch returns a watch.Interface that watches the globalNetworkPolicies that match the
// supplied options.
func (r globalNetworkPolicies) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	// Add the name prefix if name is provided
	if opts.Name != "" {
		opts.Name = names.TieredPolicyName(opts.Name)
	}

	return r.client.resources.Watch(ctx, opts, apiv3.KindGlobalNetworkPolicy, &policyConverter{})
}

func defaultPolicyTypesField(ingressRules, egressRules []apiv3.Rule, types *[]apiv3.PolicyType) {
	if len(*types) == 0 {
		// Default the Types field according to what inbound and outbound rules are present
		// in the policy.
		if len(egressRules) == 0 {
			// Policy has no egress rules, so apply this policy to ingress only.  (Note:
			// intentionally including the case where the policy also has no ingress
			// rules.)
			*types = []apiv3.PolicyType{apiv3.PolicyTypeIngress}
		} else if len(ingressRules) == 0 {
			// Policy has egress rules but no ingress rules, so apply this policy to
			// egress only.
			*types = []apiv3.PolicyType{apiv3.PolicyTypeEgress}
		} else {
			// Policy has both ingress and egress rules, so apply this policy to both
			// ingress and egress.
			*types = []apiv3.PolicyType{apiv3.PolicyTypeIngress, apiv3.PolicyTypeEgress}
		}
	}
}

func convertPolicyNameForStorage(name string) string {
	// Do nothing on names prefixed with "knp."
	if strings.HasPrefix(name, "knp.") {
		return name
	}
	return "default." + name
}

func convertPolicyNameFromStorage(name string) string {
	// Do nothing on names prefixed with "knp."
	if strings.HasPrefix(name, "knp.") {
		return name
	}
	parts := strings.SplitN(name, ".", 2)
	return parts[len(parts)-1]
}

type policyConverter struct{}

func (pc *policyConverter) Convert(r resource) resource {
	retName, err := names.ClientTieredPolicyName(r.GetObjectMeta().GetName())
	if err != nil {
		log.WithError(err).Infof("Stored policy name is invalid: %s", r.GetObjectMeta().GetName())
		return r
	}
	r.GetObjectMeta().SetName(retName)
	return r
}
