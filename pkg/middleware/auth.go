package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/apiserver/pkg/authentication"
	lmaauth "github.com/tigera/lma/pkg/auth"
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
