/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admission

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

const (
	policyDelim = "."
)

type CalicoAdmission struct {
	admission.Interface
	authorizer.Authorizer
}

func NewCalicoAdmission(ad admission.Interface, az authorizer.Authorizer) *CalicoAdmission {
	ca := &CalicoAdmission{
		ad,
		az,
	}

	return ca
}

func getTierPolicy(policyName string) (string, string, error) {
	policySlice := strings.Split(policyName, policyDelim)
	if len(policySlice) < 2 {
		return "", "", fmt.Errorf("error parsing policy name %s. expecting <tier>.<policy>", policyName)
	}
	return policySlice[0], policySlice[1], nil
}

func (c *CalicoAdmission) Admit(a admission.Attributes) (err error) {

	resourceType := a.GetResource().Resource
	if resourceType == "policies" {
		policy := a.GetObject().(*calico.Policy)
		tierName, policyName, err := getTierPolicy(policy.Name)
		if err != nil {
			return apierrors.NewBadRequest("Policy name not formatted")
		}
		attrs := authorizer.AttributesRecord{}
		attrs.APIGroup = a.GetKind().Group
		attrs.APIVersion = a.GetKind().Version
		attrs.Name = tierName
		attrs.Resource = "tiers"
		attrs.User = a.GetUserInfo()
		switch a.GetOperation() {
		case "CREATE":
			attrs.Verb = "POST"
		case "UPDATE":
			attrs.Verb = "PUT"
		case "DELETE":
			attrs.Verb = "DELETE"
		}
		authorized, reason, err := c.Authorizer.Authorize(attrs)
		glog.Infof("For policy %s tier %s is getting authorized first and its a %s. Reason: %s", policyName, tierName, authorized, reason)
		if !authorized {
			return apierrors.NewForbidden(a.GetResource().GroupResource(), policy.Name, fmt.Errorf("Policy resource cannot be created based on assocaited Tier's role binding"))
		}
	}

	if c.Interface != nil {
		return c.Interface.Admit(a)
	}
	return nil
}

func (c *CalicoAdmission) Handles(operation admission.Operation) bool {
	return true
}
