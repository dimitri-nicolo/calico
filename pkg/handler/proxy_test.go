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
	var proxyServer, target *httptest.Server

	BeforeEach(func() {
		testmux := http.NewServeMux()
		testmux.Handle("/test200", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Target-Name", targetName)
			w.WriteHeader(200)
			w.Write([]byte(targetName))
		}))
		testmux.Handle("/test400", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			w.Header().Add("X-Target-Name", targetName)
		}))
		target = httptest.NewUnstartedServer(testmux)
		targetURLStr := "http://" + target.Listener.Addr().String()
		targetURL, err := url.Parse(targetURLStr)
		Expect(err).ShouldNot(HaveOccurred())
		proxyServer = httptest.NewServer(handler.NewProxy(targetURL))
	})
	It("should forward requests to the target server when the target is available", func() {
		By("Starting the target server")
		target.Start()
		client := proxyServer.Client()
		proxyServerURL, err := url.Parse(proxyServer.URL)
		Expect(err).ShouldNot(HaveOccurred())

		By("Requesting an available resource should return a 200 OK")
		proxyServerURL.Path = "/test200"
		resp, err := client.Get(proxyServerURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(200))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(targetName))

		By("Requesting an non-existent resource should return the original 404 back")
		proxyServerURL.Path = "/test123"
		resp, err = client.Get(proxyServerURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(404))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(""))

		By("Requesting an available resource but a bad request should return the errored 400 back")
		proxyServerURL.Path = "/test400"
		resp, err = client.Get(proxyServerURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(400))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(""))
	})
	It("should respond with 502 bad gateway when the server isn't available", func() {
		By("Not stopping the target server")
		target.Close()
		client := proxyServer.Client()
		proxyServerURL, err := url.Parse(proxyServer.URL)
		Expect(err).ShouldNot(HaveOccurred())

		By("Requesting an available resource however should return a 502 Bad Gateway")
		proxyServerURL.Path = "/test200"
		resp, err := client.Get(proxyServerURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(502))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(""))
	})
	AfterEach(func() {
		proxyServer.Close()
		proxyServer = nil
		target.Close()
		target = nil
	})
})
