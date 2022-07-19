package auth_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/health"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/middleware/auth"
	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/auth/testing"

	"github.com/stretchr/testify/mock"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

const (
	iss = "https://example.com/my-issuer"

	mockADServiceAccountName = "ad-service-account"
	testEndpoint             = "/clusters/cluster_name/models/dynamic/flow/port_scan.models"
)

var _ = Describe("AD API Auth test", func() {

	var (
		jwtAuth    lmaauth.JWTAuth
		mAuth      *mockAuth
		fakeK8sCli *fake.Clientset
		jwt        = testing.NewFakeJWT(iss, mockADServiceAccountName)
		userInfo   = &user.DefaultInfo{
			Name: "default",
		}

		testNamespace = "test-namespace"
	)

	BeforeEach(func() {
		mAuth = &mockAuth{}
		fakeK8sCli = new(fake.Clientset)

		var err error
		jwtAuth, err = lmaauth.NewJWTAuth(&rest.Config{BearerToken: jwt.ToString()}, fakeK8sCli, lmaauth.WithAuthenticator(iss, mAuth))
		Expect(err).NotTo(HaveOccurred())
	})

	It("passes the request to the next handler if a valid token is provided", func() {
		req, _ := http.NewRequest("GET", testEndpoint, nil)
		req.Header.Set("Authorization", jwt.BearerTokenHeader())

		mAuth.On("Authenticate", req).Return(userInfo, 200, nil)
		addAccessReviewsReactor(fakeK8sCli, true, userInfo)

		var requestReceived *http.Request
		spyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = r
		})

		authMiddleware := auth.Auth(spyHandler, jwtAuth, testNamespace)
		w := httptest.NewRecorder()
		authMiddleware.ServeHTTP(w, req)

		// request is passed to next handler
		Expect(requestReceived).ToNot(BeNil())
		Expect(requestReceived.Method).To(Equal("GET"))
		Expect(requestReceived.URL.Path).To(Equal(testEndpoint))
	})

	It("allows unauthenticated requests for public endpoint - health", func() {
		var requestReceived *http.Request
		spyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = r
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", health.HealthPath, nil)
		authMiddleware := auth.Auth(spyHandler, jwtAuth, testNamespace)

		authMiddleware.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(200))
		Expect(requestReceived).ToNot(BeNil())
		Expect(requestReceived.Method).To(Equal("GET"))
		Expect(requestReceived.URL.Path).To(Equal(health.HealthPath))
	})

	It("blocks unauthenticated requests", func() {
		var requestReceived *http.Request
		spyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = r
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", testEndpoint, nil)
		authMiddleware := auth.Auth(spyHandler, jwtAuth, testNamespace)

		authMiddleware.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(401))
		Expect(requestReceived).To(BeNil())
	})

	It("blocks unauthorized requests", func() {
		req, _ := http.NewRequest("GET", testEndpoint, nil)
		req.Header.Set("Authorization", jwt.BearerTokenHeader())

		mAuth.On("Authenticate", req).Return(userInfo, 200, nil)
		addAccessReviewsReactor(fakeK8sCli, false, userInfo)

		var requestReceived *http.Request
		spyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = r
		})

		authMiddleware := auth.Auth(spyHandler, jwtAuth, testNamespace)
		w := httptest.NewRecorder()
		authMiddleware.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(403))
		Expect(requestReceived).To(BeNil())
	})
})

type mockAuth struct {
	mock.Mock
}

func (m *mockAuth) Authenticate(r *http.Request) (user.Info, int, error) {
	args := m.Called(r)
	err := args.Get(2)
	if err != nil {
		return nil, args.Get(1).(int), err.(error)
	}
	return args.Get(0).(user.Info), args.Get(1).(int), nil
}

func addAccessReviewsReactor(fakeK8sCli *fake.Clientset, authorized bool, userInfo user.Info) {
	fakeK8sCli.AddReactor("create", "subjectaccessreviews",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			extra := make(map[string]authzv1.ExtraValue)
			for k, v := range userInfo.GetExtra() {
				extra[k] = v
			}
			createAction, ok := action.(k8stesting.CreateAction)
			Expect(ok).To(BeTrue())
			review, ok := createAction.GetObject().(*authzv1.SubjectAccessReview)

			Expect(ok).To(BeTrue())
			Expect(review.Spec.User).To(Equal(userInfo.GetName()))
			Expect(review.Spec.UID).To(Equal(userInfo.GetUID()))
			Expect(review.Spec.Groups).To(Equal(userInfo.GetGroups()))
			Expect(review.Spec.Extra).To(Equal(extra))
			Expect(review.Spec.ResourceAttributes.Resource).To(Equal("models"))
			Expect(review.Spec.ResourceAttributes.Verb).To(Equal("get"))
			return true, &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: authorized}}, nil
		})
}
