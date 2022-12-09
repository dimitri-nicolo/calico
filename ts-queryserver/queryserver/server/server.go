// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/config"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers"
	handler "github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/auth"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/query"
)

type Server struct {
	authHandler handler.AuthHandler
	cfg         *apiconfig.CalicoAPIConfig
	k8sClient   kubernetes.Interface
	servercfg   *config.Config

	server *http.Server
	wg     sync.WaitGroup

	stopCh chan struct{}
}

func NewServer(
	k8sClient kubernetes.Interface,
	cfg *apiconfig.CalicoAPIConfig,
	servercfg *config.Config,
	authHandler handler.AuthHandler,
) *Server {
	return &Server{
		authHandler: authHandler,
		cfg:         cfg,
		k8sClient:   k8sClient,
		servercfg:   servercfg,

		stopCh: make(chan struct{}),
	}
}

// Start the query server.
func (s *Server) Start() error {
	c, err := clientv3.New(*s.cfg)
	if err != nil {
		return err
	}

	sm := http.NewServeMux()
	qh := query.NewQuery(client.NewQueryInterface(s.k8sClient, c, s.stopCh), s.servercfg)
	sm.HandleFunc("/endpoints", s.authHandler.AuthenticationHandler(qh.Endpoints))
	sm.HandleFunc("/endpoints/", s.authHandler.AuthenticationHandler(qh.Endpoint))
	sm.HandleFunc("/policies", s.authHandler.AuthenticationHandler(qh.Policies))
	sm.HandleFunc("/policies/", s.authHandler.AuthenticationHandler(qh.Policy))
	sm.HandleFunc("/nodes", s.authHandler.AuthenticationHandler(qh.Nodes))
	sm.HandleFunc("/nodes/", s.authHandler.AuthenticationHandler(qh.Node))
	sm.HandleFunc("/summary", s.authHandler.AuthenticationHandler(qh.Summary))
	sm.HandleFunc("/metrics", s.authHandler.AuthenticationHandler(qh.Metrics))
	sm.HandleFunc("/version", handlers.VersionHandler)

	lic := handlers.License{Client: c}
	sm.HandleFunc("/license", s.authHandler.AuthenticationHandler(lic.LicenseHandler))

	s.server = &http.Server{
		Addr:      s.servercfg.ListenAddr,
		Handler:   sm,
		TLSConfig: calicotls.NewTLSConfig(s.servercfg.FIPSModeEnabled),
	}
	if s.servercfg.TLSCert != "" && s.servercfg.TLSKey != "" {
		log.WithField("Addr", s.server.Addr).Info("Starting HTTPS server")
		s.wg.Add(1)
		go func() {
			log.Warningf("%v", s.server.ListenAndServeTLS(s.servercfg.TLSCert, s.servercfg.TLSKey))
			<-s.stopCh
			s.wg.Done()
		}()
	} else {
		log.WithField("Addr", s.server.Addr).Info("Starting HTTP server")
		s.wg.Add(1)
		go func() {
			log.Warning(s.server.ListenAndServe())
			<-s.stopCh
			s.wg.Done()
		}()
	}

	return nil
}

// Wait for the query server to terminate.
func (s *Server) Wait() {
	s.wg.Wait()
}

// Stop the query server.
func (s *Server) Stop() {
	if s.server != nil {
		log.WithField("Addr", s.server.Addr).Info("Stopping HTTPS server")
		_ = s.server.Shutdown(context.Background())
		s.server = nil
		close(s.stopCh)
		s.wg.Wait()
	}
}
