package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/tigera/intrusion-detection/controller/pkg/db"

	"github.com/olivere/elastic"

	. "github.com/onsi/gomega"
)

func TestElasticFlowLogIterator(t *testing.T) {
	g := NewGomegaWithT(t)

	input := [][]db.FlowLog{
		{
			{
				SourceIP:   "1.2.3.4",
				SourceName: "source",
				DestIP:     "2.3.4.5",
				DestName:   "dest",
				StartTime:  1,
				EndTime:    2,
			},
			{
				SourceIP:   "5.6.7.8",
				SourceName: "source",
				DestIP:     "2.3.4.5",
				DestName:   "dest",
				StartTime:  2,
				EndTime:    3,
			},
		},
		{
			{
				SourceIP:   "9.10.11.12",
				SourceName: "source",
				DestIP:     "2.3.4.5",
				DestName:   "dest",
				StartTime:  3,
				EndTime:    4,
			},
		},
	}
	var expected []db.FlowLog
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
		scroll: scroll,
		ctx:    context.TODO(),
	}

	var actual []db.FlowLog
	for i.Next() {
		actual = append(actual, i.Value())
	}
	g.Expect(i.Err()).ShouldNot(HaveOccurred())

	g.Expect(actual).Should(Equal(expected), "Events are retrieved in order")
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
		scroll: scroll,
		ctx:    context.TODO(),
	}

	g.Expect(i.Next()).Should(BeFalse(), "Iterator stops immediately")
	g.Expect(i.Err()).Should(HaveOccurred())
}

type mockScrollerError struct{}

func (m *mockScrollerError) Do(context.Context) (*elastic.SearchResult, error) {
	return nil, errors.New("fail")
}
