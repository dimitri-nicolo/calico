// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package auth identifies a user against the Kubernetes API
// The identification will done by extracting tokens from an HTTP request
// The authentication will be done by validation of the user identity with Kubernetes API
// A User is identified by its name and the groups it belongs to
package auth

import (
	"fmt"
	"net/http"

	"k8s.io/client-go/rest"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"
)

// User identifies a user by Name and group
type User struct {
	Name   string
	Groups []string
}

func (user *User) String() string {
	return fmt.Sprintf("User%#v", user)
}

// Identity authenticates a User based on a token and validates against k8s api.
// The following types of authentication are supported: Basic and Bearer
type Identity struct {
	authenticators map[Token]Authenticator
}

// NewIdentity creates a new Identity using the in-cluster config for K8S
func NewIdentity(k8sAPI k8s.Interface, config *rest.Config) *Identity {
	// creates the authenticators Basic and Bearer
	authenticators := make(map[Token]Authenticator)
	authenticators[Basic] = NewBasicAuthenticator(&k8sClientGenerator{config: config})
	authenticators[Bearer] = NewBearerAuthenticator(k8sAPI)

	return &Identity{authenticators}

}

// Authenticate authenticates a User based on tokens from the http request and validated
// against k8s api. The following types of authentication are supported: Basic and Bearer
// If a User cannot be authenticated it will return an error
func (id *Identity) Authenticate(r *http.Request) (*User, error) {
	log.Debugf("Will extract token of out request %v", r)
	token, tokenType := Extract(r)
	log.Debugf("Extracted token %v with type %v", token, tokenType)
	authenticator, ok := id.authenticators[tokenType]
	if !ok {
		return nil, errors.New("Token type not supported")
	}

	return authenticator.Authenticate(token)
}
