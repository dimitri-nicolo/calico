/*
Copyright 2017 The Kubernetes Authors.

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

package v1

import (
	"github.com/projectcalico/libcalico-go/lib/net"

	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	// Add non-generated conversion functions
	err := scheme.AddConversionFuncs(
		Convert_v1_Policy_To_calico_Policy,
		Convert_calico_Policy_To_v1_Policy,
	)
	if err != nil {
		return err
	}

	return nil
}

// ruleActionAPIToBackend converts the rule action field value from the API
// value to the equivalent backend value.
func ruleActionAPIToBackend(action string) string {
	if action == "pass" {
		return "next-tier"
	}
	return action
}

// ruleActionBackendToAPI converts the rule action field value from the backend
// value to the equivalent API value.
func ruleActionBackendToAPI(action string) string {
	if action == "" {
		return "allow"
	} else if action == "next-tier" {
		return "pass"
	}
	return action
}

// ruleAPIToBackend converts an API Rule structure to a Backend Rule structure.
func ruleAPIToBackend(ar api.Rule) model.Rule {
	var icmpCode, icmpType, notICMPCode, notICMPType *int
	if ar.ICMP != nil {
		icmpCode = ar.ICMP.Code
		icmpType = ar.ICMP.Type
	}

	if ar.NotICMP != nil {
		notICMPCode = ar.NotICMP.Code
		notICMPType = ar.NotICMP.Type
	}

	return model.Rule{
		Action:      ruleActionAPIToBackend(ar.Action),
		IPVersion:   ar.IPVersion,
		Protocol:    ar.Protocol,
		ICMPCode:    icmpCode,
		ICMPType:    icmpType,
		NotProtocol: ar.NotProtocol,
		NotICMPCode: notICMPCode,
		NotICMPType: notICMPType,

		SrcTag:      ar.Source.Tag,
		SrcNet:      ar.Source.Net,
		SrcSelector: ar.Source.Selector,
		SrcPorts:    ar.Source.Ports,
		DstTag:      ar.Destination.Tag,
		DstNet:      normalizeIPNet(ar.Destination.Net),
		DstSelector: ar.Destination.Selector,
		DstPorts:    ar.Destination.Ports,

		NotSrcTag:      ar.Source.NotTag,
		NotSrcNet:      ar.Source.NotNet,
		NotSrcSelector: ar.Source.NotSelector,
		NotSrcPorts:    ar.Source.NotPorts,
		NotDstTag:      ar.Destination.NotTag,
		NotDstNet:      normalizeIPNet(ar.Destination.NotNet),
		NotDstSelector: ar.Destination.NotSelector,
		NotDstPorts:    ar.Destination.NotPorts,
	}
}

// normalizeIPNet converts an IPNet to a network by ensuring the IP address
// is correctly masked.
func normalizeIPNet(n *net.IPNet) *net.IPNet {
	if n == nil {
		return nil
	}
	return n.Network()
}

// ruleBackendToAPI convert a Backend Rule structure to an API Rule structure.
func ruleBackendToAPI(br model.Rule) api.Rule {
	var icmp, notICMP *api.ICMPFields
	if br.ICMPCode != nil || br.ICMPType != nil {
		icmp = &api.ICMPFields{
			Code: br.ICMPCode,
			Type: br.ICMPType,
		}
	}
	if br.NotICMPCode != nil || br.NotICMPType != nil {
		notICMP = &api.ICMPFields{
			Code: br.NotICMPCode,
			Type: br.NotICMPType,
		}
	}
	return api.Rule{
		Action:      ruleActionBackendToAPI(br.Action),
		IPVersion:   br.IPVersion,
		Protocol:    br.Protocol,
		ICMP:        icmp,
		NotProtocol: br.NotProtocol,
		NotICMP:     notICMP,
		Source: api.EntityRule{
			Tag:         br.SrcTag,
			Net:         br.SrcNet,
			Selector:    br.SrcSelector,
			Ports:       br.SrcPorts,
			NotTag:      br.NotSrcTag,
			NotNet:      br.NotSrcNet,
			NotSelector: br.NotSrcSelector,
			NotPorts:    br.NotSrcPorts,
		},

		Destination: api.EntityRule{
			Tag:         br.DstTag,
			Net:         br.DstNet,
			Selector:    br.DstSelector,
			Ports:       br.DstPorts,
			NotTag:      br.NotDstTag,
			NotNet:      br.NotDstNet,
			NotSelector: br.NotDstSelector,
			NotPorts:    br.NotDstPorts,
		},
	}
}

// rulesAPIToBackend converts an API Rule structure slice to a Backend Rule structure slice.
func rulesAPIToBackend(ars []api.Rule) []model.Rule {
	if ars == nil {
		return []model.Rule{}
	}

	brs := make([]model.Rule, len(ars))
	for idx, ar := range ars {
		brs[idx] = ruleAPIToBackend(ar)
	}
	return brs
}

// rulesBackendToAPI converts a Backend Rule structure slice to an API Rule structure slice.
func rulesBackendToAPI(brs []model.Rule) []api.Rule {
	if brs == nil {
		return nil
	}

	ars := make([]api.Rule, len(brs))
	for idx, br := range brs {
		ars[idx] = ruleBackendToAPI(br)
	}
	return ars
}

func Convert_v1_Policy_To_calico_Policy(in *Policy, out *calico.Policy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec.DoNotTrack = in.Spec.DoNotTrack
	out.Spec.Selector = in.Spec.Selector
	out.Spec.Order = in.Spec.Order
	out.Spec.OutboundRules = rulesAPIToBackend(in.Spec.EgressRules)
	out.Spec.InboundRules = rulesAPIToBackend(in.Spec.IngressRules)

	if err := Convert_v1_PolicyStatus_To_calico_PolicyStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

func Convert_calico_Policy_To_v1_Policy(in *calico.Policy, out *Policy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec.DoNotTrack = in.Spec.DoNotTrack
	out.Spec.Selector = in.Spec.Selector
	out.Spec.Order = in.Spec.Order
	out.Spec.EgressRules = rulesBackendToAPI(in.Spec.OutboundRules)
	out.Spec.IngressRules = rulesBackendToAPI(in.Spec.InboundRules)

	if err := Convert_calico_PolicyStatus_To_v1_PolicyStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}
