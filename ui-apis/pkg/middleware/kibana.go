// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/ui-apis/pkg/kibana"
	"github.com/projectcalico/calico/ui-apis/pkg/user"
)

const (
	kibanaURL    = "/tigera-kibana"
	dashboardURL = "/dashboard/overall-stats"

	OIDCUsersElasticsearchCredentialsSecret = "tigera-oidc-users-elasticsearch-credentials"
)

type kibanaLoginHandler struct {
	k8sCli                      datastore.ClientSet
	kibanaCli                   kibana.Client
	oidcAuthEnabled             bool
	oidcAuthIssuer              string
	esLicense                   ElasticsearchLicenseType
	elasticsearchUsernamePrefix string
}

func NewKibanaLoginHandler(
	k8sCli datastore.ClientSet,
	kibanaCli kibana.Client,
	oidcAuthEnabled bool,
	oidcAuthIssuer string, esLicense ElasticsearchLicenseType) http.Handler {
	// A tenant more or less represents a billing account. We want to prefix the users with the tenant, so that
	// we don't get overlapping usernames in elasticsearch.
	elasticsearchUsernamePrefix := os.Getenv("TENANT_ID")
	if elasticsearchUsernamePrefix != "" {
		elasticsearchUsernamePrefix = strings.TrimSuffix(elasticsearchUsernamePrefix, "-") + "-"
	}
	return &kibanaLoginHandler{
		k8sCli:                      k8sCli,
		kibanaCli:                   kibanaCli,
		oidcAuthEnabled:             oidcAuthEnabled,
		oidcAuthIssuer:              oidcAuthIssuer,
		esLicense:                   esLicense,
		elasticsearchUsernamePrefix: elasticsearchUsernamePrefix,
	}
}

// kibanaErrorResponse is the structure errors are unmarshalled into when Kibana returns an error.
type kibanaErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// ServeHTTP attempts to log the OIDC user in the request context (if available) into Kibana using the Elasticsearch
// user that corresponds to the OIDC user in the request context.
//
// ServeHTTP attempts to log the OIDC user into Kibana by sending a login request to the Kibana API. If this login
// request is succesfully, ServeHTTP sets the cookies set in the Kibana response to it's response, then redirects to
// Kibana.
//
// If Dex is not enabled, the OIDC issuer is not Dex, or the Elasticsearch license is no the basic license ServeHTTP
// redirects to the Kibana login without attempting to login.
//
// If an error occurs ServeHTTP redirects the user back to the Manager and sets the `errorStatus` and `errorMessage`
// query parameters.
func (handler *kibanaLoginHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.WithFields(log.Fields{
		"OIDC Auth Enabled":     handler.oidcAuthEnabled,
		"OIDC Auth Issuer":      handler.oidcAuthIssuer,
		"Elasticsearch License": handler.esLicense,
	}).Debug("ServeHTTP called")

	usr, ok := request.UserFrom(req.Context())
	if !ok {
		log.Error("user not found in request context")
		redirectToDashboardWithError(w, req, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	log.WithField("user", usr).Debug("User retrieved from context.")

	oidcUser, err := user.OIDCUserFromUserInfo(usr)
	if err != nil {
		log.WithError(err).Error("failed to get the OIDC user from the user info")
		redirectToDashboardWithError(w, req, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// If OIDC Auth is enabled and the Elasticsearch license is basic then attempt to log into Kibana.
	// If any one of these conditions are false redirect the user to Kibana without logging in.
	if handler.oidcAuthEnabled && oidcUser.Issuer == handler.oidcAuthIssuer && handler.esLicense == ElasticsearchLicenseTypeBasic {
		log.Debugf("Attempting to log user %s into Kibana.", oidcUser.Username)

		credsSecret, err := handler.k8sCli.CoreV1().Secrets(ElasticsearchNamespace).Get(context.Background(), OIDCUsersElasticsearchCredentialsSecret, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Error("failed to get elasticsearch password")
			redirectToDashboardWithError(w, req, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		pwdBytes, exists := credsSecret.Data[oidcUser.Base64EncodedSubjectID()]
		if !exists {
			log.Warnf("User %s doesn't exist or doesn't have permission to access Kibana.", oidcUser.Username)
			redirectToDashboardWithError(w, req, http.StatusForbidden, "User doesn't exist or doesn't have permission to access Kibana")
			return
		}
		esPassword := string(pwdBytes)
		esUsername := fmt.Sprintf("%s%s", handler.elasticsearchUsernamePrefix, oidcUser.Base64EncodedSubjectID())

		kibanaResponse, err := handler.kibanaCli.Login(fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host),
			esUsername, esPassword)
		if err != nil {
			log.WithError(err).Error("failed to log into Kibana")
			redirectToDashboardWithError(w, req, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		if kibanaResponse.StatusCode != http.StatusOK {
			body, err := io.ReadAll(kibanaResponse.Body)
			if err != nil {
				log.WithError(err).Error("failed to parse Kibana response body")
				redirectToDashboardWithError(w, req, http.StatusInternalServerError, "Internal Server Error")
				return
			}

			kibanaResponseObj := kibanaErrorResponse{}
			if err := json.Unmarshal(body, &kibanaResponseObj); err != nil {
				log.WithError(err).Error("failed to parse response body into Kibana response object")
				redirectToDashboardWithError(w, req, http.StatusInternalServerError, "Internal Server Error")
				return
			}

			redirectToDashboardWithError(w, req, kibanaResponseObj.StatusCode, kibanaResponseObj.Message)
			return
		}

		for _, cookie := range kibanaResponse.Cookies() {
			http.SetCookie(w, cookie)
		}
	}

	http.Redirect(w, req, kibanaURL, http.StatusFound)
}

// redirectToDashboardWithError redirects the user to the Manager dashboard with the given errorCode and errorMessage
// set as the `errorCode` and `errorMessage` parameters
func redirectToDashboardWithError(w http.ResponseWriter, req *http.Request, errorCode int, errorMessage string) {
	url := fmt.Sprintf("%s?errorCode=%d&errorMessage=%s", dashboardURL, errorCode, errorMessage)
	http.Redirect(w, req, url, http.StatusFound)
}
