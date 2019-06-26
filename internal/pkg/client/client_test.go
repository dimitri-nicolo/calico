package client_test

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/client"
	"github.com/tigera/voltron/pkg/tunnel"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

var _ = Describe("Client Tunneling", func() {
	var (
		lis       net.Listener
		err       error
		wg        sync.WaitGroup
		cl        *client.Client
		srv       *tunnel.Server
		srvTunnel *tunnel.Tunnel
		ts        *httptest.Server
		srvRW     io.ReadWriteCloser
	)

	It("Should start up a tunnel server, serve connections", func() {
		lis, err = net.Listen("tcp", "localhost:0")
		Expect(err).ShouldNot(HaveOccurred())

		srv, err = tunnel.NewServer()
		Expect(err).ToNot(HaveOccurred())

		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.Serve(lis)
		}()
	})

	Context("While Tunnel Server is serving", func() {
		It("Starts up mock server", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Proxied and Received!")
			}))
		})

		It("Starts up the client", func() {
			cl, err = client.New(
				lis.Addr().String(),
				client.WithProxyTargets(
					[]client.ProxyTarget{{Pattern: "^/test", Dest: ts.URL}},
				),
			)
			Expect(err).NotTo(HaveOccurred())
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = cl.ServeTunnelHTTP()
			}()
		})

		It("Server should start accepting tunnels", func() {
			srvTunnel, err = srv.AcceptTunnel()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should open the stream for the tunnel", func() {
			srvRW, err = srvTunnel.OpenStream()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Sends a request through the tunnel, srv -> cln", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/test", nil)
			Expect(err).ToNot(HaveOccurred())
			err = req.Write(srvRW)
			Expect(err).ToNot(HaveOccurred())

			reader := bufio.NewReader(srvRW)

			resp, err := http.ReadResponse(reader, req)
			message, err := ioutil.ReadAll(resp.Body)

			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			Expect(string(message)).To(Equal("Proxied and Received!"))
		})

		It("Sends a request through the tunnel with invalid path, srv -> cln", func() {
			req, err := http.NewRequest("GET", "http://"+lis.Addr().String()+"/gopher", nil)
			Expect(err).ToNot(HaveOccurred())
			err = req.Write(srvRW)
			Expect(err).ToNot(HaveOccurred())

			reader := bufio.NewReader(srvRW)

			resp, err := http.ReadResponse(reader, req)
			message, err := ioutil.ReadAll(resp.Body)

			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(400))
			Expect(string(message)).ToNot(Equal("Proxied and Received!"))
		})

	})

	It("should clean up", func() {
		err = cl.Close()
		Expect(err).ToNot(HaveOccurred())
		srv.Stop()
		wg.Wait()
	})

})
