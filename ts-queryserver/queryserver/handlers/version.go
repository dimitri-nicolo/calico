// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

var VERSION, BUILD_DATE, GIT_DESCRIPTION, GIT_REVISION string

type version struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitTagRef string `json:"gitTagRef"`
	GitCommit string `json:"gitCommit"`
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{
		"Version":   VERSION,
		"BuildDate": BUILD_DATE,
		"GitTagRef": GIT_DESCRIPTION,
		"GitCommit": GIT_REVISION,
	}).Debug("Handling version request")

	v := version{
		Version:   VERSION,
		BuildDate: BUILD_DATE,
		GitTagRef: GIT_DESCRIPTION,
		GitCommit: GIT_REVISION,
	}

	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
	_, _ = w.Write([]byte{'\n'})
}
