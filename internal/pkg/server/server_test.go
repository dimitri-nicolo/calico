package server_test

import (
	"encoding/json"
	"net"
	"net/http"
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
