// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server_test

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/server"
	"github.com/tigera/voltron/internal/pkg/test"
	"github.com/tigera/voltron/internal/pkg/utils"
	"github.com/tigera/voltron/pkg/tunnel"
	"k8s.io/client-go/kubernetes/fake"
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

var _ = Describe("Server", func() {
	var (
		err error
		wg  sync.WaitGroup
		srv *server.Server
		lis net.Listener
	)

	client := fake.NewSimpleClientset()
	test.AddJaneIdentity(client)

	It("should fail to use invalid path", func() {
		_, err := server.New(
			nil,
			server.WithCredsFiles("dog/gopher.crt", "dog/gopher.key"),
		)
		Expect(err).To(HaveOccurred())
	})

	It("should start a server", func() {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		srv, err = server.New(
			client,
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
			server.WithAuthentication(),
		)
		Expect(err).NotTo(HaveOccurred())
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = srv.ServeHTTP(lis)
		}()
	})

	Context("when server is up", func() {
		It("should get empty list", func() {
			var list []clusters.Cluster

			Eventually(func() int {
				var code int
				list, code = listClusters(lis.Addr().String())
				return code
			}).Should(Equal(200))
			Expect(len(list)).To(Equal(0))
		})

		It("should be able to register a new cluster", func() {
			addCluster(lis.Addr().String(), "clusterA", "A")
		})

		It("should be able to get the clusters creds", func() {
			cert, key, err := srv.ClusterCreds("clusterA")
			Expect(err).NotTo(HaveOccurred())
			Expect(cert != nil && key != nil).To(BeTrue())
		})

		It("should be able to list the cluster", func() {
			list, code := listClusters(lis.Addr().String())
			Expect(code).To(Equal(200))
			Expect(len(list)).To(Equal(1))
			Expect(list[0].ID).To(Equal("clusterA"))
			Expect(list[0].DisplayName).To(Equal("A"))
		})

		It("should be able to update a cluster", func() {
			addCluster(lis.Addr().String(), "clusterA", "AAA")
		})

		It("should be able to register another cluster", func() {
			addCluster(lis.Addr().String(), "clusterB", "BB")
		})

		It("should be able to get sorted list of clusters", func() {
			list, code := listClusters(lis.Addr().String())
			Expect(code).To(Equal(200))
			Expect(len(list)).To(Equal(2))
			Expect(list[0].ID).To(Equal("clusterA"))
			Expect(list[0].DisplayName).To(Equal("AAA"))
			Expect(list[1].ID).To(Equal("clusterB"))
			Expect(list[1].DisplayName).To(Equal("BB"))
		})

		It("should be able to delete a cluster", func() {
			Expect(deleteCluster(lis.Addr().String(), "clusterB")).To(BeTrue())
		})

		It("should be able to get list without the deleted cluster", func() {
			list, code := listClusters(lis.Addr().String())
			Expect(code).To(Equal(200))
			Expect(len(list)).To(Equal(1))
			Expect(list[0].ID).To(Equal("clusterA"))
		})

		It("should not be able to delete the cluster again", func() {
			Expect(deleteCluster(lis.Addr().String(), "clusterB")).NotTo(BeTrue())
		})

		It("Should not proxy anywhere - no header", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	})

	It("should stop the server", func(done Done) {
		cerr := srv.Close()
		Expect(cerr).NotTo(HaveOccurred())
		wg.Wait()
		Expect(err).To(HaveOccurred())
		close(done)
	})
})

var _ = Describe("Server Proxy to tunnel", func() {
	var (
		err    error
		wg     sync.WaitGroup
		srv    *server.Server
		lis    net.Listener
		lis2   net.Listener
		lisTun net.Listener
		key    []byte
		cert   []byte
	)

	client := fake.NewSimpleClientset()
	test.AddJaneIdentity(client)

	defaultServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	It("Should get some credentials for the server", func() {
		key, _ = utils.KeyPEMEncode(srvPrivKey)
		cert = utils.CertPEMEncode(srvCert)
	})

	startServer := func(opts ...server.Option) {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		xcert, _ := tls.X509KeyPair(cert, key)

		lis2, e = tls.Listen("tcp", "localhost:0", &tls.Config{
			Certificates: []tls.Certificate{xcert},
			NextProtos:   []string{"h2"},
		})

		lisTun, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		defaultURL, e := url.Parse(defaultServer.URL)
		Expect(e).NotTo(HaveOccurred())

		defaultProxy, e := proxy.New([]proxy.Target{{
			Path: "/",
			Dest: defaultURL,
		}})
		Expect(e).NotTo(HaveOccurred())

		opts = append(opts,
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
			server.WithAuthentication(),
			server.WithDefaultProxy(defaultProxy),
		)

		srv, err = server.New(
			client,
			opts...,
		)
		Expect(err).NotTo(HaveOccurred())

		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.ServeHTTP(lis)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.ServeHTTP(lis2)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.ServeTunnelsTLS(lisTun)
		}()
	}

	It("should start a server", func() {
		startServer()
	})

	Context("when server is up", func() {
		It("Should not proxy anywhere - invalid cluster", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add(server.ClusterHeaderField, "zzzzzzz")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Should not proxy anywhere - multiple headers", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add(server.ClusterHeaderField, "clusterA")
			req.Header.Add(server.ClusterHeaderField, "helloworld")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should not be able to proxy to a cluster without a tunnel", func() {
			addCluster(lis.Addr().String(), "clusterA", "A")
			clientHelloReq(lis.Addr().String(), "clusterA", 503)
		})

		It("Should proxy to default if no header", func() {
			resp, err := http.Get(defaultServer.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})

		var clnT *tunnel.Tunnel

		var certPemA, keyPemA []byte

		It("should be possible to open a tunnel", func() {
			var err error

			certPemA, keyPemA, err = srv.ClusterCreds("clusterA")
			Expect(err).NotTo(HaveOccurred())

			cert, err := tls.X509KeyPair(certPemA, keyPemA)
			Expect(err).NotTo(HaveOccurred())

			cfg := &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      rootCAs,
			}

			clnT, err = tunnel.DialTLS(lisTun.Addr().String(), cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		// assumes clnT to be set to the tunnel we test
		testClnT := func() {
			It("should be possible to make HTTP/2 connection", func() {
				var wg sync.WaitGroup

				wg.Add(1)
				go func() {
					defer wg.Done()
					http2Srv(clnT)
				}()

				clnt := &http.Client{
					Transport: &http2.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
							NextProtos:         []string{"h2"},
						},
					},
				}

				req, err := http.NewRequest("GET",
					"https://"+lis2.Addr().String()+"/some/path", strings.NewReader("HELLO"))
				Expect(err).NotTo(HaveOccurred())
				req.Header[server.ClusterHeaderField] = []string{"clusterA"}
				test.AddJaneToken(req)

				var resp *http.Response

				Eventually(func() bool {
					var err error
					resp, err = clnt.Do(req)
					return err == nil && resp.StatusCode == 200
				}).Should(BeTrue())

				i := 0
				for {
					data := make([]byte, 100)
					n, err := resp.Body.Read(data)
					if err != nil {
						break
					}
					Expect(string(data[:n])).To(Equal(fmt.Sprintf("tick %d\n", i)))
					i++
				}
				wg.Wait()
			})
		}

		Context("when client tunnel exists", func() {
			testClnT()
		})

		When("opening another tunnel", func() {
			var certPem, keyPem []byte

			It("should fail to get creds if it does not exist yet", func() {
				var err error
				certPem, keyPem, err = srv.ClusterCreds("clusterB")
				Expect(err).To(HaveOccurred())
			})

			It("should be able to register another cluster", func() {
				addCluster(lis.Addr().String(), "clusterB", "BB")
			})

			When("another cluster is registered", func() {
				var cfgB *tls.Config

				It("shold be possible to get creds for clusterB", func() {
					var err error
					certPem, keyPem, err = srv.ClusterCreds("clusterB")
					Expect(err).NotTo(HaveOccurred())

					cert, err := tls.X509KeyPair(certPem, keyPem)
					Expect(err).NotTo(HaveOccurred())

					cfgB = &tls.Config{
						Certificates: []tls.Certificate{cert},
						RootCAs:      rootCAs,
					}
				})

				var tunB *tunnel.Tunnel

				It("should be possible to open tunnel from clusterB", func() {
					var err error

					tunB, err = tunnel.DialTLS(lisTun.Addr().String(), cfgB)
					Expect(err).NotTo(HaveOccurred())
				})

				It("eventually accepting connections succeeds", func() {
					testSrv := httptest.NewUnstartedServer(
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
					testSrv.Listener = tunB
					testSrv.Start()

					clientHelloReq(lis.Addr().String(), "clusterB", 502 /* non-TLS srv -> err */)
				})

				It("should be possible to open a second tunnel from clusterB", func() {
					var err error

					tunB, err = tunnel.DialTLS(lisTun.Addr().String(), cfgB)
					Expect(err).NotTo(HaveOccurred())
				})

				It("eventually accepting connections fails as the tunnel is rejected", func() {
					_, err := tunB.Accept()
					Expect(err).Should(HaveOccurred())
				})

				It("should be possible to delete the cluster", func() {
					Expect(deleteCluster(lis.Addr().String(), "clusterB")).To(BeTrue())
				})

				It("eventually accepting connections fails", func() {
					_, err := tunB.Accept()
					Expect(err).Should(HaveOccurred())
				})

				It("should be possible to open tunnel from unregistered clusterB", func() {
					var err error

					tunB, err = tunnel.DialTLS(lisTun.Addr().String(), cfgB)
					Expect(err).NotTo(HaveOccurred())
				})

				It("eventually accepting connections fails as the tunnel is rejected", func() {
					_, err := tunB.Accept()
					Expect(err).Should(HaveOccurred())
				})

				It("should be able to register clusterB again", func() {
					addCluster(lis.Addr().String(), "clusterB", "B again")
				})

				It("should be possible to open tunnel from clusterB with outdated creds", func() {
					var err error

					tunB, err = tunnel.DialTLS(lisTun.Addr().String(), cfgB)
					Expect(err).NotTo(HaveOccurred())
				})

				It("eventually accepting connections fails as the tunnel is rejected", func() {
					_, err := tunB.Accept()
					Expect(err).Should(HaveOccurred())
				})
			})
		})

		It("should stop the servers", func(done Done) {
			err := srv.Close()
			Expect(err).NotTo(HaveOccurred())
			wg.Wait()
			close(done)
		})

		It("should re-start a server with auto registration", func() {
			startServer(server.WithAutoRegister())
		})

		Context("When auto-registration is enabled", func() {
			It("should be possible to open a tunnel with certs for clusterA", func() {
				cert, err := tls.X509KeyPair(certPemA, keyPemA)
				Expect(err).NotTo(HaveOccurred())

				cfg := &tls.Config{
					Certificates: []tls.Certificate{cert},
					RootCAs:      rootCAs,
				}

				clnT, err = tunnel.DialTLS(lisTun.Addr().String(), cfg)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when client tunnel exists", func() {
				testClnT()
			})
		})

	})

	It("should stop the servers", func(done Done) {
		err := srv.Close()
		Expect(err).NotTo(HaveOccurred())
		wg.Wait()
		close(done)
	})
})

var _ = Describe("Server authenticates requests", func() {

	k8sAPI := fake.NewSimpleClientset()
	var wg sync.WaitGroup
	var srv *server.Server
	var tun *tunnel.Tunnel
	var lisHTTPS net.Listener
	var lisHTTP net.Listener
	var lisTun net.Listener
	var xCert tls.Certificate

	By("Creating credentials for server", func() {
		srvCert, _ = test.CreateSelfSignedX509Cert("voltron", true)

		block, _ := pem.Decode([]byte(test.PrivateRSA))
		srvPrivKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)

		rootCAs = x509.NewCertPool()
		rootCAs.AddCert(srvCert)

		key, _ := utils.KeyPEMEncode(srvPrivKey)
		cert := utils.CertPEMEncode(srvCert)

		xCert, _ = tls.X509KeyPair(cert, key)
	})

	It("Should start the server", func() {
		var err error

		lisHTTP, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		lisHTTPS, err = tls.Listen("tcp", "localhost:0", &tls.Config{
			Certificates: []tls.Certificate{xCert},
			NextProtos:   []string{"h2"},
		})
		Expect(err).NotTo(HaveOccurred())

		lisTun, err = net.Listen("tcp", "localhost:0")
		Expect(err).NotTo(HaveOccurred())

		srv, err = server.New(
			k8sAPI,
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
			server.WithAuthentication(),
		)
		Expect(err).NotTo(HaveOccurred())

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = srv.ServeHTTP(lisHTTPS)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = srv.ServeHTTP(lisHTTP)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = srv.ServeTunnelsTLS(lisTun)
		}()
	})

	It("Should add cluster A", func() {
		Eventually(func() bool {
			cluster, _ := json.Marshal(&clusters.Cluster{ID: "clusterA", DisplayName: "ClusterA"})
			req, _ := http.NewRequest("PUT",
				"http://"+lisHTTP.Addr().String()+"/voltron/api/clusters?", bytes.NewBuffer(cluster))
			resp, err := http.DefaultClient.Do(req)
			return err == nil && resp.StatusCode == 200
		}).Should(Equal(true))
	})

	It("Should open a tunnel for cluster A", func() {
		certPem, keyPem, _ := srv.ClusterCreds("clusterA")
		cert, _ := tls.X509KeyPair(certPem, keyPem)

		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      rootCAs,
		}

		Eventually(func() error {
			var err error
			tun, err = tunnel.DialTLS(lisTun.Addr().String(), cfg)
			return err
		}).ShouldNot(HaveOccurred())
	})

	It("should authenticate Jane", func() {
		test.AddJaneIdentity(k8sAPI)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			test.HTTPSBin(tun, xCert, func(r *http.Request) {
				Expect(r.Header.Get("Impersonate-User")).To(Equal(test.Jane))
				Expect(r.Header.Get("Impersonate-Group")).To(Equal(test.Developers))
				Expect(r.Header.Get("Authorization")).NotTo(Equal(test.JaneBearerToken))
			})
		}()

		clnt := configureHTTPSClient()
		req := requestToClusterA(lisHTTPS.Addr().String())
		test.AddJaneToken(req)
		resp, err := clnt.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))
	})
	It("should not authenticate Bob", func() {
		test.AddBobIdentity(k8sAPI)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			test.HTTPSBin(tun, xCert, func(r *http.Request) {
				Expect(r.Header.Get("Impersonate-User")).To(Equal(""))
				Expect(r.Header.Get("Impersonate-Group")).To(Equal(""))
				Expect(r.Header.Get("Authorization")).To(Equal(""))
			})
		}()

		clnt := configureHTTPSClient()
		req := requestToClusterA(lisHTTPS.Addr().String())
		test.AddBobToken(req)
		resp, err := clnt.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(401))
	})
	It("should return 401 on missing tokens", func() {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			test.HTTPSBin(tun, xCert, func(r *http.Request) {
				Expect(r.Header.Get("Impersonate-User")).To(Equal(""))
				Expect(r.Header.Get("Impersonate-Group")).To(Equal(""))
				Expect(r.Header.Get("Authorization")).To(Equal(""))
			})
		}()

		clnt := configureHTTPSClient()
		req := requestToClusterA(lisHTTPS.Addr().String())
		resp, err := clnt.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(401))
	})

	It("should stop the server", func(done Done) {
		err := srv.Close()
		Expect(err).NotTo(HaveOccurred())
		wg.Wait()
		close(done)
	})
})

func requestToClusterA(address string) *http.Request {
	defer GinkgoRecover()
	req, err := http.NewRequest("GET",
		"https://"+address+"/some/path", strings.NewReader("HELLO"))
	req.Header[server.ClusterHeaderField] = []string{"clusterA"}
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

func addCluster(server, id, name string) {
	cluster, err := json.Marshal(&clusters.Cluster{ID: id, DisplayName: name})
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest("PUT",
		"http://"+server+"/voltron/api/clusters?", bytes.NewBuffer(cluster))
	Expect(err).NotTo(HaveOccurred())
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(200))
}

func deleteCluster(server, id string) bool {
	cluster, err := json.Marshal(&clusters.Cluster{ID: id})
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest("DELETE",
		"http://"+server+"/voltron/api/clusters?", bytes.NewBuffer(cluster))
	Expect(err).NotTo(HaveOccurred())
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())

	return resp.StatusCode == 200
}

func listClusters(server string) ([]clusters.Cluster, int) {
	resp, err := http.Get("http://" + server + "/voltron/api/clusters")
	Expect(err).NotTo(HaveOccurred())

	if resp.StatusCode != 200 {
		return nil, resp.StatusCode
	}

	var list []clusters.Cluster

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	Expect(err).NotTo(HaveOccurred())

	return list, 200
}

func clientHelloReq(addr string, target string, expectStatus int) *http.Response {
	defer GinkgoRecover()
	req, err := http.NewRequest("GET", "http://"+addr+"/some/path", strings.NewReader("HELLO"))
	Expect(err).NotTo(HaveOccurred())

	req.Header[server.ClusterHeaderField] = []string{target}
	test.AddJaneToken(req)

	var resp *http.Response

	Eventually(func() bool {
		var err error
		resp, err := http.DefaultClient.Do(req)
		return err == nil && resp.StatusCode == expectStatus
	}, 2*time.Second, 400*time.Millisecond).Should(BeTrue())

	return resp
}

func http2Srv(t *tunnel.Tunnel) {
	// we need some credentials
	key, _ := utils.KeyPEMEncode(srvPrivKey)
	cert := utils.CertPEMEncode(srvCert)

	xcert, _ := tls.X509KeyPair(cert, key)

	mux := http.NewServeMux()
	httpsrv := &http.Server{
		Handler: mux,
	}

	var reqWg sync.WaitGroup
	reqWg.Add(1)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer reqWg.Done()
		f, ok := w.(http.Flusher)
		Expect(ok).To(BeTrue())

		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "tick %d\n", i)
			f.Flush()
			time.Sleep(300 * time.Millisecond)
		}
	})

	lisTLS := tls.NewListener(t, &tls.Config{
		Certificates: []tls.Certificate{xcert},
		NextProtos:   []string{"h2"},
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		httpsrv.Serve(lisTLS)
	}()

	// we only handle one request, we wait until it is done
	reqWg.Wait()

	httpsrv.Close()
	wg.Wait()
}
