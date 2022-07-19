// Copyright (c) 2022 Tigera All rights reserved.
package auth

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/health"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
)

const (
	adDetectorsResourceGroup = "detectors.tigera.io"
	adModelsResourceName     = "models"
	adMetadataResourceName   = "metadata"
)

var (
	publicSystemsEndpoint = map[string]bool{
		health.HealthPath: true,
	}
)

type RequestAccess struct {
	Path   string
	Method string
}

// Auth acts as a middleware for verifying the Authorization: bearer <jwt-auth-token>
// against the roles set for accessing the AD API
// fails with http error 401 if not authenticated, 403 if not authorized
func Auth(h http.Handler, authenticator lmaauth.JWTAuth, namespace string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if _, ok := publicSystemsEndpoint[path]; ok {
			// pass - skip authentication for public system info endpoints
			log.Infof("Accessing public endpoint %s", path)
			h.ServeHTTP(w, req)
			return
		}

		// authenticate
		usr, stat, err := authenticator.Authenticate(req)
		if err != nil { // err http status one of: 401, 500
			log.WithError(err).Infof("%d error autheticating user", stat)
			api_error.WriteStatusErrorToHeader(w, stat)
			return
		}

		resources, apiError := GetRBACResoureAttribute(namespace, req)
		if apiError != nil {
			api_error.WriteAPIErrorToHeader(w, apiError)
		}

		// preemptive exit with 405 to disregard continuing
		if len(resources) == 0 {
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
