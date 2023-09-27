// package cors includes middleware to support the w3 cors spec
//
// https://www.w3.org/TR/2020/SPSD-cors-20200602#resource-processing-model
package cors

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type CORS struct {
	corsOriginRegexp *regexp.Regexp

	// ignorePath is a url path which the CORS handler should not modify requests to.
	// this is useful for destinations which already set the appropriate cors headers.
	ignorePaths []string
}

func New(expr string, ignorePaths ...string) (*CORS, error) {
	r, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regexp for cors host '%s': %v", expr, err)
	}
	return &CORS{
		corsOriginRegexp: r,
		ignorePaths:      ignorePaths,
	}, nil
}

// NewHandlerFunc generates a handler which is wrapped by a CORS handler.
func (c *CORS) NewHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// skip cors headers if the ignorePath is configured and matches.
		// we ignore this setting and set headers anyways for Options requests since the voltron middleware
		// incorrectly checks authentication for these types of requests.
		if r.Method != http.MethodOptions {
			for _, path := range c.ignorePaths {
				if path != "" && strings.HasPrefix(r.URL.Path, path) {
					h.ServeHTTP(w, r)
					return
				}
			}
		}

		origin := r.Header.Get("origin")

		if origin == "" {
			// if the Origin header is not present terminate this set of steps.
			h.ServeHTTP(w, r)
			return
		}

		// if the value of the Origin header is not a case-sensitive match,
		// do not set any additional headers.
		if !c.corsOriginRegexp.MatchString(origin) {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)

		// since we use a regex statement to allow sharing with multiple Origins,
		// the returned header value for Access-Control-Allow-Origin is dynamic.
		// As a consequence of this, send a Vary: Origin HTTP header to prevent caching.
		w.Header().Set("Vary", "Origin")

		// since we support credentials, add the Access-Control-Allow-Credentials header
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Auth-IdToken, x-cluster-id")
			// we would ideally allow the options request through to some backend services like prometheus which respond with the correct cors settings
			// but voltron is configured to strictly check credentials, even if it is incorrect to do so on options requests, so we won't allow
			// the request to continue through and instead will respond on behalf of the backend services.
		} else {
			// if not an options request, continue processing
			h.ServeHTTP(w, r)
		}
	})
}
