// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/projectcalico/calico/linseed/pkg/server"
	v1 "k8s.io/api/authorization/v1"
)

type myHandler struct {
	sleepSeconds int
	t            *testing.T
}

func (h *myHandler) APIS() []handler.API {
	return []handler.API{{
		Method:          "GET",
		URL:             "/sleep",
		Handler:         h,
		AuthzAttributes: &v1.ResourceAttributes{},
	}}
}

func (h *myHandler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	time.Sleep(time.Duration(h.sleepSeconds) * time.Second)
	_, err := rw.Write([]byte("Hello, world!"))
	Expect(err).NotTo(HaveOccurred())
}

// Prior to increasing the Linseed server's WriteTimeout to 60s, this test got a "tls: bad record
// MAC" error like this:
//
//	Unexpected error:
//	    <*url.Error | 0xc00042a660>:
//	    Get "https://localhost:9999/sleep": local error: tls: bad record MAC
//	    {
//	        Op: "Get",
//	        URL: "https://localhost:9999/sleep",
//	        Err: <*tls.permanentError | 0xc0009883e0>{
//	            err: <*tls.permanentError | 0xc0009883d0>{
//	                err: <*net.OpError | 0xc0009a45a0>{Op: "local error", Net: "", Source: nil, Addr: nil, Err: <tls.alert>20},
//	            },
//	        },
//	    }
//	occurred
func TestServerSlowRequest(t *testing.T) {
	RegisterTestingT(t)

	// Get the current working directory, which we expect to be the fv dir.
	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	// Turn it to an absolute path.
	cwd, err = filepath.Abs(cwd)
	Expect(err).NotTo(HaveOccurred())

	caCertFile := fmt.Sprintf("%s/cert/RootCA.crt", cwd)
	certFile := fmt.Sprintf("%s/cert/localhost.crt", cwd)
	keyFile := fmt.Sprintf("%s/cert/localhost.key", cwd)

	server := server.NewServer(
		"0.0.0.0:9999",
		server.WithRoutes(server.UnpackRoutes(&myHandler{sleepSeconds: 15, t: t})...),
		server.WithClientCACerts(caCertFile),
	)
	Expect(server).NotTo(BeNil())
	go func() {
		err := server.ListenAndServeTLS(certFile, keyFile)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("http: Server closed"))
	}()

	// Give time for server to be running.
	time.Sleep(3 * time.Second)

	// Create a client and send a request that will take 15 seconds.
	caCertPool := x509.NewCertPool()
	caCert, err := os.ReadFile(caCertFile)
	Expect(err).NotTo(HaveOccurred())
	caCertPool.AppendCertsFromPEM(caCert)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	Expect(err).NotTo(HaveOccurred())
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{cert},
		},
	}}
	_, err = client.Get("https://localhost:9999/sleep")
	Expect(err).NotTo(HaveOccurred())

	// Shut down the server.
	err = server.Shutdown(context.TODO())
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(3 * time.Second)
}
