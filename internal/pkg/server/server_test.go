// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package server_test

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"io/ioutil"
	"net/http/httptest"
	"net/url"

	apiv3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientv3 "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"k8s.io/client-go/kubernetes"

	"github.com/coreos/go-oidc"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/lma/pkg/auth"
	"k8s.io/client-go/rest"

	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/apiserver/pkg/authentication"
	"github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/regex"
	"github.com/tigera/voltron/internal/pkg/server"
	"github.com/tigera/voltron/internal/pkg/test"
	"github.com/tigera/voltron/pkg/tunnel"

	"github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const (
	k8sIssuer           = "kubernetes/serviceaccount"
	dexIssuer           = "https://127.0.0.1:9443/dex"
	managerSAAuthHeader = "Bearer tigera-manager-token"

	// Exp that is very far in the future.
	exp = 9600964804
)

var (
	clusterA = "clusterA"
	clusterB = "clusterB"
	config   = &rest.Config{BearerToken: "tigera-manager-token"}

	// Tokens issued by k8s.
	janeBearerToken = authHeader(k8sIssuer, "jane@example.io", exp)
	bobBearerToken  = authHeader(k8sIssuer, "bob@example.io", exp)

	// Token issued by dex.
	jennyToken      = authHeader(dexIssuer, "jenny@dex.io", exp)
	expiredDexToken = authHeader(dexIssuer, "jenny@dex.io", 0)
)

func authHeader(iss, email string, exp int) string {
	hdrhdr := "eyJhbGciOiJSUzI1NiIsImtpZCI6Ijk3ODM2YzRiMjdmN2M3ZmVjMjk1MTk0NTFkNDc5MmUyNjQ4M2RmYWUifQ" // rs256 header
	payload := map[string]interface{}{
		"iss":            iss,
		"sub":            "ChUxMDkxMzE",
		"aud":            "tigera-manager",
		"exp":            exp,
		"iat":            1600878403,
		"nonce":          "35e32c66028243f592cc3103c7c2dfb2",
		"at_hash":        "jOq0F62t_NE9a3UXtNJkYg",
		"email":          email,
		"email_verified": true,
		"groups": []string{
			"all-van@tigera.io",
		},
		"name": "John Doe",
	}
	payloadJson, _ := json.Marshal(payload)
	payloadStr := base64.RawURLEncoding.EncodeToString(payloadJson)
	return fmt.Sprintf("Bearer %s.%s.%s", hdrhdr, payloadStr, "e30")
}

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

type k8sClient struct {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

var _ = Describe("Server Proxy to tunnel", func() {
	var (
		keyset *mockKeySet
		k8sAPI bootstrap.K8sClient

		voltronTunnelCert      *x509.Certificate
		voltronTunnelTLSCert   tls.Certificate
		voltronTunnelPrivKey   *rsa.PrivateKey
		voltronExtHttpsCert    *x509.Certificate
		voltronExtHttpsPrivKey *rsa.PrivateKey
		voltronIntHttpsCert    *x509.Certificate
		voltronIntHttpsPrivKey *rsa.PrivateKey
		voltronTunnelCAs       *x509.CertPool
		voltronHttpsCAs        *x509.CertPool
	)

	BeforeEach(func() {
		var err error

		keyset = &mockKeySet{}
		k8sAPI = &k8sClient{
			Interface:                k8sfake.NewSimpleClientset(),
			ProjectcalicoV3Interface: fake.NewSimpleClientset().ProjectcalicoV3(),
		}

		voltronTunnelCertTemplate := test.CreateCACertificateTemplate("voltron")
		voltronTunnelPrivKey, voltronTunnelCert, err = test.CreateCertPair(voltronTunnelCertTemplate, nil, nil)
		Expect(err).ShouldNot(HaveOccurred())

		voltronTunnelTLSCert, err = tls.X509KeyPair(test.CertToPemBytes(voltronTunnelCert), test.KeyToPemBytes(voltronTunnelPrivKey))
		Expect(err).ShouldNot(HaveOccurred())

		voltronExtHttpCertTemplate := test.CreateServerCertificateTemplate("localhost")
		voltronExtHttpsPrivKey, voltronExtHttpsCert, err = test.CreateCertPair(voltronExtHttpCertTemplate, nil, nil)
		Expect(err).ShouldNot(HaveOccurred())

		voltronIntHttpCertTemplate := test.CreateServerCertificateTemplate("tigera-manager.tigera-manager.svc")
		voltronIntHttpsPrivKey, voltronIntHttpsCert, err = test.CreateCertPair(voltronIntHttpCertTemplate, nil, nil)
		Expect(err).ShouldNot(HaveOccurred())

		voltronTunnelCAs = x509.NewCertPool()
		voltronTunnelCAs.AppendCertsFromPEM(test.CertToPemBytes(voltronTunnelCert))

		voltronHttpsCAs = x509.NewCertPool()
		voltronHttpsCAs.AppendCertsFromPEM(test.CertToPemBytes(voltronExtHttpsCert))
		voltronHttpsCAs.AppendCertsFromPEM(test.CertToPemBytes(voltronIntHttpsCert))
	})

	It("should fail to start the server when the paths to the external credentials are invalid", func() {
		authenticator := authentication.NewFakeAuthenticator()
		authenticator.AddValidApiResponse(janeBearerToken, "jane", []string{"developers"})
		_, err := server.New(
			k8sAPI,
			config,
			authenticator,
			server.WithExternalCredsFiles("dog/gopher.crt", "dog/gopher.key"),
			server.WithInternalCredFiles("dog/gopher.crt", "dog/gopher.key"),
		)
		Expect(err).To(HaveOccurred())
	})

	Context("Server is running", func() {
		var (
			httpsAddr, tunnelAddr string
			srvWg                 *sync.WaitGroup
			srv                   *server.Server
			defaultServer         *httptest.Server
		)

		BeforeEach(func() {
			var err error

			authenticator := authentication.NewFakeAuthenticator()
			authenticator.AddValidApiResponse(authHeader(k8sIssuer, "jane@example.io", exp), "jane", []string{"developers"})

			defaultServer = httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Echo the token, such that we can determine if the auth header was successfully swapped.
					w.Header().Set(authentication.AuthorizationHeader, r.Header.Get(authentication.AuthorizationHeader))
				}))

			defaultURL, err := url.Parse(defaultServer.URL)
			Expect(err).NotTo(HaveOccurred())

			defaultProxy, e := proxy.New([]proxy.Target{
				{Path: "/", Dest: defaultURL},
				{Path: "/compliance/", Dest: defaultURL},
			})
			Expect(e).NotTo(HaveOccurred())

			tunnelTargetWhitelist, err := regex.CompileRegexStrings([]string{`^/$`, `^/some/path$`})
			Expect(err).ShouldNot(HaveOccurred())

			k8sTargets, err := regex.CompileRegexStrings([]string{`^/api/?`, `^/apis/?`})
			Expect(err).ShouldNot(HaveOccurred())

			srv, httpsAddr, tunnelAddr, srvWg = createAndStartServer(k8sAPI,
				config,
				auth.NewAggregateAuthenticator(dex(keyset), authenticator),
				server.WithTunnelCreds(voltronTunnelCert, voltronTunnelPrivKey),
				server.WithExternalCreds(test.CertToPemBytes(voltronExtHttpsCert), test.KeyToPemBytes(voltronExtHttpsPrivKey)),
				server.WithInternalCreds(test.CertToPemBytes(voltronIntHttpsCert), test.KeyToPemBytes(voltronIntHttpsPrivKey)),
				server.WithDefaultProxy(defaultProxy),
				server.WithKubernetesAPITargets(k8sTargets),
				server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
			)
		})

		AfterEach(func() {
			Expect(srv.Close()).NotTo(HaveOccurred())
			defaultServer.Close()
			srvWg.Wait()
		})

		Context("Adding / removing managed clusters", func() {
			It("should get an empty list if no managed clusters are registered", func() {
				var list []clusters.ManagedCluster

				Eventually(func() int {
					var code int
					list, code = listClusters(httpsAddr)
					return code
				}).Should(Equal(200))
				Expect(len(list)).To(Equal(0))
			})

			It("should be able to register multiple clusters", func() {
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterA},
				}, metav1.CreateOptions{})

				Expect(err).ShouldNot(HaveOccurred())
				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})

				Expect(err).ShouldNot(HaveOccurred())
				var list []clusters.ManagedCluster
				var code int
				Eventually(func() int {
					list, code = listClusters(httpsAddr)
					return len(list)
				}, 5*time.Second, 200*time.Millisecond).Should(Equal(2))

				Expect(code).To(Equal(200))
				Expect(list).To(Equal([]clusters.ManagedCluster{
					{ID: "clusterA"},
					{ID: "clusterB"},
				}))
			})

			It("should be able to list the remaining clusters after deleting one", func() {
				By("adding two cluster")
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterA},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				var list []clusters.ManagedCluster
				var code int
				Eventually(func() int {
					list, code = listClusters(httpsAddr)
					return len(list)
				}, 5*time.Second, 200*time.Millisecond).Should(Equal(2))

				Expect(code).To(Equal(200))
				Expect(list).To(Equal([]clusters.ManagedCluster{
					{ID: "clusterA"}, {ID: "clusterB"},
				}))

				By("removing one cluster")
				Expect(k8sAPI.ManagedClusters().Delete(context.Background(), clusterB, metav1.DeleteOptions{})).ShouldNot(HaveOccurred())
				Eventually(func() int {
					list, code = listClusters(httpsAddr)
					return len(list)
				}, 5*time.Second, 200*time.Millisecond).Should(Equal(1))

				Expect(code).To(Equal(200))
				Expect(list).To(Equal([]clusters.ManagedCluster{
					{ID: "clusterA"},
				}))
			})

			It("should be able to register clusterB after it's been deleted again", func() {
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(k8sAPI.ManagedClusters().Delete(context.Background(), clusterB, metav1.DeleteOptions{})).ShouldNot(HaveOccurred())
				Eventually(func() int {
					list, _ := listClusters(httpsAddr)
					return len(list)
				}).Should(Equal(0))

				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() int {
					list, _ := listClusters(httpsAddr)
					return len(list)
				}).Should(Equal(1))
			})
		})

		Context("Proxying requests over the tunnel", func() {
			It("should not proxy anywhere without valid headers", func() {
				resp, err := http.Get("http://" + httpsAddr + "/")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
			})

			It("Should reject requests to clusters that don't exist", func() {
				req, err := http.NewRequest("GET", "http://"+httpsAddr+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add(server.ClusterHeaderField, "zzzzzzz")
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
			})
			It("Should not proxy anywhere - multiple headers", func() {
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
						ServerName:         "localhost",
					},
				}
				client := &http.Client{Transport: tr}
				req, err := http.NewRequest("GET", "https://"+httpsAddr+"/", nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add(server.ClusterHeaderField, clusterA)
				req.Header.Add(server.ClusterHeaderField, "helloworld")
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
			})

			It("should not be able to proxy to a cluster without a tunnel", func() {
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					TypeMeta: metav1.TypeMeta{
						Kind:       calicov3.KindManagedCluster,
						APIVersion: calicov3.GroupVersionCurrent,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterA,
						//Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: test.CertificateFingerprint(leafCert)},
					},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				clientHelloReq(httpsAddr, clusterA, 400)
			})

			It("Should proxy to default if no header", func() {
				resp, err := http.Get(defaultServer.URL)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
			})

			It("Should proxy to default even with header, if request path matches one of bypass tunnel targets", func() {
				req, err := http.NewRequest(
					"GET",
					"https://"+httpsAddr+"/compliance/reports",
					strings.NewReader("HELLO"),
				)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add(server.ClusterHeaderField, clusterA)
				req.Header.Set(authentication.AuthorizationHeader, janeBearerToken)

				var httpClient = &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				}
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
			})

			It("Should swap the auth header and impersonate the user for requests to k8s (a)api server", func() {
				req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", httpsAddr, "/api/v1/namespaces"), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set(authentication.AuthorizationHeader, janeBearerToken)

				resp, err := configureHTTPSClient().Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get(authentication.AuthorizationHeader)).To(Equal(managerSAAuthHeader))
			})

			It("Should authenticate dex users for k8s api call", func() {
				payload, _ := base64.RawURLEncoding.DecodeString(strings.Split(jennyToken, ".")[1])
				keyset.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(jennyToken, "Bearer "))).Return(payload, nil)
				req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", httpsAddr, "/api/v1/namespaces"), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set(authentication.AuthorizationHeader, jennyToken)

				resp, err := configureHTTPSClient().Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get(authentication.AuthorizationHeader)).To(Equal(managerSAAuthHeader))

			})

			It("Should reject dex users with expired token for k8s api call", func() {
				req, err := http.NewRequest(
					"GET",
					fmt.Sprintf("https://%s%s", httpsAddr, "/api/v1/namespaces"), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add(server.ClusterHeaderField, clusterA)
				req.Header.Set(authentication.AuthorizationHeader, expiredDexToken)

				var httpClient = &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				}
				resp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(401))
			})

			Context("A single cluster is registered", func() {
				var (
					clusterATLSCert tls.Certificate
				)

				BeforeEach(func() {
					clusterACertTemplate := test.CreateClientCertificateTemplate(clusterA, "localhost")
					clusterAPrivKey, clusterACert, err := test.CreateCertPair(clusterACertTemplate, voltronTunnelCert, voltronTunnelPrivKey)
					Expect(err).ShouldNot(HaveOccurred())

					payload, _ := base64.RawURLEncoding.DecodeString(strings.Split(jennyToken, ".")[1])
					keyset.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(jennyToken, "Bearer "))).Return(payload, nil)

					_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:        clusterA,
							Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: test.CertificateFingerprint(clusterACert)},
						},
					}, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					clusterATLSCert, err = tls.X509KeyPair(test.CertToPemBytes(clusterACert), test.KeyToPemBytes(clusterAPrivKey))
					Expect(err).NotTo(HaveOccurred())
				})

				It("can send requests from the server to the cluster", func() {
					tun, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
						Certificates: []tls.Certificate{clusterATLSCert},
						RootCAs:      voltronTunnelCAs,
					})
					Expect(err).NotTo(HaveOccurred())

					WaitForClusterToConnect(k8sAPI, clusterA)

					cli := &http.Client{
						Transport: &http2.Transport{
							TLSClientConfig: &tls.Config{
								NextProtos: []string{"h2"},
								RootCAs:    voltronHttpsCAs,
								ServerName: "localhost",
							},
						},
					}

					req, err := http.NewRequest("GET", "https://"+httpsAddr+"/some/path", strings.NewReader("HELLO"))
					Expect(err).NotTo(HaveOccurred())

					req.Header[server.ClusterHeaderField] = []string{clusterA}
					req.Header.Set(authentication.AuthorizationHeader, jennyToken)

					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer GinkgoRecover()
						defer wg.Done()

						_, err := cli.Do(req)
						Expect(err).ShouldNot(HaveOccurred())
					}()

					serve := &http.Server{
						Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							f, ok := w.(http.Flusher)
							Expect(ok).To(BeTrue())

							body, err := ioutil.ReadAll(r.Body)
							Expect(err).ShouldNot(HaveOccurred())

							Expect(string(body)).Should(Equal("HELLO"))

							f.Flush()
						}),
					}

					defer serve.Close()
					go func() {
						defer GinkgoRecover()
						err := serve.Serve(tls.NewListener(tun, &tls.Config{
							Certificates: []tls.Certificate{clusterATLSCert},
							NextProtos:   []string{"h2"},
						}))
						Expect(err).Should(Equal(fmt.Errorf("http: Server closed")))
					}()

					wg.Wait()
				})

				Context("A second cluster is registered", func() {
					var (
						clusterBTLSCert tls.Certificate
					)

					BeforeEach(func() {
						clusterBCertTemplate := test.CreateClientCertificateTemplate(clusterB, "localhost")
						clusterBPrivKey, clusterBCert, err := test.CreateCertPair(clusterBCertTemplate, voltronTunnelCert, voltronTunnelPrivKey)

						Expect(err).NotTo(HaveOccurred())
						_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
							ObjectMeta: metav1.ObjectMeta{
								Name:        clusterB,
								Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: test.CertificateFingerprint(clusterBCert)},
							},
						}, metav1.CreateOptions{})
						Expect(err).ShouldNot(HaveOccurred())

						clusterBTLSCert, err = tls.X509KeyPair(test.CertToPemBytes(clusterBCert), test.KeyToPemBytes(clusterBPrivKey))
						Expect(err).NotTo(HaveOccurred())
					})

					It("can send requests from the server to the second cluster", func() {
						tun, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
							Certificates: []tls.Certificate{clusterBTLSCert},
							RootCAs:      voltronTunnelCAs,
						})
						Expect(err).NotTo(HaveOccurred())

						WaitForClusterToConnect(k8sAPI, clusterB)

						cli := &http.Client{
							Transport: &http2.Transport{
								TLSClientConfig: &tls.Config{
									NextProtos: []string{"h2"},
									RootCAs:    voltronHttpsCAs,
									ServerName: "localhost",
								},
							},
						}

						req, err := http.NewRequest("GET", "https://"+httpsAddr+"/some/path", strings.NewReader("HELLO"))
						Expect(err).NotTo(HaveOccurred())

						req.Header[server.ClusterHeaderField] = []string{clusterB}
						req.Header.Set(authentication.AuthorizationHeader, jennyToken)

						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							defer GinkgoRecover()
							defer wg.Done()

							_, err := cli.Do(req)
							Expect(err).ShouldNot(HaveOccurred())
						}()

						serve := &http.Server{
							Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								f, ok := w.(http.Flusher)
								Expect(ok).To(BeTrue())

								body, err := ioutil.ReadAll(r.Body)
								Expect(err).ShouldNot(HaveOccurred())

								Expect(string(body)).Should(Equal("HELLO"))

								f.Flush()
							}),
						}

						defer serve.Close()
						go func() {
							defer GinkgoRecover()
							err := serve.Serve(tls.NewListener(tun, &tls.Config{
								Certificates: []tls.Certificate{clusterBTLSCert},
								NextProtos:   []string{"h2"},
							}))
							Expect(err).Should(Equal(fmt.Errorf("http: Server closed")))
						}()

						wg.Wait()
					})

					It("should not be possible to open a two tunnels to the same cluster", func() {
						_, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
							Certificates: []tls.Certificate{clusterBTLSCert},
							RootCAs:      voltronTunnelCAs,
						})
						Expect(err).NotTo(HaveOccurred())

						tunB2, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
							Certificates: []tls.Certificate{clusterBTLSCert},
							RootCAs:      voltronTunnelCAs,
						})
						Expect(err).NotTo(HaveOccurred())

						_, err = tunB2.Accept()
						Expect(err).Should(HaveOccurred())
					})
				})
			})
		})
	})

	Context("Voltron tunnel configured with tls certificate with invalid Key Extension", func() {
		var (
			wg                    *sync.WaitGroup
			srv                   *server.Server
			httpsAddr, tunnelAddr string
			defaultServer         *httptest.Server
		)

		BeforeEach(func() {
			authenticator := authentication.NewFakeAuthenticator()
			authenticator.AddValidApiResponse(authHeader(k8sIssuer, "jane@example.io", exp), "jane", []string{"developers"})
			defaultServer = httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

			defaultURL, err := url.Parse(defaultServer.URL)
			Expect(err).NotTo(HaveOccurred())

			defaultProxy, e := proxy.New([]proxy.Target{
				{Path: "/", Dest: defaultURL},
				{Path: "/compliance/", Dest: defaultURL},
				{Path: "/api/v1/namespaces", Dest: defaultURL},
			})
			Expect(e).NotTo(HaveOccurred())

			tunnelTargetWhitelist, _ := regex.CompileRegexStrings([]string{
				`^/$`,
				`^/some/path$`,
			})

			// Recreate the voltron certificate specifying client auth key usage
			voltronTunnelCertTemplate := test.CreateCACertificateTemplate("voltron")
			voltronTunnelCertTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

			voltronTunnelPrivKey, voltronTunnelCert, err = test.CreateCertPair(voltronTunnelCertTemplate, nil, nil)
			Expect(err).ShouldNot(HaveOccurred())

			voltronTunnelCAs = x509.NewCertPool()
			voltronTunnelCAs.AppendCertsFromPEM(test.CertToPemBytes(voltronTunnelCert))

			srv, httpsAddr, tunnelAddr, wg = createAndStartServer(
				k8sAPI,
				config,
				authenticator,
				server.WithTunnelCreds(voltronTunnelCert, voltronTunnelPrivKey),
				server.WithDefaultProxy(defaultProxy),
				server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
				server.WithInternalCreds(test.CertToPemBytes(voltronIntHttpsCert), test.KeyToPemBytes(voltronIntHttpsPrivKey)),
				server.WithExternalCreds(test.CertToPemBytes(voltronExtHttpsCert), test.KeyToPemBytes(voltronExtHttpsPrivKey)),
			)

			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(srv.Close()).NotTo(HaveOccurred())
			wg.Wait()
		})

		It("server with invalid key types will not accept connections", func() {
			var err error

			certTemplate := test.CreateClientCertificateTemplate(clusterA, "localhost")
			privKey, cert, err := test.CreateCertPair(certTemplate, voltronTunnelCert, voltronTunnelPrivKey)
			Expect(err).ShouldNot(HaveOccurred())

			By("adding ClusterA")
			_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        clusterA,
					Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: test.CertificateFingerprint(cert)},
				},
			}, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				list, _ := listClusters(httpsAddr)
				return len(list)
			}).Should(Equal(1))

			// Try to connect clusterA to the new fake voltron, should fail
			tlsCert, err := tls.X509KeyPair(test.CertToPemBytes(cert), test.KeyToPemBytes(privKey))
			Expect(err).NotTo(HaveOccurred())

			_, err = tunnel.DialTLS(tunnelAddr, &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
				RootCAs:      voltronTunnelCAs,
				ServerName:   "voltron",
			})
			Expect(err).Should(MatchError("tcp.tls.Dial failed: x509: certificate specifies an incompatible key usage"))
		})
	})

	Context("Server authenticates requests", func() {
		var (
			wg                   *sync.WaitGroup
			srv                  *server.Server
			httpAddr, tunnelAddr string

			authenticator authentication.FakeAuthenticator
		)

		BeforeEach(func() {
			var err error
			authenticator = authentication.NewFakeAuthenticator()

			tunnelTargetWhitelist, err := regex.CompileRegexStrings([]string{`^/?`})
			Expect(err).ShouldNot(HaveOccurred())

			srv, httpAddr, tunnelAddr, wg = createAndStartServer(
				k8sAPI,
				config,
				auth.NewAggregateAuthenticator(dex(keyset), authenticator),
				server.WithTunnelCreds(voltronTunnelCert, voltronTunnelPrivKey),
				server.WithInternalCreds(test.CertToPemBytes(voltronIntHttpsCert), test.KeyToPemBytes(voltronIntHttpsPrivKey)),
				server.WithExternalCreds(test.CertToPemBytes(voltronExtHttpsCert), test.KeyToPemBytes(voltronExtHttpsPrivKey)),
				server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
			)
		})

		Context("Single managed cluster", func() {
			var bin *test.HTTPSBin
			binC := make(chan struct{}, 1)
			authJane := func() {
				clnt := configureHTTPSClient()
				req := requestToClusterA(httpAddr)
				req.Header.Set(authentication.AuthorizationHeader, janeBearerToken)
				resp, err := clnt.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				// would timeout the test if the reply is not from the serve and the test were not executed
				<-binC
			}

			BeforeEach(func() {
				tmpl := test.CreateClientCertificateTemplate(clusterA, "voltron")
				key, cert, err := test.CreateCertPair(tmpl, voltronTunnelCert, voltronTunnelPrivKey)
				Expect(err).ShouldNot(HaveOccurred())

				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &apiv3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterA,
						Annotations: map[string]string{
							server.AnnotationActiveCertificateFingerprint: test.CertificateFingerprint(cert),
						},
					},
				}, metav1.CreateOptions{})

				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() int {
					list, _ := listClusters(httpAddr)
					return len(list)
				}).Should(Equal(1))

				tlsCert, err := tls.X509KeyPair(test.CertToPemBytes(cert), test.KeyToPemBytes(key))
				Expect(err).ShouldNot(HaveOccurred())
				tun, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
					Certificates: []tls.Certificate{tlsCert},
					RootCAs:      voltronTunnelCAs,
				})
				Expect(err).NotTo(HaveOccurred())

				bin = test.NewHTTPSBin(tun, voltronTunnelTLSCert, func(r *http.Request) {
					switch r.Header.Get("Impersonate-User") {
					case "jane":
						Expect(r.Header.Get("Impersonate-Group")).To(Equal("developers"))
						Expect(r.Header.Get("Authorization")).NotTo(Equal(janeBearerToken))
						binC <- struct{}{}
					case "jenny@dex.io":
						Expect(r.Header.Get("Impersonate-Group")).To(Equal("all-van@tigera.io"))
						Expect(r.Header.Get("Authorization")).NotTo(Equal(janeBearerToken))
						binC <- struct{}{}
					default:
						panic("unexpected user was impersonated")
					}
				})

				WaitForClusterToConnect(k8sAPI, clusterA)

				authenticator.AddValidApiResponse(janeBearerToken, "jane", []string{"developers"})
				authJane()
			})

			AfterEach(func() {
				Expect(srv.Close()).NotTo(HaveOccurred())
				wg.Wait()
				bin.Close()
			})

			It("Should authenticate dex users for tunnel call", func() {
				payload, _ := base64.RawURLEncoding.DecodeString(strings.Split(jennyToken, ".")[1])
				keyset.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(jennyToken, "Bearer "))).Return(payload, nil)
				clnt := configureHTTPSClient()
				req := requestToClusterA(httpAddr)
				req.Header.Set(authentication.AuthorizationHeader, jennyToken)
				resp, err := clnt.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				<-binC
			})

			It("Should reject dex users with expired token for tunnel call", func() {
				clnt := configureHTTPSClient()
				req := requestToClusterA(httpAddr)
				req.Header.Set(authentication.AuthorizationHeader, expiredDexToken)
				resp, err := clnt.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(401))
			})

			It("should not authenticate Bob - Bob exists, does not have rights", func() {
				authenticator.AddErrorAPIServerResponse(bobBearerToken, nil, http.StatusUnauthorized)
				clnt := configureHTTPSClient()
				req := requestToClusterA(httpAddr)
				req.Header.Set(authentication.AuthorizationHeader, bobBearerToken)
				resp, err := clnt.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(401))
			})

			It("should not authenticate user that does not exist", func() {
				clnt := configureHTTPSClient()
				req := requestToClusterA(httpAddr)
				randomToken := "Bearer someRandomTokenThatShouldNotMatch"
				authenticator.AddErrorAPIServerResponse(randomToken, nil, http.StatusUnauthorized)
				resp, err := clnt.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(401))
			})

			It("should return 401 on missing tokens", func() {
				clnt := configureHTTPSClient()
				req := requestToClusterA(httpAddr)
				resp, err := clnt.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(401))
			})

			It("should authenticate Jane again after errors", func() {
				authJane()
			})
		})
	})
	// Will be fixed in SAAS-769
	/*		When("long lasting connection is in progress", func() {
			var slowTun *tunnel.Tunnel
			var xCert tls.Certificate

			It("should get some certs for test server", func() {
				key, _ := utils.KeyPEMEncode(tunnelPrivKey)
				cert := utils.CertPEMEncode(tunnelCert)

				xCert, _ = tls.X509KeyPair(cert, key)
			})

			It("Should add cluster", func() {
				Expect(k8sAPI.AddCluster("slow", "slow")).ShouldNot(HaveOccurred())
				Expect(<-watchSync).NotTo(HaveOccurred())
			})

			var slow *test.HTTPSBin
			slowC := make(chan struct{})
			slowWaitC := make(chan struct{})

			It("Should open a tunnel", func() {
				certPem, keyPem, _ := srv.ClusterCreds("slow")
				cert, _ := tls.X509KeyPair(certPem, keyPem)

				cfg := &tls.Config{
					Certificates: []tls.Certificate{cert},
					RootCAs:      rootCAs,
				}

				Eventually(func() error {
					var err error
					slowTun, err = tunnel.DialTLS(lisTun.Addr().String(), cfg)
					return err
				}).ShouldNot(HaveOccurred())

				slow = test.NewHTTPSBin(slowTun, xCert, func(r *http.Request) {
					// the connection is set up, let the test proceed
					close(slowWaitC)
					// block here to emulate long lasting connection
					<-slowC
				})

			})

			It("should be able to update a cluster - test race SAAS-226", func() {
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					clnt := configureHTTPSClient()
					req, err := http.NewRequest("GET",
						"https://"+lis2.Addr().String()+"/some/path", strings.NewReader("HELLO"))
					Expect(err).NotTo(HaveOccurred())
					req.Header[server.ClusterHeaderField] = []string{"slow"}
					test.AddJaneToken(req)
					resp, err := clnt.Do(req)
					log.Infof("resp = %+v\n", resp)
					log.Infof("err = %+v\n", err)
					Expect(err).NotTo(HaveOccurred())
				}()

				<-slowWaitC
				Expect(k8sAPI.UpdateCluster("slow")).ShouldNot(HaveOccurred())
				Expect(<-watchSync).NotTo(HaveOccurred())
				close(slowC) // let the call handler exit
				slow.Close()
				wg.Wait()
			})
		})*/
})

func requestToClusterA(address string) *http.Request {
	defer GinkgoRecover()
	req, err := http.NewRequest("GET",
		"https://"+address+"/some/path", strings.NewReader("HELLO"))
	Expect(err).ShouldNot(HaveOccurred())
	req.Header[server.ClusterHeaderField] = []string{clusterA}
	Expect(err).NotTo(HaveOccurred())
	return req
}

func configureHTTPSClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h2"},
			},
		},
	}
}

func listClusters(server string) ([]clusters.ManagedCluster, int) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "localhost",
		},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get("https://" + server + "/voltron/api/clusters")
	Expect(err).NotTo(HaveOccurred())

	if resp.StatusCode != 200 {
		return nil, resp.StatusCode
	}

	var list []clusters.ManagedCluster

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	Expect(err).NotTo(HaveOccurred())

	return list, 200
}

func clientHelloReq(addr string, target string, expectStatus int) {
	Eventually(func() bool {
		req, err := http.NewRequest("GET", "https://"+addr+"/some/path", strings.NewReader("HELLO"))
		Expect(err).NotTo(HaveOccurred())

		req.Header[server.ClusterHeaderField] = []string{target}
		req.Header.Set(authentication.AuthorizationHeader, janeBearerToken)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         "localhost",
			},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)

		return err == nil && resp.StatusCode == expectStatus
	}, 2*time.Second, 400*time.Millisecond).Should(BeTrue())
}

type mockKeySet struct {
	mock.Mock
}

// Test Verify method.
func (t *mockKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	args := t.Called(ctx, jwt)
	err := args.Get(1)
	if err != nil {
		return nil, err.(error)
	}
	return args.Get(0).([]byte), nil
}

func dex(keySet oidc.KeySet) authentication.Authenticator {
	dex, _ := auth.NewDexAuthenticator("https://127.0.0.1:9443/dex", "tigera-manager", "email",
		auth.WithKeySet(keySet),
		auth.WithGroupsClaim("groups"),
		auth.WithUsernamePrefix("-"),
	)
	return dex
}

func createAndStartServer(k8sAPI bootstrap.K8sClient, config *rest.Config, authenticator authentication.Authenticator,
	options ...server.Option) (*server.Server, string, string, *sync.WaitGroup) {

	srv, err := server.New(k8sAPI, config, authenticator, options...)
	Expect(err).ShouldNot(HaveOccurred())

	lisHTTPS, err := net.Listen("tcp", "localhost:0")
	Expect(err).NotTo(HaveOccurred())

	lisTun, err := net.Listen("tcp", "localhost:0")
	Expect(err).NotTo(HaveOccurred())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.ServeHTTPS(lisHTTPS, "", "")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.ServeTunnelsTLS(lisTun)
	}()

	go func() {
		_ = srv.WatchK8s()
	}()

	return srv, lisHTTPS.Addr().String(), lisTun.Addr().String(), &wg
}

func WaitForClusterToConnect(k8sAPI bootstrap.K8sClient, clusterName string) {
	Eventually(func() calicov3.ManagedClusterStatus {
		managedCluster, err := k8sAPI.ManagedClusters().Get(context.Background(), clusterName, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		return managedCluster.Status
	}, 5*time.Second, 100*time.Millisecond).Should(Equal(calicov3.ManagedClusterStatus{
		Conditions: []calicov3.ManagedClusterStatusCondition{
			{Status: calicov3.ManagedClusterStatusValueTrue, Type: calicov3.ManagedClusterStatusTypeConnected},
		},
	}))
}
