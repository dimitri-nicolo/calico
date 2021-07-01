// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware

import (
	"fmt"
	"net/http"

	"github.com/tigera/packetcapture-api/pkg/cache"

	log "github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

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
		req, stat, err := authentication.AuthenticateRequest(auth.authN, req)
		if err != nil {
			log.WithError(err).Errorf("failed to authenticate user")
			http.Error(w, err.Error(), stat)
			return
		}
		handlerFunc.ServeHTTP(w, req)
	}
}

// Authorize is a middleware handler that authorizes a request for aceess to
// subresource packet captures/files in a given namespace
func (auth *Auth) Authorize(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var clusterID = ClusterIDFromContext(req.Context())
		var authorizer, err = auth.cache.GetAuthorizer(clusterID)
		if err != nil {
			log.WithError(err).Error("Failed to create create authorizer")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resAtr := &authzv1.ResourceAttributes{
			Verb:        "get",
			Group:       "projectcalico.org",
			Resource:    "packetcaptures",
			Subresource: "files",
			Name:        CaptureNameFromContext(req.Context()),
			Namespace:   NamespaceFromContext(req.Context()),
		}

		usr, ok := request.UserFrom(req.Context())
		if !ok {
			var err = fmt.Errorf("missing user from request")
			log.WithError(err).Error("no user found in request context")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		isAuthorized, err := authorizer.Authorize(usr, resAtr, nil)
		if err != nil {
			log.WithError(err).Error("Kubernetes authorization failure")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if !isAuthorized {
			err := fmt.Errorf("%s is not authorized to get packetcaptures/files", usr.GetName())
			log.WithError(err).Error("User is not authorized")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		handlerFunc.ServeHTTP(w, req)
	}
}
