// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package fv_test

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"

	"github.com/projectcalico/calico/apiserver/pkg/authentication"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/calico/voltron/internal/pkg/client"
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
)

func init() {
	var err error
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)

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
	Expect(err).NotTo(HaveOccurred())
	resp, err := c.http.Do(req)
	Expect(err).NotTo(HaveOccurred())

	if resp.StatusCode != 200 {
		return "", errors.Errorf("error status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	return string(body), nil
}

func (c *testClient) doHTTPRequest(clusterID string) (string, error) {
	req, err := c.request(clusterID, "http", c.voltronHTTP)
	Expect(err).NotTo(HaveOccurred())
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())

	if resp.StatusCode != 200 {
		return "", errors.Errorf("error status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	return string(body), nil
}

func (c *testClient) request(clusterID string, schema string, address string) (*http.Request, error) {
	req, err := http.NewRequest("GET", schema+"://"+address+"/some/path", strings.NewReader("HELLO"))
	Expect(err).NotTo(HaveOccurred())
	req.Header[server.ClusterHeaderField] = []string{clusterID}
	req.Header.Set(authentication.AuthorizationHeader, "Bearer jane")
	Expect(err).NotTo(HaveOccurred())
	return req, err
}

type testServer struct {
	msg  string
	http *http.Server
}

func (s *testServer) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, s.msg)
}

var _ = Describe("Voltron-Guardian interaction", func() {
	var (
		voltron   *server.Server
		lisHTTP11 net.Listener
		lisHTTP2  net.Listener
		lisTun    net.Listener

		guardian  *client.Client
		guardian2 *client.Client

		ts    *testServer
		lisTs net.Listener

		ts2    *testServer
		lisTs2 net.Listener

		wgSrvCnlt sync.WaitGroup
	)

	clusterID := "external-cluster"
	clusterID2 := "other-cluster"

	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
	authenticator := new(auth.MockJWTAuth)
	authenticator.On("Authenticate", mock.Anything).Return(&user.DefaultInfo{Name: "jane", Groups: []string{"developers"}}, 0, nil)
	watchSync := make(chan error)

	// client to be used to interact with voltron (mimic UI)
	ui := &testClient{
		http: &http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	It("should start test servers", func() {
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

	It("should start voltron", func() {
		var err error

		lisHTTP11, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		lisHTTP2, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		lisTun, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		tunnelTargetWhitelist, _ := regex.CompileRegexStrings([]string{
			`^/$`,
			`^/some/path$`,
		})

		voltron, err = server.New(
			k8sAPI,
			&rest.Config{BearerToken: "manager-token"},
			authenticator,
			server.WithTunnelSigningCreds(tunnelCert),
			server.WithTunnelCert(tunnelTLS),
			server.WithExternalCredsFiles("../../internal/pkg/server/testdata/localhost.pem", "../../internal/pkg/server/testdata/localhost.key"),
			server.WithInternalCredFiles("../../internal/pkg/server/testdata/tigera-manager-svc.pem", "../../internal/pkg/server/testdata/tigera-manager-svc.key"),
			server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
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

	var certPemID1, keyPemID1, certPemID2, keyPemID2 []byte
	var fingerprintID1, fingerprintID2 string

	It("should register 2 clusters", func() {
		k8sAPI.WaitForManagedClustersWatched()
		var err error
		certPemID1, keyPemID1, fingerprintID1, err = test.GenerateTestCredentials(clusterID, tunnelCert, tunnelPrivKey)
		Expect(err).NotTo(HaveOccurred())
		annotationsID1 := map[string]string{server.AnnotationActiveCertificateFingerprint: fingerprintID1}

		Expect(k8sAPI.AddCluster(clusterID, clusterID, annotationsID1)).ShouldNot(HaveOccurred())
		Expect(<-watchSync).NotTo(HaveOccurred())

		certPemID2, keyPemID2, fingerprintID2, err = test.GenerateTestCredentials(clusterID2, tunnelCert, tunnelPrivKey)
		Expect(err).NotTo(HaveOccurred())
		annotationsID2 := map[string]string{server.AnnotationActiveCertificateFingerprint: fingerprintID2}

		Expect(k8sAPI.AddCluster(clusterID2, clusterID2, annotationsID2)).ShouldNot(HaveOccurred())
		Expect(<-watchSync).NotTo(HaveOccurred())
	})

	It("should start guardian", func() {
		var err error

		guardian, err = client.New(
			lisTun.Addr().String(),
			client.WithTunnelCreds(certPemID1, keyPemID1),
			client.WithTunnelRootCA(rootCAs),
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

	It("should start guardian2", func() {
		var err error

		guardian2, err = client.New(
			lisTun.Addr().String(),
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
		)
		Expect(err).NotTo(HaveOccurred())
		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			_ = guardian2.ServeTunnelHTTP()
		}()
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

	It("should be possible to stop guardian", func() {
		_ = guardian.Close()
	})

	It("should not be possible to reach the test server", func() {
		_, err := ui.doRequest(clusterID)
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

	It("should clean up", func(done Done) {
		_ = voltron.Close()
		_ = guardian.Close()
		_ = guardian2.Close()
		_ = ts.http.Close()
		_ = ts2.http.Close()
		wgSrvCnlt.Wait()

		close(done)
	})
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
