// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/olivere/elastic"
	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

func TestElasticFlowLogIterator(t *testing.T) {
	g := NewGomegaWithT(t)

	input := [][]events.FlowLogJSONOutput{
		{
			{
				SourceIP:   util.Sptr("1.2.3.4"),
				SourceName: "source",
				DestIP:     util.Sptr("2.3.4.5"),
				DestName:   "dest",
			},
			{
				SourceIP:   util.Sptr("5.6.7.8"),
				SourceName: "source",
				DestIP:     util.Sptr("2.3.4.5"),
				DestName:   "dest",
			},
		},
		{
			{
				SourceIP:   util.Sptr("9.10.11.12"),
				SourceName: "source",
				DestIP:     util.Sptr("2.3.4.5"),
				DestName:   "dest",
			},
		},
	}
	var expected []events.FlowLogJSONOutput
	var results []*elastic.SearchResult
	for _, logs := range input {
		r := &elastic.SearchResult{
			Hits: &elastic.SearchHits{},
		}

		for _, flowLog := range logs {
			expected = append(expected, flowLog)

			b, err := json.Marshal(&flowLog)
			g.Expect(err).ShouldNot(HaveOccurred())

			raw := json.RawMessage(b)
			hit := elastic.SearchHit{Source: &raw}
			r.Hits.Hits = append(r.Hits.Hits, &hit)
		}

		results = append(results, r)
	}
	junk := &elastic.SearchResult{
		Hits: &elastic.SearchHits{
			Hits: []*elastic.SearchHit{
				{
					Source: &json.RawMessage{byte('{')},
				},
			},
		},
	}
	results = append(results, junk)

	scroll := &mockScroller{
		results: results,
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	expectedKey := "source_ip"
	i := flowLogIterator{
		scrollers: []scrollerEntry{{expectedKey, scroll, nil}},
		ctx:       ctx,
	}

	var actual []events.SuspiciousIPSecurityEvent
	for i.Next() {
		actual = append(actual, i.Value().(events.SuspiciousIPSecurityEvent))
	}
	g.Expect(i.Err()).ShouldNot(HaveOccurred())

	g.Expect(actual).Should(HaveLen(len(expected)), "All events are retrieved.")
	for idx := range actual {
		g.Expect(actual[idx].SourceIP).Should(Equal(expected[idx].SourceIP), "Events are retrieved in order.")
	}
}

type mockScroller struct {
	results []*elastic.SearchResult
}

func (m *mockScroller) Do(context.Context) (*elastic.SearchResult, error) {
	if len(m.results) == 0 {
		return nil, io.EOF
	}

	result := m.results[0]
	m.results = m.results[1:]
	return result, nil
}

func TestElasticFlowLogIteratorWithError(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	scroll := &mockScrollerError{}
	i := flowLogIterator{
		scrollers: []scrollerEntry{{"dest_ip", scroll, nil}},
		ctx:       ctx,
	}

	g.Expect(i.Next()).Should(BeFalse(), "Iterator stops immediately")
	g.Expect(i.Err()).Should(HaveOccurred())
}

func TestElasticFlowLogIteratorWithTwoScrollers(t *testing.T) {
	g := NewGomegaWithT(t)

	source_log := events.FlowLogJSONOutput{
		SourceType: "wep",
		SourceIP:   util.Sptr("1.2.3.4"),
		SourceName: "source",
		DestType:   "hep",
		DestIP:     util.Sptr("2.3.4.5"),
		DestName:   "dest",
	}
	b, err := json.Marshal(&source_log)
	g.Expect(err).ShouldNot(HaveOccurred())
	source_msg := json.RawMessage(b)

	dest_log := events.FlowLogJSONOutput{
		SourceType: "net",
		SourceIP:   util.Sptr("3.4.5.6"),
		SourceName: "source",
		DestType:   "ns",
		DestIP:     util.Sptr("4.5.6.7"),
		DestName:   "dest",
	}
	b, err = json.Marshal(&dest_log)
	g.Expect(err).ShouldNot(HaveOccurred())
	dest_msg := json.RawMessage(b)

	scrollers := []scrollerEntry{
		{"source_ip", &mockScroller{
			[]*elastic.SearchResult{
				{
					Hits: &elastic.SearchHits{
						Hits: []*elastic.SearchHit{
							{
								Source: &source_msg,
							},
						},
					},
				},
			},
		}, nil},
		{"dest_ip", &mockScroller{
			[]*elastic.SearchResult{
				{
					Hits: &elastic.SearchHits{
						Hits: []*elastic.SearchHit{
							{
								Source: &dest_msg,
							},
						},
					},
				},
			},
		}, nil},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	i := flowLogIterator{
		scrollers: scrollers,
		ctx:       ctx,
		name:      "mock",
	}

	var results []events.SuspiciousIPSecurityEvent
	for i.Next() {
		results = append(results, i.Value().(events.SuspiciousIPSecurityEvent))
	}
	g.Expect(i.Err()).ShouldNot(HaveOccurred(), "No errors from the iterator")
	g.Expect(results).Should(HaveLen(2), "Should have gotten back two results")
	g.Expect(results[0].SourceIP).ShouldNot(Equal(results[1].SourceIP), "Both have different source IPs")

	// Order is random. Swap them to make the tests simpler.
	if *results[0].SourceIP == *dest_log.SourceIP {
		results[1], results[0] = results[0], results[1]
	}

	g.Expect(results[0].Description).Should(Equal("suspicious IP 1.2.3.4 from list mock connected to hep /dest"))
	g.Expect(results[1].Description).Should(Equal("net /source connected to suspicious IP 4.5.6.7 from list mock"))
}

type mockScrollerError struct{}

func (m *mockScrollerError) Do(context.Context) (*elastic.SearchResult, error) {
	return nil, errors.New("fail")
}
