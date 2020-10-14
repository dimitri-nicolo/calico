// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/tigera/apiserver/pkg/authentication"

	log "github.com/sirupsen/logrus"

	authzv1 "k8s.io/api/authorization/v1"
	k8suser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
)

type K8sAuthInterface interface {
	KubernetesAuthn(http.Handler) http.Handler
	Authenticate(*http.Request) (*http.Request, int, error)
	Authorize(*http.Request) (int, error)
	KubernetesAuthnAuthz(http.Handler) http.Handler
}

// Returns a K8sAuthInterface. For the default AuthnClient, see DefaultAuthnClient().
func NewK8sAuth(k k8s.Interface, authenticator authentication.Authenticator) K8sAuthInterface {
	return &k8sauth{k8sApi: k, authenticator: authenticator}
}

type k8sauth struct {
	k8sApi        k8s.Interface
	authenticator authentication.Authenticator
}

// The handler returned by this will authenticate and authorize the request
// passed to the handler based off the Authorization header and a
// ResourceAttribute on the context of the request. Upon successful authn/authz
// the handler passed in will be called, otherwise the ResponseWriter will be
// updated with the appropriate status and a message with details.
func (ka *k8sauth) KubernetesAuthnAuthz(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req, stat, err := ka.Authenticate(req)
		if err != nil {
			log.WithError(err).Debug("Kubernetes authn failure")
			http.Error(w, err.Error(), stat)
			return
		}

		stat, err = ka.Authorize(req)
		if err != nil {
			log.WithError(err).Debug("Kubernetes auth failure")
			http.Error(w, err.Error(), stat)
			return
		}
		h.ServeHTTP(w, req)
	})
}

// The handler returned by this will authenticate the request passed
// to the handler based off the Authorization header. Upon successful
// authn the handler passed in will be called, otherwise the
// ResponseWriter will be updated with the appropriate status and a
// message with details.
func (ka *k8sauth) KubernetesAuthn(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req, stat, err := ka.Authenticate(req)
		if err != nil {
			log.WithError(err).Debug("Kubernetes auth failure")
			http.Error(w, err.Error(), stat)
			return
		}
		h.ServeHTTP(w, req)
	})
}

// This method should be called after a request has been successfully authenticated, which has resulted in adding the
// user information to the context.
//
// The  User, Resource and NonResource attributes are taken from the context as the basis for a SubjectAccessReview to
// the k8s-apiserver.
func (ka *k8sauth) Authorize(req *http.Request) (status int, err error) {
	info, ok := request.UserFrom(req.Context())
	if !ok {
		// Should never be possible to happen, unless authn was skipped.
		return http.StatusUnauthorized, fmt.Errorf("invalid or missing user authentication")
	}
	// User info was present in request context. This means the user has already been authenticated and we just
	// need to authorize.
	log.Debugf("User has been authenticated, the user is: %v", info.GetName())

	return ka.authorizeUser(req, info)
}

// This method authenticates the user by making a call to the tigera-apiserver. If successful, an updated request is
// returned which has user information added to the context.
func (ka *k8sauth) Authenticate(req *http.Request) (r *http.Request, status int, err error) {
	return authentication.AuthenticateRequest(ka.authenticator, req)
}

// authorizeUser will check that the user passed in is authorized to access the
// ResourceAttributes attached to the context of the request.
// If there is a failure status will be set and an appropriate error, otherwise
// err is nil.
func (ka *k8sauth) authorizeUser(req *http.Request, user k8suser.Info) (status int, err error) {
	res, resOK := FromContextGetReviewResource(req.Context())
	nonRes, nonResOK := FromContextGetReviewNonResource(req.Context())
	// Continue only if we have at least one resource or non-resource attribute to check.
	if !resOK && !nonResOK {
		return http.StatusForbidden, fmt.Errorf("no resource available to authorize")
	}
	return ka.subjectAccessReview(res, nonRes, user)
}

// subjectAccessReview authorizes that the user has permission to access the resource.
func (ka *k8sauth) subjectAccessReview(resource *authzv1.ResourceAttributes, nonResource *authzv1.NonResourceAttributes, user k8suser.Info) (status int, err error) {
	sar := authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			ResourceAttributes:    resource,
			NonResourceAttributes: nonResource,
			User:                  user.GetName(),
			Groups:                user.GetGroups(),
			Extra:                 make(map[string]authzv1.ExtraValue),
			UID:                   user.GetUID(),
		},
	}
	for k, v := range user.GetExtra() {
		sar.Spec.Extra[k] = authzv1.ExtraValue(v)
	}
	var res *authzv1.SubjectAccessReview
	res, err = ka.k8sApi.AuthorizationV1().SubjectAccessReviews().Create(&sar)
	if err != nil {
		log.Debugf("Response to access review: %v", res.Status)
		return http.StatusInternalServerError, fmt.Errorf("error performing AccessReview: %v", err)
	}
	log.Debugf("Response to access review: %v", res.Status)

	if res.Status.Allowed {
		return 0, nil
	}
	return http.StatusForbidden, fmt.Errorf("AccessReview Status %#v", res.Status)
}

// Not exported to avoid collisions.
type contextKey int

const (
	ResourceAttributeKey contextKey = iota
	NonResourceAttributeKey
)

func NewContextWithReviewResource(
	ctx context.Context,
	ra *authzv1.ResourceAttributes,
) context.Context {
	return context.WithValue(ctx, ResourceAttributeKey, ra)
}

func NewContextWithReviewNonResource(
	ctx context.Context,
	ra *authzv1.NonResourceAttributes,
) context.Context {
	return context.WithValue(ctx, NonResourceAttributeKey, ra)
}

func FromContextGetReviewResource(ctx context.Context) (*authzv1.ResourceAttributes, bool) {
	ra, ok := ctx.Value(ResourceAttributeKey).(*authzv1.ResourceAttributes)
	return ra, ok
}

func FromContextGetReviewNonResource(ctx context.Context) (*authzv1.NonResourceAttributes, bool) {
	nra, ok := ctx.Value(NonResourceAttributeKey).(*authzv1.NonResourceAttributes)
	return nra, ok
}

// aggregateAuthenticator will authenticate the provided authenticator args in order. If an authenticator returns an
// HTTP 421 misdirected error code, it tries the next, until it it reaches an authenticator that can authenticate
// the authorization header.
type aggregateAuthenticator struct {
	authenticators []authentication.Authenticator
}

// Authenticate will authenticate based on its provided authenticators. If an authenticator returns an HTTP 421 misdirected
// error code, it tries the next, until it it reaches an authenticator that can authenticate the authorization header.
func (a *aggregateAuthenticator) Authenticate(token string) (k8suser.Info, int, error) {
	if a.authenticators == nil || len(a.authenticators) == 0 {
		return nil, 500, errors.New("authenticator was not configured correctly")
	}
	for _, auth := range a.authenticators {
		if auth != nil {
			usr, stat, err := auth.Authenticate(token)
			if stat != 421 {
				return usr, stat, err
			}
		}
	}
	return nil, 401, errors.New("no authenticator can authenticate user")
}

// NewAggregateAuthenticator will create an authenticator that combines multiple authenticators into one.
func NewAggregateAuthenticator(authenticators ...authentication.Authenticator) authentication.Authenticator {
	var auths []authentication.Authenticator
	for _, a := range authenticators {
		if a != nil {
			auths = append(auths, a)
		}
	}
	return &aggregateAuthenticator{auths}
}
