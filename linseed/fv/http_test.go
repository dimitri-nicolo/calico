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

func xndJSONPostHTTPReqSpec(url, tenant, cluster string, body []byte) httpReqSpec {
	return httpReqSpec{
		method: "POST",
		url:    url,
		headers: map[string]string{
			"x-cluster-id": cluster,
			"x-tenant-id":  tenant,
			"Content-Type": "application/x-ndjson",
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
	// Get root CA for TLS verification of the server cert.
	certPool, _ := x509.SystemCertPool()
	if certPool == nil {
		certPool = x509.NewCertPool()
	}
	caCert, err := certs.ReadFile("cert/RootCA.crt")
	require.NoError(t, err)
	certPool.AppendCertsFromPEM(caCert)

	// Get client certificate for mTLS.
	cert, err := tls.LoadX509KeyPair("cert/localhost.crt", "cert/localhost.key")
	require.NoError(t, err)

	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{cert},
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	return client
}
