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

	requestAndCheckResult := func(path string, expectedStatusCode int, expectedTarget string) {
		client := proxyServer.Client()
		proxyServerURL, err := url.Parse(proxyServer.URL)
		Expect(err).ShouldNot(HaveOccurred())
		proxyServerURL.Path = path
		resp, err := client.Get(proxyServerURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resp.StatusCode).Should(Equal(expectedStatusCode))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(expectedTarget))
	}
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

		By("Requesting an available resource should return a 200 OK")
		requestAndCheckResult("/test200", 200, targetName)

		By("Requesting an non-existent resource should return the original 404 back")
		requestAndCheckResult("/test123", 404, "")

		By("Requesting an available resource but a bad request should return the errored 400 back")
		requestAndCheckResult("/test400", 400, "")
	})
	It("should respond with 502 bad gateway when the server isn't available", func() {
		By("Stopping the target server")
		target.Close()

		By("Requesting an available resource however should return a 502 Bad Gateway")
		requestAndCheckResult("/test200", 502, "")
	})
	AfterEach(func() {
		proxyServer.Close()
		proxyServer = nil
		target.Close()
		target = nil
	})
})
