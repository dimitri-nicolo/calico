package handler_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/tigera/es-proxy/pkg/handler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const insertedHeaderValue = "This is an test header inserted by a response modifier"

var _ = Describe("Proxy Handler", func() {
	targetName := "target"
	var proxyServer, target *httptest.Server

	requestResult := func(path string) *http.Response {
		client := proxyServer.Client()
		proxyServerURL, err := url.Parse(proxyServer.URL)
		Expect(err).ShouldNot(HaveOccurred())
		proxyServerURL.Path = path
		resp, err := client.Get(proxyServerURL.String())
		Expect(err).ShouldNot(HaveOccurred())
		return resp
	}

	requestAndCheckResult := func(path string, expectedStatusCode int, expectedTarget string) {
		resp := requestResult(path)
		Expect(resp.StatusCode).Should(Equal(expectedStatusCode))
		Expect(resp.Header.Get("X-Target-Name")).Should(Equal(expectedTarget))
	}

	getTestMux := func() *http.ServeMux {
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
		return testmux
	}

	assertsWhenTestHeaderHasBeenAdded := func() {
		It("should respond with a header inserted by the response modifier", func() {
			By("Requesting an available resource should include inserted header.")
			resp := requestResult("/test200")
			Expect(resp.Header.Get("X-TestResponseModified")).Should(Equal(insertedHeaderValue))
		})
	}

	assertsWhenTestHeaderHasNotBeenAdded := func() {
		It("should respond without a header inserted by the response modifier", func() {
			By("Requesting an available resource should not include the inserted header.")
			resp := requestResult("/test200")
			Expect(resp.Header.Get("X-TestResponseModified")).Should(Equal(""))
		})
	}

	assertsWhenTargetAvailable := func(startTLSTarget bool) {
		It("should forward requests to the target server when the target is available", func() {

			By("Requesting an available resource should return a 200 OK")
			requestAndCheckResult("/test200", 200, targetName)

			By("Requesting an non-existent resource should return the original 404 back")
			requestAndCheckResult("/test123", 404, "")

			By("Requesting an available resource but a bad request should return the errored 400 back")
			requestAndCheckResult("/test400", 400, "")
		})
	}
	assertsWhenTargetDown := func(startTLSTarget bool) {
		It("should respond with 502 bad gateway when the server isn't available", func() {
			By("Stopping the target server")
			target.Close()

			By("Requesting an available resource however should return a 502 Bad Gateway")
			requestAndCheckResult("/test200", 502, "")
		})
	}

	hostHeaderChecker := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Per RFC 7230, proxy should set the host header of
			// the request-target.
			By("Checking that the host header is set to the target")
			targetURL, err := url.Parse(target.URL)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(r.Host).Should(Equal(targetURL.Host))
			h.ServeHTTP(w, r)
		})
	}

	nilResponseModifier := func(r *http.Response) error {
		//does not modify the response
		return nil
	}

	insertingResponseModifier := func(r *http.Response) error {
		r.Header.Add("X-TestResponseModified", insertedHeaderValue)
		return nil
	}

	setupServers := func(tlsTarget bool, responseModifier handler.ResponseModifier) {
		testmux := getTestMux()

		var tc *tls.Config
		if tlsTarget {
			target = httptest.NewTLSServer(hostHeaderChecker(testmux))
			pool := x509.NewCertPool()
			pool.AddCert(target.Certificate())
			tc = &tls.Config{
				RootCAs:            pool,
				InsecureSkipVerify: false,
			}
		} else {
			target = httptest.NewServer(hostHeaderChecker(testmux))
		}

		targetURL, err := url.Parse(target.URL)
		Expect(err).ShouldNot(HaveOccurred())

		pc := &handler.ProxyConfig{
			TargetURL:       targetURL,
			TLSConfig:       tc,
			ConnectTimeout:  time.Second,
			KeepAlivePeriod: time.Second,
			IdleConnTimeout: time.Second,
		}
		proxy := handler.NewProxy(pc)

		proxy.AddResponseModifier(responseModifier)

		proxyServer = httptest.NewServer(proxy)
	}

	Context("No TLS", func() {
		BeforeEach(func() {
			setupServers(false, nilResponseModifier)
		})
		assertsWhenTargetAvailable(false)
		assertsWhenTestHeaderHasNotBeenAdded()
		assertsWhenTargetDown(false)
	})
	Context("TLS", func() {
		BeforeEach(func() {
			setupServers(true, insertingResponseModifier)
		})
		assertsWhenTargetAvailable(true)
		assertsWhenTestHeaderHasBeenAdded()
		assertsWhenTargetDown(true)
	})
	AfterEach(func() {
		proxyServer.Close()
		proxyServer = nil
		target.Close()
		target = nil
	})
})
