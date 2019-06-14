// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/tigera/voltron/internal/pkg/clusters"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/server"
	"github.com/tigera/voltron/pkg/tunnel"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
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

		srv, err = server.New()
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

		It("should be able to list the cluster", func() {
			list := listClusters(lis.Addr().String())
			Expect(len(list)).To(Equal(1))
			Expect(list[0].DisplayName).To(Equal("clusterA"))
		})

		It("should be able to register another cluster", func() {
			addCluster(lis.Addr().String(), "clusterB", "BB")
		})

		It("should be able to get sorted list of clusters", func() {
			list := listClusters(lis.Addr().String())
			Expect(len(list)).To(Equal(2))
			Expect(list[0].DisplayName).To(Equal("clusterA"))
			Expect(list[1].DisplayName).To(Equal("clusterB"))
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

		srv, err = server.New()
		Expect(err).NotTo(HaveOccurred())

		wg.Add(1)
		go func() {
			defer wg.Done()
			err = srv.ServeHTTP(lis)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.ServeTunnels(lisTun)
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
			clnT, err = tunnel.Dial(lisTun.Addr().String())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be possible to open another tunnel", func() {
			_, err = tunnel.Dial(lisTun.Addr().String())
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

				clientHelloReq(lis.Addr().String(), clnT.Addr().String(), 502 /* due to the EOF */)

				wg.Wait()
				Expect(acceptErr).NotTo(HaveOccurred())
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

func addCluster(server, name, dest string) {
	addr, err := url.Parse(dest)
	Expect(err).NotTo(HaveOccurred())
	cluster, err := clusters.Cluster{ID: name, DisplayName: name, TargetURL: *addr}.MarshalJSON()
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest("PUT", "http://"+server+"/voltron/api/clusters?", bytes.NewBuffer(cluster))
	Expect(err).NotTo(HaveOccurred())
	_, err = http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
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
	req, err := http.NewRequest("GET", "http://"+addr+"/some/path", strings.NewReader("HELLO"))
	Expect(err).NotTo(HaveOccurred())

	req.Header[server.ClusterHeaderField] = []string{target}

	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(expectStatus))

	return resp
}
