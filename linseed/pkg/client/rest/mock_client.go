package rest

import (
	"fmt"
	"net/http"
)

func NewMockClient(results ...MockResult) RESTClient {
	return &mockRestClient{results: results}
}

type mockRestClient struct {
	results []MockResult
	called  int
}

func (m *mockRestClient) Verb(v string) Request {
	// Return the nth request with its expected result.
	if m.called > len(m.results) {
		panic(fmt.Sprintf("Mock client called %d times, but only have %d results", m.called+1, len(m.results)))
	} else if len(m.results) == 0 {
		panic("Mock client called, but have no results to return!")
	}
	result := m.results[m.called]
	m.called++
	return NewMockRequest(m, result).Verb(v)
}

func (m *mockRestClient) Post() Request {
	return m.Verb("POST")
}

// BaseURL should never be used by the mock client.
func (m *mockRestClient) BaseURL() string {
	panic("not implemented")
}

// Tenant should never be used by the mock client.
func (m *mockRestClient) Tenant() string {
	panic("not implemented")
}

// HTTPClient should never be used by the mock client.
func (m *mockRestClient) HTTPClient() *http.Client {
	panic("not implemented")
}
