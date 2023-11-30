// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.

package flows

import (
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

func BuildPolicyMatchQuery(policyMatches []v1.PolicyMatch) (*elastic.BoolQuery, error) {
	if len(policyMatches) == 0 {
		return nil, nil
	}

	// Filter-in any flow logs that match any of the given policy matches.
	b := elastic.NewBoolQuery()
	for _, m := range policyMatches {
		// only build query for non-empty PolicyMatch. Return error if there is an empty PolicyMatch.
		if (m == v1.PolicyMatch{}) {
			return nil, fmt.Errorf("PolicyMatch passed to BuildPolicyMatchQuery cannot be empty")
		}
		b.Should(policyQuery(m))
	}
	b.MinimumNumberShouldMatch(1)
	return b, nil
}

func policyQuery(m v1.PolicyMatch) elastic.Query {
	index := "*"
	tier := "*"
	name := "*"
	action := "*"
	if m.Tier != "" {
		tier = m.Tier
	}

	// Names can look differently depending on the type of hit.
	// - Namespaced policy: <namespace>/<tier>.<name>
	// - Global / Profile: <tier>.<name>
	if m.Name != nil {
		name = fmt.Sprintf("%s.%s", tier, *m.Name)
	}
	if m.Namespace != nil {
		name = fmt.Sprintf("%s/%s", *m.Namespace, name)
	}
	if m.Action != nil {
		action = string(*m.Action)
	}

	// Policy strings are formatted like so:
	// <index> | <tier> | <name> | <action> | <ruleID>
	matchString := fmt.Sprintf("%s|%s|%s|%s*", index, tier, name, action)
	logrus.WithField("match", matchString).Debugf("Matching on policy string")
	wildcard := elastic.NewWildcardQuery("policies.all_policies", matchString)
	return elastic.NewNestedQuery("policies", wildcard)
}
