package test

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/onsi/ginkgo"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/pkg/tunnel"
)

// HTTPSBin is a bin server that listens on the other end of the tunnel. Its parameters can be used to inspect
// the requests and make assertion on it. HTTPSBin will return 200 OK for every request
func HTTPSBin(t *tunnel.Tunnel, xCert tls.Certificate, inspectRequest func(r *http.Request)) {
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
	}

	var reqWg sync.WaitGroup
	reqWg.Add(1)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer reqWg.Done()
		defer ginkgo.GinkgoRecover()
		log.Infof("Received request %v", r)
		inspectRequest(r)
	})

	lisTLS := tls.NewListener(t, &tls.Config{
		Certificates: []tls.Certificate{xCert},
		NextProtos:   []string{"h2"},
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = srv.Serve(lisTLS)
	}()

	reqWg.Wait()
	wg.Wait()
}
