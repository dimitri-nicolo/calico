// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	k8sserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/tigera/packetcapture-api/pkg/cache"

	log "github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	authenticationv1 "k8s.io/api/authentication/v1"

	"github.com/projectcalico/apiserver/pkg/authentication"
)

// Auth is used to authenticate/authorize requests for PacketCapture files access
type Auth struct {
	authN authentication.Authenticator
	cache cache.ClientCache
}

// NewAuth will return an *Auth based on the passed in configuration and multi-cluster setup
// Authentication can be checked against the K8S Api server or against the Dex service
// Authorization can be checked against the management cluster or the managed cluster
// authenticator is a custom authenticator based on the give config
// cache.ClientCache will create/return specialized authorizer based on the request given
func NewAuth(authenticator authentication.Authenticator, cache cache.ClientCache) *Auth {
	return &Auth{authN: authenticator, cache: cache}
}

// Authenticate is a middleware handler that authenticates a request
func (auth *Auth) Authenticate(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Authenticate the user/service account based on the Authorization token
		// For a standalone and a management cluster, the Authorization token will
		// authenticate the initiator of the request. For a managed cluster, it
		// will authenticate the service account that tries to impersonate the initiator
		// of the request
		req, stat, err := authentication.AuthenticateRequest(auth.authN, req)
		if err != nil {
			log.WithError(err).Error("failed to authenticate user")
			http.Error(w, err.Error(), stat)
			return
		}

		// Extract username, groups and extras based on the Impersonation headers
		userName, groups, extras := auth.extractUserFromImpersonationHeaders(req)
		// We are rejecting requests that are missing Impersonate-User, but are providing groups
		// or extra headers. For example, requests with only Impersonate-group with value
		// "system:authenticated" will be rejected
		if len(userName) == 0 && (len(groups) != 0 || len(extras) != 0) {
			var err = fmt.Errorf("missing impersonation headers")
			log.WithError(err).Error("Impersonation headers are missing impersonate user header")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(userName) != 0 {
			// build resource attributes needed to perform authorization for impersonation
			var resAttributes = auth.buildResourceAttributesForImpersonation(userName, groups, extras)
			// authorize impersonation for user, service accounts, groups, extras
			for _, resAtr := range resAttributes {
				var err, status = auth.authorize(req, resAtr)
				if err != nil {
					http.Error(w, err.Error(), status)
					return
				}
			}

			// Override the user.Info set on the context of the request
			var user user.Info = &user.DefaultInfo{
				Name:   userName,
				Groups: groups,
				Extra:  extras,
			}
			req = req.WithContext(request.WithUser(req.Context(), user))
		}

		handlerFunc.ServeHTTP(w, req)
	}
}

func (auth *Auth) buildResourceAttributesForImpersonation(userName string, groups []string, extras map[string][]string) []*authzv1.ResourceAttributes {
	var result []*authzv1.ResourceAttributes

	if len(userName) > 0 {
		namespace, name, err := k8sserviceaccount.SplitUsername(userName)
		if err == nil {
			result = append(result, &authzv1.ResourceAttributes{
				Verb:      "impersonate",
				Resource:  "serviceaccounts",
				Name:      name,
				Namespace: namespace,
			})
		} else {
			result = append(result, &authzv1.ResourceAttributes{
				Verb:     "impersonate",
				Resource: "users",
				Name:     userName,
			})
		}
	}

	if len(groups) > 0 {
		for _, group := range groups {
			result = append(result, &authzv1.ResourceAttributes{
				Verb:     "impersonate",
				Resource: "groups",
				Name:     group,
			})
		}
	}

	if len(extras) > 0 {
		for key, extra := range extras {
			for _, value := range extra {
				result = append(result, &authzv1.ResourceAttributes{
					Verb:        "impersonate",
					Resource:    "userextras",
					Subresource: key,
					Name:        value,
				})
			}
		}
	}

	return result
}

func (auth *Auth) extractUserFromImpersonationHeaders(req *http.Request) (string, []string, map[string][]string) {
	var userName = req.Header.Get(authenticationv1.ImpersonateUserHeader)
	var groups = req.Header[authenticationv1.ImpersonateGroupHeader]
	var extras = make(map[string][]string)
	for headerName, value := range req.Header {
		if strings.HasPrefix(headerName, authenticationv1.ImpersonateUserExtraHeaderPrefix) {
			encodedKey := strings.ToLower(headerName[len(authenticationv1.ImpersonateUserExtraHeaderPrefix):])
			extraKey, err := url.PathUnescape(encodedKey)
			if err != nil {
				var err = fmt.Errorf("malformed extra key for impersonation request")
				log.WithError(err).Errorf("Could not decode extra key %s", encodedKey)
			}
			extras[extraKey] = value
		}
	}
	return userName, groups, extras
}

// Authorize is a middleware handler that authorizes a request for aceess to
// subresource packet captures/files in a given namespace
func (auth *Auth) Authorize(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		resAtr := &authzv1.ResourceAttributes{
			Verb:        "get",
			Group:       "projectcalico.org",
			Resource:    "packetcaptures",
			Subresource: "files",
			Name:        CaptureNameFromContext(req.Context()),
			Namespace:   NamespaceFromContext(req.Context()),
		}

		var err, status = auth.authorize(req, resAtr)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		handlerFunc.ServeHTTP(w, req)
	}
}

func (auth *Auth) authorize(req *http.Request, resAtr *authzv1.ResourceAttributes) (error, int) {
	var clusterID = ClusterIDFromContext(req.Context())
	var authorizer, err = auth.cache.GetAuthorizer(clusterID)
	if err != nil {
		log.WithError(err).Error("Failed to create authorizer")
		return err, http.StatusInternalServerError
	}
	usr, ok := request.UserFrom(req.Context())
	if !ok {
		var err = fmt.Errorf("missing user from request")
		log.WithError(err).Error("no user found in request context")
		return err, http.StatusBadRequest
	}

	isAuthorized, err := authorizer.Authorize(usr, resAtr, nil)
	if err != nil {
		log.WithError(err).Error("Kubernetes authorization failure")
		return err, http.StatusUnauthorized
	}

	if !isAuthorized {
		var err error
		if len(resAtr.Subresource) == 0 {
			err = fmt.Errorf("%s is not authorized to %s for %s", usr.GetName(), resAtr.Verb, resAtr.Resource)
		} else {
			err = fmt.Errorf("%s is not authorized to %s for %s/%s", usr.GetName(), resAtr.Verb, resAtr.Resource, resAtr.Subresource)
		}
		log.WithError(err).Error("User is not authorized")
		return err, http.StatusUnauthorized
	}

	return nil, 0
}
