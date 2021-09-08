// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/tigera/intrusion-detection/controller/pkg/db"

	"github.com/olivere/elastic/v7"
	. "github.com/onsi/gomega"
)

func TestElasticFlowLogIterator(t *testing.T) {
	g := NewGomegaWithT(t)

	input := [][]elastic.SearchHit{
		{
			{
				Index: "idx1",
				Id:    "id1",
			},
			{
				Index: "idx2",
				Id:    "id2",
			},
		},
		{
			{
				Index: "idx3",
				Id:    "id3",
			},
		},
	}
	var expected []elastic.SearchHit
	var results []*elastic.SearchResult
	for _, hits := range input {
		r := &elastic.SearchResult{
			Hits: &elastic.SearchHits{},
		}

		for i, hit := range hits {
			expected = append(expected, hit)
			// Take the address of hits[i]; the address of hit is constant since
			// hit is a local variable
			r.Hits.Hits = append(r.Hits.Hits, &hits[i])
		}

		results = append(results, r)
	}

	scroll := &mockScroller{
		results: results,
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	expectedKey := db.QueryKeyFlowLogSourceIP
	i := newQueryIterator(
		ctx,
		[]scrollerEntry{{expectedKey, scroll, nil}},
		"test")

	var actualHits []*elastic.SearchHit
	var actualKeys []db.QueryKey
	for i.Next() {
		k, h := i.Value()
		actualKeys = append(actualKeys, k)
		actualHits = append(actualHits, h)
	}
	g.Expect(i.Err()).ShouldNot(HaveOccurred())

	g.Expect(actualHits).Should(HaveLen(len(expected)), "All events are retrieved.")
	for idx := range actualHits {
		g.Expect(*actualHits[idx]).Should(Equal(expected[idx]), "Events are retrieved in order.")
		g.Expect(actualKeys[idx]).Should(Equal(expectedKey))
	}

	g.Expect(scroll.clearCalled).Should(BeTrue())
}

type mockScroller struct {
	results     []*elastic.SearchResult
	clearCalled bool
}

func (m *mockScroller) Do(context.Context) (*elastic.SearchResult, error) {
	if len(m.results) == 0 {
		return nil, io.EOF
	}

	result := m.results[0]
	m.results = m.results[1:]
	return result, nil
}

func (m *mockScroller) Clear(context.Context) error {
	m.clearCalled = true
	return nil
}

func TestElasticFlowLogIteratorWithError(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	scroll := &mockScrollerError{}
	i := queryIterator{
		scrollers: []scrollerEntry{{db.QueryKeyFlowLogDestIP, scroll, nil}},
		ctx:       ctx,
	}

	g.Expect(i.Next()).Should(BeFalse(), "Iterator stops immediately")
	g.Expect(i.Err()).Should(HaveOccurred())
}

func TestElasticFlowLogIteratorWithTwoScrollers(t *testing.T) {
	g := NewGomegaWithT(t)

	sourceHit := elastic.SearchHit{
		Index: "source",
		Id:    "source",
	}
	destHit := elastic.SearchHit{
		Index: "dest",
		Id:    "dest",
	}

	scrollers := []scrollerEntry{
		{db.QueryKeyFlowLogSourceIP, &mockScroller{
			results: []*elastic.SearchResult{
				{
					Hits: &elastic.SearchHits{
						Hits: []*elastic.SearchHit{&sourceHit},
					},
				},
			},
		}, nil},
		{db.QueryKeyFlowLogDestIP, &mockScroller{
			results: []*elastic.SearchResult{
				{
					Hits: &elastic.SearchHits{
						Hits: []*elastic.SearchHit{&destHit},
					},
				},
			},
		}, nil},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	i := newQueryIterator(ctx, scrollers, "mock")

	var results []*elastic.SearchHit
	for i.Next() {
		_, h := i.Value()
		results = append(results, h)
	}
	g.Expect(i.Err()).ShouldNot(HaveOccurred(), "No errors from the iterator")
	g.Expect(results).Should(HaveLen(2), "Should have gotten back two results")
	g.Expect(*results[0]).ShouldNot(Equal(*results[1]), "Both have different source IPs")

	// Order is random. Swap them to make the tests simpler.
	if results[0].Index == destHit.Index {
		results[1], results[0] = results[0], results[1]
	}

	g.Expect(results[0].Index).Should(Equal(sourceHit.Index))
	g.Expect(results[1].Index).Should(Equal(destHit.Index))

	for _, scroller := range scrollers {
		g.Expect(scroller.scroller.(*mockScroller).clearCalled).Should(BeTrue())
	}
}

type mockScrollerError struct{}

func (m *mockScrollerError) Do(context.Context) (*elastic.SearchResult, error) {
	return nil, errors.New("fail")
}

func (m *mockScrollerError) Clear(context.Context) error {
	return nil
}
