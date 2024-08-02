package tunnel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/elazarl/goproxy.v1"

	calicoTLS "github.com/projectcalico/calico/crypto/pkg/tls"
)

// The following UTs ensure that the failure cases for proxying are covered by tests. They also ensure that HTTPS
// connection to proxy is under test. The mainline case / happy path is handled by the FV tests.
var _ = Describe("tlsDialViaHTTPProxy", func() {
	var httpProxy, httpsProxy *goproxy.ProxyHttpServer
	var httpProxyServer, httpsProxyServer *http.Server
	var httpProxyURL, httpsProxyURL *url.URL
	var tunnelClientTLSConfig, proxyClientTLSConfig *tls.Config
	var wg sync.WaitGroup
	BeforeEach(func() {
		httpProxy = goproxy.NewProxyHttpServer()
		httpsProxy = goproxy.NewProxyHttpServer()
		tunnelClientTLSConfig = calicoTLS.NewTLSConfig()
		proxyClientTLSConfig = calicoTLS.NewTLSConfig()

		// Silence warnings from connections being closed. The proxy server lib only accepts the unstructured std logger.
		silentLogger := log.New(io.Discard, "", log.LstdFlags)
		httpProxy.Logger = silentLogger
		httpsProxy.Logger = silentLogger

		// Instantiate the HTTP server.
		httpProxyServer = &http.Server{
			Addr:    ":3128",
			Handler: httpProxy,
		}
		httpProxyURL = &url.URL{
			Scheme: "http",
			Host:   "localhost:3128",
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = httpProxyServer.ListenAndServe()
		}()

		// Wait for the server to be ready.
		Eventually(func() error {
			_, err := net.Dial("tcp", "localhost:3128")
			return err
		}).WithTimeout(5 * time.Second).ShouldNot(HaveOccurred())

		// Instantiate the HTTPS server.
		// Create a CA.
		caKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).NotTo(HaveOccurred())
		caTemplate := &x509.Certificate{
			Subject: pkix.Name{
				Organization: []string{"Tigera, Inc."},
			},
			SerialNumber:          big.NewInt(123),
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(1, 0, 0),
			IsCA:                  true,
			BasicConstraintsValid: true,
		}
		caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
		Expect(err).NotTo(HaveOccurred())
		caCert, err := x509.ParseCertificate(caCertBytes)
		Expect(err).NotTo(HaveOccurred())
		certPool := x509.NewCertPool()
		certPool.AddCert(caCert)
		proxyClientTLSConfig.RootCAs = certPool

		// Issue a cert from that CA that has the expected names.
		proxyKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).NotTo(HaveOccurred())
		proxyTemplate := &x509.Certificate{
			Subject: pkix.Name{
				CommonName:   "localhost",
				Organization: []string{"proxy-co"},
			},
			DNSNames:              []string{"localhost"},
			IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
			SerialNumber:          big.NewInt(456),
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(1, 0, 0),
			BasicConstraintsValid: true,
		}
		proxyCertBytes, err := x509.CreateCertificate(rand.Reader, proxyTemplate, caCert, &proxyKey.PublicKey, caKey)
		Expect(err).NotTo(HaveOccurred())
		proxyCert, err := x509.ParseCertificate(proxyCertBytes)
		Expect(err).NotTo(HaveOccurred())

		// Set the CA as a RootCA on the proxyTLSConfig
		httpsProxyServer = &http.Server{
			Addr:    ":3129",
			Handler: httpsProxy,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{proxyCert.Raw, caCert.Raw},
						PrivateKey:  proxyKey,
					},
				},
			},
		}
		httpsProxyURL = &url.URL{
			Scheme: "https",
			Host:   "localhost:3129",
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = httpsProxyServer.ListenAndServeTLS("", "")
		}()

		// Wait for the server to be ready.
		Eventually(func() error {
			// Silence logging, as we'll see handshake failures from opening a TCP connection without a handshake.
			httpsProxyServer.ErrorLog = silentLogger
			_, err := net.Dial("tcp", "localhost:3129")
			httpProxyServer.ErrorLog = nil
			return err
		}).WithTimeout(5 * time.Second).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = httpProxyServer.Close()
		_ = httpsProxyServer.Close()
		wg.Wait()
	})

	It("errors when the scheme of the proxy URL is not http or https", func() {
		_, err := tlsDialViaHTTPProxy(
			newDialer(time.Second),
			"someplace:443",
			&url.URL{
				Scheme: "socks5",
				Host:   "localhost:3128",
			},
			tunnelClientTLSConfig,
			proxyClientTLSConfig,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("socks5"))
	})

	It("errors when the proxy URL is invalid", func() {
		_, err := tlsDialViaHTTPProxy(
			newDialer(time.Second),
			"someplace:443",
			&url.URL{
				Scheme: "http",
				Host:   "http://url-not-host",
			},
			tunnelClientTLSConfig,
			proxyClientTLSConfig,
		)

		Expect(err).To(HaveOccurred())
	})

	for _, tls := range []bool{false, true} {
		It(fmt.Sprintf("errors when the CONNECT request is rejected (tls: %v)", tls), func() {
			var proxyServer *goproxy.ProxyHttpServer
			var proxyURL *url.URL
			if tls {
				proxyServer = httpsProxy
				proxyURL = httpsProxyURL
			} else {
				proxyServer = httpProxy
				proxyURL = httpProxyURL
			}

			// Set up the proxy server to reject CONNECT requests with a 401.
			proxyServer.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				ctx.Resp = &http.Response{
					Status:     http.StatusText(http.StatusUnauthorized),
					StatusCode: http.StatusUnauthorized,
				}
				return goproxy.RejectConnect, host
			}))

			_, err := tlsDialViaHTTPProxy(
				newDialer(time.Second),
				"someplace:443",
				proxyURL,
				tunnelClientTLSConfig,
				proxyClientTLSConfig,
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unauthorized"))
		})
	}

	// The function under test can detect a specific misbehaviour of the proxy: sending data immediately after accepting the CONNECT.
	// However, there are a couple of scenarios in which it __cannot__ reliably detect this misbehaviour:
	// 1. When it reads data from the underlying client connection before the proxy has sent the unexpected data.
	// 2. When it's connection to the proxy is HTTPS, since TLS records are read from the underlying client connection one at a time.
	// In both of these cases, this function only reads the 200 response to the CONNECT from the connection, and therefore does not observe
	// any misbehaviour. In these cases, we still receive a generic error during mTLS handshake failure, but mTLS is not in scope of this test.
	It("errors explicitly when the server continues to send data after it accepts the CONNECT request", func() {
		// Set up a proxy server that writes extra data to the connection after accepting the CONNECT request.
		httpProxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			return &goproxy.ConnectAction{
				Action: goproxy.ConnectHijack,
				Hijack: func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
					n, err := client.Write([]byte("hello, I shouldn't be speaking right now"))
					Expect(n).ToNot(BeZero())
					Expect(err).NotTo(HaveOccurred(), "Failed to write data to hijacked connection")
				},
			}, host
		}))

		var err error
		// Validate that our dial function notices the extra bytes following the 200.
		// We wrap this test in an Eventually in case our first tries hit the race condition described in (1) above.
		// Hitting the race condition is not a problem in practice, it just means we have a less precise error message.
		Eventually(func() string {
			_, err = tlsDialViaHTTPProxy(
				newDialer(time.Second),
				"someplace:443",
				httpProxyURL,
				tunnelClientTLSConfig,
				proxyClientTLSConfig,
			)
			return err.Error()
		}).WithTimeout(5 * time.Second).Should(ContainSubstring("buffered data"))
	})
})

var _ = Describe("GetHTTPProxyURL", func() {
	var originalHTTPProxy, originalHTTPSProxy, originalNoProxy string
	httpProxyHost := "http-proxy"
	httpsProxyHost := "https-proxy"

	BeforeEach(func() {
		originalHTTPProxy = os.Getenv("HTTP_PROXY")
		originalHTTPSProxy = os.Getenv("HTTPS_PROXY")
		originalNoProxy = os.Getenv("NO_PROXY")
		_ = os.Setenv("HTTP_PROXY", "http://"+httpProxyHost)
		_ = os.Setenv("HTTPS_PROXY", "https://"+httpsProxyHost)
	})

	AfterEach(func() {
		_ = os.Setenv("HTTP_PROXY", originalHTTPProxy)
		_ = os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
		_ = os.Setenv("NO_PROXY", originalNoProxy)
	})

	It("returns the HTTPS proxy for a given target", func() {
		url := GetHTTPProxyURL("voltron:9449")
		Expect(url.Scheme).To(Equal("https"))
		Expect(url.Host).To(Equal(httpsProxyHost))
	})

	It("returns the no proxy for a given target if not HTTPS proxy present", func() {
		_ = os.Setenv("HTTPS_PROXY", "")
		url := GetHTTPProxyURL("voltron:9449")
		Expect(url).To(BeNil())
	})

	It("respects NO_PROXY for a given DNS target", func() {
		_ = os.Setenv("NO_PROXY", "voltron,8.8.8.8")
		url := GetHTTPProxyURL("voltron:9449")
		Expect(url).To(BeNil())
	})

	It("respects NO_PROXY for a given IP target", func() {
		_ = os.Setenv("NO_PROXY", "voltron,8.8.8.8")
		url := GetHTTPProxyURL("8.8.8.8:9449")
		Expect(url).To(BeNil())
	})
})
