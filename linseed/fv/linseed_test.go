// Copyright (c) 2022 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFV_Linseed(t *testing.T) {
	var addr = "localhost:8444"
	var tests = []struct {
		name string
		path string
	}{
		{name: "should return 404 for / request", path: "/"},
		{name: "should return 404 for /foo request", path: "/foo"},
		{name: "should return 404 for /api/v1/flows/foo request", path: "/api/v1/flows/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := secureHTTPClient(t)
			res, resBody := doRequest(t, client, "GET", fmt.Sprintf("https://%s%s", addr, tt.path))

			assert.Equal(t, http.StatusNotFound, res.StatusCode)
			assert.Equal(t, "404 page not found", strings.Trim(string(resBody), "\n"))

			return
		})
	}

	t.Run("should deny any HTTP connection", func(t *testing.T) {
		var client = &http.Client{}
		res, resBody := doRequest(t, client, "GET", fmt.Sprintf("http://%s/", addr))

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "Client sent an HTTP request to an HTTPS server.", strings.Trim(string(resBody), "\n"))
	})
}
