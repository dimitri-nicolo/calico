// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package tunnel_test

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/test"
	"github.com/tigera/voltron/pkg/tunnel"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

var _ = Describe("Stream Server", func() {
	var (
		addr net.Addr
		srv  *tunnel.Server

		cconns []net.Conn
		sconns []io.ReadWriteCloser
	)

	It("should start listening", func() {
		srv, addr = startServer()
	})

	It("should accept a few connections", func(done Done) {
		var (
			wg sync.WaitGroup
		)

		N := 3

		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 1; i < N; i++ {
				c, err := net.Dial("tcp", addr.String())
				Expect(err).ShouldNot(HaveOccurred())
				cconns = append(cconns, c)
			}
		}()

		for i := 1; i < N; i++ {
			c, err := srv.Accept()
			Expect(err).ShouldNot(HaveOccurred())
			sconns = append(sconns, c)
		}

		wg.Wait()
		close(done)
	})

	It("srv.Stop() should fail all connections", func() {
		srv.Stop()

		for _, c := range sconns {
			data := make([]byte, 3)
			_, err := c.Read(data)
			Expect(err).Should(HaveOccurred())
		}

		for _, c := range cconns {
			data := make([]byte, 3)
			_, err := c.Read(data)
			Expect(err).Should(HaveOccurred())
		}
	})

})

var _ = Describe("Tunnel server", func() {
	var (
		addr net.Addr
		srv  *tunnel.Server
	)

	It("should start listening", func() {
		srv, addr = startServer()
	})

	var (
		srvT *tunnel.Tunnel
		clnT *tunnel.Tunnel
	)

	It("should setup a tunnel connection", func() {
		srvT, clnT = setupTunnel(srv, addr.String())
	})

	var srvS, clnS io.ReadWriteCloser

	It("should be able to setup a regular tunneled stream c -> s", func() {
		srvS, clnS = setupTunneledStream(srvT, clnT, true)
	})

	Context("when regular stream is open", func() {
		It("should be able to send data s -> c", func(done Done) {
			recvMsg, err := test.DataFlow(clnS, srvS, []byte("HELLO"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(recvMsg)).To(Equal("HELLO"))
			close(done)
		})

		It("should be able to send data s <- c", func(done Done) {
			recvMsg, err := test.DataFlow(srvS, clnS, []byte("WORLD"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(recvMsg)).To(Equal("WORLD"))
			close(done)
		})
	})

	It("should be able to setup a reverse tunneled stream s -> c", func() {
		srvS, clnS = setupTunneledStream(srvT, clnT, true)
	})

	Context("when reverse stream is open", func() {
		It("should be able to send data s -> c", func(done Done) {
			recvMsg, err := test.DataFlow(clnS, srvS, []byte("HELLO"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(recvMsg)).To(Equal("HELLO"))
			close(done)
		})

		It("should be able to send data s <- c", func(done Done) {
			recvMsg, err := test.DataFlow(srvS, clnS, []byte("WORLD"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(recvMsg)).To(Equal("WORLD"))
			close(done)
		})

		var srvS2, clnS2 io.ReadWriteCloser

		It("should be able to setup another reverse tunneled stream s -> c", func() {
			srvS2, clnS2 = setupTunneledStream(srvT, clnT, true)
		})

		It("should be able to send and recv on both streams simultaneously", func(done Done) {
			var wg sync.WaitGroup

			rwRun := func(r io.Reader, w io.Writer, msg string) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					test.DataFlow(r, w, []byte(msg))
				}()
			}

			rwRun(srvS, clnS, "clnS says hi to srvS")
			rwRun(clnS, srvS, "srvS says hi back to clnS")
			rwRun(srvS2, clnS2, "clnS2 says hi to srvS2")
			rwRun(clnS2, srvS2, "srvS2 says hi back to clnS2")

			wg.Wait()
			close(done)
		})

		It("should be possible to close stream", func() {
			err := srvS2.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("after the stream is closed again", func() {
			It("should not be possible to read from the other side (all data are aleady consumed)",
				func() {
					data := make([]byte, 1)
					_, err := clnS2.Read(data)
					Expect(err).To(HaveOccurred())
				})

			It("should be possible to close it again", func() {
				err := srvS2.Close()
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})

	Context("when server stops", func() {
		It("should fail client accept", func(done Done) {
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()

				err := srvT.Close()
				Expect(err).ShouldNot(HaveOccurred())
			}()

			_, err := clnT.Accept()
			Expect(err).Should(HaveOccurred())
			close(done)
		})

		It("should fail tunneled streams", func() {
			data := make([]byte, 1)

			_, err := srvS.Read(data)
			Expect(err).Should(HaveOccurred())

			_, err = clnS.Read(data)
			Expect(err).Should(HaveOccurred())
		})
	})

})

func startServer() (*tunnel.Server, net.Addr) {
	lis, err := net.Listen("tcp", "localhost:0")
	Expect(err).ShouldNot(HaveOccurred())

	srv, err := tunnel.NewServer()
	Expect(err).NotTo(HaveOccurred())

	go func() {
		srv.Serve(lis)
	}()

	return srv, lis.Addr()
}

func setupTunnel(srv *tunnel.Server, dialTarget string) (*tunnel.Tunnel, *tunnel.Tunnel) {

	var (
		srvT *tunnel.Tunnel
		clnT *tunnel.Tunnel
		err  error
		wg   sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		var err error

		clnT, err = tunnel.Dial(dialTarget)
		Expect(err).ShouldNot(HaveOccurred())
	}()

	srvT, err = srv.AcceptTunnel()
	Expect(err).ShouldNot(HaveOccurred())

	wg.Wait()

	return srvT, clnT
}

func setupTunneledStream(srvT, clnT *tunnel.Tunnel,
	reverse bool) (io.ReadWriteCloser, io.ReadWriteCloser) {

	var (
		s, c io.ReadWriteCloser
		err  error
	)

	// N.B. we can only do this in a single thread because Accept backlog is 1
	// by default
	if reverse {
		s, err = srvT.OpenStream()
		Expect(err).ShouldNot(HaveOccurred())
		c, err = clnT.AcceptStream()
		Expect(err).ShouldNot(HaveOccurred())
	} else {
		c, err = clnT.OpenStream()
		Expect(err).ShouldNot(HaveOccurred())
		s, err = srvT.AcceptStream()
		Expect(err).ShouldNot(HaveOccurred())
	}

	return s, c
}

var _ = Describe("TLS Stream", func() {
	var (
		err     error
		srvCert *x509.Certificate
		clnCert *x509.Certificate
	)

	It("should  create a cert for server", func() {
		srvCert, err = test.CreateSelfSignedX509Cert("voltron", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(srvCert.EmailAddresses)).To(Equal(1))
		Expect(srvCert.EmailAddresses[0]).To(Equal("voltron"))
	})

	It("should  create a cert for client", func() {
		clnCert, err = test.CreateSignedX509Cert("guardian", srvCert)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(clnCert.EmailAddresses)).To(Equal(1))
		Expect(clnCert.EmailAddresses[0]).To(Equal("guardian"))
	})

	var (
		lis net.Listener
		srv *tunnel.Server
	)

	It("should start TLS server", func() {
		lis, err = net.Listen("tcp", "localhost:0")
		Expect(err).ShouldNot(HaveOccurred())

		var err error
		srv, err = tunnel.NewServer(
			tunnel.WithCredsPEM(test.PemEncodeCert(srvCert), []byte(test.PrivateRSA)),
			tunnel.WithTLSHandshakeTimeout(200*time.Millisecond),
		)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			srv.ServeTLS(lis)
		}()
	})

	var srvS, clnS io.ReadWriteCloser

	It("should be possible to open a mTLS connection with a correct cert", func() {
		var (
			err error
			wg  sync.WaitGroup
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			srvS, err = srv.Accept()
			Expect(err).ShouldNot(HaveOccurred())
		}()

		certPem := test.PemEncodeCert(clnCert)
		cert, err := tls.X509KeyPair(certPem, []byte(test.PrivateRSA))

		rootCAs := x509.NewCertPool()
		rootCAs.AddCert(srvCert)
		clnS, err = tls.Dial("tcp", lis.Addr().String(), &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      rootCAs,
		})
		Expect(err).ShouldNot(HaveOccurred())
		wg.Wait()
	})

	Context("when tls stream is open", func() {
		It("should be able to send data s -> c", func(done Done) {
			recvMsg, err := test.DataFlow(clnS, srvS, []byte("HELLO"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(recvMsg)).To(Equal("HELLO"))
			close(done)
		})

		It("should be able to send data s <- c", func(done Done) {
			recvMsg, err := test.DataFlow(srvS, clnS, []byte("WORLD"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(recvMsg)).To(Equal("WORLD"))
			close(done)
		})
	})

	var clnC net.Conn

	It("should be ok to initiate non-TLS connection", func() {
		var err error
		clnC, err = net.Dial("tcp", lis.Addr().String())
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should fail to accept non-TLS connection", func() {
		_, err := srv.Accept()
		Expect(err).Should(HaveOccurred())
	})

	It("eventually server closes and it should not be possible to use the client side", func() {
		Eventually(func() error {
			_, err := clnC.Write([]byte("blah"))
			return err
		}).ShouldNot(Succeed())
	})
})
