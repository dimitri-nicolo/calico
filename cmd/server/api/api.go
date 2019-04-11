package api

import (
	"context"
	"net/http"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/compliance/pkg/report"
)

var (
	server *http.Server
	wg     sync.WaitGroup
	rep    report.ReportRetriever
)

// Start the compliance api server
func Start(addr string, key string, cert string) error {
	sm := http.NewServeMux()

	sm.HandleFunc("/listreports", HandleListReports)
	sm.HandleFunc("/downloadreport", HandleDownloadReports)

	server = &http.Server{
		Addr:    addr,
		Handler: sm,
	}

	if key != "" && cert != "" {
		log.WithField("Addr", server.Addr).Info("Starting HTTPS server")
		wg.Add(1)
		go func() {
			log.Warning(server.ListenAndServeTLS(cert, key))
			wg.Done()
		}()
	} else {
		log.WithField("Addr", server.Addr).Info("Starting HTTP server")
		wg.Add(1)
		go func() {
			log.Warning(server.ListenAndServe())
			wg.Done()
		}()
	}

	return nil
}

func SetReportRetriever(rr report.ReportRetriever) {
	log.Info("Server API report retriever set")
	rep = rr
}

// Wait for the compliance server to terminate.
func Wait() {
	log.Info("Waiting")
	wg.Wait()
}

// Stop the compliance server.
func Stop() {
	if server != nil {
		log.WithField("Addr", server.Addr).Info("Stopping HTTPS server")
		e := server.Shutdown(context.Background())
		if e != nil {
			log.Fatal("Server graceful shutdown fail")
			os.Exit(1)
		}
		server = nil
		wg.Wait()
	}
}
