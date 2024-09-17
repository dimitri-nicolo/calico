// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package main

import (
	"log"
	"net/http"

	"github.com/projectcalico/calico/l7-admission-controller/cmd/l7-admission-controller/config"
	"github.com/projectcalico/calico/l7-admission-controller/sidecar"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	http.Handle("/sidecar-webhook", sidecar.NewSidecarHandler(cfg))
	http.HandleFunc("/live", liveHandler)

	if err := http.ListenAndServeTLS(":6443", cfg.TLSCert, cfg.TLSKey, nil); err != nil {
		log.Fatal("Server stopped unexpected: ", err)
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
