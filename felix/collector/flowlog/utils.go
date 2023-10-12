// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package flowlog

import (
	"fmt"
	"strings"

	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/collector/types/metric"
	"github.com/projectcalico/calico/felix/rules"
)

const (
	FieldNotIncluded                 = "-"
	fieldNotIncludedForNumericFields = 0
	fieldAggregated                  = "*"

	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"

	ReporterSrc reporterType = "src"
	ReporterDst reporterType = "dst"
)

// GetActionAndReporterFromRuleID converts the action to a string value.
func getActionAndReporterFromRuleID(r *calc.RuleID) (a Action, flr reporterType) {
	switch r.Action {
	case rules.RuleActionDeny:
		a = ActionDeny
	case rules.RuleActionAllow:
		a = ActionAllow
	}
	switch r.Direction {
	case rules.RuleDirIngress:
		flr = ReporterDst
	case rules.RuleDirEgress:
		flr = ReporterSrc
	}
	return
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

func getService(svc metric.ServiceInfo) FlowService {
	if svc.Name == "" {
		return FlowService{
			Namespace: FieldNotIncluded,
			Name:      FieldNotIncluded,
			PortName:  FieldNotIncluded,
			PortNum:   fieldNotIncludedForNumericFields,
		}
	} else if svc.Port == "" { // proxy.ServicePortName.Port refers to the PortName
		// A single port for a service may not have a name.
		return FlowService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
			PortName:  FieldNotIncluded,
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
