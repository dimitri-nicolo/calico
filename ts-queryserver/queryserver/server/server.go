// Copyright (c) 2018-2020, 2022 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/config"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers"
	handler "github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/auth"
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
	qh := handlers.NewQuery(client.NewQueryInterface(c))
	sm.HandleFunc("/endpoints", authHandler.AuthenticationHandler(qh.Endpoints))
	sm.HandleFunc("/endpoints/", authHandler.AuthenticationHandler(qh.Endpoint))
	sm.HandleFunc("/policies", authHandler.AuthenticationHandler(qh.Policies))
	sm.HandleFunc("/policies/", authHandler.AuthenticationHandler(qh.Policy))
	sm.HandleFunc("/nodes", authHandler.AuthenticationHandler(qh.Nodes))
	sm.HandleFunc("/nodes/", authHandler.AuthenticationHandler(qh.Node))
	sm.HandleFunc("/summary", authHandler.AuthenticationHandler(qh.Summary))
	sm.HandleFunc("/version", handlers.VersionHandler)

	lic := handlers.License{Client: c}
	sm.HandleFunc("/license", authHandler.AuthenticationHandler(lic.LicenseHandler))

	server = &http.Server{
		Addr:      servercfg.ListenAddr,
		Handler:   sm,
		TLSConfig: NewTLSConfig(servercfg.FIPSModeEnabled),
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

// NewTLSConfig returns a tls.Config with the recommended default settings for Calico Enterprise components.
// Read more recommendations here in Chapter 3:
// https://www.gsa.gov/cdnstatic/SSL_TLS_Implementation_%5BCIO_IT_Security_14-69_Rev_6%5D_04-06-2021docx.pdf
//
// todo: remove after monorepo merge has taken place.
func NewTLSConfig(fipsMode bool) *tls.Config {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	if fipsMode {
		cfg.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		}
		cfg.CurvePreferences = []tls.CurveID{tls.CurveP384, tls.CurveP256}
		cfg.MinVersion = tls.VersionTLS12
		// Our certificate for FIPS validation does not mention validation for v1.3.
		cfg.MaxVersion = tls.VersionTLS12
		cfg.Renegotiation = tls.RenegotiateNever
	}
	return cfg
}
