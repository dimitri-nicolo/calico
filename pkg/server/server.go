package server

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bmizerany/pat"
	log "github.com/sirupsen/logrus"

	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/report"
)

// New creates a new server.
func New(rr report.ReportRetriever, rcg ReportConfigurationGetter, rhf RbacHelperFactory, addr string, key string, cert string) ServerControl {
	s := &server{
		key:  key,
		cert: cert,
		rr:   rr,
		rcg:  rcg,
		rhf:  rhf,
	}

	// Create a new pattern matching MUX.
	mux := pat.New()
	mux.Get(UrlVersion, http.HandlerFunc(s.handleVersion))
	// TODO(rlb): Should really handle get on a report too.
	// mux.Get(urlGet, http.HandlerFunc(s.handleVersion))
	mux.Get(UrlList, http.HandlerFunc(s.handleListReports))
	mux.Get(UrlDownload, http.HandlerFunc(s.handleDownloadReports))

	// Create a new server using the MUX.
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s
}

// server implements the compliance server, and implements the ServerControl interface.
type server struct {
	running bool
	server  *http.Server
	key     string
	cert    string
	wg      sync.WaitGroup
	rr      report.ReportRetriever
	rcg     ReportConfigurationGetter
	rhf     RbacHelperFactory

	// Track all of the reports and report types. We don't expect these to change too often, so we only need to
	// update the lists every so often. Access to this data should be through getReportTypes.
	reportLock  sync.RWMutex
	lastUpdate  time.Time
	reportTypes map[string]*calicov3.ReportTypeSpec
}

// Start will start the compliance api server and return. Call Wait() to block until server termination.
func (s *server) Start() {

	if s.key != "" && s.cert != "" {
		log.WithField("Addr", s.server.Addr).Info("Starting HTTPS server")
		s.wg.Add(1)
		go func() {
			log.Warning(s.server.ListenAndServeTLS(s.cert, s.key))
			s.wg.Done()
		}()
	} else {
		log.WithField("Addr", s.server.Addr).Info("Starting HTTP server")
		s.wg.Add(1)
		go func() {
			log.Warning(s.server.ListenAndServe())
			s.wg.Done()
		}()
	}
	s.running = true
	return
}

// Wait for the compliance server to terminate.
func (s *server) Wait() {
	log.Info("Waiting")
	s.wg.Wait()
}

// Stop the compliance server.
func (s *server) Stop() {
	if s.running {
		log.WithField("Addr", s.server.Addr).Info("Stopping HTTPS server")
		e := s.server.Shutdown(context.Background())
		if e != nil {
			log.Fatal("ServerControl graceful shutdown fail")
			os.Exit(1)
		}
		s.wg.Wait()
		s.running = false
	}
}
