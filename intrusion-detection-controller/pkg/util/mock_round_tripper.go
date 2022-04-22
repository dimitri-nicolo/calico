// Copyright 2019 Tigera Inc. All rights reserved.

package util

import "net/http"

type MockRoundTripper struct {
	Response *http.Response
	Error    error
	Count    uint
}

func (r *MockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	r.Count++
	return r.Response, r.Error
}
