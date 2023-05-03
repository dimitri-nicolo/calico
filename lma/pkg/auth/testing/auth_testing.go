// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package testing

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	. "github.com/onsi/gomega"
	authzv1 "k8s.io/api/authorization/v1"

	authnv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/projectcalico/calico/lma/pkg/auth"
)

// The code in this file is meant to be used in (unit) tests that are related to authN.

const (
	// DefaultJWTHeader Typical rsa256 header.
	DefaultJWTHeader = "eyJhbGciOiJSUzI1NiIsImtpZCI6Ijk3ODM2YzRiMjdmN2M3ZmVjMjk1MTk0NTFkNDc5MmUyNjQ4M2RmYWUifQ"
)

// NewFakeJWT Convenience method for creating bearer tokens that can then be used for authn in testing. Some common args can be passed,
// while OverrideClaims lets you pretty much override anything.
func NewFakeJWT(issuer, name string) *FakeJWT {
	jwt := &FakeJWT{
		Header: DefaultJWTHeader,
		PayloadMap: map[string]interface{}{
			auth.ClaimNameIss:           issuer,
			auth.ClaimNameSub:           name,
			auth.ClaimNameEmail:         name,
			auth.ClaimNameExp:           9600964803, // Very far in the future
			"iat":                       1600878403,
			"nonce":                     "35e32c66028243f592cc3103c7c2dfb2",
			auth.ClaimNameEmailVerified: true,
			auth.ClaimNameGroups: []string{
				"system:authenticated",
			},
			auth.ClaimNameName: name,
		},
	}
	jwt.refresh()
	return jwt
}

// refresh computes JWT fields.
func (f *FakeJWT) refresh() {
	payloadJSON, _ := json.Marshal(f.PayloadMap)
	f.PayloadJSON = string(payloadJSON)
	f.Payload = base64.RawURLEncoding.EncodeToString(payloadJSON)
}

func (f *FakeJWT) WithClaim(claimName string, claimValue interface{}) *FakeJWT {
	f.PayloadMap[claimName] = claimValue
	f.refresh()
	return f
}

// NewFakeServiceAccountJWT Convenience method for creating bearer tokens associated with service accounts.
func NewFakeServiceAccountJWT() *FakeJWT {
	payload := ServiceAccountPayload{
		Iss:                                  auth.ServiceAccountIss,
		KubernetesIoServiceaccountNamespace:  "tigera-prometheus",
		KubernetesIoServiceaccountSecretName: "default-token-vsznx",
		KubernetesIoServiceaccountServiceAccountName: "default",
		KubernetesIoServiceaccountServiceAccountUid:  "2ae2ecb2-bca6-43ab-8be5-5131c14bb64c",
	}

	payloadJSON, _ := json.Marshal(payload)
	payloadMap := make(map[string]interface{})
	err := json.Unmarshal(payloadJSON, &payloadMap)
	if err != nil {
		panic(err) // should not be possible.
	}
	payloadStr := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return &FakeJWT{
		Header:                DefaultJWTHeader,
		PayloadJSON:           string(payloadJSON),
		PayloadMap:            payloadMap,
		Payload:               payloadStr,
		ServiceAccountPayload: &payload,
	}
}

// FakeJWT is a struct that should help setting up tests that involve authentication. When created using
// NewFakeServiceAccountJWT() or NewFakeJWT(), you can easily use separate parts of a JWT for creating auth headers or
// other test code.
type FakeJWT struct {
	JWT                   string
	Header                string
	Payload               string
	PayloadMap            map[string]interface{}
	PayloadJSON           string
	ServiceAccountPayload *ServiceAccountPayload
	Signature             string
}

// ToString returns a jwt in web safe format.
func (f *FakeJWT) ToString() string {
	return fmt.Sprintf("%s.%s.%s", f.Header, f.Payload, f.Signature)
}

// BearerTokenHeader returns a jwt such that it can be used as an authorization header.
func (f *FakeJWT) BearerTokenHeader() string {
	return fmt.Sprintf("Bearer %s.%s.%s", f.Header, f.Payload, f.Signature)
}

func (f *FakeJWT) UserName() string {
	if f.ServiceAccountPayload != nil {
		return fmt.Sprintf("%s:%s", f.ServiceAccountPayload.KubernetesIoServiceaccountNamespace, f.ServiceAccountPayload.KubernetesIoServiceaccountServiceAccountName)
	}
	return fmt.Sprintf("%s", f.PayloadMap[auth.ClaimNameName])
}

// ServiceAccountPayload models a service account token as issued by Kubernetes.
type ServiceAccountPayload struct {
	Iss                                          string `json:"iss"`
	KubernetesIoServiceaccountNamespace          string `json:"kubernetes.io/serviceaccount/namespace"`
	KubernetesIoServiceaccountSecretName         string `json:"kubernetes.io/serviceaccount/secret.name"`
	KubernetesIoServiceaccountServiceAccountName string `json:"kubernetes.io/serviceaccount/service-account.name"`
	KubernetesIoServiceaccountServiceAccountUid  string `json:"kubernetes.io/serviceaccount/service-account.uid"`
	Sub                                          string `json:"sub"`
}

// SetTokenReviewsReactor adds a reactor to your fake clientset. This helps you to add one or more authenticated users,
// based on their FakeJWT.
func SetTokenReviewsReactor(fakeK8sCli *fake.Clientset, tokens ...*FakeJWT) {
	tokenMap := map[string]*FakeJWT{}
	for _, tkn := range tokens {
		tokenMap[tkn.ToString()] = tkn
	}
	fakeK8sCli.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction, ok := action.(k8stesting.CreateAction)
		Expect(ok).To(BeTrue())
		review, ok := createAction.GetObject().(*authnv1.TokenReview)
		Expect(ok).To(BeTrue())

		token, ok := tokenMap[review.Spec.Token]
		Expect(ok).To(BeTrue(), "Token unknown to token reviews reactor.")

		Expect(review.Spec).To(Equal(authnv1.TokenReviewSpec{
			Token: token.ToString(),
		}))
		return true, &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{User: authnv1.UserInfo{Username: fmt.Sprintf("%v", token.UserName())}, Authenticated: true}}, nil
	})
}

type UserPermissions struct {
	Username string
	Attrs    []authzv1.ResourceAttributes
}

func SetSubjectAccessReviewsReactor(fakeK8sCli *fake.Clientset, userPermissions ...UserPermissions) {
	userMap := map[string][]authzv1.ResourceAttributes{}
	for _, up := range userPermissions {
		userMap[up.Username] = up.Attrs
	}
	fakeK8sCli.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction, ok := action.(k8stesting.CreateAction)
		Expect(ok).To(BeTrue())
		review, ok := createAction.GetObject().(*authzv1.SubjectAccessReview)
		Expect(ok).To(BeTrue())
		Expect(review.Spec.ResourceAttributes).ToNot(BeNil(), "only ResourceAttributes supported currently")

		permittedAttrs, ok := userMap[review.Spec.User]
		Expect(ok).To(BeTrue(), "user unknown to subject access reviews reactor.", review.Spec.User)

		allowed := false
		specAttrs := *review.Spec.ResourceAttributes
		for _, permittedAttr := range permittedAttrs {
			if permittedAttr == specAttrs {
				allowed = true
				break
			}
		}

		return true, &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{
			Allowed: allowed,
			Denied:  !allowed,
		}}, nil
	})
}
