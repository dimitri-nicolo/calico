// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/projectcalico/calico/es-gateway/pkg/cache"
)

type GatewayContextKey string

const (
	ESUserKey    GatewayContextKey = "esUser"
	ClusterIDKey GatewayContextKey = "clusterID"
)

// User contains the revelant user metadata we want to pass between middlewares.
type User struct {
	Username string
}

// elasticAuthHandler returns an HTTP handler which acts as a middleware to authenticate a request against
// Elasticsearch. If authentication is successful, then we set the user info into the request context for
// later use.
func elasticAuthHandler(c cache.SecretsCache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Attempting ES authentication for request with URI %s", r.RequestURI)
		var err error

		authValue, ok := r.Header["Authorization"]
		if !ok || len(authValue) == 0 {
			log.Error("unable to authenticate user: Authorization header not found")
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		username, password, err := extractCredentials(authValue[0])
		if err != nil {
			log.Error("unable to authenticate user: Authorization header contains invalid value")
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		// Authenticate user credentials by comparing with secret containing gateway ES credentials.
		secretName := fmt.Sprintf("%s-%s", username, ESGatewayPasswordSecretSuffix)
		hashed, err := getHashedESCredentials(c, secretName)
		if err != nil {
			log.Errorf("unable to authenticate user: %s", err)
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		// Credentials on the request must match the expected hashed value from the secret
		err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
		if err != nil {
			log.Errorf("unable to authenticate user: %s", err)
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		// Is authenticate call was successful, load the user into the request context for later use (credential swapping).
		reqContext := context.WithValue(r.Context(), ESUserKey, &User{Username: username})

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r.WithContext(reqContext))
	})
}

// NewAuthMiddleware returns an initialized version of elasticAuthHandler.
func NewAuthMiddleware(c cache.SecretsCache) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return elasticAuthHandler(c, h)
	}
}

// extractCredentials attempts to parse out the username and password from the given hash value.
func extractCredentials(value string) (string, string, error) {
	hashValue := strings.TrimPrefix(value, "Basic ")
	b, err := base64.StdEncoding.DecodeString(hashValue)
	if err != nil {
		return "", "", err
	}
	val := strings.Split(string(b), ":")
	if len(val) != 2 {
		return "", "", errors.New("could not extract username & password: invalid format")
	}
	username := val[0]
	password := val[1]
	return username, password, nil
}

// getHashedESCredentials attempts to retrieve a hashed credentials value from the given secretName using the provided k8s
// client for the given request.
func getHashedESCredentials(c cache.SecretsCache, secretName string) (string, error) {
	secret, err := c.GetSecret(secretName)
	if err != nil {
		return "", err
	}
	// Extract the hashed value representing the credentials from the ES credential secret
	data := secret.Data
	hashed, hashedFound := data[SecretDataFieldPassword]
	if !hashedFound {
		return "", fmt.Errorf("k8s secret did not contain hashed field")
	}

	return string(hashed), nil
}
