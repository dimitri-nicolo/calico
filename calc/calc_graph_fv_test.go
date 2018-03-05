// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.

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

// This file tests the mapping from datastore content - expressed as KVUpdates using model.* objects
// - to proto.* messages.

package calc_test

import (
	. "github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/dataplane/mock"

	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/config"
	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/health"
)

// Each entry in baseTests contains a series of states to move through (defined in
// states_for_test.go). Apart from running each of these, we'll also expand each of them by
// passing it through the expansion functions below.  In particular, we'll do each of them
// in reversed order and reversed KV injection order.
var baseTests = []StateList{
	// Empty should be empty!
	{},

	// Add one endpoint then remove it and add another with overlapping IP.
	{localEp1WithPolicy, localEp2WithPolicy},

	// Same but ingress-only policy on ep1.
	{localEp1WithIngressPolicy, localEp2WithPolicy},

	// Add one endpoint then another with an overlapping IP, then remove
	// first.
	{localEp1WithPolicy, localEpsWithPolicy, localEp2WithPolicy},

	// Add both endpoints, then return to empty, then add them both back.
	{localEpsWithPolicy, initialisedStore, localEpsWithPolicy},

	// IP updates.
	{localEpsWithPolicy, localEpsWithPolicyUpdatedIPs, localEp1WithIngressPolicy},

	// Add a profile and a couple of endpoints.  Then update the profile to
	// use different tags and selectors.
	{localEpsWithProfile, localEpsWithUpdatedProfile},

	// Tests of policy ordering.  Each state has one tier but we shuffle
	// the order of the policies within it.
	{localEp1WithOneTierPolicy123,
		localEp1WithOneTierPolicy321,
		localEp1WithOneTierPolicyAlpha},

	// Test mutating the profile list of some endpoints.
	{localEpsWithNonMatchingProfile, localEpsWithProfile},

	// Host endpoint tests.
	{hostEp1WithPolicy, hostEp2WithPolicy, hostEp1WithIngressPolicy, hostEp1WithEgressPolicy},

	// Network set tests.
	{hostEp1WithPolicy,
		hostEp1WithPolicyAndANetworkSet,
		hostEp1WithPolicyAndANetworkSetMatchingBEqB,
		hostEp2WithPolicy,
		hostEp1WithPolicyAndANetworkSet,
		hostEp1WithPolicyAndTwoNetworkSets},

	// Untracked policy on its own.
	{hostEp1WithUntrackedPolicy},
	// Mixed policy.
	{hostEp1WithTrackedAndUntrackedPolicy},
	// Single policy switches between tracked/untracked.
	{hostEp1WithUntrackedPolicy, hostEp1WithPolicy, hostEp1WithIngressPolicy},
	{hostEp1WithUntrackedPolicy, hostEp1WithTrackedAndUntrackedPolicy, hostEp1WithPolicy},

	// Pre-DNAT policy, then egress-only policy.
	{hostEp1WithPreDNATPolicy, hostEp1WithEgressPolicy},

	// Tag to label inheritance.  Tag foo should be inherited as label
	// foo="".
	{withProfileTagInherit, localEpsWithTagInheritProfile},
	// But if there's an explicit label, it overrides the tag.
	{localEpsWithTagOverriddenProfile, withProfileTagOverriden},

	// Named ports. Simple cases.
	{localEp1WithNamedPortPolicy},
	{localEp1WithNamedPortPolicyUDP},
	{localEpsAndNamedPortPolicyDuplicatePorts},
	{localEp1WithNamedPortPolicyNoSelector},
	{localEp1WithNegatedNamedPortPolicyNoSelector},
	{localEp1WithNegatedNamedPortPolicy},
	{localEp1WithNegatedNamedPortPolicyDest},
	// Host endpoints have named ports too.
	{localHostEp1WithNamedPortPolicy},
	{hostEp1WithPolicy, localHostEp1WithNamedPortPolicy, hostEp1WithPolicy},
	// Endpoints with overlapping IPs.
	{localEpsWithNamedPortsPolicy},
	{localEp1WithNamedPortPolicy, localEpsWithNamedPortsPolicy},
	// Endpoints with overlapping IPs but different port numbers.
	{localEpsWithNamedPortsPolicyTCPPort2},
	// Policy has protocol=TCP but named ports defined as UDP and vice-versa.
	{localEpsWithMismatchedNamedPortsPolicy},
	// Handling a port update.
	{localEpsWithNamedPortsPolicy, localEpsWithNamedPortsPolicyTCPPort2},
	// Add named ports to policy and then remove them.
	{hostEp1WithPolicy, localEp1WithNamedPortPolicy, hostEp1WithPolicy},
	{hostEp1WithPolicy, localEp1WithNamedPortPolicyNoSelector, hostEp1WithPolicy},
	{hostEp1WithPolicy, localEpsWithNamedPortsPolicy, hostEp1WithPolicy},
	// In this scenario, the endpoint only matches the selector of the named port due to
	// inheriting a label from its profile.
	{
		// Start with the endpoints and profile but no policy.
		localEpsWithOverlappingIPsAndInheritedLabels,
		// Policy added, matches EP1 due to its inheritance.
		localEpsAndNamedPortPolicyMatchingInheritedLabelOnEP1,
		// Add label to EP2 via inheritance.
		localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs,
		// Then change inherited label on EP2 to stop the match.
		localEpsAndNamedPortPolicyNoLongerMatchingInheritedLabelOnEP2,
		// Ditto for EP1.  Now matches none of the EPs.
		localEpsAndNamedPortPolicyNoLongerMatchingInheritedLabelOnEP1},
	// This scenario introduces ports with duplicate names.
	{
		// Start with endpoints and policy.
		localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs,
		// Adjust workload 1 to have duplicate ports.
		localEpsAndNamedPortPolicyDuplicatePorts,
		// Then go back...
		localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs,
	},
	// In this scenario, we remove the profiles from the endpoints rather than changing the labels.
	{
		// Start with both matching, as in the middle of the above test.
		localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs,
		// Remove the profiles from EP2.
		localEpsAndNamedPortPolicyEP2ProfileRemoved,
		// Ditto for EP1.  Named port now matches none of the EPs.
		localEpsAndNamedPortPolicyBothEPsProfilesRemoved,
		// Add everything back.
		localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs,
	},

	// Repro of a particular named port index update failure case.  The inherited profile was
	// improperly cleaned up, so, when it was added back in again we ended up with multiple copies.
	{localEpsWithTagInheritProfile,
		localEp1WithPolicy,
		localEpsWithProfile},

	// A long, fairly random sequence of updates.
	{
		localEpsWithProfile,
		localEp1WithOneTierPolicy123,
		localEpsWithNonMatchingProfile,
		localEpsWithTagInheritProfile,
		localEpsWithPolicy,
		localEpsWithPolicyUpdatedIPs,
		hostEp1WithPolicy,
		localEpsWithUpdatedProfile,
		withProfileTagInherit,
		localEp1WithIngressPolicy,
		localEpsWithNonMatchingProfile,
		localEpsWithUpdatedProfileNegatedTags,
		hostEp1WithUntrackedPolicy,
		localEpsWithTagInheritProfile,
		localEp1WithPolicy,
		localEpsWithProfile,
	},

	// And another one.
	{
		localEpsWithProfile,
		localEp1WithOneTierPolicy123,
		localEpsWithNonMatchingProfile,
		localEpsWithTagInheritProfile,
		hostEp1WithUntrackedPolicy,
		localEpsWithTagInheritProfile,
		localEpsWithMismatchedNamedPortsPolicy,
		localEp1WithPolicy,
		localEp1WithNamedPortPolicyNoSelector,
		localEpsWithProfile,
		localEpsAndNamedPortPolicyMatchingInheritedLabelBothEPs,
		localEp1WithIngressPolicy,
		localEpsWithNonMatchingProfile,
		localEpsWithUpdatedProfileNegatedTags,
		localEpsWithPolicy,
		localEp1WithNamedPortPolicyNoSelector,
		localEpsWithPolicyUpdatedIPs,
		hostEp1WithPolicy,
		localEpsWithUpdatedProfile,
		withProfileTagInherit,
		localEp1WithNamedPortPolicyUDP,
		localEp1WithNamedPortPolicyUDP,
	},

	// And another.
	{localEpsWithProfile,
		localEp1WithOneTierPolicy123,
		localEpsWithNonMatchingProfile,
		localEpsWithTagInheritProfile,
		localEpsWithPolicy,
		localEpsWithPolicyUpdatedIPs,
		hostEp1WithPolicyAndANetworkSetMatchingBEqB,
		hostEp1WithPolicy,
		localEpsWithUpdatedProfile,
		withProfileTagInherit,
		hostEp1WithPolicyAndTwoNetworkSets,
		localEp1WithIngressPolicy,
		localEpsWithNonMatchingProfile,
		localEpsWithUpdatedProfileNegatedTags,
		hostEp1WithUntrackedPolicy,
		localEpsWithTagInheritProfile,
		localEp1WithPolicy,
		localEpsWithProfile,
		hostEp1WithPolicyAndANetworkSet,
	},

	// TODO(smc): Test config calculation
	// TODO(smc): Test mutation of endpoints
	// TODO(smc): Test mutation of host endpoints
	// TODO(smc): Test validation
	// TODO(smc): Test rule conversions
}

var testExpanders = []func(baseTest StateList) (desc string, mappedTests []StateList){
	identity,
	reverseKVOrder,
	reverseStateOrder,
	insertEmpties,
	splitStates,
	squashStates,
}

// These tests drive the calculation graph directly (and synchronously).
// They take the StateLists in baseTests, expand them using the test expansion
// functions and then drive the graph through the expanded states.  These tests
// also deterministically decide when to flush the calculation graph's buffers
// so they should be deterministic overall.  Any non-determinism is likely to
// come from iterations over maps or sets inside the calculation graph.
//
// Debugging note: since the tests get expanded, a single failure in a base
// test often creates many fails in the output as each expansion of that test
// is also likely to fail.  A good strategy for debugging is to focus on the
// base tests first.
var _ = Describe("Calculation graph state sequencing tests:", func() {
	describeSyncTests(baseTests)
})

func describeSyncTests(baseTests []StateList) {
	for _, test := range baseTests {
		baseTest := test
		for _, expander := range testExpanders {
			expanderDesc, expandedTests := expander(baseTest)
			for _, expandedTest := range expandedTests {
				// Always worth adding an empty to the end of the test.
				expandedTest = append(expandedTest, empty)
				desc := fmt.Sprintf("with input states %v %v", baseTest, expanderDesc)
				Describe(desc+" flushing after each KV", func() {
					doStateSequenceTest(expandedTest, afterEachKV)
				})
				Describe(desc+" flushing after each KV and duplicating each update", func() {
					doStateSequenceTest(expandedTest, afterEachKVAndDupe)
				})
				Describe(desc+" flushing after each state", func() {
					doStateSequenceTest(expandedTest, afterEachState)
				})
				Describe(desc+" flushing at end only", func() {
					doStateSequenceTest(expandedTest, atEnd)
				})
			}
		}
	}
}

// These tests use the same expansion logic as the synchronous tests above
// but they drive the calculation graph via its asynchronous channel interface.
// Since they don't have control over when the graph gets flushed, they are
// less deterministic than the tests above and they can't test the output after
// every state is reached.
//
// Debugging note: only spend time debugging these tests once the equivalent
// synchronous test above is passing.  It's much easier to debug a
// deterministic test!
var _ = Describe("Async calculation graph state sequencing tests:", func() {
	//describeAsyncTests(baseTests)
})

func describeAsyncTests(baseTests []StateList) {
	for _, test := range baseTests {
		if len(test) == 0 {
			continue
		}
		baseTest := test

		for _, expander := range testExpanders {
			expanderDesc, expandedTests := expander(baseTest)
			for _, test := range expandedTests {
				test := test
				It("should handle: "+baseTest.String()+" "+expanderDesc, func() {
					// Create the calculation graph.
					conf := config.New()
					conf.FelixHostname = localHostname
					outputChan := make(chan interface{})
					asyncGraph := NewAsyncCalcGraph(conf, outputChan, nil)
					// And a validation filter, with a channel between it
					// and the async graph.
					validator := NewValidationFilter(asyncGraph)
					toValidator := NewSyncerCallbacksDecoupler()
					// Start the validator in one thread.
					go toValidator.SendTo(validator)
					// And the calc graph in another.
					asyncGraph.Start()
					// Channel to tell us when the input is done.
					done := make(chan bool, 2)
					// Start a thread to inject the KVs.
					go func() {
						log.Info("Input injector thread started")
						lastState := empty
						for _, state := range test {
							log.WithField("state", state).Info("Injecting next state")
							kvDeltas := state.KVDeltas(lastState)
							toValidator.OnUpdates(kvDeltas)
							lastState = state
						}
						toValidator.OnStatusUpdated(api.InSync)

						// Wait for the graph to flush.  We've seen this
						// take >1s on a heavily-loaded test server so we
						// give it a long timeout.
						time.Sleep(10 * time.Second)
						done <- true
					}()

					// Now drain the output from the output channel.
					mockDataplane := mock.NewMockDataplane()
					inSyncReceived := false
				readLoop:
					for {
						select {
						case <-done:
							log.Info("Got done message, stopping.")
							Expect(inSyncReceived).To(BeTrue(), "Timed out before we got an in-sync message")
							break readLoop
						case update := <-outputChan:
							log.WithField("update", update).Info("Update from channel")
							Expect(inSyncReceived).To(BeFalse(), "Unexpected update after in-sync")
							mockDataplane.OnEvent(update)
							if _, ok := update.(*proto.InSync); ok {
								// InSync should be the last message, to make sure, give
								// the graph another few ms before we stop.
								inSyncReceived = true
								go func() {
									time.Sleep(20 * time.Millisecond)
									done <- true
								}()
							}
						}
					}
					state := test[len(test)-1]

					// Async tests are slower to run so we do all the assertions
					// on each test rather than as separate It() blocks.
					Expect(mockDataplane.IPSets()).To(Equal(state.ExpectedIPSets),
						"IP sets didn't match expected state after moving to state: %v",
						state.Name)

					Expect(mockDataplane.ActivePolicies()).To(Equal(state.ExpectedPolicyIDs),
						"Active policy IDs were incorrect after moving to state: %v",
						state.Name)

					Expect(mockDataplane.ActiveProfiles()).To(Equal(state.ExpectedProfileIDs),
						"Active profile IDs were incorrect after moving to state: %v",
						state.Name)

					Expect(mockDataplane.EndpointToPolicyOrder()).To(Equal(state.ExpectedEndpointPolicyOrder),
						"Endpoint policy order incorrect after moving to state: %v",
						state.Name)

					Expect(mockDataplane.EndpointToPreDNATPolicyOrder()).To(Equal(state.ExpectedPreDNATEndpointPolicyOrder),
						"Endpoint pre-DNAT policy order incorrect after moving to state: %v",
						state.Name)

					Expect(mockDataplane.EndpointToUntrackedPolicyOrder()).To(Equal(state.ExpectedUntrackedEndpointPolicyOrder),
						"Endpoint untracked policy order incorrect after moving to state: %v",
						state.Name)
				})
			}
		}
	}
}

type flushStrategy int

const (
	afterEachKV flushStrategy = iota
	afterEachKVAndDupe
	afterEachState
	atEnd
)

func doStateSequenceTest(expandedTest StateList, flushStrategy flushStrategy) {
	var validationFilter *ValidationFilter
	var calcGraph *dispatcher.Dispatcher
	var mockDataplane *mock.MockDataplane
	var eventBuf *EventSequencer
	var lastState State
	var state State
	var sentInSync bool

	BeforeEach(func() {
		mockDataplane = mock.NewMockDataplane()
		eventBuf = NewEventSequencer(mockDataplane)
		eventBuf.Callback = mockDataplane.OnEvent
		calcGraph = NewCalculationGraph(eventBuf, localHostname)
		validationFilter = NewValidationFilter(calcGraph)
		sentInSync = false
		lastState = empty
		state = empty
	})

	// iterStates iterates through the states in turn,
	// executing the expectation function after each
	// state.
	iterStates := func(expectation func()) func() {
		return func() {
			var ii int
			for ii, state = range expandedTest {
				By(fmt.Sprintf("(%v) Moving from state %#v to %#v",
					ii, lastState.Name, state.Name))
				kvDeltas := state.KVDeltas(lastState)
				for _, kv := range kvDeltas {
					fmt.Fprintf(GinkgoWriter, "       -> Injecting KV: %#v = %#v\n",
						kv.Key, kv.Value)
					validationFilter.OnUpdates([]api.Update{kv})
					if flushStrategy == afterEachKV || flushStrategy == afterEachKVAndDupe {
						if !sentInSync {
							validationFilter.OnStatusUpdated(api.InSync)
							sentInSync = true
						}
						eventBuf.Flush()
					}
					if flushStrategy == afterEachKVAndDupe {
						validationFilter.OnUpdates([]api.Update{kv})
						eventBuf.Flush()
					}
				}
				fmt.Fprintln(GinkgoWriter, "       -- <<FLUSH>>")
				if flushStrategy == afterEachState {
					if !sentInSync {
						validationFilter.OnStatusUpdated(api.InSync)
						sentInSync = true
					}
					eventBuf.Flush()
				}
				if flushStrategy == afterEachState ||
					flushStrategy == afterEachKV ||
					flushStrategy == afterEachKVAndDupe {
					expectation()
				}
				lastState = state
			}
		}
	}

	// Note: these used to be separate It() blocks but combining them knocks ~10s off the
	// runtime, which is worthwhile!
	It("should result in correct active state", iterStates(func() {
		Expect(mockDataplane.IPSets()).To(Equal(state.ExpectedIPSets),
			"IP sets didn't match expected state after moving to state: %v",
			state.Name)
		Expect(mockDataplane.ActivePolicies()).To(Equal(state.ExpectedPolicyIDs),
			"Active policy IDs were incorrect after moving to state: %v",
			state.Name)
		Expect(mockDataplane.ActiveProfiles()).To(Equal(state.ExpectedProfileIDs),
			"Active profile IDs were incorrect after moving to state: %v",
			state.Name)
		Expect(mockDataplane.EndpointToPolicyOrder()).To(Equal(state.ExpectedEndpointPolicyOrder),
			"Endpoint policy order incorrect after moving to state: %v",
			state.Name)
		Expect(mockDataplane.EndpointToUntrackedPolicyOrder()).To(Equal(state.ExpectedUntrackedEndpointPolicyOrder),
			"Untracked endpoint policy order incorrect after moving to state: %v",
			state.Name)
		Expect(mockDataplane.EndpointToPreDNATPolicyOrder()).To(Equal(state.ExpectedPreDNATEndpointPolicyOrder),
			"Untracked endpoint policy order incorrect after moving to state: %v",
			state.Name)
		Expect(mockDataplane.ActiveUntrackedPolicies()).To(Equal(state.ExpectedUntrackedPolicyIDs),
			"Untracked policies incorrect after moving to state: %v",
			state.Name)
		Expect(mockDataplane.ActivePreDNATPolicies()).To(Equal(state.ExpectedPreDNATPolicyIDs),
			"PreDNAT policies incorrect after moving to state: %v",
			state.Name)
	}))
}

var _ = Describe("calc graph with health state", func() {

	It("should be constructable", func() {
		// Create the calculation graph.
		conf := config.New()
		conf.FelixHostname = localHostname
		outputChan := make(chan interface{})
		healthAggregator := health.NewHealthAggregator()
		asyncGraph := NewAsyncCalcGraph(conf, outputChan, healthAggregator)
		Expect(asyncGraph).NotTo(BeNil())
	})
})
