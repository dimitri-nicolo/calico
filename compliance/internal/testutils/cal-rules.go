// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package testutils

import (
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func getCalicoAction(a Action) apiv3.Action {
	switch a {
	case Allow:
		return apiv3.Allow
	default:
		return apiv3.Deny
	}
}

func getCalicoNets(n Net) []string {
	var nets []string
	if n&Public != 0 {
		nets = append(nets, "1.1.1.1/32")
	}
	if n&Private != 0 {
		nets = append(nets, "10.0.100.0/24")
	}
	return nets
}

func CalicoRuleNets(a Action, e Entity, n Net) apiv3.Rule {
	var s apiv3.EntityRule
	var d apiv3.EntityRule
	if e&Source != 0 {
		s = apiv3.EntityRule{
			Nets: getCalicoNets(n),
		}
	}
	if e&Destination != 0 {
		d = apiv3.EntityRule{
			Nets: getCalicoNets(n),
		}
	}
	return apiv3.Rule{
		Action:      getCalicoAction(a),
		Source:      s,
		Destination: d,
	}
}

func CalicoRuleSelectors(a Action, e Entity, sel Selector, nsSel Selector) apiv3.Rule {
	var s apiv3.EntityRule
	var d apiv3.EntityRule
	if e&Source != 0 {
		s = apiv3.EntityRule{
			Selector:          selectorByteToSelector(sel),
			NamespaceSelector: selectorByteToSelector(nsSel),
		}

	}
	if e&Destination != 0 {
		d = apiv3.EntityRule{
			Selector:          selectorByteToSelector(sel),
			NamespaceSelector: selectorByteToSelector(nsSel),
		}
	}
	return apiv3.Rule{
		Action:      getCalicoAction(a),
		Source:      s,
		Destination: d,
	}
}

var (
	CalicoRule = apiv3.Rule{
		Action: apiv3.Allow,
		Source: apiv3.EntityRule{
			Nets:              []string{},
			Selector:          "",
			NamespaceSelector: "",
			ServiceAccounts: &apiv3.ServiceAccountMatch{
				Names:    []string{},
				Selector: "",
			},
		},
		Destination: apiv3.EntityRule{
			Nets:              []string{},
			Selector:          "",
			NamespaceSelector: "",
			ServiceAccounts: &apiv3.ServiceAccountMatch{
				Names:    []string{},
				Selector: "",
			},
		},
	}
)
