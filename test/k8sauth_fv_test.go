// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/tigera/es-proxy/pkg/middleware"
	fv "github.com/tigera/es-proxy/test"
	"github.com/tigera/lma/pkg/auth"

	"github.com/projectcalico/apiserver/pkg/authentication"

	authzv1 "k8s.io/api/authorization/v1"
	k8s "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

// HttpHandler to see that the 'next' handler was called or not
type DummyHttpHandler struct {
	serveCalled bool
}

func (dhh *DummyHttpHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	dhh.serveCalled = true
}

var tigeraFlowPath = "/tigera_secure_ee_flows*/_search"
var pathToSomething = "/path/to/something"

func genPath(q string) string {
	return fmt.Sprintf("/%s/_search", q)
}

var _ = Describe("Authenticate against K8s apiserver", func() {
	var k8sClient k8s.Interface
	var dhh *DummyHttpHandler
	var rr *httptest.ResponseRecorder
	var authorizer auth.RBACAuthorizer
	var authenticator authentication.Authenticator

	BeforeEach(func() {
		restCfg := restclient.Config{}
		restCfg.Host = "https://localhost:6443"
		restCfg.Insecure = true
		if restCfg.RateLimiter == nil && restCfg.QPS > 0 {
			restCfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(restCfg.QPS, restCfg.Burst)
		}

		k8sClient = k8s.NewForConfigOrDie(&restCfg)
		Expect(k8sClient).NotTo(BeNil())

		dhh = &DummyHttpHandler{serveCalled: false}
		rr = httptest.NewRecorder()
		authenticator = fv.NewAuthnClient()
		authorizer = auth.NewRBACAuthorizer(k8sClient)
	})
	AfterEach(func() {
	})

	// This is really more of a test that RequestToResource does not add a
	// ResourceAttribute to the context and that K8sAuth interprets that as
	// Forbidden.
	It("Should cause StatusForbidden with valid token but missing URL", func() {
		By("authenticating the token", func() {
			uut := middleware.RequestToResource(
				middleware.AuthenticateRequest(authenticator,
					middleware.AuthorizeRequest(authorizer, dhh)))
			req := &http.Request{Header: http.Header{"Authorization": []string{"Bearer deadbeef"}}}
			uut.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusForbidden), "Token deadbeef authentication failed")
			Expect(dhh.serveCalled).To(BeFalse())
		})
	})

	It("Should cause StatusForbidden with valid basic but user doesn't have SelfSubject RBAC", func() {
		By("authenticating the token", func() {
			req := &http.Request{
				Header: http.Header{
					"Authorization": []string{
						fmt.Sprintf("Basic %s",
							base64.StdEncoding.EncodeToString([]byte("basicusernoselfaccess:basicpwnos")))},
				},
				URL: &url.URL{Path: tigeraFlowPath},
			}

			uut := middleware.RequestToResource(
				middleware.AuthenticateRequest(authenticator,
					middleware.AuthorizeRequest(authorizer, dhh)))
			uut.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusForbidden),
				fmt.Sprintf("Status unexpected with msg: %s", rr.Body.String()))
			Expect(dhh.serveCalled).To(BeFalse())
		})
	})

	DescribeTable("Invalid login causes StatusUnauthorized",
		func(req *http.Request) {
			uut := middleware.RequestToResource(
				middleware.AuthenticateRequest(authenticator,
					middleware.AuthorizeRequest(authorizer, dhh)))
			uut.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusUnauthorized),
				fmt.Sprintf("Message in response writer: %s", rr.Body.String()))
			Expect(dhh.serveCalled).To(BeFalse())
		},

		Entry("Bad basic auth",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s",
						base64.StdEncoding.EncodeToString([]byte("basicuserall:badpw")))},
				},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Bad bearer token",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer d00dbeef"}},
				URL:    &url.URL{Path: tigeraFlowPath},
			}),
	)

	// These test that tokens are mapping to users that have access to certain
	// paths/resources. See the test folder for the users (in *.csv) and roles
	// and bindings for them.
	DescribeTable("Test valid Authorization Headers",
		func(req *http.Request) {
			uut := middleware.RequestToResource(
				middleware.AuthenticateRequest(authenticator,
					middleware.AuthorizeRequest(authorizer, dhh)))
			uut.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK),
				fmt.Sprintf("Should get OK status, message: %s", rr.Body.String()))
			Expect(dhh.serveCalled).To(BeTrue())
		},

		Entry("Allow all token access flow",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef"}},
				URL:    &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Allow all token access audit*",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster.*")},
			}),
		Entry("Allow all token access audit_ee",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit_ee.cluster.*")},
			}),
		Entry("Allow all token access audit_kube",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit_kube.cluster.*")},
			}),
		Entry("Allow all token access events",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_events*")},
			}),
		Entry("Flow token access flow",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeeff"}},
				URL:    &url.URL{Path: tigeraFlowPath},
			}),
		Entry("All Audit token access audit*",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeefaa"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster.*")},
			}),
		Entry("All Audit token access audit_ee",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeefaa"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit_ee.cluster.*")},
			}),
		Entry("Audit kube token access audit_kube",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeefak"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit_kube.cluster.*")},
			}),

		Entry("Allow all basic auth access flow",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Allow all basic auth access audit*",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster.*")},
			}),
		Entry("Allow all basic auth access audit_ee",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit_ee.cluster.*")},
			}),
		Entry("Allow all basic auth access events",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_events*")},
			}),
		Entry("Flow basic auth access flow",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
		Entry("All audit basic auth access audit*",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit_*.cluster.*")},
			}),
		Entry("All audit basic auth access audit_ee",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit_ee.cluster.*")},
			}),
		Entry("Audit kube basic auth access audit_kube",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserall:basicpw")))}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit_kube.cluster.*")},
			}),
		Entry("Allow all basic auth with group binding can access flow",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserallgrp:basicpwgrp")))}},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
	)

	DescribeTable("Test valid Authorization Headers to unauthorized resource causes Forbidden",
		func(req *http.Request) {
			uut := middleware.RequestToResource(
				middleware.AuthenticateRequest(authenticator,
					middleware.AuthorizeRequest(authorizer, dhh)))
			uut.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusForbidden),
				fmt.Sprintf("Should get %d status, message: %s",
					http.StatusForbidden, rr.Body.String()))
			Expect(dhh.serveCalled).To(BeFalse())
		},

		Entry("Token for user tokenuserauditonly try to access flows",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeefaa"}},
				URL:    &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Token with no access (user tokenusernone) try to access flows",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef0"}},
				URL:    &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Token with only audit_kube access try to access audit*",
			&http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeefak"}},
				URL:    &url.URL{Path: genPath("tigera_secure_ee_audit*")},
			}),
		Entry("Basic auth with user basicuserauditonly try to access flows",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{
						fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserauditonly:basicpwaa"))),
					}},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Basic auth with no access (user basicusernone) try to access flows",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{
						fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicusernone:basicpw0"))),
					}},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Basic auth with audit* access try to access flows",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{
						fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserauditonly:basicpwaa"))),
					}},
				URL: &url.URL{Path: tigeraFlowPath},
			}),
		Entry("Basic auth with audit_kube access try to access audit*",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{
						fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserauditkubeonly:basicpwak"))),
					}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit*")},
			}),
		Entry("Basic auth with audit_kube access try to access audit_ee*",
			&http.Request{
				Header: http.Header{
					"Authorization": []string{
						fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicuserauditkubeonly:basicpwak"))),
					}},
				URL: &url.URL{Path: genPath("tigera_secure_ee_audit_ee*")},
			}),
	)

	It("Should cause an InternalServer error when no ResourceAttribute is set on the context", func() {
		By("authorizing the request", func() {
			uut := middleware.AuthenticateRequest(authenticator, middleware.AuthorizeRequest(authorizer, dhh))
			req := &http.Request{
				Header: http.Header{"Authorization": []string{"Bearer deadbeef"}},
				// The URL should not matter but include it anyway to ensure the
				// KubernetesAuthnAuthz does not parse the path.
				URL: &url.URL{Path: tigeraFlowPath},
			}
			uut.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError),
				fmt.Sprintf("The message written to the request writer: %s", rr.Body.String()))
			Expect(dhh.serveCalled).To(BeFalse())
		})
	})

	Context("Test non resource URL", func() {
		DescribeTable("RBAC enforcement on access to non resource URL",
			func(req *http.Request, statusCode int, isServeCalled bool) {
				uut := dummyNonResourceMiddleware(
					middleware.AuthenticateRequest(authenticator,
						middleware.AuthorizeRequest(authorizer, dhh)))
				uut.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(statusCode),
					fmt.Sprintf("Should get %d status, message: %s", statusCode, rr.Body.String()))
				Expect(dhh.serveCalled).To(Equal(isServeCalled))
			},

			Entry("Token for user tokenusernru try to access /path/to/something is allowed",
				&http.Request{
					Header: http.Header{"Authorization": []string{"Bearer deadbeefnru"}},
					URL:    &url.URL{Path: pathToSomething},
				}, http.StatusOK, true),
			Entry("Basic auth for user basicusernonresourceurl try to access /path/to/something is allowed",
				&http.Request{
					Header: http.Header{
						"Authorization": []string{
							fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicusernonresourceurl:basicpwnru"))),
						}},
					URL: &url.URL{Path: pathToSomething},
				}, http.StatusOK, true),
			Entry("Token for user tokenusernone try to access /path/to/something is forbidden",
				&http.Request{
					Header: http.Header{"Authorization": []string{"Bearer deadbeef0"}},
					URL:    &url.URL{Path: pathToSomething},
				}, http.StatusForbidden, false),
			Entry("Basic auth for user basicusernone try to accesss /path/to/something is forbidden",
				&http.Request{
					Header: http.Header{
						"Authorization": []string{
							fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("basicusernone:basicpw0"))),
						}},
					URL: &url.URL{Path: pathToSomething},
				}, http.StatusForbidden, false),
		)
	})

})

func dummyNonResourceMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(w, req.WithContext(auth.NewContextWithReviewNonResource(req.Context(), getNonResourceAttributes(req.URL.Path))))
	})
}

func getNonResourceAttributes(path string) *authzv1.NonResourceAttributes {
	return &authzv1.NonResourceAttributes{
		Verb: "get",
		Path: path,
	}
}
