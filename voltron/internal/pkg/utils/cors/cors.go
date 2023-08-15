package cors

import (
	"net/http"
	"regexp"
)

const (
	// vary header
	vary = "Vary"

	// accessControlAllowOrigin CORS header for allowed origin
	accessControlAllowOrigin = "Access-Control-Allow-Origin"

	// accessControlAllowMethods Proxy CORS header for allowed methods
	accessControlAllowMethods = "Access-Control-Allow-Methods"

	// accessControlAllowHeaders Proxy CORS header for allowed headers
	accessControlAllowHeaders = "Access-Control-Allow-Headers"

	// accessControlAllowCredentials Proxy CORS header for allowed credentials
	accessControlAllowCredentials = "Access-Control-Allow-Credentials"

	allowedMethods = "GET, POST, PUT, DELETE, PATCH, OPTIONS"
	allowedHeaders = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Auth-IdToken, x-cluster-id"
)

type ModifyResponse func(r *http.Response) error
type PreflightRequestHandler func(r *http.Request, headersOnly bool) http.HandlerFunc

func setCommonHeaders(origin string, headers http.Header) {
	headers.Set(vary, "Origin")
	headers.Set(accessControlAllowOrigin, origin)
	headers.Set(accessControlAllowCredentials, "true")
}

func HandlePreflight(origin string, w http.ResponseWriter, headersOnly bool) {
	setCommonHeaders(origin, w.Header())
	w.Header().Set(accessControlAllowMethods, allowedMethods)
	w.Header().Set(accessControlAllowHeaders, allowedHeaders)
	if !headersOnly {
		w.WriteHeader(http.StatusOK)
	}
}

func ResponseHandler(corsOriginRegexp *regexp.Regexp) ModifyResponse {
	return func(r *http.Response) error {
		origin := r.Request.Header.Get("origin")
		if corsOriginRegexp.MatchString(origin) {
			setCommonHeaders(origin, r.Header)
		}
		return nil
	}
}
