package middlewares

import (
	"context"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-gateway/pkg/elastic"
)

type GatewayContextKey string

const ESUserKey GatewayContextKey = "esUser"

func elasticAuthHandler(client elastic.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Attempting ES authentication for request with URI %s", r.RequestURI)

		authValue, ok := r.Header["Authorization"]
		if !ok || len(authValue) == 0 {
			log.Error("unable to authenticate user: Authorization header not found")
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		// Make API call to ES to authenticate user credentials
		user, err := client.AuthenticateUser(authValue[0])
		if err != nil {
			log.Errorf("unable to authenticate user: %s", err)
			http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
			return
		}

		// Is authenticate call was successful, load the user into the request context for later use.
		reqContext := context.WithValue(r.Context(), ESUserKey, user)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r.WithContext(reqContext))
	})
}

// NewAuthMiddleware returns an initialized version of elasticAuthHandler.
func NewAuthMiddleware(client elastic.Client) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return elasticAuthHandler(client, h)
	}
}
