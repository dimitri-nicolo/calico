package rest

import (
	"context"
	"net/http"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

// Produce a new mock request, used to mock request results from Linseed.
func NewMockRequest(c RESTClient, result MockResult) Request {
	return &mockRequest{
		realRequest: NewRequest(c).(*request),
		result:      result,
	}
}

type mockRequest struct {
	// Wrap a real request so that we can rely on its logic where needed.
	realRequest *request

	// Mock result to return on call to Do()
	result MockResult
}

func (m *mockRequest) Verb(v string) Request {
	m.realRequest.Verb(v)
	return m
}

func (m *mockRequest) Params(p any) Request {
	m.realRequest.Params(p)
	return m
}

func (m *mockRequest) BodyJSON(b any) Request {
	m.realRequest.BodyJSON(b)
	return m
}

func (m *mockRequest) Path(p string) Request {
	m.realRequest.Path(p)
	return m
}

func (m *mockRequest) Cluster(c string) Request {
	m.realRequest.Cluster(c)
	return m
}

func (m *mockRequest) ContentType(t string) Request {
	m.realRequest.ContentType(t)
	return m
}

// This is where the magic happens. Do() simulates a
// real response from Linseed. The mock client stack provides a
// hook for callers to return custom results here.
func (m *mockRequest) Do(ctx context.Context) *Result {
	return &Result{
		err:        m.result.Err,
		body:       m.result.body(),
		statusCode: m.result.statusCode(),
		path:       "/mock/request",
	}
}

type MockResult struct {
	Err        error
	Body       interface{}
	StatusCode int
}

func (m *MockResult) body() []byte {
	bs, err := json.Marshal(m.Body)
	if err != nil {
		panic(err)
	}
	return bs
}

func (m *MockResult) statusCode() int {
	if m.StatusCode != 0 {
		return m.StatusCode
	}
	return http.StatusOK
}
