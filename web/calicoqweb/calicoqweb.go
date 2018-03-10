// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package main

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/calicoq/web/calicoqweb/handlers"
	"github.com/tigera/calicoq/web/pkg/clientmgr"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

func main() {
	// TODO: Make this a better check than just pulling variables
	// Possibly switch this to use the golang TLSConfig if necessary.
	webKey := os.Getenv("CALICOQ_WEB_KEY")
	webCert := os.Getenv("CALICOQ_WEB_CERT")

	c, err := clientmgr.NewClient("")
	if err != nil {
		panic(err)
	}
	h := handlers.NewQuery(client.NewQueryInterface(c))

	http.HandleFunc("/endpoints", h.Endpoints)
	http.HandleFunc("/endpoints/", h.Endpoint)
	http.HandleFunc("/policies", h.Policies)
	http.HandleFunc("/policies/", h.Policy)
	http.HandleFunc("/nodes", h.Nodes)
	http.HandleFunc("/nodes/", h.Node)
	http.HandleFunc("/summary", h.Summary)
	http.HandleFunc("/version", handlers.VersionHandler)

	if webKey != "" && webCert != "" {
		log.Fatal(http.ListenAndServeTLS(":10443", webCert, webKey, nil))
	} else {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
}
