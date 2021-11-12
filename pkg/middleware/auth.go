package middleware

//
//import (
//	"context"
//	"fmt"
//	"net/http"
//	"net/url"
//	"strings"
//
//	"github.com/SermoDigital/jose/jws"
//	"github.com/projectcalico/apiserver/pkg/authentication"
//	log "github.com/sirupsen/logrus"
//	authnv1 "k8s.io/api/authentication/v1"
//	authzv1 "k8s.io/api/authorization/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	k8sserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
//	"k8s.io/apiserver/pkg/authentication/user"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/rest"
//)
//
//type AuthNAuthZ interface {
//	// Authenticate checks if a request is authenticated. It accepts only JWT bearer tokens.
//	// If it has impersonation headers, it will also check if the authenticated user is authorized
//	//to impersonate. The resulting user info will be that of the impersonated user.
//	Authenticate(r *http.Request) (userInfo user.Info, httpStatusCode int, err error)
//
//	// Authorize makes a SubjectAccessReview request to k8s in order to check if a user is allowed to access a resource.
//	Authorize(user user.Info, resAtr *authzv1.ResourceAttributes) (authorized bool, err error)
//
//	// AddAuthenticator associates an authenticator with an issuer for incoming bearer tokens.
//	AddAuthenticator(issuer string, authenticator authentication.Authenticator)
//}
//
//type auth struct {
//	k8sCli         kubernetes.Interface
//	k8sAuthn       authentication.Authenticator
//	authenticators map[string]authentication.Authenticator
//}
//
//func (a *auth) AddAuthenticator(issuer string, authenticator authentication.Authenticator) {
//	a.authenticators[issuer] = authenticator
//}
//
//func (a *auth) Authenticate(req *http.Request) (user.Info, int, error) {
//	jwt, err := jws.ParseJWTFromRequest(req)
//	if err != nil {
//		return nil, 401, jws.ErrNoTokenInRequest
//	}
//
//	issuer, ok := jwt.Claims().Issuer()
//	if !ok {
//		return nil, 401, jws.ErrIsNotJWT
//	}
//
//	authHeader := req.Header.Get("Authorization")
//	// Strip the "Bearer " part of the token. We know this is possible, since it has been validated above.
//	token := authHeader[7:]
//
//	authn, ok := a.authenticators[issuer]
//	var userInfo user.Info
//	if ok {
//		usr, stat, err := authn.Authenticate(req.Header.Get("Authorization"))
//		if err != nil {
//			return usr, stat, err
//		}
//		userInfo = usr
//	} else {
//		authn = a.k8sAuthn
//		tknReview, err := a.k8sCli.AuthenticationV1().TokenReviews().Create(
//			context.Background(),
//			&authnv1.TokenReview{
//				Spec: authnv1.TokenReviewSpec{Token: token},
//			},
//			metav1.CreateOptions{})
//
//		if err != nil {
//			return nil, 500, err
//		}
//		if !tknReview.Status.Authenticated {
//			return nil, 401, fmt.Errorf("user is not authenticated")
//		}
//		userInfo = &user.DefaultInfo{
//			Name:   tknReview.Status.User.Username,
//			Groups: tknReview.Status.User.Groups,
//			Extra:  toExtra(tknReview.Status.User.Extra),
//		}
//	}
//
//	// If this user was impersonated, see if the impersonating user is allowed to do so.
//	impersonatedUser, err := a.extractUserFromImpersonationHeaders(req)
//	if err != nil {
//		return nil, 401, err
//	}
//	if impersonatedUser != nil {
//		attributes := a.buildResourceAttributesForImpersonation(impersonatedUser)
//
//		for _, resAtr := range attributes {
//			ok, err = a.Authorize(userInfo, resAtr)
//			if err != nil {
//				return nil, 500, err
//			} else if !ok {
//				return nil, 401, fmt.Errorf("user is not allowed to impersonate")
//			}
//		}
//		userInfo = impersonatedUser
//	}
//	return userInfo, 200, nil
//}
//
//func toExtra(extra map[string]authnv1.ExtraValue) map[string][]string {
//	ret := make(map[string][]string)
//	for k, v := range extra {
//		ret[k] = v
//	}
//	return ret
//}
//
//// NewAuthNAuthZ creates an object adhering to the Auth interface. It can perform authN and authZ.
//func NewAuthNAuthZ() (AuthNAuthZ, error) {
//	restConfig, err := rest.InClusterConfig()
//	if err != nil {
//		return nil, err
//	}
//
//	k8sCli, err := kubernetes.NewForConfig(restConfig)
//	if err != nil {
//		return nil, err
//	}
//
//	k8sAuth, err := authentication.New()
//	if err != nil {
//		return nil, err
//	}
//
//	return &auth{
//		k8sCli:         k8sCli,
//		k8sAuthn:       k8sAuth,
//		authenticators: make(map[string]authentication.Authenticator),
//	}, nil
//}
//
//func (a *auth) extractUserFromImpersonationHeaders(req *http.Request) (user.Info, error) {
//	var userName = req.Header.Get(authnv1.ImpersonateUserHeader)
//	var groups = req.Header[authnv1.ImpersonateGroupHeader]
//	var extras = make(map[string][]string)
//	for headerName, value := range req.Header {
//		if strings.HasPrefix(headerName, authnv1.ImpersonateUserExtraHeaderPrefix) {
//			encodedKey := strings.ToLower(headerName[len(authnv1.ImpersonateUserExtraHeaderPrefix):])
//			extraKey, err := url.PathUnescape(encodedKey)
//			if err != nil {
//				var err = fmt.Errorf("malformed extra key for impersonation request")
//				log.WithError(err).Errorf("Could not decode extra key %s", encodedKey)
//			}
//			extras[extraKey] = value
//		}
//	}
//
//	if len(userName) == 0 && (len(groups) != 0 || len(extras) != 0) {
//		return nil, fmt.Errorf("impersonation headers are missing impersonate user header")
//	}
//
//	if len(userName) != 0 {
//		return &user.DefaultInfo{
//			Name:   userName,
//			Groups: groups,
//			Extra:  extras,
//		}, nil
//	}
//	return nil, nil
//}
//
//func (a *auth) buildResourceAttributesForImpersonation(usr user.Info) []*authzv1.ResourceAttributes {
//	var result []*authzv1.ResourceAttributes
//	namespace, name, err := k8sserviceaccount.SplitUsername(usr.GetName())
//	if err == nil {
//		result = append(result, &authzv1.ResourceAttributes{
//			Verb:      "impersonate",
//			Resource:  "serviceaccounts",
//			Name:      name,
//			Namespace: namespace,
//		})
//	} else {
//		result = append(result, &authzv1.ResourceAttributes{
//			Verb:     "impersonate",
//			Resource: "users",
//			Name:     usr.GetName(),
//		})
//	}
//
//	for _, group := range usr.GetGroups() {
//		result = append(result, &authzv1.ResourceAttributes{
//			Verb:     "impersonate",
//			Resource: "groups",
//			Name:     group,
//		})
//	}
//
//	for key, extra := range usr.GetExtra() {
//		for _, value := range extra {
//			result = append(result, &authzv1.ResourceAttributes{
//				Verb:        "impersonate",
//				Resource:    "userextras",
//				Subresource: key,
//				Name:        value,
//			})
//		}
//	}
//
//	return result
//}
//
//func (a *auth) Authorize(user user.Info, resAtr *authzv1.ResourceAttributes) (bool, error) {
//	sar := authzv1.SubjectAccessReview{
//		Spec: authzv1.SubjectAccessReviewSpec{
//			ResourceAttributes:    resAtr,
//			NonResourceAttributes: nil,
//			User:                  user.GetName(),
//			Groups:                user.GetGroups(),
//			Extra:                 make(map[string]authzv1.ExtraValue),
//			UID:                   user.GetUID(),
//		},
//	}
//	res, err := a.k8sCli.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), &sar, metav1.CreateOptions{})
//	if err != nil {
//		return false, fmt.Errorf("error performing AccessReview: %v", err)
//	}
//
//	return res.Status.Allowed, nil
//}
