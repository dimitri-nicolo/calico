// Copyright (c) 2022 Tigera All rights reserved.
package auth

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	authzv1 "k8s.io/api/authorization/v1"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
)

// GetRBACResoureAttribute returns the RBAC Attributes for the Anomaly Detection API generated
// from the API Config
func GetRBACResoureAttribute(config *config.Config) map[string][]*authzv1.ResourceAttributes {
	return map[string][]*authzv1.ResourceAttributes{
		http.MethodHead: {
			{
				Namespace: config.HostedNamespace,
				Group:     "detectors.tigera.io",
				Resource:  "models",
				Verb:      "get",
			},
		},
		http.MethodGet: {
			{
				Namespace: config.HostedNamespace,
				Group:     "detectors.tigera.io",
				Resource:  "models",
				Verb:      "get",
			},
		},
		http.MethodPost: {
			{
				Namespace: config.HostedNamespace,
				Group:     "detectors.tigera.io",
				Resource:  "models",
				Verb:      "create",
			},
			{
				Namespace: config.HostedNamespace,
				Group:     "detectors.tigera.io",
				Resource:  "models",
				Verb:      "update",
			},
		},
	}
}

// Auth acts as a middleware for verifying the Authorization: bearer <jwt-auth-token>
// against the roles set for accessing the AD API
// fails with http error 401 if not authenticated, 403 if not authorized
func Auth(h http.Handler, authenticator lmaauth.JWTAuth, rbacAttributes map[string][]*authzv1.ResourceAttributes) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// authenticate
		usr, stat, err := authenticator.Authenticate(req)
		if err != nil { // err http status one of: 401, 500
			log.WithError(err).Infof("%d error autheticating user", stat)
			api_error.WriteStatusErrorToHeader(w, stat)
			return
		}

		resources, found := rbacAttributes[req.Method]

		// preemptive exit with 405 to disregard continuing
		if !found {
			api_error.WriteStatusErrorToHeader(w, http.StatusMethodNotAllowed)
			return
		}

		// authorize role once user recieved
		// compare the role we are expecting to one retrieved by lma dependent on the
		// requests attempted auth
		authorized := false
		// Check if the user has all the expected permission to access the endpoint
		for _, res := range resources {
			log.Infof("authozing resource %+v with user %s ,%+v, %+v", res, usr.GetName(), usr.GetGroups(), usr.GetExtra())
			allowed, err := authenticator.Authorize(usr, res, nil)
			if err != nil {
				log.WithError(err).Infof("error authorizing request")
				api_error.WriteStatusErrorToHeader(w, http.StatusInternalServerError)
				return
			}

			if allowed {
				authorized = true
			} else {
				// if any of the permission requirements are not met unauthorize the
				// request since at this point we know the user does not have  complete
				// permissions requirement
				authorized = false
				break
			}
		}

		if !authorized {
			api_error.WriteStatusErrorToHeader(w, http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, req)
	})
}
