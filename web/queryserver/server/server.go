// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package server

import (
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

	if webKey != "" && webCert != "" {
		server = &http.Server{
			Addr:    addr,
			Handler: sm,
		}
		log.Debug("Starting HTTPS server")
		wg.Add(1)
		go func() {
			log.Warning(server.ListenAndServeTLS(webCert, webKey))
			wg.Done()
		}()
	} else {
		server = &http.Server{
			Addr:    addr,
			Handler: sm,
		}
		log.Debug("Starting HTTP server")
		wg.Add(1)
		go func() {
			log.Warning(server.ListenAndServe())
			wg.Done()
		}()
	}

	return nil
}

func Wait() {
	wg.Wait()
}

func Stop() {
	if server != nil {
		server.Shutdown(nil)
		server = nil
		wg.Wait()
	}
}
