// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.
package fv_test

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/elazarl/goproxy.v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kscheme "k8s.io/client-go/kubernetes/scheme"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/apiserver/pkg/authentication"

	logrus "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/mock"

	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/projectcalico/calico/voltron/internal/pkg/bootstrap"
	"github.com/projectcalico/calico/voltron/internal/pkg/client"
	vcfg "github.com/projectcalico/calico/voltron/internal/pkg/config"
	"github.com/projectcalico/calico/voltron/internal/pkg/proxy"
	"github.com/projectcalico/calico/voltron/internal/pkg/regex"
	"github.com/projectcalico/calico/voltron/internal/pkg/server"
	"github.com/projectcalico/calico/voltron/internal/pkg/test"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"

	"github.com/projectcalico/calico/lma/pkg/auth"

	"golang.org/x/net/http2"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/rest"
)

var (
	tunnelCert    *x509.Certificate
	tunnelPrivKey *rsa.PrivateKey
	rootCAs       *x509.CertPool
	tunnelTLS     tls.Certificate

	// testServerName is the hostname to use for the test server.
	// Voltron will proxy connections to this hostname to the test server using SNI.
	testServerName = "test-server-name"
)

func init() {
	var err error
	logrus.SetOutput(GinkgoWriter)
	logrus.SetLevel(logrus.DebugLevel)

	tunnelCert, err = test.CreateSelfSignedX509Cert("voltron", true)
	if err != nil {
		panic(err)
	}

	block, _ := pem.Decode([]byte(test.PrivateRSA))
	tunnelPrivKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

	rootCAs = x509.NewCertPool()
	rootCAs.AddCert(tunnelCert)

	certPEM := utils.CertPEMEncode(tunnelCert)
	tunnelTLS, _ = tls.X509KeyPair(certPEM, []byte(test.PrivateRSA))
}

type testClient struct {
	http         *http.Client
	voltronHTTPS string
	voltronHTTP  string
}

func (c *testClient) doRequest(clusterID string) (string, error) {
	req, err := c.request(clusterID, "https", c.voltronHTTPS)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("error status: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *testClient) doHTTPRequest(clusterID string) (string, error) {
	req, err := c.request(clusterID, "http", c.voltronHTTP)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *testClient) request(clusterID string, schema string, address string) (*http.Request, error) {
	req, err := http.NewRequest("GET", schema+"://"+address+"/some/path", strings.NewReader("HELLO"))
	Expect(err).NotTo(HaveOccurred())
	req.Header[utils.ClusterHeaderField] = []string{clusterID}
	req.Header.Set(authentication.AuthorizationHeader, "Bearer jane")
	Expect(err).NotTo(HaveOccurred())
	return req, err
}

type testServer struct {
	msg string
	// store the last auth header received
	authHeader string
	http       *http.Server
}

func (s *testServer) handler(w http.ResponseWriter, r *http.Request) {
	s.authHeader = r.Header.Get(authentication.AuthorizationHeader)
	fmt.Fprint(w, s.msg)
}

func describe(name string, testFn func(string, bool)) bool {
	Describe(name+" cluster-scoped", func() { testFn("", false) })
	Describe(name+" namespace-scoped", func() { testFn("resource-ns", false) })
	Describe(name+" cluster-scoped (proxied)", func() { testFn("", true) })
	Describe(name+" namespace-scoped (proxied)", func() { testFn("resource-ns", true) })
	return true
}

var _ = describe("basic functionality", func(clusterNamespace string, proxied bool) {
	var (
		voltron   *server.Server
		lisHTTP11 net.Listener
		lisHTTP2  net.Listener
		lisTun    net.Listener

		guardian          *client.Client
		guardianTokenFile *os.File
		guardian2         *client.Client

		ts    *testServer
		lisTs net.Listener

		ts2    *testServer
		lisTs2 net.Listener

		proxyServer         *http.Server
		proxyURL            *url.URL
		proxiedRequestCount int

		wgSrvCnlt                                    sync.WaitGroup
		certPemID1, keyPemID1, certPemID2, keyPemID2 []byte

		// client to be used to interact with voltron (mimic UI)
		ui *testClient

		watchSync chan error
	)

	clusterID := "external-cluster"
	clusterID2 := "other-cluster"
	clusterNS := clusterNamespace

	authenticator := new(auth.MockJWTAuth)
	authenticator.On("Authenticate", mock.Anything).Return(&user.DefaultInfo{Name: "jane", Groups: []string{"developers"}}, 0, nil)

	AfterEach(func() {
		_ = guardian.Close()
		_ = os.Remove(guardianTokenFile.Name())
		_ = voltron.Close()
		_ = guardian.Close()
		_ = guardian2.Close()
		_ = ts.http.Close()
		_ = ts2.http.Close()
		if proxied {
			// Cleanup.
			_ = proxyServer.Close()

			// Validate requests were proxied.
			Expect(proxiedRequestCount).ToNot(BeZero(), "No requests were received by the proxy - expected traffic to route through it")
			proxiedRequestCount = 0
		}
		close(watchSync)

		wgSrvCnlt.Wait()
	})

	BeforeEach(func() {
		var err error

		ui = &testClient{
			http: &http.Client{
				Transport: &http2.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		}

		watchSync = make(chan error)

		// Instantiate a new fake client for each test.
		k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
		scheme := kscheme.Scheme
		err = v3.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()

		// Create a Voltron server for use in each test.
		lisHTTP11, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		lisHTTP2, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		// Bind to 0.0.0.0 as localhost addresses cause proxy to be bypassed - we want to test proxying the tunnel connection.
		lisTun, err = net.Listen("tcp", "0.0.0.0:0")
		Expect(err).NotTo(HaveOccurred())

		By("starting test servers", func() {
			var err error

			lisTs, err = net.Listen("tcp", "localhost:0")
			Expect(err).NotTo(HaveOccurred())

			ts = newTestServer("you reached me")

			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = ts.http.Serve(lisTs)
			}()

			lisTs2, err = net.Listen("tcp", "localhost:0")
			Expect(err).NotTo(HaveOccurred())

			ts2 = newTestServer("you reached the other me")

			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = ts2.http.Serve(lisTs2)
			}()
		})

		if proxied {
			By("starting the HTTP proxy", func() {
				httpProxy := goproxy.NewProxyHttpServer()

				// Count the amount of CONNECT requests made to the proxy.
				httpProxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
					proxiedRequestCount++
					return goproxy.OkConnect, host
				}))

				// Ensure the proxy does not try to dial through to the configured proxy (i.e. itself)
				httpProxy.ConnectDial = nil

				// Silence warnings from connections being closed. The proxy server lib only accepts the unstructured std logger.
				httpProxy.Logger = log.New(io.Discard, "", log.LstdFlags)

				// Instantiate the server.
				proxyServer = &http.Server{
					Addr:    ":3128",
					Handler: httpProxy,
				}
				proxyURL = &url.URL{
					Scheme: "http",
					Host:   proxyServer.Addr,
				}

				// Start the proxy.
				wgSrvCnlt.Add(1)
				go func() {
					defer wgSrvCnlt.Done()
					_ = proxyServer.ListenAndServe()
				}()
			})
		}

		By("starting Voltron", func() {
			tunnelTargetWhitelist, _ := regex.CompileRegexStrings([]string{
				`^/$`,
				`^/some/path$`,
			})

			voltron, err = server.New(
				k8sAPI,
				fakeClient,
				&rest.Config{BearerToken: "manager-token"},
				vcfg.Config{TenantNamespace: clusterNS},
				authenticator,
				server.WithTunnelSigningCreds(tunnelCert),
				server.WithTunnelCert(tunnelTLS),
				server.WithExternalCredFiles("../../internal/pkg/server/testdata/localhost.pem", "../../internal/pkg/server/testdata/localhost.key"),
				server.WithInternalCredFiles("../../internal/pkg/server/testdata/tigera-manager-svc.pem", "../../internal/pkg/server/testdata/tigera-manager-svc.key"),
				server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
				server.WithForwardingEnabled(true),

				// This config routes requests over the tunnel with hostname testServerName to the test server at listHTTP2.
				server.WithSNIServiceMap(map[string]string{testServerName: lisHTTP2.Addr().String()}),
			)
			Expect(err).NotTo(HaveOccurred())

			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = voltron.ServeHTTPS(lisHTTP11, "", "")
			}()

			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = voltron.ServeHTTPS(lisHTTP2, "", "")
			}()

			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = voltron.ServeTunnelsTLS(lisTun)
			}()

			go func() {
				_ = voltron.WatchK8sWithSync(watchSync)
			}()

			ui.voltronHTTPS = lisHTTP2.Addr().String()
			ui.voltronHTTP = lisHTTP11.Addr().String()
		})

		By("registering 2 managed clusters", func() {
			var fingerprintID1, fingerprintID2 string

			var err error
			certPemID1, keyPemID1, fingerprintID1, err = test.GenerateTestCredentials(clusterID, tunnelCert, tunnelPrivKey)
			Expect(err).NotTo(HaveOccurred())
			annotationsID1 := map[string]string{server.AnnotationActiveCertificateFingerprint: fingerprintID1}

			err = fakeClient.Create(context.Background(), &v3.ManagedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        clusterID,
					Namespace:   clusterNS,
					Annotations: annotationsID1,
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(<-watchSync).NotTo(HaveOccurred())

			certPemID2, keyPemID2, fingerprintID2, err = test.GenerateTestCredentials(clusterID2, tunnelCert, tunnelPrivKey)
			Expect(err).NotTo(HaveOccurred())
			annotationsID2 := map[string]string{server.AnnotationActiveCertificateFingerprint: fingerprintID2}

			err = fakeClient.Create(context.Background(), &v3.ManagedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        clusterID2,
					Namespace:   clusterNS,
					Annotations: annotationsID2,
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(<-watchSync).NotTo(HaveOccurred())
		})

		// It should also start Guardian.
		By("starting guardian", func() {
			var err error
			// Setup to read token from a file like we do with the service account token
			// so we can update it and test that we re-load the token when it changes
			guardianTokenFile, err = os.CreateTemp("", "guardianToken")
			Expect(err).NotTo(HaveOccurred())
			_, err = guardianTokenFile.WriteString("initialToken")
			Expect(err).NotTo(HaveOccurred())
			err = guardianTokenFile.Close()
			Expect(err).NotTo(HaveOccurred())
			targets, err := bootstrap.ProxyTargets([]bootstrap.Target{
				{
					Path:      "/some/path",
					Dest:      listenerURL(lisTs).String(),
					TokenPath: guardianTokenFile.Name(),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			guardian, err = client.New(
				lisTun.Addr().String(),
				"voltron",
				client.WithTunnelCreds(certPemID1, keyPemID1),
				client.WithTunnelRootCA(rootCAs),
				client.WithProxyTargets(targets),
				client.WithHTTPProxyURL(proxyURL),
			)
			Expect(err).NotTo(HaveOccurred())
			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = guardian.ServeTunnelHTTP()
			}()
		})

		By("starting guardian2", func() {
			var err error

			guardian2, err = client.New(
				lisTun.Addr().String(),
				"voltron",
				client.WithTunnelCreds(certPemID2, keyPemID2),
				client.WithTunnelRootCA(rootCAs),
				client.WithProxyTargets(
					[]proxy.Target{
						{
							Path: "/some/path",
							Dest: listenerURL(lisTs2),
						},
					},
				),
				client.WithHTTPProxyURL(proxyURL),
			)
			Expect(err).NotTo(HaveOccurred())
			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = guardian2.ServeTunnelHTTP()
			}()
		})

		By("indicating that before each is complete", func() {
			logrus.Info("[TEST] BeforeEach complete")
		})
	})

	It("should be possible to reach the test server on http2", func() {
		var msg string
		Eventually(func() error {
			var err error
			msg, err = ui.doRequest(clusterID)
			return err
		}, "10s", "1s").ShouldNot(HaveOccurred())
		Expect(msg).To(Equal(ts.msg))
	})

	It("should pickup refreshed token", func() {
		var err error
		Eventually(func() string {
			_, err = ui.doRequest(clusterID)
			if err != nil {
				return ""
			}
			return ts.authHeader
		}, "10s", "1s").Should(Equal("Bearer initialToken"))

		guardianTokenFile, err = os.OpenFile(guardianTokenFile.Name(), os.O_RDWR, 0644)
		Expect(err).NotTo(HaveOccurred())
		_, err = guardianTokenFile.WriteString("updatedToken")
		Expect(err).NotTo(HaveOccurred())
		err = guardianTokenFile.Close()
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() string {
			_, err = ui.doRequest(clusterID)
			if err != nil {
				return ""
			}
			return ts.authHeader
		}, "1m", "1s").Should(Equal("Bearer updatedToken"))
	})

	It("should be possible to reach the other test server on http2", func() {
		var msg string
		Eventually(func() error {
			var err error
			msg, err = ui.doRequest(clusterID2)
			return err
		}, "10s", "1s").ShouldNot(HaveOccurred())
		Expect(msg).To(Equal(ts2.msg))
	})

	It("should not be possible to reach the test server on http", func() {
		_, err := ui.doHTTPRequest(clusterID)
		Expect(err).To(HaveOccurred())
	})

	It("should be possible to send a request to Voltron via Guardian", func() {
		// We need to Listen for connections to pass to Guardian.
		listener, err := net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		// Start listening for connections on the test listener.
		By("Starting guardian AcceptAndProxy")
		go func() {
			_ = guardian.AcceptAndProxy(listener)
		}()

		// Esablish a connection to the listener.
		By("Establishing a connection to the test listener at " + listener.Addr().String())
		var resp string
		Eventually(func() error {
			tc := &testClient{
				// Point the client at the listener that we passed to Guardian above. This will
				// create connections to Guardian that will be forwarded to Voltron.
				voltronHTTPS: listener.Addr().String(),
				http: &http.Client{
					Timeout: 3 * time.Second,
					Transport: &http2.Transport{
						TLSClientConfig: &tls.Config{
							ServerName:         testServerName,
							InsecureSkipVerify: true,
						},
					},
				},
			}
			resp, err = tc.doRequest(clusterID)
			return err
		}).ShouldNot(HaveOccurred())

		Expect(resp).To(Equal(ts.msg))

		By("Establishing many connections at once")

		// numConns needs to be less than the maximum number of connections that Voltron will
		// accept simultaneously from a single tunnel. This is currently 500.
		numConns := 499
		numReqs := 30
		done := make(chan struct{})
		for i := 0; i < numConns; i++ {
			go func() {
				defer GinkgoRecover()

				tc := &testClient{
					// Point the client at the listener that we passed to Guardian above. This will
					// create connections to Guardian that will be forwarded to Voltron.
					voltronHTTPS: listener.Addr().String(),
					http: &http.Client{
						Timeout: 3 * time.Second,
						Transport: &http2.Transport{
							TLSClientConfig: &tls.Config{
								ServerName:         testServerName,
								InsecureSkipVerify: true,
							},
						},
					},
				}

				for j := 0; j < numReqs; j++ {
					resp, err = tc.doRequest(clusterID)
					Expect(err).NotTo(HaveOccurred())
					done <- struct{}{}
				}
			}()
		}

		dur := time.Duration(numReqs) * time.Second
		timeout := time.After(dur)
		By(fmt.Sprintf("Waiting for all requests to complete in %s", dur))
		numDone := 0
		for i := 0; i < numConns*numReqs; i++ {
			select {
			case <-done:
				numDone++
				continue
			case <-timeout:
				Expect(true).To(BeFalse(), "timed out waiting for all requests to complete (%d done)", numDone)
			}
		}
		Expect(numDone).To(Equal(numConns*numReqs), "Not all connections completed")
	})

	It("should not be possible to reach the test server after guardian is terminated", func() {
		By("Terminating guardian")
		err := guardian.Close()
		Expect(err).NotTo(HaveOccurred())

		_, err = ui.doRequest(clusterID)
		Expect(err).To(HaveOccurred())
	})

	// To be fixed in SAAS-768
	/*	It("should start guardian again", func() {
			cert, key, err := voltron.ClusterCreds(clusterID)
			Expect(err).NotTo(HaveOccurred())

			guardian, err = client.New(
				lisTun.Addr().String(),
				client.WithTunnelCreds(cert, key, rootCAs),
				client.WithProxyTargets(
					[]proxy.Target{
						{
							Path: "/some/path",
							Dest: listenerURL(lisTs),
						},
					},
				),
			)
			Expect(err).NotTo(HaveOccurred())
			wgSrvCnlt.Add(1)
			go func() {
				defer wgSrvCnlt.Done()
				_ = guardian.ServeTunnelHTTP()
			}()
		})

		It("should be possible to reach the test server again", func() {
			var msg string
			Eventually(func() error {
				var err error
				msg, err = ui.doRequest(clusterID)
				return err
			}, "10s", "1s").ShouldNot(HaveOccurred())
			Expect(msg).To(Equal(ts.msg))
		})*/
})

func newTestServer(msg string) *testServer {
	cert, _ := test.CreateSelfSignedX509Cert("test-server", true)
	certPEM := utils.CertPEMEncode(cert)
	xcert, err := tls.X509KeyPair(certPEM, []byte(test.PrivateRSA))
	Expect(err).NotTo(HaveOccurred())

	mux := http.NewServeMux()
	ts := &testServer{
		msg: msg,
		http: &http.Server{
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{xcert},
				NextProtos:   []string{"h2"},
			},
			Handler: mux,
		},
	}

	mux.HandleFunc("/", ts.handler)

	return ts
}

func listenerURL(l net.Listener) *url.URL {
	u, err := url.Parse("http://" + l.Addr().String())
	Expect(err).NotTo(HaveOccurred())
	return u
}
