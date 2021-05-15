// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middleware

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	lmaauth "github.com/tigera/lma/pkg/auth"

	"github.com/projectcalico/apiserver/pkg/authentication"
)

// AuthenticateRequest uses the given Authenticator to authenticate the request then passes the request to the next Handler
// if the authentication was successful.
func AuthenticateRequest(authn authentication.Authenticator, handlerFunc http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		req, stat, err := authentication.AuthenticateRequest(authn, req)
		if err != nil {
			log.WithError(err).Debug("Failed to authenticate request.")
			http.Error(w, err.Error(), stat)
			return
		}
		handlerFunc.ServeHTTP(w, req)
	}
}

// AuthorizeRequest uses the given RBACAuthorizer to authorize the request against the RBAC attributes stored in the request
// context and passes the request to the next handler if the authorization was successful.
func AuthorizeRequest(authz lmaauth.RBACAuthorizer, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		usr, res, nonRes := lmaauth.ExtractRBACFieldsFromContext(req.Context())

		if authorized, err := authz.Authorize(usr, res, nonRes); err != nil {
			log.WithError(err).Debug("Failed to authorize request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !authorized {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		handler.ServeHTTP(w, req)
	}
}

// SetAuthorizationHeaderFromCookie takes the bearer token from the cookie named "Bearer" and adds the authorization
// header that's needed for authentication and authorization.
func SetAuthorizationHeaderFromCookie(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bearerCookie, err := req.Cookie("Bearer")
		if err != nil {
			log.WithError(err).Error("failed to get the bearer cookie")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if bearerCookie == nil {
			log.Debug("No bearer cookie found.")
			return
		}

		newReq := req
		newReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerCookie.Value))

		h.ServeHTTP(w, newReq)
	})
}
