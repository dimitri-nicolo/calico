package auth

import (
	"errors"

	k8suser "k8s.io/apiserver/pkg/authentication/user"

	"github.com/projectcalico/apiserver/pkg/authentication"
)

// aggregateAuthenticator will authenticate the provided authenticator args in order. If an authenticator returns an
// HTTP 421 misdirected error code, it tries the next, until it it reaches an authenticator that can authenticate
// the authorization header.
type aggregateAuthenticator struct {
	authenticators []authentication.Authenticator
}

// Authenticate will authenticate based on its provided authenticators. If an authenticator returns an HTTP 421 misdirected
// error code, it tries the next, until it it reaches an authenticator that can authenticate the authorization header.
func (a *aggregateAuthenticator) Authenticate(token string) (k8suser.Info, int, error) {
	if a.authenticators == nil || len(a.authenticators) == 0 {
		return nil, 500, errors.New("authenticator was not configured correctly")
	}
	for _, auth := range a.authenticators {
		if auth != nil {
			usr, stat, err := auth.Authenticate(token)
			if stat != 421 {
				return usr, stat, err
			}
		}
	}
	return nil, 401, errors.New("no authenticator can authenticate user")
}

// NewAggregateAuthenticator will create an authenticator that combines multiple authenticators into one.
func NewAggregateAuthenticator(authenticators ...authentication.Authenticator) authentication.Authenticator {
	var auths []authentication.Authenticator
	for _, a := range authenticators {
		if a != nil {
			auths = append(auths, a)
		}
	}
	return &aggregateAuthenticator{auths}
}
