// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func NewRequest(c *RESTClient) *Request {
	return &Request{
		client: c,
	}
}

// Request is a helper struct for building an HTTP request.
type Request struct {
	client  *RESTClient
	headers http.Header
	verb    string
	params  any
}

// Verb sets the verb this request will use.
func (r *Request) Verb(verb string) *Request {
	r.verb = verb
	return r
}

// Params sets parameters to pass in the request body.
func (r *Request) Params(p any) *Request {
	r.params = p
	return r
}

// Set HTTP headers on the request.
func (r *Request) SetHeader(key string, values ...string) *Request {
	if r.headers == nil {
		r.headers = http.Header{}
	}
	r.headers.Del(key)
	for _, value := range values {
		r.headers.Add(key, value)
	}
	return r
}

func (r *Request) Do(ctx context.Context) *Result {
	request, err := json.Marshal(r.params)

	// Build the request.
	req, err := http.NewRequestWithContext(
		ctx,
		r.verb,
		r.client.config.URL,
		bytes.NewBuffer(request),
	)
	req.Header.Set("x-cluster-id", r.client.clusterID)
	req.Header.Set("x-tenant-id", r.client.tenantID)
	req.Header.Set("Content-Type", "application/json")

	// Perform the request.
	response, err := r.client.client.Do(req)
	if err != nil {
		return &Result{
			err: fmt.Errorf("error connecting linseed API: %s", err),
		}
	}
	defer response.Body.Close()

	// Build the response.
	responseByte, err := io.ReadAll(response.Body)
	return &Result{
		err:        err,
		body:       responseByte,
		statusCode: response.StatusCode,
	}
}

type Result struct {
	err        error
	body       []byte
	statusCode int
}

// Into decodes the body of the result into the given structure. obj should be a pointer.
func (r *Result) Into(obj any) error {
	if r.err != nil {
		return r.err
	}
	if len(r.body) == 0 {
		return fmt.Errorf("no body returned from request. status=%d", r.statusCode)
	}

	err := json.Unmarshal(r.body, obj)
	if err != nil {
		return err
	}

	return nil
}
