// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"sort"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/benchmark"
	"github.com/tigera/compliance/pkg/docindex"
)

// addBenchmarks reads benchmarks from storage, filters them based on the report configuration and adds the data
// to the reports.
func (r *reporter) addBenchmarks() error {
	// Track summary stats for this report.
	r.data.CISBenchmarkSummary = apiv3.CISBenchmarkSummary{
		Type: string(benchmark.TypeKubernetes),
	}

	// Create a filter from the report spec configuration.
	filter := newTestFilter(r.data.ReportSpec)

	// Track node results along with the node names so that we can present them in sorted order.
	nodeResults := make(map[string]*apiv3.CISBenchmarkNode)
	nodeNames := []string{}

	// Determine the high/med thresholds that we use for our aggregated results status.
	highThreshold := 100
	medThreshold := 50
	if r.data.ReportSpec.CIS != nil {
		if r.data.ReportSpec.CIS.HighThreshold != nil {
			highThreshold = *r.data.ReportSpec.CIS.HighThreshold
		}
		if r.data.ReportSpec.CIS.MedThreshold != nil {
			medThreshold = *r.data.ReportSpec.CIS.MedThreshold
		}
	}

	// Grab the latest benchmarks for each node.
	for b := range r.benchmarker.RetrieveLatestBenchmarks(
		r.ctx, benchmark.TypeKubernetes, nil, r.cfg.ParsedReportStart, r.cfg.ParsedReportEnd,
	) {
		// If we received an error then log and exit.
		if b.Err != nil {
			r.clog.WithError(b.Err).Error("Error querying benchmarks from store")
			return b.Err
		}

		// Benchmarks are returned for a given node, so create an entry for this node.
		//TODO(rlb): What about nodes that are failing to gather results (i.e. Err != nil).
		node := &apiv3.CISBenchmarkNode{
			NodeName: b.Benchmarks.NodeName,
		}

		// Collate sections and tests. Gather section IDs and result IDs so that we can present them in sorted order.
		sectionResults := make(map[string]*apiv3.CISBenchmarkSectionResult)
		sectionIds := docindex.SortableDocIndexes{}
		resultsBySection := make(map[string]map[string]*apiv3.CISBenchmarkResult)
		resultIdsBySection := make(map[string]docindex.SortableDocIndexes)

		// Iterate through the tests filtering in or out as per the report configuration.
		// TODO(rlb) How do we decide which CIS benchmark types to include:
		// -  ReportType has only an on/off for CIS
		// -  If we always include everything, how to do we filter includes/excludes based on CIS benchmark type
		for _, t := range b.Benchmarks.Tests {
			if !filter.include(t) {
				continue
			}

			// If this is a new section, initialize the maps for this section. We will sort the sections by section ID
			// once we have the full set of results and sections for this set of benchmarks.
			if _, ok := sectionResults[t.Section]; !ok {
				//TODO(rlb): What is the section status? Not sure it should be there since it doesn't make sense to use
				// the same percentages as the per node stats.
				sectionResults[t.Section] = &apiv3.CISBenchmarkSectionResult{
					Section: t.Section,
					Desc:    t.SectionDesc,
				}
				sectionIds = append(sectionIds, docindex.New(t.Section))
				resultsBySection[t.Section] = make(map[string]*apiv3.CISBenchmarkResult)
			}

			// Sanity check that we don't have multiple of the same test number. Best not to fail, but we should
			// at least error log.
			if _, ok := resultsBySection[t.Section][t.TestNumber]; ok {
				r.clog.Errorf("Duplicate test found: test %s in section %s", t.TestNumber, t.Section)
				continue
			}

			// Store the result and result ID ready for sorting by test number.
			resultsBySection[t.Section][t.TestNumber] = &apiv3.CISBenchmarkResult{
				TestNumber: t.TestNumber,
				TestDesc:   t.TestDesc,
				TestInfo:   t.TestInfo,
				Status:     t.Status,
				Scored:     t.Scored,
			}
			resultIdsBySection[t.Section] = append(resultIdsBySection[t.Section], docindex.New(t.TestNumber))
		}

		// Sort the section IDs for this node.
		sort.Sort(sectionIds)

		// Append the section to the node data in section ID order.
		for _, sid := range sectionIds {
			// Add the tests to the section, and increment the section stats based on the test result.
			section := sectionResults[sid.Index()]
			results := resultsBySection[sid.Index()]
			resultIds := resultIdsBySection[sid.Index()]

			// Sort the result IDs.
			sort.Sort(resultIds)

			// Add the results in result ID order and increment section stats.
			for _, rid := range resultIds {
				result := results[rid.Index()]
				//TODO(rlb): Status values should be declared in libcalico-go?
				switch result.Status {
				case "PASS":
					section.Pass++
				case "FAIL":
					section.Fail++
				case "INFO":
					section.Info++
				}
				section.Results = append(section.Results, *result)
			}

			// Update the section status based on section results and thresholds.
			var p int
			if len(section.Results) > 0 {
				p = 100 * section.Fail / len(section.Results)
			}
			switch {
			case p >= highThreshold:
				//TODO(rlb): Status values should be declared in libcalico-go?
				section.Status = "HIGH"
			case p < highThreshold && p >= medThreshold:
				section.Status = "MED"
			default:
				section.Status = "LOW"
			}

			// Update the node stats from the section.
			node.Summary.TotalPass += section.Pass
			node.Summary.TotalFail += section.Fail
			node.Summary.TotalInfo += section.Info
			node.Summary.Total += len(section.Results)

			// Add the section to the node data.
			node.Results = append(node.Results, *section)
		}

		// Update the node status and benchmark summaries based on the node stats and thresholds.
		var p int
		if node.Summary.Total > 0 {
			p = 100 * node.Summary.TotalFail / node.Summary.Total
		}
		switch {
		case p >= highThreshold || b.Benchmarks.Error != "":
			// Either we have tipped the high threshold, or the benchmark did not run. In either case flag as
			// high.
			//TODO(rlb): Status values should be declared in libcalico-go?
			node.Summary.Status = "HIGH"
			r.data.CISBenchmarkSummary.HighCount++
		case p < highThreshold && p >= medThreshold:
			node.Summary.Status = "MED"
			r.data.CISBenchmarkSummary.MedCount++
		default:
			node.Summary.Status = "LOW"
			r.data.CISBenchmarkSummary.LowCount++
		}

		// Store the node ready for sorting by node name.
		nodeResults[node.NodeName] = node
		nodeNames = append(nodeNames, node.NodeName)
	}

	// Sort the node names
	sort.Strings(nodeNames)

	// Add the nodes to the report
	for _, nodeName := range nodeNames {
		r.data.CISBenchmark = append(r.data.CISBenchmark, *nodeResults[nodeName])
	}

	return nil
}

// testFilter is used to filter in or out benchmark tests from the report.
type testFilter struct {
	// Whether the filter contains excludes only.
	excludesOnly bool

	// The set of includes and excludes compressed into a single map where (true indicates includes, false indicates
	// exclude).
	includes map[string]bool

	// Whether the report should include unscored tests.
	includeUnscored bool
}

// newTestFilter creates a new testFilter from the supplied Report configuration.
func newTestFilter(r apiv3.ReportSpec) *testFilter {
	f := &testFilter{}

	if r.CIS != nil {
		f.includeUnscored = r.CIS.IncludeUnscoredTests

		if len(r.CIS.Include) != 0 || len(r.CIS.Exclude) != 0 {
			f.includes = make(map[string]bool)

			for _, e := range r.CIS.Exclude {
				f.includes[e] = false
			}
			for _, e := range r.CIS.Include {
				f.includes[e] = true
			}

			// Keep track of whether only excludes have been specified.
			f.excludesOnly = len(r.CIS.Include) == 0
		}
	}

	return f
}

// include returns whether or not a benchmark test should be included in the report.
func (f *testFilter) include(t benchmark.Test) bool {
	if f.includeUnscored && f.includes == nil {
		// Short circuit including everything.
		return true
	}
	if !t.Scored && !f.includeUnscored {
		// Not scored, and we are not including those.
		return false
	}
	if len(f.includes) == 0 {
		// No includes/excludes, so just include it.
		return true
	}

	if inc, ok := f.includes[t.TestNumber]; ok {
		// Test is explicitly specified, use that value.
		return inc
	}

	if inc, ok := f.includes[t.Section]; ok {
		// Specified at the section level, use that value.
		return inc
	}

	// Otherwise, we include if the filter only contains exclusions, or exclude if the filter contains
	// inclusions which we didn't match.
	return f.excludesOnly
}
