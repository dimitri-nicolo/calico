// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package auth

import (
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	authn "k8s.io/api/authentication/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// Authenticator authenticates a token to a User
type Authenticator interface {
	Authenticate(token string) (*User, error)
}

// BearerAuthenticator is a wrapper that authenticates Bearer tokens against K8s Api
type BearerAuthenticator struct {
	k8sAPI k8s.Interface
}

// NewBearerAuthenticator creates a Bearer Authenticator
func NewBearerAuthenticator(client k8s.Interface) *BearerAuthenticator {
	return &BearerAuthenticator{client}
}

// Authenticate attempts to authenticate a Bearer token to a known User against K8s Api
func (id BearerAuthenticator) Authenticate(token string) (*User, error) {
	if len(token) == 0 {
		return nil, errors.New("Invalid token")
	}

	review := &authn.TokenReview{
		Spec: authn.TokenReviewSpec{
			Token: token,
		}}

	result, err := id.k8sAPI.AuthenticationV1().TokenReviews().Create(review)
	if err != nil {
		return nil, err
	}

	if result.Status.Authenticated {
		user := &User{Name: result.Status.User.Username, Groups: result.Status.User.Groups}
		log.Debugf("User was authenticated as %v", user)
		return user, nil
	}

	return nil, errors.New("Token does not authenticate the user")
}

// BasicAuthenticator is a wrapper that authenticates Basic tokens
type BasicAuthenticator struct {
}

// Authenticate attempts to authenticate a Basic token to a User
func (id BasicAuthenticator) Authenticate(token string) (*User, error) {
	userPwd, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	slice := strings.Split(string(userPwd), ":")
	if len(slice) != 2 {
		return nil, errors.New("Could not parse basic token")
	}

	user := &User{Name: slice[0], Groups: []string{}}
	log.Debugf("User was authenticated as %v", user)
	return user, nil
}
