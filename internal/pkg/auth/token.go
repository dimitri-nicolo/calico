// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth

import (
	"net/http"
	"strings"
)

// Token type for different authorization methods against K8S
type Token = int

const (
	// Unknown Token type
	Unknown Token = iota

	// Basic Token type
	Basic

	// Bearer Token type
	Bearer
)

// Extract extracts a token and token type from an HTTP request
func Extract(r *http.Request) (token string, tokenType Token) {
	bearer, present := extractBearer(r)
	if present {
		return bearer, Bearer
	}
	basic, present := extractBasic(r)
	if present {
		return basic, Basic
	}

	return "", Unknown
}

func extractBearer(r *http.Request) (token string, present bool) {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 0 {
		slice := strings.Split(bearer, " ")
		if len(slice) == 2 && slice[0] == "Bearer" {
			return slice[1], true
		}
	}

	return "", false
}

func extractBasic(r *http.Request) (token string, present bool) {
	basic := r.Header.Get("Auth")
	if len(basic) > 0 {
		return basic, true
	}

	return "", false
}
