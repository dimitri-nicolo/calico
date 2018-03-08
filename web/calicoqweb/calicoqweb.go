// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package main

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/calicoq/web/calicoqweb/handlers"
	"github.com/tigera/calicoq/web/pkg/clientmgr"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

func main() {
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
	log.Fatal(http.ListenAndServe(":8080", nil))
}
