// package cors includes middleware to support the w3 cors spec
//
// https://www.w3.org/TR/2020/SPSD-cors-20200602#resource-processing-model
package cors

import (
	"fmt"
	"net/http"
	"regexp"
)

type CORS struct {
	corsOriginRegexp *regexp.Regexp
}

func New(expr string) (*CORS, error) {
	r, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regexp for cors host '%s': %v", expr, err)
	}
	return &CORS{
		corsOriginRegexp: r,
	}, nil
}

// NewHandlerFunc generates a handler which is wrapped by a CORS handler.
func (c *CORS) NewHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		} else {
			// if not an options request, continue processing
			h.ServeHTTP(w, r)
		}
	})
}
