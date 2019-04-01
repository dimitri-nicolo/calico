package handler_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/tigera/es-proxy/pkg/handler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Proxy Handler", func() {
	targetName := "target"
	var server, target *httptest.Server

	BeforeEach(func() {
		target = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Target-Name", targetName)
			w.Write([]byte(targetName))
			w.WriteHeader(200)
		}))
		targetURL, err := url.Parse(target.URL)
		Expect(err).ShouldNot(HaveOccurred())
		server = httptest.NewServer(handler.NewProxy(targetURL))
	})
	AfterEach(func() {
		server.Close()
		target.Close()
	})
	It("should forward requests to the target server", func() {
		client := server.Client()
		resp, err := client.Get(server.URL)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(200))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(targetName))
	})
})
