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

	Context("Proxy should proxy a response OK", func() {
		BeforeEach(func() {
			target = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Target-Name", targetName)
				w.WriteHeader(200)
				w.Write([]byte(targetName))
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
	Context("Status code must be sent unchanged by the proxy", func() {
		BeforeEach(func() {
			target = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Target-Name", targetName)
				w.WriteHeader(400)
			}))
			targetURL, err := url.Parse(target.URL)
			Expect(err).ShouldNot(HaveOccurred())
			server = httptest.NewServer(handler.NewProxy(targetURL))
		})
		AfterEach(func() {
			server.Close()
			target.Close()
		})
		It("should not change the response code", func() {
			client := server.Client()
			resp, err := client.Get(server.URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(400))
			Expect(resp.Header.Get("X-Target-Name")).Should(Equal(targetName))
		})
	})
	Context("Proxy should timeout the request when the backend doesn't exist", func() {
		BeforeEach(func() {
			target = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Target-Name", targetName)
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
		It("should respond with 502 bad gateway", func() {
			client := server.Client()
			resp, err := client.Get(server.URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(502))
			Expect(resp.Header.Get("X-Target-Name")).Should(Equal(""))
		})
	})
})
