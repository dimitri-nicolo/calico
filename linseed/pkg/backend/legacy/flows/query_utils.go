// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.

package flows

import (
	"errors"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/names"
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
		query, err := policyQuery(m)
		if err != nil {
			return nil, err
		}
		b.Should(query)
	}
	b.MinimumNumberShouldMatch(1)
	return b, nil
}

func policyQuery(m v1.PolicyMatch) (elastic.Query, error) {
	indexMatch, nameMatch, actionMatch := "*", "*", "*"

	// Validate and set the tier according to the policy type
	tier, err := validateAndSetTierValue(m.Tier, m.Type)
	if err != nil {
		return nil, err
	}

	// Set the action if an action is provided, otherwise action should be set to `*` to match against all actions
	if m.Action != nil && *m.Action != "" {
		actionMatch = string(*m.Action)
	}

	// Set policy name if it is provided, otherwise name should be set to `*` to match against all names
	if m.Name != nil && *m.Name != "" {
		nameMatch = *m.Name
	}

	// Policy combined-name in flowlogs is constructed differently depending on the type of hit.
	// The formatting can be found in: https://github.com/tigera/calico-private/blob/master/felix/calc/policy_lookup_cache.go
	// - Namespaced policy: <namespace>/<tier>.<name>
	// - Global / Profile policy: <tier>.<name>
	// - kubernetes policy: <namespace>/knp.default.<name>
	// - kubernetes admin policy: kanp.adminnetworkpolicy.<name>
	// m.Type defines how the name should be constructed
	switch m.Type {
	case v1.KNP:
		// staged kubernetes network policy format:
		// "<index>|<namespace>|<namespace>/<staged:>knp.default.<name>|<action>|<rule>"
		nameMatch = fmt.Sprintf("%s%s", names.K8sNetworkPolicyNamePrefix, nameMatch)
		if m.Staged {
			nameMatch = fmt.Sprintf("staged:%s", nameMatch)
		}
	case v1.KANP:
		// staged kubernetes admin network policies:
		// "<index>|adminnetworkpolicy|adminnetworkpolicy.staged:<name>|<action>|<rule>"
		if m.Staged {
			nameMatch = fmt.Sprintf("staged:%s", nameMatch)
		}
		nameMatch = fmt.Sprintf("%s%s", names.K8sAdminNetworkPolicyNamePrefix, nameMatch)
	default:
		// Calico staged policy:
		// staged namespaced policies: <namespace>/<tier>.<staged:><name>
		// staged global policies: <tier>.staged:<name>
		if m.Staged {
			nameMatch = fmt.Sprintf("staged:%s", nameMatch)
		}
		nameMatch = fmt.Sprintf("%s.%s", tier, nameMatch)
	}

	// Set namespace
	if m.Namespace != nil && *m.Namespace != "" {
		if m.Tier == "__PROFILE__" {
			return nil, errors.New("namespace cannot be set when tier==__PROFILE__")
		}
		if m.Type == v1.KANP {
			return nil, errors.New("namespace cannot be set for kubernetes admin network policies")
		}
		nameMatch = fmt.Sprintf("%s/%s", *m.Namespace, nameMatch)
	}

	// Policy strings are formatted like so:
	// <index> | <tier> | <combined-name> | <action> | <ruleID>
	matchString := fmt.Sprintf("%s|%s|%s|%s|*", indexMatch, tier, nameMatch, actionMatch)
	logrus.WithField("match", matchString).Debugf("Matching on policy string")

	wildcard := elastic.NewWildcardQuery("policies.all_policies", matchString)
	return elastic.NewNestedQuery("policies", wildcard), nil
}

func validateAndSetTierValue(tier string, policyType v1.PolicyType) (string, error) {
	// Validate tier for knp
	if policyType == v1.KNP {
		if tier != "" && tier != names.DefaultTierName {
			return "", fmt.Errorf("tier cannot be set to %v for policy type %v", tier, policyType)
		} else {
			return names.DefaultTierName, nil
		}
	}

	// Validate tier for kanp
	if policyType == v1.KANP {
		if tier != "" && tier != names.AdminNetworkPolicyTierName {
			return "", fmt.Errorf("tier cannot be set to %v for policy type %v", tier, policyType)
		} else {
			return names.AdminNetworkPolicyTierName, nil
		}
	}

	if tier != "" {
		return tier, nil
	} else {
		// Match against all tiers if m.Tier is empty
		return "*", nil
	}
}
