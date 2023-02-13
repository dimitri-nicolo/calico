// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package rest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const (
	ContentTypeJSON          = "application/json"
	ContentTypeMultilineJSON = "application/x-ndjson"
)

func NewRequest(c *RESTClient) *Request {
	return &Request{
		client: c,
	}
}

// Request is a helper struct for building an HTTP request.
type Request struct {
	client      *RESTClient
	contentType string
	verb        string
	params      any
	body        any
	path        string
	clusterID   string
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

// BodyJSON sets the body
func (r *Request) BodyJSON(p any) *Request {
	r.body = p
	return r
}

func (r *Request) Path(p string) *Request {
	r.path = p
	return r
}

// Cluster sets the x-cluster-id header for this request.
func (r *Request) Cluster(c string) *Request {
	r.clusterID = c
	return r
}

func (r *Request) ContentType(c string) *Request {
	r.contentType = c
	return r
}

func (r *Request) Do(ctx context.Context) *Result {
	if r.body != nil && r.params != nil {
		return &Result{
			err: fmt.Errorf("cannot specify body and params on same requst"),
		}
	}

	var err error
	var body []byte
	if r.params != nil {
		body, err = json.Marshal(r.params)
		if err != nil {
			return &Result{
				err: fmt.Errorf("error marshalling request param: %s", err),
			}
		}
	}
	if r.body != nil {
		var ok bool
		body, ok = r.body.([]byte)
		if !ok {
			return &Result{
				err: fmt.Errorf("body must be a slice of bytes"),
			}
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
		bytes.NewBuffer(body),
	)
	if err != nil {
		return &Result{
			err: fmt.Errorf("error creating new request: %s", err),
		}
	}
	req.Header.Set("x-cluster-id", r.clusterID)
	req.Header.Set("x-tenant-id", r.client.tenantID)

	if r.contentType == "" {
		req.Header.Set("Content-Type", ContentTypeJSON)
	} else {
		req.Header.Set("Content-Type", r.contentType)
	}

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
		httpError := v1.HTTPError{}
		err := json.Unmarshal(r.body, &httpError)
		if err != nil {
			return fmt.Errorf("failed to unmarshal error response: %s", err)
		}
		return fmt.Errorf("[status %d] %s", httpError.Status, httpError.Msg)
	}

	// Got an OK response - unmarshal it into the expected type.
	err := json.Unmarshal(r.body, obj)
	if err != nil {
		logrus.WithField("body", string(r.body)).Errorf("Error unmarshalling response")
		return fmt.Errorf("error unmarshalling response: %s", err)
	}

	return nil
}
