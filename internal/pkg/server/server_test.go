// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server_test

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/server"
	"github.com/tigera/voltron/pkg/tunnel"
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

	It("should fail to use invalid path", func() {
		_, err := server.New(
			server.WithCredsFiles("dog/gopher.crt", "dog/gopher.key"),
		)
		Expect(err).To(HaveOccurred())
	})

	It("should start a server", func() {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		srv, err = server.New(
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
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
			list := listClusters(lis.Addr().String())
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
			list := listClusters(lis.Addr().String())
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
			list := listClusters(lis.Addr().String())
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
			list := listClusters(lis.Addr().String())
			Expect(len(list)).To(Equal(1))
			Expect(list[0].ID).To(Equal("clusterA"))
		})

		It("should not be able to delete the cluster again", func() {
			Expect(deleteCluster(lis.Addr().String(), "clusterB")).NotTo(BeTrue())
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
		lisTun net.Listener
	)

	It("should start a server", func() {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		lisTun, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		srv, err = server.New(
			server.WithKeepClusterKeys(),
			server.WithTunnelCreds(srvCert, srvPrivKey),
		)
		Expect(err).NotTo(HaveOccurred())

		wg.Add(1)
		go func() {
			defer wg.Done()
			err = srv.ServeHTTP(lis)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.ServeTunnelsTLS(lisTun)
		}()

	})

	Context("when server is up", func() {
		It("Should not proxy anywhere - return 400", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Should not proxy anywhere - no header", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

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

		var clnT *tunnel.Tunnel

		It("should be possible to open a tunnel", func() {
			certPem, keyPem, err := srv.ClusterCreds("clusterA")
			Expect(err).NotTo(HaveOccurred())

			cert, err := tls.X509KeyPair(certPem, keyPem)
			Expect(err).NotTo(HaveOccurred())

			cfg := &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      rootCAs,
			}

			clnT, err = tunnel.DialTLS(lisTun.Addr().String(), cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when client tunnel exists", func() {
			It("should be possible to accept proxied connections", func() {
				var wg sync.WaitGroup
				var acceptErr error

				wg.Add(1)
				go func() {
					defer wg.Done()

					var c net.Conn
					c, acceptErr = clnT.Accept()
					Expect(acceptErr).NotTo(HaveOccurred())

					data := make([]byte, 1)
					c.Read(data)
					c.Close()
				}()

				clientHelloReq(lis.Addr().String(), "clusterA", 502 /* due to the EOF */)

				wg.Wait()
				Expect(acceptErr).NotTo(HaveOccurred())
			})
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
					var wg sync.WaitGroup

					wg.Add(1)
					go func() {
						defer wg.Done()
						clientHelloReq(lis.Addr().String(), "clusterB", 502 /* due to the EOF */)
					}()

					c, err := tunB.Accept()
					Expect(err).ShouldNot(HaveOccurred())
					c.Close()
					wg.Wait()
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
	})

	It("should stop the servers", func(done Done) {
		cerr := srv.Close()
		Expect(cerr).NotTo(HaveOccurred())
		wg.Wait()
		Expect(err).To(HaveOccurred())
		close(done)
	})
})

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

func listClusters(server string) []clusters.Cluster {
	resp, err := http.Get("http://" + server + "/voltron/api/clusters")
	Expect(err).NotTo(HaveOccurred())

	var list []clusters.Cluster

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	Expect(err).NotTo(HaveOccurred())

	return list
}

func clientHelloReq(addr string, target string, expectStatus int) *http.Response {
	defer GinkgoRecover()
	req, err := http.NewRequest("GET", "http://"+addr+"/some/path", strings.NewReader("HELLO"))
	Expect(err).NotTo(HaveOccurred())

	req.Header[server.ClusterHeaderField] = []string{target}

	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(expectStatus))

	return resp
}
