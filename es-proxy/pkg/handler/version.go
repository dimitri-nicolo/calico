// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package handler

import (
	"encoding/json"
	"net/http"
)

var BUILD_DATE, GIT_COMMIT, GIT_TAG, VERSION string

type version struct {
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
	GitTag    string `json:"gitTag"`
	Version   string `json:"version"`
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	v := version{
		BuildDate: BUILD_DATE,
		GitCommit: GIT_COMMIT,
		GitTag:    GIT_TAG,
		Version:   VERSION,
	}

	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write([]byte{'\n'})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
