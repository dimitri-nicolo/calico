package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/es-proxy/pkg/middleware"
)

var _ = Describe("Test PathValidator Middleware", func() {
	targetName := "target"
	var (
		target    *httptest.Server
		targetURL *url.URL
		err       error
	)

	BeforeEach(func() {
		paths := map[string]struct{}{
			"/tsee": struct{}{},
		}
		target = httptest.NewServer(middleware.PathValidator(paths, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Target-Name", targetName)
			w.Write([]byte(targetName))
			w.WriteHeader(200)
		})))
		targetURL, err = url.Parse(target.URL)
		Expect(err).ShouldNot(HaveOccurred())
	})
	AfterEach(func() {
		target.Close()
	})
	It("should allow requests to the target server for accepted paths", func() {
		client := target.Client()
		targetURL.Path = "tsee"
		resp, err := client.Get(targetURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(200))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(targetName))
	})
	It("should reject requests for paths that should not handled", func() {
		client := target.Client()
		targetURL.Path = "cnx"
		resp, err := client.Get(targetURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(400))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(""))
	})
})
