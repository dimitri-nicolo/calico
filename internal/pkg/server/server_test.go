package server_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/server"
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

	It("should fail to use a bad target", func() {
		_, err := server.New(
			server.WithProxyTargets(
				[]server.ProxyTarget{{Pattern: "some bad url", Dest: "(*&&%&^$"}},
			),
		)
		Expect(err).To(HaveOccurred())
	})

	It("should start a server", func() {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		srv, e = server.New()
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
			Expect(list[0]).To(Equal("clusterA"))
		})

		It("should be able to register another cluster", func() {
			addCluster(lis.Addr().String(), "clusterB", "BB")
		})

		It("should be able to get sorted list of clusters", func() {
			list := listClusters(lis.Addr().String())
			Expect(len(list)).To(Equal(2))
			Expect(list[0]).To(Equal("clusterA"))
			Expect(list[1]).To(Equal("clusterB"))
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

var _ = Describe("Server Header Test", func() {
	var (
		err error
		wg  sync.WaitGroup
		srv *server.Server
		lis net.Listener
		ts  *httptest.Server
	)

	It("should start a server", func() {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		srv, e = server.New()
		Expect(err).NotTo(HaveOccurred())
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = srv.ServeHTTP(lis)
		}()
	})

	Context("when server is up", func() {
		It("Should not proxy anywhere - return 400", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should be able to register a new cluster & start test server", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello clusterA")
			}))

			addCluster(lis.Addr().String(), "clusterA", ts.URL)
		})

		It("Should not proxy anywhere - no header", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Should proxy to clusterA", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Target", "clusterA")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			message, err := ioutil.ReadAll(resp.Body)

			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			Expect(string(message)).To(Equal("Hello clusterA"))
		})

		It("Should not proxy anywhere - invalid cluster", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Target", "zzzzzzz")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Should not proxy anywhere - multiple headers", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Target", "clusterA")
			req.Header.Add("X-Target", "helloworld")
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	})

	It("should stop the servers", func(done Done) {
		cerr := srv.Close()
		Expect(cerr).NotTo(HaveOccurred())
		wg.Wait()
		Expect(err).To(HaveOccurred())
		if ts != nil {
			ts.Close()
		}
		close(done)
	})
})

func addCluster(server, name, dest string) {
	v := url.Values{}
	v.Add("name", name)
	v.Add("target", dest)

	req, err := http.NewRequest("PUT", "http://"+server+"/targets?"+v.Encode(), nil)
	Expect(err).NotTo(HaveOccurred())
	_, err = http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
}

func listClusters(server string) []string {
	resp, err := http.Get("http://" + server + "/targets")
	Expect(err).NotTo(HaveOccurred())

	var list []string

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	Expect(err).NotTo(HaveOccurred())

	return list
}
