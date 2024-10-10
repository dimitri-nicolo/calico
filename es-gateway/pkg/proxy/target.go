// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// Route represents one (of possibly many) route paths that a Target will be matched for. This means
// we only accept requests destined to the Target if it matches one of the defined Routes for that Target.
type Route struct {
	// String representing the path of the given route. This value can be a regex pattern,
	// but if and only if the IsPathPrefix flag is not true. This is because Gorilla Mux (the
	// routing library we are using) does not support path prefixes that are also regex expressions.
	Path string
	// IsPathPrefix indicates whether this Route should be treated as a path prefix. Note that
	// if it is to be treated as a prefix, then it cannot contain a regrex pattern.
	IsPathPrefix bool
	// RequireAuth indicates whether this Route requires authentication.
	RequireAuth bool
	// RejectUnacceptableContentType mitigates CVE-2020-28491, which Elasticsearch is vulnerable to. We reject calls
	// with content-type application/cbor or application/smile.
	RejectUnacceptableContentType bool
	// HTTPMethods specifies which HTTP method the Route should match with. If empty, then Route
	// will match with any HTTP method.
	HTTPMethods []string
	// Name helps identify this Route (for logging purposes).
	Name           string
	EnforceTenancy bool
}

// Routes is a listing of Route paths that a Target possesses.
type Routes []Route

// Target defines a destination URL to which HTTP requests will be proxied. The path prefix dictates
// which requests will be proxied to this particular target.
type Target struct {
	// Dest is the destination of this Target. Requests that match with a Route from this Target will
	// be sent to this destination.
	Dest *url.URL

	// List of Routes that apply to this Target. Any request that matches one of these Routes will be
	// directed to the Dest of this Target.
	Routes Routes

	// Catch-all Route for this Traget that is evaluated last (by virtue of being configured last).
	// This is optional (defaults to nil).
	CatchAllRoute *Route

	// Provides the CA cert to use for TLS verification.
	CAPem string

	// Provides the client cert to use for mTLS.
	ClientCert string

	// Provides the client key to use for mTLS.
	ClientKey string

	// Allows mTLS for this target.
	EnableMutualTLS bool

	// Transport to use for this target. If nil, a transport will be provided. This is useful for testing.
	Transport http.RoundTripper

	// Allow TLS without the verify step. This is useful for testing.
	AllowInsecureTLS bool
}

// CreateTarget returns a Target instance based on the provided parameter values.
func CreateTarget(catchAllRoute *Route, routes Routes, dest, caCertPath, clientCertPath, clientKeyPath string, enableMTLS, allowInsecureTLS bool) (*Target, error) {
	var err error

	target := &Target{
		Routes:           routes,
		AllowInsecureTLS: allowInsecureTLS,
		EnableMutualTLS:  enableMTLS,
	}

	if len(routes) < 1 {
		return nil, errors.New("target configuration requires at least one route")
	}

	if catchAllRoute != nil {
		target.CatchAllRoute = catchAllRoute
	}

	target.Dest, err = url.Parse(dest)
	if err != nil {
		return nil, fmt.Errorf("could not parse destination URL %q with associcated routes %+v: %s", dest, routes, err)
	}

	if target.Dest.Scheme == "https" && !allowInsecureTLS && caCertPath == "" {
		return nil, fmt.Errorf("target for '%s' must specify the CA bundle if AllowInsecureTLS is false when the scheme is https", dest)
	}

	if caCertPath != "" {
		target.CAPem = caCertPath
	}

	if clientCertPath != "" && clientKeyPath != "" {
		target.ClientCert = clientCertPath
		target.ClientKey = clientKeyPath
	}

	return target, nil
}
