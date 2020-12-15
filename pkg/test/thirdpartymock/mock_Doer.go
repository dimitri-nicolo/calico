package thirdpartymock

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type MockDoer struct {
	mock.Mock
}

func (mock *MockDoer) Do(req *http.Request) (*http.Response, error) {
	ret := mock.Called(req)

	var r0 *http.Response
	if rf, ok := ret.Get(0).(func(*http.Request) *http.Response); ok {
		r0 = rf(req)
	} else {
		r0 = ret.Get(0).(*http.Response)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*http.Request) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
