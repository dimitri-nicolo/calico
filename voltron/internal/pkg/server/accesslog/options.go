package accesslog

import (
	"fmt"
	"net/http"
	"strings"
)

type Option func(c *config) error

func WithPath(path string) Option {
	return func(c *config) error {
		c.zapConfig.OutputPaths = append(c.zapConfig.OutputPaths, path)
		return nil
	}
}

// WithStringJWTClaim if both claimName & logFieldName are not empty, the claim with that name will be logged
func WithStringJWTClaim(claimName, logFieldName string) Option {
	return func(c *config) error {
		claimName = strings.TrimSpace(claimName)
		logFieldName = strings.TrimSpace(logFieldName)
		if claimName != "" && logFieldName != "" {
			c.stringClaims = append(c.stringClaims, fieldMapping{inputName: claimName, logFieldName: logFieldName})
		}
		return nil
	}
}

// WithStringArrayJWTClaim if both claimName & logFieldName are not empty, the claim with that name will be logged
func WithStringArrayJWTClaim(claimName, logFieldName string) Option {
	return func(c *config) error {
		claimName = strings.TrimSpace(claimName)
		logFieldName = strings.TrimSpace(logFieldName)
		if claimName != "" && logFieldName != "" {
			c.stringArrayClaims = append(c.stringArrayClaims, fieldMapping{inputName: claimName, logFieldName: logFieldName})
		}
		return nil
	}
}

// WithStandardJWTClaims log standard claims, "iss", "sub", "aud", "sid", "nonce"
func WithStandardJWTClaims() Option {
	return func(c *config) error {
		c.stringClaims = append(c.stringClaims, fieldMapping{inputName: "iss", logFieldName: "iss"})
		c.stringClaims = append(c.stringClaims, fieldMapping{inputName: "sub", logFieldName: "sub"})
		c.stringClaims = append(c.stringClaims, fieldMapping{inputName: "aud", logFieldName: "aud"})
		c.stringClaims = append(c.stringClaims, fieldMapping{inputName: "sid", logFieldName: "sid"})
		c.stringClaims = append(c.stringClaims, fieldMapping{inputName: "nonce", logFieldName: "nonce"})
		return nil
	}
}

// WithRequestHeader if both claimName & logFieldName are not empty, the claim with that name will be logged
func WithRequestHeader(headerName, logFieldName string) Option {
	return func(c *config) error {
		headerName = http.CanonicalHeaderKey(strings.TrimSpace(headerName))
		if headerName == authorizationHeaderName || headerName == cookieHeaderName {
			return fmt.Errorf("%v header is not permitted", headerName)
		}
		logFieldName = strings.TrimSpace(logFieldName)

		if headerName != "" && logFieldName != "" {
			c.requestHeaders = append(c.requestHeaders, fieldMapping{inputName: headerName, logFieldName: logFieldName})
		}
		return nil
	}
}
