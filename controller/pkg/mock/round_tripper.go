// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import "net/http"

type RoundTripper struct {
	Response *http.Response
	Error    error
	Count    uint
}

func (r *RoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	r.Count++
	return r.Response, r.Error
}
