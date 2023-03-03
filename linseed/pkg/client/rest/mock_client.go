package rest

import (
	"fmt"
	"net/http"
)

func NewMockClient(results ...MockResult) RESTClient {
	return &MockRESTClient{results: results, requests: []*MockRequest{}}
}

type MockRESTClient struct {
	results  []MockResult
	called   int
	requests []*MockRequest
}

// Returns the MockRequets made by this client.
func (m *MockRESTClient) Requests() []*MockRequest {
	return m.requests
}

func (m *MockRESTClient) Verb(v string) Request {
	// Return the nth request with its expected result.
	if m.called > len(m.results) {
		panic(fmt.Sprintf("Mock client called %d times, but only have %d results", m.called+1, len(m.results)))
	} else if len(m.results) == 0 {
		panic("Mock client called, but have no results to return!")
	}
	result := m.results[m.called]
	m.called++
	req := NewMockRequest(m, &result).Verb(v)
	mr := req.(*MockRequest)
	m.requests = append(m.requests, mr)
	return req
}

func (m *MockRESTClient) Post() Request {
	return m.Verb("POST")
}

// BaseURL should never be used by the mock client.
func (m *MockRESTClient) BaseURL() string {
	panic("not implemented")
}

// Tenant should never be used by the mock client.
func (m *MockRESTClient) Tenant() string {
	panic("not implemented")
}

// HTTPClient should never be used by the mock client.
func (m *MockRESTClient) HTTPClient() *http.Client {
	panic("not implemented")
}
