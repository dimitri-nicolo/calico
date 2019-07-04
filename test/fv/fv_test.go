// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package fv_test

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/tigera/voltron/internal/pkg/client"
	"github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/proxy"
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

func (c *testClient) addCluster(id string) error {
	cluster, err := json.Marshal(&clusters.Cluster{ID: id, DisplayName: id})
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest("PUT",
		"https://"+c.voltronHTTPS+"/voltron/api/clusters?", bytes.NewBuffer(cluster))
	Expect(err).NotTo(HaveOccurred())

	resp, err := c.http.Do(req)
	Expect(err).NotTo(HaveOccurred())

	if resp.StatusCode != 200 {
		return errors.Errorf("addCluster: StatusCode %d", resp.StatusCode)
	}

	return nil
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

	k8sAPI := fake.NewSimpleClientset()
	test.AddJaneIdentity(k8sAPI)

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
			ts.http.Serve(lisTs)
		}()

		lisTs2, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		ts2 = newTestServer("you reached the other me")

		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			ts2.http.Serve(lisTs2)
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

		voltron, err = server.New(
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
			server.WithAuthentication(true, k8sAPI),
		)
		Expect(err).NotTo(HaveOccurred())

		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			voltron.ServeHTTP(lisHTTP11)
		}()

		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			voltron.ServeHTTP(lisHTTP2)
		}()

		wgSrvCnlt.Add(1)
		go func() {
			defer wgSrvCnlt.Done()
			voltron.ServeTunnelsTLS(lisTun)
		}()

		ui.voltronHTTPS = lisHTTP2.Addr().String()
		ui.voltronHTTP = lisHTTP11.Addr().String()
	})

	It("should register 2 clusters", func() {
		err := ui.addCluster(clusterID)
		Expect(err).NotTo(HaveOccurred())

		err = ui.addCluster(clusterID2)
		Expect(err).NotTo(HaveOccurred())
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
			guardian.ServeTunnelHTTP()
		}()
	})

	It("should wait for tunnel", func() {
		err := guardian.WaitForTunnel()
		Expect(err).NotTo(HaveOccurred())
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
			guardian2.ServeTunnelHTTP()
		}()
	})

	It("should wait for tunnel", func() {
		err := guardian2.WaitForTunnel()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be possible to reach the test server on http2", func() {
		var msg string
		Eventually(func() error {
			var err error
			msg, err = ui.doRequest(clusterID)
			return err
		}).ShouldNot(HaveOccurred())
		Expect(msg).To(Equal(ts.msg))
	})

	It("should be possible to reach the other test server on http2", func() {
		var msg string
		Eventually(func() error {
			var err error
			msg, err = ui.doRequest(clusterID2)
			return err
		}).ShouldNot(HaveOccurred())
		Expect(msg).To(Equal(ts2.msg))
	})

	It("should be possible to reach the test server on http", func() {
		msg, err := ui.doHTTPRequest(clusterID)
		Expect(err).NotTo(HaveOccurred())
		Expect(msg).To(Equal(ts.msg))
	})

	It("should be possible to stop guardian", func() {
		guardian.Close()
	})

	It("should not be possible to reach the test server", func() {
		_, err := ui.doRequest(clusterID)
		Expect(err).To(HaveOccurred())
	})

	It("should start guardian again", func() {
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
			guardian.ServeTunnelHTTP()
		}()
	})

	It("should wait for tunnel to come up again", func() {
		err := guardian.WaitForTunnel()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be possible to reach the test server again", func() {
		var msg string
		Eventually(func() error {
			var err error
			msg, err = ui.doRequest(clusterID)
			return err
		}).ShouldNot(HaveOccurred())
		Expect(msg).To(Equal(ts.msg))
	})

	It("should clean up", func(done Done) {
		voltron.Close()
		guardian.Close()
		guardian2.Close()
		ts.http.Close()
		ts2.http.Close()
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
