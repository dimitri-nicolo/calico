// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package auth identifies a user against the Kubernetes API
// The identification will done by extracting tokens from an HTTP request
// The authentication will be done by validation of the user identity with Kubernetes API
// A User is identified by its name and the groups it belongs to
package auth

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
func NewIdentity() (*Identity, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the client for k8s
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// creates the authenticators Basic and Bearer
	authenticators := make(map[Token]Authenticator)
	authenticators[Basic] = &BasicAuthenticator{}
	authenticators[Bearer] = NewBearerAuthenticator(client)

	return &Identity{authenticators}, nil

}

// Authenticate authenticates a User based on tokens from the http request and validated
// against k8s api. The following types of authentication are supported: Basic and Bearer
// If a User cannot be authenticated it will return an error
func (id *Identity) Authenticate(r *http.Request) (*User, error) {
	token, tokenType := Extract(r)
	authenticator, ok := id.authenticators[tokenType]
	if !ok {
		return nil, errors.New("Token type not supported")
	}

	return authenticator.Authenticate(token)
}
