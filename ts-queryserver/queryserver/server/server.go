// Copyright (c) 2018-2020, 2022 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/config"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers"
	handler "github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/auth"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/query"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

// Start the query server.
func Start(cfg *apiconfig.CalicoAPIConfig, servercfg *config.Config, authHandler handler.AuthHandler) error {
	c, err := clientv3.New(*cfg)
	if err != nil {
		return err
	}

	sm := http.NewServeMux()
	qh := query.NewQuery(client.NewQueryInterface(c))
	sm.HandleFunc("/endpoints", authHandler.AuthenticationHandler(qh.Endpoints))
	sm.HandleFunc("/endpoints/", authHandler.AuthenticationHandler(qh.Endpoint))
	sm.HandleFunc("/policies", authHandler.AuthenticationHandler(qh.Policies))
	sm.HandleFunc("/policies/", authHandler.AuthenticationHandler(qh.Policy))
	sm.HandleFunc("/nodes", authHandler.AuthenticationHandler(qh.Nodes))
	sm.HandleFunc("/nodes/", authHandler.AuthenticationHandler(qh.Node))
	sm.HandleFunc("/summary", authHandler.AuthenticationHandler(qh.Summary))
	sm.HandleFunc("/metrics", authHandler.AuthenticationHandler(qh.Metrics))
	sm.HandleFunc("/version", handlers.VersionHandler)

	lic := handlers.License{Client: c}
	sm.HandleFunc("/license", authHandler.AuthenticationHandler(lic.LicenseHandler))

	server = &http.Server{
		Addr:      servercfg.ListenAddr,
		Handler:   sm,
		TLSConfig: calicotls.NewTLSConfig(servercfg.FIPSModeEnabled),
	}
	if servercfg.TLSCert != "" && servercfg.TLSKey != "" {
		log.WithField("Addr", server.Addr).Info("Starting HTTPS server")
		wg.Add(1)
		go func() {
			log.Warningf("%v", server.ListenAndServeTLS(servercfg.TLSCert, servercfg.TLSKey))
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
		_ = server.Shutdown(context.Background())
		server = nil
		wg.Wait()
	}
}
