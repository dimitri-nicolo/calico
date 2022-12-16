// Copyright (c) 2022 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed cert
var certs embed.FS

func doRequest(t *testing.T, client *http.Client, method, url string) (*http.Response, []byte) {
	var req, err = http.NewRequest(method, url, nil)
	assert.NoError(t, err)

	var res = &http.Response{}
	res, err = client.Do(req)
	assert.NoError(t, err)
	defer res.Body.Close()

	var resBody []byte
	resBody, err = io.ReadAll(res.Body)
	assert.NoError(t, err)
	return res, resBody
}

func secureHTTPClient(t *testing.T) *http.Client {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	rootCaCert, err := certs.ReadFile("cert/RootCA.crt")
	assert.NoError(t, err)
	rootCAs.AppendCertsFromPEM(rootCaCert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	return client
}
