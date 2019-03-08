package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/olivere/elastic"
	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/events"
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

	i := elasticFlowLogIterator{
		scrollers: []Scroller{scroll},
		ctx:       context.TODO(),
	}

	var actual []events.SecurityEvent
	for i.Next() {
		actual = append(actual, i.Value())
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

	scroll := &mockScrollerError{}
	i := elasticFlowLogIterator{
		scrollers: []Scroller{scroll},
		ctx:       context.TODO(),
	}

	g.Expect(i.Next()).Should(BeFalse(), "Iterator stops immediately")
	g.Expect(i.Err()).Should(HaveOccurred())
}

type mockScrollerError struct{}

func (m *mockScrollerError) Do(context.Context) (*elastic.SearchResult, error) {
	return nil, errors.New("fail")
}
