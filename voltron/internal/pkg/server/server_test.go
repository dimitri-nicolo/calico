// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package server_test

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/http2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/projectcalico/calico/apiserver/pkg/authentication"
	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/auth/testing"
	"github.com/projectcalico/calico/voltron/internal/pkg/bootstrap"
	"github.com/projectcalico/calico/voltron/internal/pkg/proxy"
	"github.com/projectcalico/calico/voltron/internal/pkg/regex"
	"github.com/projectcalico/calico/voltron/internal/pkg/server"
	"github.com/projectcalico/calico/voltron/internal/pkg/test"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"
	"github.com/projectcalico/calico/voltron/pkg/tunnel"

	calicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	clientv3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

const (
	k8sIssuer           = "kubernetes/serviceaccount"
	managerSAAuthHeader = "Bearer tigera-manager-token"
)

var (
	clusterA = "clusterA"
	clusterB = "clusterB"
	config   = &rest.Config{BearerToken: "tigera-manager-token"}

	// Tokens issued by k8s.
	janeBearerToken = testing.NewFakeJWT(k8sIssuer, "jane@example.io")
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

type k8sClient struct {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

var _ = Describe("Server Proxy to tunnel", func() {
	fipsmode := false
	var (
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

		k8sAPI = &k8sClient{
			Interface:                k8sfake.NewSimpleClientset(),
			ProjectcalicoV3Interface: fake.NewSimpleClientset().ProjectcalicoV3(),
		}

		voltronTunnelCertTemplate := test.CreateCACertificateTemplate("voltron")
		voltronTunnelPrivKey, voltronTunnelCert, err = test.CreateCertPair(voltronTunnelCertTemplate, nil, nil)
		Expect(err).ShouldNot(HaveOccurred())

		// convert x509 cert to tls cert
		voltronTunnelTLSCert, err = test.X509CertToTLSCert(voltronTunnelCert, voltronTunnelPrivKey)
		Expect(err).NotTo(HaveOccurred())

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
		mockAuthenticator := new(auth.MockJWTAuth)
		_, err := server.New(
			k8sAPI,
			config,
			mockAuthenticator,
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

			mockAuthenticator := new(auth.MockJWTAuth)
			mockAuthenticator.On("Authenticate", mock.Anything).Return(
				&user.DefaultInfo{
					Name:   "jane@example.io",
					Groups: []string{"developers"},
				}, 0, nil)

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
				mockAuthenticator,
				server.WithTunnelSigningCreds(voltronTunnelCert),
				server.WithTunnelCert(voltronTunnelTLSCert),
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
				list, err := k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(list.Items).To(HaveLen(0))
			})

			It("should be able to register multiple clusters", func() {
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterA},
				}, metav1.CreateOptions{})

				Expect(err).ShouldNot(HaveOccurred())
				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})

				Expect(err).ShouldNot(HaveOccurred())

				list, err := k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(list.Items).To(HaveLen(2))
				Expect(list.Items[0].GetObjectMeta().GetName()).To(Equal("clusterA"))
				Expect(list.Items[1].GetObjectMeta().GetName()).To(Equal("clusterB"))
			})

			It("should be able to list the remaining clusters after deleting one", func() {
				By("adding two cluster")
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterA},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				list, err := k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(list.Items).To(HaveLen(2))
				Expect(list.Items[0].GetObjectMeta().GetName()).To(Equal("clusterA"))
				Expect(list.Items[1].GetObjectMeta().GetName()).To(Equal("clusterB"))

				By("removing one cluster")
				Expect(k8sAPI.ManagedClusters().Delete(context.Background(), clusterB, metav1.DeleteOptions{})).ShouldNot(HaveOccurred())

				list, err = k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(list.Items).To(HaveLen(1))
				Expect(list.Items[0].GetObjectMeta().GetName()).To(Equal("clusterA"))
			})

			It("should be able to register clusterB after it's been deleted again", func() {
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(k8sAPI.ManagedClusters().Delete(context.Background(), clusterB, metav1.DeleteOptions{})).ShouldNot(HaveOccurred())

				list, err := k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(list.Items).To(HaveLen(0))

				_, err = k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{Name: clusterB},
				}, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				list, err = k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(list.Items).To(HaveLen(1))
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
				_, err := k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
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
				req.Header.Set(authentication.AuthorizationHeader, janeBearerToken.BearerTokenHeader())

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
				req.Header.Set(authentication.AuthorizationHeader, janeBearerToken.BearerTokenHeader())

				resp, err := configureHTTPSClient().Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get(authentication.AuthorizationHeader)).To(Equal(managerSAAuthHeader))
			})

			Context("A single cluster is registered", func() {
				var (
					clusterATLSCert tls.Certificate
				)

				BeforeEach(func() {
					clusterACertTemplate := test.CreateClientCertificateTemplate(clusterA, "localhost")
					clusterAPrivKey, clusterACert, err := test.CreateCertPair(clusterACertTemplate, voltronTunnelCert, voltronTunnelPrivKey)
					Expect(err).ShouldNot(HaveOccurred())

					_, err = k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:        clusterA,
							Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: utils.GenerateFingerprint(fipsmode, clusterACert)},
						},
					}, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					clusterATLSCert, err = test.X509CertToTLSCert(clusterACert, clusterAPrivKey)
					Expect(err).NotTo(HaveOccurred())
				})

				It("can send requests from the server to the cluster", func() {
					tun, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
						Certificates: []tls.Certificate{clusterATLSCert},
						RootCAs:      voltronTunnelCAs,
					}, 5*time.Second)
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
					req.Header.Set(authentication.AuthorizationHeader, janeBearerToken.BearerTokenHeader())

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
						_, err = k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
							ObjectMeta: metav1.ObjectMeta{
								Name:        clusterB,
								Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: utils.GenerateFingerprint(fipsmode, clusterBCert)},
							},
						}, metav1.CreateOptions{})
						Expect(err).ShouldNot(HaveOccurred())

						clusterBTLSCert, err = test.X509CertToTLSCert(clusterBCert, clusterBPrivKey)
						Expect(err).NotTo(HaveOccurred())
					})

					It("can send requests from the server to the second cluster", func() {
						tun, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
							Certificates: []tls.Certificate{clusterBTLSCert},
							RootCAs:      voltronTunnelCAs,
						}, 5*time.Second)
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
						req.Header.Set(authentication.AuthorizationHeader, janeBearerToken.BearerTokenHeader())

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
						}, 5*time.Second)
						Expect(err).NotTo(HaveOccurred())

						tunB2, err := tunnel.DialTLS(tunnelAddr, &tls.Config{
							Certificates: []tls.Certificate{clusterBTLSCert},
							RootCAs:      voltronTunnelCAs,
						}, 5*time.Second)
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
			wg            *sync.WaitGroup
			srv           *server.Server
			tunnelAddr    string
			defaultServer *httptest.Server
		)

		BeforeEach(func() {
			mockAuthenticator := new(auth.MockJWTAuth)
			mockAuthenticator.On("Authenticate", mock.Anything).Return(
				&user.DefaultInfo{
					Name:   "jane@example.io",
					Groups: []string{"developers"},
				}, 0, nil)
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

			// convert x509 cert to tls cert
			voltronTunnelTLSCert, err = test.X509CertToTLSCert(voltronTunnelCert, voltronTunnelPrivKey)
			Expect(err).NotTo(HaveOccurred())

			voltronTunnelCAs = x509.NewCertPool()
			voltronTunnelCAs.AppendCertsFromPEM(test.CertToPemBytes(voltronTunnelCert))

			srv, _, tunnelAddr, wg = createAndStartServer(
				k8sAPI,
				config,
				mockAuthenticator,
				server.WithTunnelSigningCreds(voltronTunnelCert),
				server.WithTunnelCert(voltronTunnelTLSCert),
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
			_, err = k8sAPI.ManagedClusters().Create(context.Background(), &calicov3.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        clusterA,
					Annotations: map[string]string{server.AnnotationActiveCertificateFingerprint: utils.GenerateFingerprint(fipsmode, cert)},
				},
			}, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			list, err := k8sAPI.ManagedClusters().List(context.Background(), metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Items).To(HaveLen(1))

			// Try to connect clusterA to the new fake voltron, should fail
			tlsCert, err := test.X509CertToTLSCert(cert, privKey)
			Expect(err).NotTo(HaveOccurred())

			_, err = tunnel.DialTLS(tunnelAddr, &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
				RootCAs:      voltronTunnelCAs,
				ServerName:   "voltron",
			}, 5*time.Second)
			Expect(err).Should(MatchError("tcp.tls.Dial failed: x509: certificate specifies an incompatible key usage"))
		})
	})
})

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

func clientHelloReq(addr string, target string, expectStatus int) {
	Eventually(func() bool {
		req, err := http.NewRequest("GET", "https://"+addr+"/some/path", strings.NewReader("HELLO"))
		Expect(err).NotTo(HaveOccurred())

		req.Header[server.ClusterHeaderField] = []string{target}
		req.Header.Set(authentication.AuthorizationHeader, janeBearerToken.BearerTokenHeader())
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

func createAndStartServer(k8sAPI bootstrap.K8sClient, config *rest.Config, authenticator auth.JWTAuth,
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
