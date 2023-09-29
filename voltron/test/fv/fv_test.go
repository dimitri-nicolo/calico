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
	"net"
	"net/http"
	"net/url"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kscheme "k8s.io/client-go/kubernetes/scheme"

	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"

	"github.com/projectcalico/calico/apiserver/pkg/authentication"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/mock"

	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

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
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		return "", errors.Errorf("error status: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
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

	body, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

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
	msg  string
	http *http.Server
}

func (s *testServer) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, s.msg)
}

func describe(name string, testFn func(string)) bool {
	Describe(name+" cluster-scoped", func() { testFn("") })
	Describe(name+" namespace-scoped", func() { testFn("resource-ns") })
	return true
}

var _ = describe("basic functionality", func(clusterNamespace string) {
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
		fipsMode  = true
	)

	clusterID := "external-cluster"
	clusterID2 := "other-cluster"
	clusterNS := clusterNamespace

	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
	scheme := kscheme.Scheme
	err := v3.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()

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
			fakeClient,
			&rest.Config{BearerToken: "manager-token"},
			vcfg.Config{TenantNamespace: clusterNS},
			authenticator,
			server.WithTunnelSigningCreds(tunnelCert),
			server.WithTunnelCert(tunnelTLS),
			server.WithExternalCredFiles("../../internal/pkg/server/testdata/localhost.pem", "../../internal/pkg/server/testdata/localhost.key"),
			server.WithInternalCredFiles("../../internal/pkg/server/testdata/tigera-manager-svc.pem", "../../internal/pkg/server/testdata/tigera-manager-svc.key"),
			server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
			server.WithFIPSModeEnabled(fipsMode),
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

	It("should register 2 clusters", func() {
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

	It("should start guardian", func() {

		var err error

		guardian, err = client.New(
			lisTun.Addr().String(),
			"voltron",
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
