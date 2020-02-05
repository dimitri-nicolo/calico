// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8suser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type K8sAuthInterface interface {
	KubernetesAuthnAuthz(http.Handler) http.Handler
	Authorize(*http.Request) (int, error)
	KubernetesAuthn(http.Handler) http.Handler
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
//
// If the user has already been authenticated (indicated by the context containing a user info) then this performs
// authorization. Note that basic auth will never have the user info context since we are unable to determine that
// information - in this case this handler will always perform Authn and Authz using a SelfSubjectAccessReview.
//
// For token auth, the request will be updated with the user info so that subsequent handlers will not need to
// re-authenticate.
func (ka *k8sauth) KubernetesAuthnAuthz(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, stat, err := ka.authorize(req)
		if err != nil {
			log.WithError(err).Debug("Kubernetes auth failure")
			http.Error(w, err.Error(), stat)
			return
		}
		if user != nil {
			// If we were able to authenticate a user, update the request to include it in the context.
			req = req.WithContext(request.WithUser(req.Context(), user))
		}
		h.ServeHTTP(w, req)
	})
}

// The handler returned by this will authenticate the request passed
// to the handler based off the Authorization header. Upon successful
// auth the handler passed in will be called, otherwise the
// ResponseWriter will be updated with the appropriate status and a
// message with details.
//
// For token auth, the request will be updated with the user info so
// that subsequent handlers will not need to re-authenticate.
func (ka *k8sauth) KubernetesAuthn(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var user k8suser.Info
		var err error
		var stat int
		token := getAuthToken(req)
		if token != "" {
			user, stat, err = ka.authenticateToken(req)
		} else if usr, pw, found := req.BasicAuth(); found && usr != "" && pw != "" {
			log.Debugf("Will authenticate and authorize user based on basic token")
			user, stat, err = ka.basicAuthenticate(usr, pw)
		}
		if err != nil {
			log.WithError(err).Debug("Kubernetes authn failure")
			http.Error(w, err.Error(), stat)
			return
		}

		if user != nil {
			// If we were able to authenticate a user, update the request to include it in the context.
			req = req.WithContext(request.WithUser(req.Context(), user))
		}
		h.ServeHTTP(w, req)
	})
}

// Authenticate a request based on the basic auth header and return user, status and error,
// if error is nil then the user can be authenticated, otherwise an http Status code is
// returned and an error describing the cause.
// The user groups that the authenticated user belong to cannot be derived by us, so only
// "system:authenticated" will be returned as a group in the userInfo.
func (ka *k8sauth) basicAuthenticate(user, pw string) (k8suser.Info, int, error) {
	client, err := getUserK8sClient(ka.config, user, pw)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	// Create a self access review authn purposes only.
	selfAccessReview := authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{},
		},
	}

	log.Debug("Perform a call to Kube Api server to validate username and password")
	response, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(&selfAccessReview)

	// If a user's password and username do not match with the basic configuration on the kubelet, an error will occur.
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}

	// We are not interested in status.Allowed, since we used an empty SSAR. However,
	// if permission is Denied it means that the user is not authenticated.
	if response.Status.Denied {
		return nil, http.StatusUnauthorized, errors.New("user not authenticated")
	}

	return &k8suser.DefaultInfo{
		Name:   user,
		Groups: []string{"system:authenticated"},
	}, http.StatusOK, nil
}

// For authentication and authorization we handle Token and Basic differently.
// With Token authentication we can use the TokenReview API and then use the
// user info it provides to then do authorization with SubjectAccessReview.
// With Basic authorization we create a k8s client object with the Basic credentials
// and then issue a SelfSubjectAccessReview, this is because there is not a way
// to obtain the groups needed for a SubjectAccessReview.

// Authenticate and authorize a request and return status and error, if error is nil then the
// request is authorized, otherwise an http Status code is returned and an error
// describing the cause.
//
// If a user info is attached to the context then the user has already been authenticated and this method
// will just authorize the request for the attached user.
func (ka *k8sauth) Authorize(req *http.Request) (status int, err error) {
	_, stat, err := ka.authorize(req)
	return stat, err
}

// Authorize a request and return user, status and error, if error is nil then the
// request is authorized, otherwise an http Status code is returned and an error
// describing the cause.
//
// -  If a user info is attached to the context then the user has already been authenticated, we just
//    need to authorize.
// -  For token auth, the user will be authenticated using TokenReview. The user info will be returned if authenticated.
// -  For basic auth, the returned user info will be nil, even if the request is authorized.
func (ka *k8sauth) authorize(req *http.Request) (user k8suser.Info, status int, err error) {
	if ui, ok := request.UserFrom(req.Context()); ok {
		// User info was present in request context. This means the user has already been authenticated and we just
		// need to authorize.
		log.Debugf("User authenticated, just perform authorization")
		status, err := ka.authorizeUser(req, ui)
		return ui, status, err
	}

	// User info was not present in context and so we need to authenticate and authorize. Process here depends on
	// whether token auth or not.
	token := getAuthToken(req)
	if token != "" {
		log.Debugf("Will authenticate user based on bearer token, and then authorize")
		return ka.TokenAuthorize(req)
	} else if _, _, found := req.BasicAuth(); found {
		log.Debugf("Will authenticate and authorize user based on basic token")
		status, err := ka.BasicAuthorize(req)
		return nil, status, err
	}

	return nil, http.StatusUnauthorized, fmt.Errorf("invalid or no user authentication credentials")
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
func (ka *k8sauth) TokenAuthorize(req *http.Request) (user k8suser.Info, status int, err error) {
	user, status, err = ka.authenticateToken(req)
	if err != nil {
		return user, status, err
	}

	status, err = ka.authorizeUser(req, user)
	return user, status, err
}

// authenticateToken will take the token from the req Header and issue a
// TokenReview against the K8s apiserver and return the user info and err of nil,
// if there is a failure status will be set and an appropriate error.
func (ka *k8sauth) authenticateToken(req *http.Request) (u k8suser.Info, status int, err error) {
	tok := getAuthToken(req)
	if tok == "" {
		return nil, http.StatusUnauthorized, fmt.Errorf("no token in request")
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
		return UserInfoToInfo(&result.Status.User), 0, nil
	}

	return nil, http.StatusUnauthorized, fmt.Errorf("token review did not authenticate user: %v", result)
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
		return http.StatusInternalServerError, fmt.Errorf("error performing AccessReview: %v", err)
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
		return http.StatusForbidden, fmt.Errorf("no resource available to authorize")
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

	return "", "", http.StatusUnauthorized, fmt.Errorf("basic authentication not valid format")
}

// getUserK8sClient takes the config passed in, copies it, and adds the user
// and password so the client returned will be acting as the user passed in.
// If successful error is nil and the interface returned will make requests as
// the passed in user, otherwise an appropriate error is returned.
func getUserK8sClient(cfg *restclient.Config, user, pw string) (k8s.Interface, error) {
	usrCfg := restclient.AnonymousClientConfig(cfg)
	usrCfg.Username = user
	usrCfg.Password = pw

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
		return http.StatusUnauthorized, fmt.Errorf("error performing AccessReview: %v", err)
	}

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error performing AccessReview: %v", err)
	}

	if res.Status.Allowed {
		return 0, nil
	}
	return http.StatusForbidden, fmt.Errorf("AccessReview %#v", res.Status)
}

// UserInfoToInfo converts the UserInfo struct found in the k8s APIs to the user.Info interface that is passed around
// in the request contexts.
func UserInfoToInfo(ui *authnv1.UserInfo) k8suser.Info {
	extra := make(map[string][]string)
	for k, v := range ui.Extra {
		extra[k] = authzv1.ExtraValue(v)
	}

	return &k8suser.DefaultInfo{
		Name:   ui.Username,
		Groups: ui.Groups,
		Extra:  extra,
	}
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
