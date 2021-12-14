package calico

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func rulesHaveDNSDomain(rules []v3.Rule) bool {
	for _, r := range rules {
		if len(r.Destination.Domains) != 0 {
			return true
		}
		if len(r.Source.Domains) != 0 {
			return true
		}
	}

	return false
}
