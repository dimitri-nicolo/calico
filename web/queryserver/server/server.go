// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
	"github.com/tigera/calicoq/web/queryserver/handlers"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

// Start the query server.
func Start(addr string, cfg *apiconfig.CalicoAPIConfig, webKey, webCert string) error {
	c, err := clientv3.New(*cfg)
	if err != nil {
		return err
	}
	sm := http.NewServeMux()
	qh := handlers.NewQuery(client.NewQueryInterface(c))
	sm.HandleFunc("/endpoints", qh.Endpoints)
	sm.HandleFunc("/endpoints/", qh.Endpoint)
	sm.HandleFunc("/policies", qh.Policies)
	sm.HandleFunc("/policies/", qh.Policy)
	sm.HandleFunc("/nodes", qh.Nodes)
	sm.HandleFunc("/nodes/", qh.Node)
	sm.HandleFunc("/summary", qh.Summary)
	sm.HandleFunc("/version", handlers.VersionHandler)

	lic := handlers.License{c}
	sm.HandleFunc("/license", lic.LicenseHandler)

	server = &http.Server{
		Addr:    addr,
		Handler: sm,
	}
	if webKey != "" && webCert != "" {
		log.WithField("Addr", server.Addr).Info("Starting HTTPS server")
		wg.Add(1)
		go func() {
			log.Warning(server.ListenAndServeTLS(webCert, webKey))
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

// Wait for the query server to terminate.
func Wait() {
	wg.Wait()
}

// Stop the query server.
func Stop() {
	if server != nil {
		log.WithField("Addr", server.Addr).Info("Stopping HTTPS server")
		server.Shutdown(context.Background())
		server = nil
		wg.Wait()
	}
}
