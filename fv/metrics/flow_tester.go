// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package metrics

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/fv/infrastructure"
)

type Aggregation int

const (
	None Aggregation = iota
	BySourcePort
	ByPodPrefix
)

// The expected policies for the flow.
type ExpectedPolicy struct {
	Reporter string
	Action   string
	Policies []string
}

// FlowTester is a helper utility to parse and check flows.
type FlowTester struct {
	destPort       int
	destPortStr    string
	expectLabels   bool
	expectPolicies bool
	felixes        []*infrastructure.Felix
	flowsStarted   []map[collector.FlowMeta]int
	flowsCompleted []map[collector.FlowMeta]int
	packets        []map[collector.FlowMeta]int
	policies       []map[collector.FlowMeta][]string
}

// NewFlowTester creates a new FlowTester initialized for the supplied felix instances.
func NewFlowTester(felixes []*infrastructure.Felix, expectLabels, expectPolicies bool, destPort int) *FlowTester {
	return &FlowTester{
		destPort:       destPort,
		destPortStr:    fmt.Sprint(destPort),
		expectLabels:   expectLabels,
		expectPolicies: expectPolicies,
		felixes:        felixes,
		flowsStarted:   make([]map[collector.FlowMeta]int, len(felixes)),
		flowsCompleted: make([]map[collector.FlowMeta]int, len(felixes)),
		packets:        make([]map[collector.FlowMeta]int, len(felixes)),
		policies:       make([]map[collector.FlowMeta][]string, len(felixes)),
	}
}

// PopulateFromFlowLogs initializes the flow tester from the flow logs.
func (t *FlowTester) PopulateFromFlowLogs(flowLogsOutput string) error {
	for ii, f := range t.felixes {
		t.flowsStarted[ii] = make(map[collector.FlowMeta]int)
		t.flowsCompleted[ii] = make(map[collector.FlowMeta]int)
		t.packets[ii] = make(map[collector.FlowMeta]int)
		t.policies[ii] = make(map[collector.FlowMeta][]string)

		cwlogs, err := f.ReadFlowLogs(flowLogsOutput)
		if err != nil {
			return err
		}

		for _, fl := range cwlogs {
			if fl.Tuple.GetDestPort() != t.destPort {
				continue
			}

			// If endpoint Labels are expected, and
			// aggregation permits this, check that they are
			// there.
			labelsExpected := t.expectLabels
			if labelsExpected {
				if fl.FlowLabels.SrcLabels == nil {
					return errors.New(fmt.Sprintf("Missing src Labels in %v: Meta %v", fl.FlowLabels, fl.FlowMeta))
				}
				if fl.FlowLabels.DstLabels == nil {
					return errors.New(fmt.Sprintf("Missing dst Labels in %v", fl.FlowLabels))
				}
			} else {
				if fl.FlowLabels.SrcLabels != nil {
					return errors.New(fmt.Sprintf("Unexpected src Labels in %v", fl.FlowLabels))
				}
				if fl.FlowLabels.DstLabels != nil {
					return errors.New(fmt.Sprintf("Unexpected dst Labels in %v", fl.FlowLabels))
				}
			}

			// Now discard Labels so that our expectation code
			// below doesn't ever have to specify them.
			fl.FlowLabels.SrcLabels = nil
			fl.FlowLabels.DstLabels = nil

			if t.expectPolicies {
				if len(fl.FlowPolicies) == 0 {
					return errors.New(fmt.Sprintf("Missing Policies in %v", fl.FlowMeta))
				}
				pols := []string{}
				for p := range fl.FlowPolicies {
					pols = append(pols, p)
				}
				t.policies[ii][fl.FlowMeta] = pols
			} else if len(fl.FlowPolicies) != 0 {
				return errors.New(fmt.Sprintf("Unexpected Policies %v in %v", fl.FlowPolicies, fl.FlowMeta))
			}

			// Accumulate flow and packet counts for this FlowMeta.
			if _, ok := t.flowsStarted[ii][fl.FlowMeta]; !ok {
				t.flowsStarted[ii][fl.FlowMeta] = 0
			}
			t.flowsStarted[ii][fl.FlowMeta] += fl.NumFlowsStarted

			if _, ok := t.flowsCompleted[ii][fl.FlowMeta]; !ok {
				t.flowsCompleted[ii][fl.FlowMeta] = 0
			}
			t.flowsCompleted[ii][fl.FlowMeta] += fl.NumFlowsCompleted

			if _, ok := t.packets[ii][fl.FlowMeta]; !ok {
				t.packets[ii][fl.FlowMeta] = 0
			}
			t.packets[ii][fl.FlowMeta] += fl.PacketsIn + fl.PacketsOut
		}
		for meta, count := range t.flowsStarted[ii] {
			log.Infof("started: %d %v", count, meta)
		}
		for meta, count := range t.flowsCompleted[ii] {
			log.Infof("completed: %d %v", count, meta)
		}

		for meta, pols := range t.policies[ii] {
			log.Infof("Policies: %v %v", pols, meta)
		}

		// For each distinct FlowMeta, the counts of flows started
		// and completed should be the same.
		for meta, count := range t.flowsCompleted[ii] {
			if count != t.flowsStarted[ii][meta] {
				return errors.New(fmt.Sprintf("Wrong started count (%d != %d) for %v",
					t.flowsStarted[ii][meta], count, meta))
			}
		}

		// Check that we have non-zero packets for each flow.
		for meta, count := range t.packets[ii] {
			if count == 0 {
				return errors.New(fmt.Sprintf("No packets for %v", meta))
			}
		}
	}

	return nil
}

// CheckFlow flow logs with the given src/dst metadata and IPs.
// Specifically there should be numMatchingMetas distinct
// FlowMetas that match those, and numFlowsPerMeta flows for each
// distinct FlowMeta.  actions indicates the expected handling on
// each host: "allow" or "deny"; or "" if the flow isn't
// explicitly allowed or denied on that host (which means that
// there won't be a flow log).
func (t *FlowTester) CheckFlow(srcMeta, srcIP, dstMeta, dstIP string, numMatchingMetas, numFlowsPerMeta int, actionsPolicies []ExpectedPolicy) error {

	var errs []string

	// Host loop.
	for ii, handling := range actionsPolicies {
		// Skip if the handling for this host is "".
		if handling.Action == "" && handling.Reporter == "" {
			continue
		}
		reporter := handling.Reporter
		action := handling.Action
		expectedPolicies := []string{}
		expectedPoliciesStr := "-"
		if t.expectPolicies {
			expectedPolicies = handling.Policies
			expectedPoliciesStr = "[" + strings.Join(expectedPolicies, ",") + "]"
		}

		// Build a FlowMeta with the metadata and IPs that we are looking for.
		var template string
		if dstIP != "" {
			template = "1 2 " + srcMeta + " - " + dstMeta + " - " + srcIP + " " + dstIP + " 6 0 " + t.destPortStr + " 1 1 0 " + reporter + " 4 6 260 364 " + action + " " + expectedPoliciesStr + " - 0"
		} else {
			template = "1 2 " + srcMeta + " - " + dstMeta + " - - - 6 0 " + t.destPortStr + " 1 1 0 " + reporter + " 4 6 260 364 " + action + " " + expectedPoliciesStr + " - 0"
		}
		fl := &collector.FlowLog{}
		err := fl.Deserialize(template)
		Expect(err).ToNot(HaveOccurred())
		log.WithField("template", template).WithField("meta", fl.FlowMeta).Info("Looking for")
		if t.expectPolicies {
			for meta, actualPolicies := range t.policies[ii] {
				fl.FlowMeta.Tuple.SetSourcePort(meta.Tuple.GetSourcePort())
				if meta != fl.FlowMeta {
					continue
				}

				// Sort the policies - they should be identical.
				sort.Strings(expectedPolicies)
				sort.Strings(actualPolicies)

				if !reflect.DeepEqual(expectedPolicies, actualPolicies) {
					errs = append(errs, fmt.Sprintf("Expected Policies %v to be present in %v", expectedPolicies, actualPolicies))
				}

				// Record that we've ticked off this flow.
				t.policies[ii][meta] = []string{}
			}
			fl.FlowMeta.Tuple.SetSourcePort(0)
		}

		matchingMetas := 0
		for meta, count := range t.flowsCompleted[ii] {
			fl.FlowMeta.Tuple.SetSourcePort(meta.Tuple.GetSourcePort())
			if meta == fl.FlowMeta {
				// This flow log matches what
				// we're looking for.
				if count != numFlowsPerMeta {
					errs = append(errs, fmt.Sprintf("Wrong flow count (%d != %d) for %v", count, numFlowsPerMeta, meta))
				}
				matchingMetas += 1
				// Record that we've ticked off this flow.
				t.flowsCompleted[ii][meta] = 0
			}
		}
		fl.FlowMeta.Tuple.SetSourcePort(0)
		if matchingMetas != numMatchingMetas {
			errs = append(errs, fmt.Sprintf("Wrong log count (%d != %d) for %v", matchingMetas, numMatchingMetas, fl.FlowMeta))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n==============\n"))
}

func (t *FlowTester) CheckAllFlowsAccountedFor() error {
	// Finally check that there are no remaining flow logs that we did not expect.
	for ii := range t.felixes {
		for meta, count := range t.flowsCompleted[ii] {
			if count != 0 {
				return errors.New(fmt.Sprintf("Unexpected flow logs (%d) for %v",
					count, meta))
			}
		}
	}
	return nil
}
