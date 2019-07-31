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

type k8sFake = fake.Clientset

// K8sFakeClient is the actual client
type K8sFakeClient struct {
	*k8sFake
}

// NewK8sSimpleFakeClient returns a new aggregated fake client that satisfies
// server.K8sClient interface to access k8s
func NewK8sSimpleFakeClient(k8sObj []runtime.Object) *K8sFakeClient {
	return &K8sFakeClient{
		k8sFake: fake.NewSimpleClientset(k8sObj...),
	}
}

// K8sFake returns the Fake struct to acces k8s (re)actions
func (c *K8sFakeClient) K8sFake() *k8stesting.Fake {
	return &c.k8sFake.Fake
}

// AddJaneIdentity mocks k8s authentication response for Jane
// Expect username to match Jane and groups to match Developers
func (c *K8sFakeClient) AddJaneIdentity() {
	c.k8sFake.PrependReactor(
		"create", "tokenreviews",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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
func (c *K8sFakeClient) AddBobIdentity() {
	c.k8sFake.PrependReactor(
		"create", "tokenreviews",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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
