// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/voltron/internal/pkg/bootstrap"
	"github.com/projectcalico/calico/voltron/internal/pkg/proxy"
	"github.com/projectcalico/calico/voltron/internal/pkg/server"

	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

var _ = Describe("Creating an HTTPS server that only proxies traffic", func() {
	var (
		k8sAPI bootstrap.K8sClient

		mockAuthenticator  *auth.MockJWTAuth
		srv                *server.Server
		externalCertFile   string
		externalKeyFile    string
		internalCertFile   string
		internalKeyFile    string
		externalCACert     string
		externalServerName string
		internalCACert     string
		internalServerName string
		listener           net.Listener
		address            net.Addr
	)

	JustBeforeEach(func() {
		var err error
		k8sAPI = &k8sClient{
			Interface:                k8sfake.NewSimpleClientset(),
			ProjectcalicoV3Interface: fake.NewSimpleClientset().ProjectcalicoV3(),
		}

		mockAuthenticator = new(auth.MockJWTAuth)

		By("Creating a default destination server that return 200 OK")
		defaultServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte("Success"))
			}))

		defaultURL, err := url.Parse(defaultServer.URL)
		Expect(err).NotTo(HaveOccurred())

		By("Creating a default proxy to proxy traffic to the default destination")
		defaultProxy, err := proxy.New([]proxy.Target{
			{
				Path: "/",
				Dest: defaultURL,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		By("Creating TCP listener to help communication")
		listener, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())
		address = listener.Addr()
		Expect(address).ShouldNot(BeNil())
		Expect(err).NotTo(HaveOccurred())

		By("Creating and starting server that only serves HTTPS traffic")
		var opts = []server.Option{
			server.WithDefaultAddr(address.String()),
			server.WithKeepAliveSettings(true, 100),
			server.WithExternalCredsFiles(externalCertFile, externalKeyFile),
			server.WithInternalCredFiles(internalCertFile, internalKeyFile),
			server.WithDefaultProxy(defaultProxy),
		}

		srv, err = server.New(
			k8sAPI,
			config,
			mockAuthenticator,
			opts...,
		)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			_ = srv.ServeHTTPS(listener, "", "")
		}()
	})

	assertHTTPSServerBehaviour := func() {
		It("Does not initiate a tunnel server when the tunnel destination doesn't have tls certificates", func() {
			var err = srv.ServeTunnelsTLS(listener)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("No tunnel server was initiated"))
		})

		It("Receives 200 OK when reaching the proxy server using HTTPS using the external server name", func() {
			var err error

			var rootCAs = x509.NewCertPool()
			caCert, err := ioutil.ReadFile(externalCACert)
			Expect(err).NotTo(HaveOccurred())
			rootCAs.AppendCertsFromPEM(caCert)

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    rootCAs,
					ServerName: externalServerName,
				},
			}
			client := &http.Client{Transport: tr}
			req, err := http.NewRequest("GET", "https://"+address.String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("Success"))
		})

		It("Receives 200 OK when reaching the proxy server using HTTPS using the internal server name", func() {
			var err error

			var rootCAs = x509.NewCertPool()
			caCert, err := ioutil.ReadFile(internalCACert)
			Expect(err).NotTo(HaveOccurred())
			rootCAs.AppendCertsFromPEM(caCert)

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    rootCAs,
					ServerName: internalServerName,
				},
			}
			client := &http.Client{Transport: tr}
			req, err := http.NewRequest("GET", "https://"+address.String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("Success"))
		})

		It("Receives 200 OK when reaching the proxy server using HTTP 2", func() {
			var err error

			var rootCAs = x509.NewCertPool()
			caCert, err := ioutil.ReadFile(externalCACert)
			Expect(err).NotTo(HaveOccurred())
			rootCAs.AppendCertsFromPEM(caCert)

			tr := &http2.Transport{
				TLSClientConfig: &tls.Config{
					NextProtos: []string{"h2"},
					RootCAs:    rootCAs,
					ServerName: externalServerName,
				},
			}

			client := &http.Client{Transport: tr}
			req, err := http.NewRequest("GET", "https://"+address.String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("Success"))
		})

		It("Receives 400 when reaching the proxy server using HTTP", func() {
			var err error
			req, err := http.NewRequest("GET", "http://"+address.String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	}

	Context("Using self signed root CA certs with DNS set to voltron", func() {
		BeforeEach(func() {
			internalCertFile = "testdata/rootCA-tunnel-generation.pem"
			externalCertFile = "testdata/rootCA-tunnel-generation.pem"
			internalKeyFile = "testdata/rootCA-tunnel-generation.key"
			externalKeyFile = "testdata/rootCA-tunnel-generation.key"
			externalCACert = "testdata/rootCA-tunnel-generation.pem"
			internalCACert = "testdata/rootCA-tunnel-generation.pem"
			externalServerName = "voltron"
			internalServerName = "voltron"
		})

		assertHTTPSServerBehaviour()
	})

	Context("Using certs that have PKCS8 Key format", func() {
		BeforeEach(func() {
			internalCertFile = "testdata/cert-pkcs8-format.pem"
			externalCertFile = "testdata/cert-pkcs8-format.pem"
			internalKeyFile = "testdata/key-pkcs8-format.pem"
			externalKeyFile = "testdata/key-pkcs8-format.pem"
			externalCACert = "testdata/cert-pkcs8-format.pem"
			internalCACert = "testdata/cert-pkcs8-format.pem"
			externalServerName = "cnx-manager.calico-monitoring.svc"
			internalServerName = "cnx-manager.calico-monitoring.svc"
		})

		assertHTTPSServerBehaviour()
	})

	Context("Using two set of certs for internal and external HTTPS traffic", func() {
		BeforeEach(func() {
			externalCertFile = "testdata/localhost.pem"
			internalCertFile = "testdata/tigera-manager-svc.pem"
			externalKeyFile = "testdata/localhost.key"
			internalKeyFile = "testdata/tigera-manager-svc.key"
			externalCACert = "testdata/localhost-intermediate-CA.pem"
			internalCACert = "testdata/tigera-manager-svc-intermediate-CA.pem"
			externalServerName = "localhost"
			internalServerName = "tigera-manager.tigera-manager.svc"
		})

		assertHTTPSServerBehaviour()
	})
})
