package test

import (
	"net/http"

	authn "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	// Jane is a generic username to be used in testing
	Jane = "jane"
	// Developers is a generic group name to be used in testing
	Developers = "developers"
	// JaneBearerToken is the Bearer token associated with Jane
	JaneBearerToken = "Bearer jane'sToken"
	// BobBearerToken is the Bearer token associated with Jane
	BobBearerToken = "Bearer bob'sToken"
)

// AddJaneIdentity mocks k8s authentication response for Jane
// Expect username to match Jane and groups to match Developers
func AddJaneIdentity(client *fake.Clientset) {
	client.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		review := &authn.TokenReview{
			Spec: authn.TokenReviewSpec{
				Token: JaneBearerToken,
			},
			Status: authn.TokenReviewStatus{
				Authenticated: true,
				User: authn.UserInfo{
					Username: Jane,
					Groups:   []string{Developers},
				},
			},
		}
		return true, review, nil
	})
}

// AddBobIdentity mocks k8s authentication response for Bob
// Expect user not be authenticated
func AddBobIdentity(client *fake.Clientset) {
	client.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		review := &authn.TokenReview{
			Spec: authn.TokenReviewSpec{
				Token: BobBearerToken,
			},
			Status: authn.TokenReviewStatus{
				Authenticated: false,
			},
		}
		return true, review, nil
	})
}

// AddJaneToken adds JaneBearerToken on the request
func AddJaneToken(req *http.Request) {
	req.Header.Add("Authorization", JaneBearerToken)
}

// AddBobToken adds BobBearerToken on the request
func AddBobToken(req *http.Request) {
	req.Header.Add("Authorization", BobBearerToken)
}
