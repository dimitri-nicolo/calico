// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec_test

import (
	"fmt"
	"reflect"
	"strings"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/set"
	"github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/policyrec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

const defaultTier = "default"

var _ = Describe("Policy Recommendation Engine", func() {
	var (
		re  policyrec.RecommendationEngine
		err error
	)
	DescribeTable("Recommend policies for matching flows and endpoint",
		// endpointName and endpointNamespace are params for which recommended policies should be generated for.
		// They are used to configure the recommendation engine.
		// matchingFlows is the input flows that are passed to ProcessFlows.
		// expectedPolicies is a slice of StagedNetworkPolicy or StagedGlobalNetworkPolicy.
		func(endpointName, endpointNamespace, policyName, policyTier string, policyOrder *float64,
			matchingFlows []api.Flow, expectedPolicies interface{}) {

			By("Initializing a recommendation engine with namespace and name")
			re = policyrec.NewEndpointRecommendationEngine(endpointName, endpointNamespace, policyName, policyTier, policyOrder)

			for _, flow := range matchingFlows {
				By("Processing matching flow")
				err = re.ProcessFlow(flow)
				Expect(err).To(BeNil())
			}

			By("Once all matched flows have been input for matching endpoint and getting recommended flows")
			recommendation, err := re.Recommend()
			Expect(err).To(BeNil())
			if endpointNamespace == "" {
				// Expect only StagedGlobalNetworkPolicies.
				policies := expectedPolicies.([]*v3.StagedGlobalNetworkPolicy)
				// We loop through each expected policy and check instead of using ConsistsOf() matcher so that
				// we can use our custom MatchPolicy() Gomega matcher.
				for _, expectedPolicy := range policies {
					Expect(recommendation.GlobalNetworkPolicies).To(ContainElement(MatchPolicy(expectedPolicy)))
				}
			} else {
				// Expect only StagedNetworkPolicies if a namespace is defined.
				policies := expectedPolicies.([]*v3.StagedNetworkPolicy)
				// We loop through each expected policy and check instead of using ConsistsOf() matcher so that
				// we can use our custom MatchPolicy() Gomega matcher.
				for _, expectedPolicy := range policies {
					Expect(recommendation.NetworkPolicies).To(ContainElement(MatchPolicy(expectedPolicy)))
				}
			}
		},
		Entry("recommend a policy with egress rule for a flow betwen 2 endpoints and matching source endpoint",
			pod1Aggr, namespace1, pod1, defaultTier, nil,
			[]api.Flow{flowPod1BlueToPod2Allow443ReporterSource, flowPod1BlueToPod2Allow443ReporterDestination},
			[]*v3.StagedNetworkPolicy{networkPolicyNamespace1Pod1BlueToPod2}),
		Entry("recommend a policy with egress rule for a flow betwen 2 endpoints with a non overlapping label - and matching source endpoint",
			pod1Aggr, namespace1, pod1, defaultTier, nil,
			[]api.Flow{flowPod1BlueToPod2Allow443ReporterSource, flowPod1BlueToPod2Allow443ReporterDestination,
				flowPod1RedToPod2Allow443ReporterSource, flowPod1RedToPod2Allow443ReporterDestination},
			[]*v3.StagedNetworkPolicy{networkPolicyNamespace1Pod1ToPod2}),
		Entry("recommend a policy with egress rule for a flow betwen 2 endpoints and external network and matching source endpoint",
			pod1Aggr, namespace1, pod1, defaultTier, nil,
			[]api.Flow{flowPod1BlueToPod2Allow443ReporterSource, flowPod1BlueToPod2Allow443ReporterDestination,
				flowPod1BlueToExternalAllow53ReporterSource},
			[]*v3.StagedNetworkPolicy{networkPolicyNamespace1Pod1BlueToPod2AndExternalNet}),
		Entry("recommend a policy with egress rule for a flow betwen 2 endpoints and matching source endpoint",
			pod1Aggr, namespace1, pod1, defaultTier, nil,
			[]api.Flow{
				flowPod1BlueToPod2Allow443ReporterSource, flowPod1BlueToPod2Allow443ReporterDestination,
				flowPod1BlueToPod3Allow5432ReporterSource, flowPod1BlueToPod3Allow5432ReporterDestination,
				flowPod1RedToPod3Allow8080ReporterSource, flowPod1RedToPod3Allow8080ReporterDestination,
			},
			[]*v3.StagedNetworkPolicy{networkPolicyNamespace1Pod1ToPod2AndPod3}),
		Entry("recommend a policy with ingress and egress rules for a flow betwen 2 endpoints and matching source and destination endpoint",
			pod2Aggr, namespace1, pod2, defaultTier, nil,
			[]api.Flow{
				flowPod1BlueToPod2Allow443ReporterSource, flowPod1BlueToPod2Allow443ReporterDestination,
				flowPod2ToPod3Allow5432ReporterSource, flowPod2ToPod3Allow5432ReporterDestination,
			},
			[]*v3.StagedNetworkPolicy{networkPolicyNamespace1Pod2}),
		Entry("recommend a policy with ingress rule for flows and matching destination endpoint",
			pod3Aggr, namespace2, pod3, defaultTier, nil,
			[]api.Flow{
				flowPod1BlueToPod3Allow5432ReporterSource, flowPod1BlueToPod3Allow5432ReporterDestination,
				flowPod1RedToPod3Allow8080ReporterSource, flowPod1RedToPod3Allow8080ReporterDestination,
				flowPod2ToPod3Allow5432ReporterSource, flowPod2ToPod3Allow5432ReporterDestination,
			},
			[]*v3.StagedNetworkPolicy{networkPolicyNamespace1Pod3}),
	)
	It("should reject flows that don't match endpoint name and namaespace", func() {
		By("Initializing a recommendationengine with namespace and name")
		re = policyrec.NewEndpointRecommendationEngine(pod1Aggr, namespace1, pod1, defaultTier, nil)

		By("Processing flow that don't match")
		err = re.ProcessFlow(flowPod2ToPod3Allow5432ReporterSource)
		Expect(err).ToNot(BeNil())
		err = re.ProcessFlow(flowPod2ToPod3Allow5432ReporterDestination)
		Expect(err).ToNot(BeNil())
	})
})

// Test Utilities

// MatchPolicy is a convenience function that returns a policyMatcher for matching
// policies in a Gomega assertion.
func MatchPolicy(expected interface{}) *policyMatcher {
	log.Debugf("Creating policy matcher")
	return &policyMatcher{expected: expected}
}

// policyMatcher implements the GomegaMatcher interface to match policies.
type policyMatcher struct {
	expected interface{}
}

func (pm *policyMatcher) Match(actual interface{}) (success bool, err error) {
	// We expect to only handle pointer to TSEE NetworkPolicy for now.
	// TODO(doublek): Support for other policy resources should be added here.
	switch actualPolicy := actual.(type) {
	case *v3.StagedNetworkPolicy:
		expectedPolicy := pm.expected.(*v3.StagedNetworkPolicy)
		success = (expectedPolicy.GroupVersionKind().Kind == actualPolicy.GroupVersionKind().Kind &&
			expectedPolicy.GroupVersionKind().Version == actualPolicy.GroupVersionKind().Version &&
			expectedPolicy.GetName() == actualPolicy.GetName() &&
			expectedPolicy.GetNamespace() == actualPolicy.GetNamespace() &&
			expectedPolicy.Spec.Tier == actualPolicy.Spec.Tier &&
			expectedPolicy.Spec.Order == actualPolicy.Spec.Order &&
			reflect.DeepEqual(expectedPolicy.Spec.Types, actualPolicy.Spec.Types) &&
			matchSelector(expectedPolicy.Spec.Selector, actualPolicy.Spec.Selector) &&
			matchRules(expectedPolicy.Spec.Ingress, actualPolicy.Spec.Ingress) &&
			matchRules(expectedPolicy.Spec.Egress, actualPolicy.Spec.Egress))
	default:
		// TODO(doublek): Remove this after testing the test.
		log.Debugf("Default case")

	}
	return
}

func matchSelector(actual, expected string) bool {
	// Currently only matches &&-ed selectors.
	// TODO(doublek): Add support for ||-ed selectors as well.
	actualSelectors := strings.Split(actual, " && ")
	expectedSelectors := strings.Split(expected, " && ")
	as := set.FromArray(actualSelectors)
	es := set.FromArray(expectedSelectors)
	es.Iter(func(item interface{}) error {
		if as.Contains(item) {
			as.Discard(item)
			return set.RemoveItem
		}
		return nil
	})
	log.Debugf("\nActual %+v\nExpected %+v\n", actual, expected)
	if es.Len() != 0 || as.Len() != 0 {
		return false
	}
	return true
}

func matchRules(actual, expected []v3.Rule) bool {
	// TODO(doublek): Make sure there aren't any extra rules left over in either params.
NEXTRULE:
	for _, actualRule := range actual {
		for i, expectedRule := range expected {
			if matchSingleRule(actualRule, expectedRule) {
				expected = append(expected[:i], expected[i+1:]...)
				continue NEXTRULE
			}
		}
		log.Debugf("\nDidn't find a match for rule\n\t%+v", actualRule)
		return false
	}
	if len(expected) != 0 {
		log.Debugf("\nDidn't find matching actual rules\n\t%+v for  expected rules\n\t%+v\n", actual, expected)
		return false
	}
	return true
}

func matchSingleRule(actual, expected v3.Rule) bool {
	return matchEntityRule(actual.Source, expected.Source) &&
		matchEntityRule(actual.Destination, expected.Destination) &&
		actual.Protocol.NumVal == expected.Protocol.NumVal
}

func matchEntityRule(actual, expected v3.EntityRule) bool {
	match := set.FromArray(actual.Nets).ContainsAll(set.FromArray(expected.Nets)) &&
		set.FromArray(actual.Ports).ContainsAll(set.FromArray(expected.Ports)) &&
		matchSelector(actual.Selector, expected.Selector) &&
		set.FromArray(actual.NotNets).ContainsAll(set.FromArray(expected.NotNets))
	if actual.ServiceAccounts != nil && expected.ServiceAccounts != nil {
		return match &&
			set.FromArray(actual.ServiceAccounts.Names).ContainsAll(set.FromArray(expected.ServiceAccounts.Names)) &&
			matchSelector(actual.ServiceAccounts.Selector, expected.ServiceAccounts.Selector)
	}
	return match
}

func (pm *policyMatcher) FailureMessage(actual interface{}) (message string) {
	message = fmt.Sprintf("Expected\n\t%#v\nto match\n\t%#v", actual, pm.expected)
	return
}

func (pm *policyMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	message = fmt.Sprintf("Expected\n\t%#v\nnot to match\n\t%#v", actual, pm.expected)
	return
}
