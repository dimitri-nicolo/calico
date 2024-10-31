// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/compliance/pkg/docindex"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const (
	// TODO(rlb): Presumably this will be parameterized or made a constant somewhere for the benchmarker code.
	// We should update to be 1.5 * benchmark interval and remove this constant.
	DayAndHalf = 36 * time.Hour
)

// addBenchmarks reads benchmarks from storage, filters them based on the report configuration and adds the data
// to the reports.
func (r *reporter) addBenchmarks() error {
	r.clog.Debug("Adding benchmark data to report")

	// Track summary stats for this report.
	r.data.CISBenchmarkSummary = apiv3.CISBenchmarkSummary{
		Type: string(v1.TypeKubernetes),
	}

	// Create a filter set from the report spec configuration.
	filters := r.newTestFilters()

	// Track node results along with the node names so that we can present them in sorted order.
	nodeResults := make(map[string]*apiv3.CISBenchmarkNode)
	nodeNames := []string{}

	// Determine the high/med thresholds that we use for our aggregated results status.
	highThreshold := 100
	medThreshold := 50
	if r.data.ReportSpec.CIS != nil {
		r.clog.Debug("Report has CIS configuration - parse threshold values")
		if r.data.ReportSpec.CIS.HighThreshold != nil {
			r.clog.Debugf("High threshold set to %d", *r.data.ReportSpec.CIS.HighThreshold)
			highThreshold = *r.data.ReportSpec.CIS.HighThreshold
		}
		if r.data.ReportSpec.CIS.MedThreshold != nil {
			r.clog.Debugf("Med threshold set to %d", *r.data.ReportSpec.CIS.MedThreshold)
			medThreshold = *r.data.ReportSpec.CIS.MedThreshold
		}
	}

	// We need to grab the latest benchmarks for each node. We always go back in time at least as far as 1.5x the
	// benchmark snapshot interval, or use the report start time if it is earlier. This means the results aren't
	// truly for the actual report interval, but to do that we'd need to track which nodes have been created and
	// deleted within the interval which is quite a bit more work for very little actual gain.
	start, end := r.cfg.ParsedReportStart, r.cfg.ParsedReportEnd
	if end.Sub(start) < DayAndHalf {
		r.clog.Debugf("Searching for benchmarks as far back as %v from the report end time", DayAndHalf)
		start = end.Add(-DayAndHalf)
	}
	r.clog.Infof("Query benchmarks from %v to %v", start, end)
	for b := range r.benchmarker.RetrieveLatestBenchmarks(
		r.ctx, v1.TypeKubernetes, nil, start, end,
	) {
		// If we received an error then log and exit.
		if b.Err != nil {
			r.clog.WithError(b.Err).Error("Error querying benchmarks from store")
			return b.Err
		}

		r.clog.WithFields(logrus.Fields{
			"Time":     b.Benchmarks.Timestamp,
			"Node":     b.Benchmarks.NodeName,
			"Type":     b.Benchmarks.Type,
			"NumTests": len(b.Benchmarks.Tests),
		}).Debugf("Processing set of benchmark results")

		// Locate the filter for these benchmarks
		filter := filters.getFilter(b.Benchmarks)

		// Benchmarks are returned for a given node, so create an entry for this node.
		// TODO(rlb): What about nodes that are failing to gather results (i.e. Err != nil).
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
				r.clog.Debugf("Filtering out test: %s", t.TestNumber)
				continue
			}
			r.clog.Debugf("Including test: %s", t.TestNumber)

			// If this is a new section, initialize the maps for this section. We will sort the sections by section ID
			// once we have the full set of results and sections for this set of benchmarks.
			if _, ok := sectionResults[t.Section]; !ok {
				// TODO(rlb): What is the section status? Not sure it should be there since it doesn't make sense to use
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
		r.clog.Debugf("Sorting %d sections", len(sectionIds))
		sort.Sort(sectionIds)

		// Append the section to the node data in section ID order.
		for _, sid := range sectionIds {
			r.clog.Debugf("Handling sections %s", sid)

			// Add the tests to the section, and increment the section stats based on the test result.
			section := sectionResults[sid.Index()]
			results := resultsBySection[sid.Index()]
			resultIds := resultIdsBySection[sid.Index()]

			// Sort the result IDs.
			r.clog.Debugf("Sorting %d results", len(resultIds))
			sort.Sort(resultIds)

			// Add the results in result ID order and increment section stats.
			sectionTotalTests := 0
			for _, rid := range resultIds {
				r.clog.Debugf("Handling result %s", rid)
				result := results[rid.Index()]
				// TODO(rlb): Status values should be declared in libcalico-go?
				switch result.Status {
				case "PASS":
					section.Pass++
					sectionTotalTests++
				case "FAIL":
					section.Fail++
					sectionTotalTests++
				case "INFO", "WARN":
					section.Info++
				}
				section.Results = append(section.Results, *result)
			}

			// Update the section status based on section results and thresholds.
			var p int
			if sectionTotalTests > 0 {
				p = 100 * section.Pass / sectionTotalTests
			}
			switch {
			case p >= highThreshold:
				// TODO(rlb): Status values should be declared in libcalico-go?
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
			node.Summary.Total += sectionTotalTests

			// Add the section to the node data.
			node.Results = append(node.Results, *section)
		}

		// Update the node status and benchmark summaries based on the node stats and thresholds.
		var p int
		if node.Summary.Total > 0 {
			p = 100 * node.Summary.TotalPass / node.Summary.Total
		}
		switch {
		case p >= highThreshold:
			// TODO(rlb): Status values should be declared in libcalico-go?
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

// newTestFilter creates a new testFilter from the supplied Report configuration.
func (r *reporter) newTestFilters() *testFilters {
	rs := r.cfg.Report.Spec

	var includeUnscored bool
	if rs.CIS != nil {
		includeUnscored = rs.CIS.IncludeUnscoredTests
	}
	fs := &testFilters{
		// Default filter is just an include-all filter.
		def: &testFilter{
			includeUnscored: includeUnscored,
			clog:            r.clog,
		},
	}

	// Create the filters from the report configuration.
	if rs.CIS != nil {
		for _, rf := range rs.CIS.ResultsFilters {
			f := &testFilter{
				includeUnscored: includeUnscored,
				clog:            r.clog,
			}
			if rf.BenchmarkSelection != nil {
				if rf.BenchmarkSelection.KubernetesVersion != "" {
					// A Kubernetes version is specified. Use the docindex to handle the version matches since they
					// use the same formats.
					f.k8sVersion = docindex.New(rf.BenchmarkSelection.KubernetesVersion)
				}
			}
			if len(rf.Include) != 0 || len(rf.Exclude) != 0 {
				f.includes = make(map[string]bool)

				for _, e := range rf.Exclude {
					f.includes[e] = false
				}
				for _, e := range rf.Include {
					f.includes[e] = true
				}

				// Keep track of whether only excludes have been specified.
				f.excludesOnly = len(rf.Include) == 0
			}
			fs.filters = append(fs.filters, f)
		}
	}

	return fs
}

// testFilters is the "compiled" set of filters for the report.
type testFilters struct {
	// All of the filters for this report.
	filters []*testFilter

	// Default filter
	def *testFilter
}

// getFilter returns the filter to use on a specific set of benchmarks.
func (t *testFilters) getFilter(b *v1.Benchmarks) *testFilter {
	// We use the first matching filter, so loop through to find the match.
	for _, f := range t.filters {
		if f.matches(b) {
			return f
		}
	}
	return t.def
}

// testFilter is used to filter in or out benchmark tests from the report.
type testFilter struct {
	// Logger
	clog *logrus.Entry

	// --- Filter match criteria ---
	k8sVersion docindex.DocIndex

	// --- Filter parameters ---

	// Whether the filter contains excludes only.
	excludesOnly bool

	// The set of includes and excludes compressed into a single map where (true indicates includes, false indicates
	// exclude).
	includes map[string]bool

	// Whether the report should include unscored tests.
	includeUnscored bool
}

// include returns whether or not a benchmark test should be included in the report.
func (f *testFilter) include(t v1.BenchmarkTest) bool {
	if f.includeUnscored && f.includes == nil {
		// Short circuit including everything.
		f.clog.Debugf("Include %s: including everything", t.TestNumber)
		return true
	}
	if !t.Scored && !f.includeUnscored {
		// Not scored, and we are not including those.
		f.clog.Debugf("Exclude %s: excluding unscored", t.TestNumber)
		return false
	}
	if f.includes == nil {
		// No includes/excludes, so just include it.
		f.clog.Debugf("Include %s: including all scored", t.TestNumber)
		return true
	}

	if inc, ok := f.includes[t.TestNumber]; ok {
		// Test is explicitly specified, use that value.
		f.clog.Debugf("Include %s?: %v", t.TestNumber, inc)
		return inc
	}

	if inc, ok := f.includes[t.Section]; ok {
		// Specified at the section level, use that value.
		f.clog.Debugf("Include %s?: %v", t.Section, inc)
		return inc
	}

	// Otherwise, we include if the filter only contains exclusions, or exclude if the filter contains
	// inclusions which we didn't match.
	return f.excludesOnly
}

// matches returns whether or not the match critera match the supplied benchmarks.
func (f *testFilter) matches(b *v1.Benchmarks) bool {
	if f.k8sVersion != nil {
		bv := docindex.New(b.KubernetesVersion)
		if !f.k8sVersion.Contains(bv) {
			return false
		}
	}
	return true
}
