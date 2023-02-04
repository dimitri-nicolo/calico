// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/httputils"
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
	path    string
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

func (r *Request) Path(p string) *Request {
	r.path = p
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
	if err != nil {
		return &Result{
			err: fmt.Errorf("error marshalling request param: %s", err),
		}
	}

	// This is temporary, until we upgrade to go1.19 which has
	// native support for this via url.JoinPath
	JoinPath := func(base string, paths ...string) string {
		p := path.Join(paths...)
		return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), strings.TrimLeft(p, "/"))
	}
	url := JoinPath(r.client.config.URL, r.path)

	// Build the request.
	req, err := http.NewRequestWithContext(
		ctx,
		r.verb,
		url,
		bytes.NewBuffer(request),
	)
	if err != nil {
		return &Result{
			err: fmt.Errorf("error creating new request: %s", err),
		}
	}
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
		path:       r.path,
	}
}

type Result struct {
	err        error
	body       []byte
	statusCode int
	path       string
}

// Into decodes the body of the result into the given structure. obj should be a pointer.
func (r *Result) Into(obj any) error {
	if r.err != nil {
		return r.err
	}
	if len(r.body) == 0 {
		return fmt.Errorf("no body returned from request. status=%d", r.statusCode)
	}

	if r.statusCode == http.StatusNotFound {
		// The path wasn't found. We shouldn't parse the response as JSON.
		return fmt.Errorf("server returned not found for path %s: %s", r.path, string(r.body))
	} else if r.statusCode != http.StatusOK {
		// A structured error returned by the server - parse it.
		httpError := httputils.HttpStatusError{}
		err := json.Unmarshal(r.body, &httpError)
		if err != nil {
			return fmt.Errorf("failed to unmarshal error response: %s", err)
		}
		return &httpError
	}

	// Got an OK response - unmarshal it into the expected type.
	err := json.Unmarshal(r.body, obj)
	if err != nil {
		logrus.WithField("body", string(r.body)).Errorf("Error unmarshalling response")
		return fmt.Errorf("error unmarshalling response : %s", err)
	}

	return nil
}
