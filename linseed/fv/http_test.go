// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed cert
var certs embed.FS

type httpReqSpec struct {
	method  string
	url     string
	body    []byte
	headers map[string]string
}

func (h *httpReqSpec) AddHeaders(headers map[string]string) {
	if h.headers == nil {
		h.headers = make(map[string]string)
	}
	for k, v := range headers {
		h.headers[k] = v
	}
}

func (h *httpReqSpec) SetBody(body string) {
	h.body = []byte(body)
}

func noBodyHTTPReqSpec(method, url, tenant, cluster string) httpReqSpec {
	return httpReqSpec{
		method: method,
		url:    url,
		headers: map[string]string{
			"x-cluster-id": tenant,
			"x-tenant-id":  cluster,
		},
	}
}

func jsonPostHTTPReqSpec(url, tenant, cluster string, body []byte) httpReqSpec {
	return httpReqSpec{
		method: "POST",
		url:    url,
		headers: map[string]string{
			"x-cluster-id": cluster,
			"x-tenant-id":  tenant,
			"Content-Type": "application/json",
		},
		body: body,
	}
}

func doRequest(t *testing.T, client *http.Client, spec httpReqSpec) (*http.Response, []byte) {
	req, err := http.NewRequest(spec.method, spec.url, bytes.NewBuffer(spec.body))
	for k, v := range spec.headers {
		req.Header.Set(k, v)
	}
	require.NoError(t, err)

	res := &http.Response{}
	res, err = client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	var resBody []byte
	resBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	return res, resBody
}

func secureHTTPClient(t *testing.T) *http.Client {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	rootCaCert, err := certs.ReadFile("cert/RootCA.crt")
	require.NoError(t, err)
	rootCAs.AppendCertsFromPEM(rootCaCert)

	tlsConfig := &tls.Config{
		RootCAs: rootCAs,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	return client
}
