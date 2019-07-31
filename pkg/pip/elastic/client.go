package elastic

import (
	"context"
	"errors"
	"github.com/olivere/elastic"

	celastic "github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
)

type Client interface {
	// The PIP ES client extends the compliance client.
	list.Destination
	event.Fetcher

	// CompositeAggregation enumeration.
	// TODO(rlb): This is totes generic and should not live here. Move to libcalico-go.
	SearchCompositeAggregations(
		context.Context, *CompositeAggregationQuery, CompositeAggregationKey,
	) (<-chan *CompositeAggregationBucket, <-chan error)
}

// client implements the Client interface.
type client struct {
	celastic.Client
	Do func(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error)
}

// NewFromComplianceClient creates a PIP elastic client from the Compliance ES client.
// TODO(rlb): The useful parts of the compliance client should be moved to libcalico, but time is not our friend at the
//            moment.
func NewFromComplianceClient(c celastic.Client) Client {
	return &client{Client: c, Do: doFunc}
}

// doFunc invokes the Do on the search service. This is added to allow us to mock out the client in test code.
func doFunc(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error) {
	return s.Do(ctx)
}

// NewMockClient creates a mock client used for testing.
func NewMockClient(doFunc func(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error)) Client {
	return &client{
		Client: mockComplianceClient{},
		Do:     doFunc,
	}
}

type mockComplianceClient struct {
	celastic.Client
}

func (m mockComplianceClient) Backend() *elastic.Client {
	return nil
}

func (m mockComplianceClient) ClusterIndex(string, string) string {
	return "fake-index"
}

// NewMockSearchClient creates a mock client used for testing search results.
func NewMockSearchClient(results []interface{}) Client {
	idx := 0

	doFunc := func(_ context.Context, _ *elastic.SearchService) (*elastic.SearchResult, error) {
		if idx >= len(results) {
			return nil, errors.New("Enumerated past end of results")
		}
		result := results[idx]
		idx++

		switch rt := result.(type) {
		case *elastic.SearchResult:
			return rt, nil
		case elastic.SearchResult:
			return &rt, nil
		case error:
			return nil, rt
		case string:
			result := new(elastic.SearchResult)
			decoder := &elastic.DefaultDecoder{}
			err := decoder.Decode([]byte(rt), result)
			return result, err
		}

		return nil, errors.New("Unexpected result type")
	}

	return NewMockClient(doFunc)
}
