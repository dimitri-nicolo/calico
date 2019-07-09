// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth

import (
	"net/http"
	"strings"
)

// Token type for different authorization methods against K8S
type Token int

const (
	// Unknown Token type
	Unknown Token = iota

	// Basic Token type
	Basic

	// Bearer Token type
	Bearer
)

var tokens = []string{"Unknown", "Basic", "Bearer"}

func (t Token) String() string {
	return tokens[t]
}

func toToken(v string) Token {
	for index, t := range tokens {
		if v == t {
			return Token(index)
		}
	}

	return Unknown
}

// Extract extracts a token and token type from an HTTP request
func Extract(r *http.Request) (token string, tokenType Token) {
	value := r.Header.Get("Authorization")
	if len(value) > 0 {
		slice := strings.Split(value, " ")
		if len(slice) == 2 {
			return slice[1], toToken(slice[0])
		}
	}

	return "", Unknown
}
