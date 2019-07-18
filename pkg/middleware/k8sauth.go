// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8s "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type K8sAuthInterface interface {
	KubernetesAuthnAuthz(http.Handler) http.Handler
	Authorize(*http.Request) (int, error)
}

type k8sauth struct {
	k8sApi k8s.Interface
	config *restclient.Config
}

func NewK8sAuth(k k8s.Interface, cfg *restclient.Config) K8sAuthInterface {
	return &k8sauth{k8sApi: k, config: cfg}
}

// The handler returned by this will authenticate and authorize the request
// passed to the handler based off the Authorization header and a
// ResourceAttribute on the context of the request. Upon successful authn/authz
// the handler passed in will be called, otherwise the ResponseWriter will be
// updated with the appropriate status and a message with details.
func (ka *k8sauth) KubernetesAuthnAuthz(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		stat, err := ka.Authorize(req)
		if err != nil {
			log.WithError(err).Debug("Kubernetes auth failure")
			http.Error(w, err.Error(), stat)
			return
		}
		h.ServeHTTP(w, req)
	})
}

// For authentication and authorization we handle Token and Basic differently.
// With Token authentication we can use the TokenReview API and then use the
// user info it provides to then do authorization with SubjectAccessReview.
// With Basic authorization we create a k8s client object with the Basic credentials
// and then issue a SelfSubjectAccessReview, this is because there is not a way
// to obtain the groups needed for a SubjectAccessReview.

// Authorize a request and return status and error, if error is nil then the
// request is authorized, otherwise an http Status code is returned and an error
// describing the cause.
func (ka *k8sauth) Authorize(req *http.Request) (status int, err error) {
	token := getAuthToken(req)
	if token != "" {
		return ka.TokenAuthorize(req)
	} else if _, _, found := req.BasicAuth(); found {
		return ka.BasicAuthorize(req)
	}

	return http.StatusUnauthorized, fmt.Errorf("Invalid or no user authentication credentials")
}

func getAuthToken(req *http.Request) string {
	auth := strings.TrimSpace(req.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	parts := strings.Split(auth, " ")
	if len(parts) < 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	token := parts[1]

	// Empty bearer tokens aren't valid
	if len(token) == 0 {
		return ""
	}

	return token
}

// TokenAuthorize will first authenticate a token in the request header and if
// successful then authorize the user represented by that token for the
// ResourceAttribute that is in the context of the request.
// If either of the above is not successful then an appropriate status and error
// message is returned, with both successful err will be nil.
func (ka *k8sauth) TokenAuthorize(req *http.Request) (status int, err error) {
	var user *authnv1.UserInfo
	user, status, err = ka.authenticateToken(req)
	if err != nil {
		return status, err
	}

	return ka.authorizeUser(req, user)
}

// authenticateToken will take the token from the req Header and issue a
// TokenReview against the K8s apiserver and return the user info and err of nil,
// if there is a failure status will be set and an appropriate error.
func (ka *k8sauth) authenticateToken(req *http.Request) (user *authnv1.UserInfo, status int, err error) {
	tok := getAuthToken(req)
	if tok == "" {
		return nil, http.StatusUnauthorized, fmt.Errorf("No token in request")
	}
	tr := &authnv1.TokenReview{
		Spec: authnv1.TokenReviewSpec{
			Token: tok,
		}}

	result, err := ka.k8sApi.AuthenticationV1().TokenReviews().Create(tr)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("TokenReview failed: %v", err)
	}

	if result.Status.Authenticated {
		return &result.Status.User, 0, nil
	}

	return nil, http.StatusUnauthorized, fmt.Errorf("Token review did not authenticate user: %v", result)
}

// authorizeUser will check that the user passed in is authorized to access the
// ResourceAttributes attached to the context of the request.
// If there is a failure status will be set and an appropriate error, otherwise
// err is nil.
func (ka *k8sauth) authorizeUser(req *http.Request, user *authnv1.UserInfo) (status int, err error) {
	res, resOK := FromContextGetReviewResource(req.Context())
	nonRes, nonResOK := FromContextGetReviewNonResource(req.Context())
	// Continue only if we have at least one resource or non-resource attribute to check.
	if !resOK && !nonResOK {
		return http.StatusForbidden, fmt.Errorf("No resource available to authorize")
	}
	return ka.subjectAccessReview(res, nonRes, user)
}

// subjectAccessReview authorizes that the user has permission to access the resource.
func (ka *k8sauth) subjectAccessReview(resource *authzv1.ResourceAttributes, nonResource *authzv1.NonResourceAttributes, user *authnv1.UserInfo) (status int, err error) {
	sar := authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			ResourceAttributes:    resource,
			NonResourceAttributes: nonResource,
			User:                  user.Username,
			Groups:                user.Groups,
			Extra:                 make(map[string]authzv1.ExtraValue),
			UID:                   user.UID,
		},
	}
	for k, v := range user.Extra {
		sar.Spec.Extra[k] = authzv1.ExtraValue(v)
	}
	var res *authzv1.SubjectAccessReview
	res, err = ka.k8sApi.AuthorizationV1().SubjectAccessReviews().Create(&sar)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error performing AccessReview %v", err)
	}

	if res.Status.Allowed {
		return 0, nil
	}
	return http.StatusForbidden, fmt.Errorf("AccessReview Status %#v", res.Status)
}

// BasicAuthorize will authenticate and authorize the user in the request header
// for the ResourceAttribute that is in the context of the request.
// If both of the above are successful then err will be nil, otherwise
// an appropriate status and error message is returned.
func (ka *k8sauth) BasicAuthorize(req *http.Request) (status int, err error) {
	var user, pw string
	user, pw, status, err = ka.getUserPw(req)
	if err != nil {
		return status, err
	}

	var client k8s.Interface
	client, err = getUserK8sClient(ka.config, user, pw)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	res, resOK := FromContextGetReviewResource(req.Context())
	nonRes, nonResOK := FromContextGetReviewNonResource(req.Context())
	// Continue only if we have at least one resource or non-resource attribute to check.
	if !resOK && !nonResOK {
		return http.StatusForbidden, fmt.Errorf("No resource available to authorize")
	}

	return ka.selfSubjectAccessReview(client, res, nonRes)
}

// getUserPw returns the user and password from the header of the req passed in
// and err is nil, if there was a problem then status is populated and error set
// appropriately.
func (ka *k8sauth) getUserPw(req *http.Request) (user, pw string, status int, err error) {
	if user, pw, found := req.BasicAuth(); found && user != "" && pw != "" {
		return user, pw, 0, nil
	}

	return "", "", http.StatusUnauthorized, fmt.Errorf("Basic authentication not valid format")
}

// getUserK8sClient takes the config passed in, copies it, and adds the user
// and password so the client returned will be acting as the user passed in.
// If successful error is nil and the interface returned will make requests as
// the passed in user, otherwise an appropriate error is returned.
func getUserK8sClient(cfg *restclient.Config, user, pw string) (k8s.Interface, error) {
	usrCfg := restclient.AnonymousClientConfig(cfg)
	usrCfg.Username = user
	usrCfg.Password = pw
	usrCfg.Impersonate = restclient.ImpersonationConfig{}

	return k8s.NewForConfig(usrCfg)
}

// selfSubjectAccessReview does a SelfSubjectAccessReview of the resource passed
// in with the client passed in returning no error in the case of access being
// allowed, otherwise a status code and appropriate error is returned.
func (ka *k8sauth) selfSubjectAccessReview(client k8s.Interface, resource *authzv1.ResourceAttributes, nonResource *authzv1.NonResourceAttributes) (status int, err error) {
	ssar := authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes:    resource,
			NonResourceAttributes: nonResource,
		},
	}
	var res *authzv1.SelfSubjectAccessReview
	res, err = client.AuthorizationV1().SelfSubjectAccessReviews().Create(&ssar)

	if apierrors.IsUnauthorized(err) {
		return http.StatusUnauthorized, fmt.Errorf("Error performing AccessReview %v", err)
	}

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error performing AccessReview %v", err)
	}

	if res.Status.Allowed {
		return 0, nil
	}
	return http.StatusForbidden, fmt.Errorf("AccessReview %#v", res.Status)
}
