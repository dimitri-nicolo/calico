// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

func NewKubernetesAuthzTracker(authz auth.RBACAuthorizer) *KubernetesAuthzTracker {
	return &KubernetesAuthzTracker{
		authorizer: authz,
		authMap:    make(map[reqdata]*authzv1.ResourceAttributes),
	}
}

type KubernetesAuthzTracker struct {
	authorizer auth.RBACAuthorizer
	authMap    map[reqdata]*authzv1.ResourceAttributes
}

func (t *KubernetesAuthzTracker) Register(v, u string, a *authzv1.ResourceAttributes) {
	url := fmt.Sprintf("/api/v1%s", u)
	logrus.Infof("Registering %s %s to authorize via %#v", v, url, a)
	r := reqdata{Verb: v, BaseURL: url}
	t.authMap[r] = a
}

// Disable disables authorization for the given endpoint and method.
func (t *KubernetesAuthzTracker) Disable(v, u string) {
	r := reqdata{Verb: v, BaseURL: fmt.Sprintf("/api/v1%s", u)}

	// A nil entry is not the same as a non-existent entry.
	t.authMap[r] = nil
}

// Attributes returns the necessary RBAC permissions to check for the endpoint. It will return an error
// if this endpoint is not registered properly.
func (t *KubernetesAuthzTracker) Attributes(v, u string) (*authzv1.ResourceAttributes, bool, error) {
	r := reqdata{Verb: v, BaseURL: u}
	attrs, ok := t.authMap[r]
	if !ok {
		return nil, true, fmt.Errorf("No matching authz options for %s %s", v, u)
	}

	// Attributes, requires-rbac, and error
	return attrs, attrs != nil, nil
}

// Represents the metadata of a given request, used to lookup the necessary
// RBAC required by the API caller.
type reqdata struct {
	Verb    string
	BaseURL string
}

type TokenChecker struct {
	authn auth.Authenticator
	authz *KubernetesAuthzTracker
}

func NewTokenAuth(authn auth.Authenticator, authz *KubernetesAuthzTracker) *TokenChecker {
	return &TokenChecker{
		authn: authn,
		authz: authz,
	}
}

func (m TokenChecker) Do() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Get the cluster / tenant for logging purposes.
			tenantID := TenantIDFromContext(req.Context())
			clusterID := ClusterIDFromContext(req.Context())
			f := logrus.Fields{"cluster": clusterID}
			if tenantID != "" {
				f["tenant"] = tenantID
			}
			log := logrus.WithFields(f)

			// Get the auth header.
			auth := req.Header.Get("Authorization")
			if auth == "" {
				httputils.JSONError(w, &v1.HTTPError{
					Status: http.StatusUnauthorized,
					Msg:    "No Authorization header provided",
				}, http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(auth, "Bearer ") {
				httputils.JSONError(w, &v1.HTTPError{
					Status: http.StatusUnauthorized,
					Msg:    "Authorization is not Bearer",
				}, http.StatusUnauthorized)
				return
			}

			// Extract the token.
			token := strings.TrimPrefix(auth, "Bearer ")
			if token == "" {
				log.Warn("No bearer token in request")
				httputils.JSONError(w, &v1.HTTPError{
					Status: http.StatusUnauthorized,
					Msg:    "No token found in request",
				}, http.StatusUnauthorized)
				return
			}

			// Authenticate the token. This makes sure the token was signed by a trusted authority -
			// either Linseed itself, or this cluster's API server - and extracts the user information
			// from the claims within the token.
			userInfo, status, err := m.authn.Authenticate(req)
			if err != nil {
				log.WithError(err).Warn("Could not authenticate token from request")
				httputils.JSONError(w, &v1.HTTPError{
					Status: status,
					Msg:    err.Error(),
				}, status)
				return
			}

			// Find the required RBAC call for this URL.
			resources, check, err := m.authz.Attributes(req.Method, req.URL.Path)
			if err != nil {
				log.WithError(err).Warn("Could not authorize token from request")
				httputils.JSONError(w, &v1.HTTPError{
					Status: http.StatusNotFound,
					Msg:    err.Error(),
				}, http.StatusNotFound)
				return
			}

			if check {
				// Update the context logger with new info.
				f = logrus.Fields{
					"user":     userInfo.GetName(),
					"verb":     resources.Verb,
					"resource": resources.Resource,
					"group":    resources.Group,
				}
				log = log.WithFields(f)

				// Authorize the user that was found within the now-validated token.
				ok, err := m.authz.authorizer.Authorize(userInfo, resources, nil)
				if err != nil {
					log.WithError(err).Warn("Error authorizing user")
					httputils.JSONError(w, &v1.HTTPError{
						Status: http.StatusUnauthorized,
						Msg:    err.Error(),
					}, http.StatusUnauthorized)
					return
				}
				if !ok {
					log.Info("User not authorized")
					httputils.JSONError(w, &v1.HTTPError{
						Status: http.StatusUnauthorized,
						Msg:    "Unauthorized",
					}, http.StatusUnauthorized)
					return
				}
			}

			next.ServeHTTP(w, req)
		})
	}
}
