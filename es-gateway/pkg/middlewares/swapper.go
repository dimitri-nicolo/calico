// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/projectcalico/calico/es-gateway/pkg/cache"
)

const (
	// We expect any ES user attached to a request to have a matching "secure" ES user with actual ES permissions.
	// We also expect that the ES credentials for the "secure" user can be found be looking for a secret named
	// using the username + this suffix.
	ElasticsearchCredsSecretSuffix = "elasticsearch-access-gateway"
	ESGatewayPasswordSecretSuffix  = "gateway-verification-credentials"
	// Below are the expected fields within the data section of an ES credential K8s secret.
	SecretDataFieldUsername    = "username"
	SecretDataFieldPassword    = "password"
	SecretDataFieldClusterName = "cluster_name"
)

// swapElasticCredHandler returns an HTTP handler which acts as a middleware to swap the credentials attached to
// a request (from gateway credentials to Elasticsearch credentials).
func swapElasticCredHandler(c cache.SecretsCache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Attempting ES credentials swap for request with URI %s", r.RequestURI)

		// This should never happen, but is possible if you mistakenly apply this handler in a chain to a route
		// without having the elasticAuthHandler applied first (it adds the user to the context).
		user, ok := r.Context().Value(ESUserKey).(*User)
		if !ok {
			log.Error("unable to authenticate user: ES user cannot be pulled from context (this is a logical bug)")
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		// Attempt to lookup a credentials for matching ES user (i.e. can be used with ES API) that matches to the current user.
		secretName := fmt.Sprintf("%s-%s", user.Username, ElasticsearchCredsSecretSuffix)
		username, password, clusterID, err := getPlainESCredentials(c, secretName)
		if err != nil {
			log.Errorf("unable to authenticate user: %s", err)
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}
		// Set swapped ES user credentials on the request
		r.SetBasicAuth(username, password)
		log.Debugf("Found ES credentials for real user [%s] for request with URI %s", username, r.RequestURI)

		// Add the clusterID to the context for other handlers to read.
		ctx := r.Context()
		ctx = context.WithValue(ctx, ClusterIDKey, clusterID)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// NewSwapElasticCredMiddlware returns an initialized version of elasticAuthHandler.
func NewSwapElasticCredMiddlware(c cache.SecretsCache) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return swapElasticCredHandler(c, h)
	}
}

// getPlainESCredentials attempts to retrieve credentials from the given secretName using the provided k8s client for given request.
func getPlainESCredentials(c cache.SecretsCache, secretName string) (string, string, string, error) {
	secret, err := c.GetSecret(secretName)
	if err != nil {
		return "", "", "", err
	}
	// Extract the username and password from the alternate ES credential secret
	data := secret.Data
	username, usernameFound := data[SecretDataFieldUsername]
	if !usernameFound {
		return "", "", "", fmt.Errorf("k8s secret did not contain username field")
	}
	password, passwordFound := data[SecretDataFieldPassword]
	if !passwordFound {
		return "", "", "", fmt.Errorf("k8s secret did not contain username field")
	}
	clusterName, ok := data[SecretDataFieldClusterName]
	var clusterNameStr string
	if ok {
		clusterNameStr = string(clusterName)
	}

	return string(username), string(password), clusterNameStr, nil
}
