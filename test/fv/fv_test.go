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

	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"

	"github.com/tigera/voltron/internal/pkg/client"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/regex"
	"github.com/tigera/voltron/internal/pkg/server"
	"github.com/tigera/voltron/internal/pkg/test"
	"github.com/tigera/voltron/internal/pkg/utils"
)

var (
	srvCert    *x509.Certificate
	srvPrivKey *rsa.PrivateKey
	rootCAs    *x509.CertPool
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)

	srvCert, _ = test.CreateSelfSignedX509Cert("voltron", true)

	block, _ := pem.Decode([]byte(test.PrivateRSA))
	srvPrivKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

	rootCAs = x509.NewCertPool()
	rootCAs.AddCert(srvCert)
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
	test.AddJaneToken(req)
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

	clusterID := "cluster"
	clusterID2 := "other-cluster"

	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
	k8sAPI.AddJaneIdentity()
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

		// we need some credentials
		key, _ := utils.KeyPEMEncode(srvPrivKey)
		cert := utils.CertPEMEncode(srvCert)

		xcert, _ := tls.X509KeyPair(cert, key)

		lisHTTP11, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		lisHTTP2, err = tls.Listen("tcp", "localhost:0", &tls.Config{
			Certificates: []tls.Certificate{xcert},
			NextProtos:   []string{"h2"},
		})
		Expect(err).NotTo(HaveOccurred())

		lisTun, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		tunnelTargetWhitelist, _ := regex.CompileRegexStrings([]string{
			`^/$`,
			`^/some/path$`,
		})

		voltron, err = server.New(
			k8sAPI,
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
			server.WithAuthentication(&rest.Config{}),
			server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
			server.WithWatchAdded(),
		)
		Expect(err).NotTo(HaveOccurred())

		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			_ = voltron.ServeHTTP(lisHTTP11)
		}()

		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			_ = voltron.ServeHTTP(lisHTTP2)
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

	It("should register 2 clusters", func() {
		var err error
		var cert []byte
		var block *pem.Block
		var certCluster, otherCertCluster *x509.Certificate

		k8sAPI.WaitForManagedClustersWatched()
		Expect(k8sAPI.AddCluster(clusterID, clusterID, nil)).ShouldNot(HaveOccurred())
		Expect(<-watchSync).NotTo(HaveOccurred())

		cert, _, err = voltron.ClusterCreds(clusterID)
		Expect(err).NotTo(HaveOccurred())
		block, _ = pem.Decode(cert)
		Expect(block).NotTo(BeNil())
		certCluster, err = x509.ParseCertificate(block.Bytes)
		Expect(err).NotTo(HaveOccurred())

		Expect(err).NotTo(HaveOccurred())
		Expect(k8sAPI.UpdateCluster(clusterID, map[string]string{server.AnnotationActiveCertificateFingerprint: utils.GenerateFingerprint(certCluster)})).ShouldNot(HaveOccurred())

		Expect(k8sAPI.AddCluster(clusterID2, clusterID2, nil)).ShouldNot(HaveOccurred())
		Expect(<-watchSync).NotTo(HaveOccurred())

		cert, _, err = voltron.ClusterCreds(clusterID2)
		Expect(err).NotTo(HaveOccurred())
		block, _ = pem.Decode(cert)
		Expect(block).NotTo(BeNil())
		otherCertCluster, err = x509.ParseCertificate(block.Bytes)
		Expect(err).NotTo(HaveOccurred())

		Expect(err).NotTo(HaveOccurred())
		Expect(k8sAPI.UpdateCluster(clusterID2, map[string]string{server.AnnotationActiveCertificateFingerprint: utils.GenerateFingerprint(otherCertCluster)})).ShouldNot(HaveOccurred())

	})

	It("should start guardian", func() {
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

	It("should start guardian2", func() {
		cert, key, err := voltron.ClusterCreds(clusterID2)
		Expect(err).NotTo(HaveOccurred())

		guardian2, err = client.New(
			lisTun.Addr().String(),
			client.WithTunnelCreds(cert, key, rootCAs),
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

	It("should be possible to reach the test server on http", func() {
		msg, err := ui.doHTTPRequest(clusterID)
		Expect(err).NotTo(HaveOccurred())
		Expect(msg).To(Equal(ts.msg))
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
