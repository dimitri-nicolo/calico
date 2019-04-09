package api

import (
	"context"
	"net/http"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

// Start the compliance api server
func Start(addr string, key string, cert string) error {
	sm := http.NewServeMux()

	sm.HandleFunc("/listreports", HandleListReports)
	sm.HandleFunc("/downloadreports", HandleDownloadReports)

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

// Wait for the compliance server to terminate.
func Wait() {
	wg.Wait()
}

// Stop the compliance server.
func Stop() {
	if server != nil {
		log.WithField("Addr", server.Addr).Info("Stopping HTTPS server")
		e := server.Shutdown(context.Background())
		if e != nil {
			log.Fatal("Server gracefull shutdown fail")
			os.Exit(1)
		}
		server = nil
		wg.Wait()
	}
}
