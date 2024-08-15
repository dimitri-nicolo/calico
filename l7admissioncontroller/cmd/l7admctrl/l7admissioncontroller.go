// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package main

import (
	"log"
	"net/http"

	"github.com/projectcalico/calico/l7admissioncontroller/cmd/l7admctrl/config"
	"github.com/projectcalico/calico/l7admissioncontroller/sidecar"
)

func main() {
	http.Handle("/sidecar-webhook", sidecar.NewSidecarHandler())
	http.HandleFunc("/live", liveHandler)

	err := http.ListenAndServeTLS(":6443", config.TLSCert, config.TLSKey,
		nil)
	if err != nil {
		log.Fatal("Server stopped unexpected: ", err)
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
