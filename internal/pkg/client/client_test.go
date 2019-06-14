package client_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/client"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

var _ = Describe("Client", func() {
	var (
		err error
		wg  sync.WaitGroup
		cl  *client.Client
		lis net.Listener
		ts  *httptest.Server
	)

	It("should fail to use invalid cert file paths", func() {
		_, err := client.New(
			client.WithCredsFiles("dog/gopher.crt", "dog/gopher.key"),
		)
		Expect(err).To(HaveOccurred())
	})

	It("Starts up mock server", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Proxied to test!")
		}))
	})

	It("Starts up the client", func() {
		var e error
		lis, e = net.Listen("tcp", "localhost:0")
		Expect(e).NotTo(HaveOccurred())

		cl, err = client.New(
			client.WithProxyTargets(
				[]client.ProxyTarget{{Pattern: "^/test", Dest: ts.URL}},
			),
		)
		Expect(err).NotTo(HaveOccurred())
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = cl.ServeHTTP(lis)
		}()
	})

	Context("When client is up", func() {
		It("should send a request to nonexistant target", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/gopher")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should send a request to the test target", func() {
			resp, err := http.Get("http://" + lis.Addr().String() + "/test")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			message, err := ioutil.ReadAll(resp.Body)

			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			Expect(string(message)).To(Equal("Proxied to test!"))
		})
	})

	It("should stop the client", func(done Done) {
		cerr := cl.Close()
		Expect(cerr).NotTo(HaveOccurred())
		wg.Wait()
		Expect(err).To(HaveOccurred())
		if ts != nil {
			ts.Close()
		}
		close(done)
	})

})
