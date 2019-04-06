// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package testutils

import (
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var (
	CalicoRuleAllowSourceNetsInternet = apiv3.Rule{
		Action: apiv3.Allow,
		Source: apiv3.EntityRule{
			Nets: []string{"1.1.1.1"},
		},
	}

	CalicoRuleAllowSourceNetsPrivate = apiv3.Rule{
		Action: apiv3.Allow,
		Source: apiv3.EntityRule{
			Nets: []string{"10.0.100.0"},
		},
	}

	CalicoRuleAllowDestinationNetsInternet = apiv3.Rule{
		Action: apiv3.Allow,
		Destination: apiv3.EntityRule{
			Nets: []string{"1.1.1.1"},
		},
	}

	CalicoRuleAllowDestinationNetsPrivate = apiv3.Rule{
		Action: apiv3.Allow,
		Destination: apiv3.EntityRule{
			Nets: []string{"10.0.100.0"},
		},
	}

	CalicoRuleAllowSourceSelector = apiv3.Rule{
		Action: apiv3.Allow,
		Source: apiv3.EntityRule{
			Selector: "x == 'y'",
		},
	}

	CalicoRuleAllowSourceNamespaceSelector = apiv3.Rule{
		Action: apiv3.Allow,
		Source: apiv3.EntityRule{
			NamespaceSelector: "x == 'y'",
		},
	}

	CalicoRuleAllowDestinationSelector = apiv3.Rule{
		Action: apiv3.Allow,
		Destination: apiv3.EntityRule{
			Selector: "x == 'y'",
		},
	}

	CalicoRuleAllowDestinationNetspaceSelector = apiv3.Rule{
		Action: apiv3.Allow,
		Destination: apiv3.EntityRule{
			NamespaceSelector: "x == 'y'",
		},
	}

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
