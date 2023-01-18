// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package handler

import (
	"encoding/json"
	"net/http"

	"github.com/projectcalico/calico/linseed/pkg/config"
)

type version struct {
	BuildDate    string `json:"buildDate"`
	GitCommit    string `json:"gitCommit"`
	GitTag       string `json:"gitTag"`
	BuildVersion string `json:"buildVersion"`
}

// VersionCheck returns the version in json format
func VersionCheck() http.HandlerFunc {
	v := version{
		BuildDate:    config.BuildDate,
		GitCommit:    config.GitCommit,
		GitTag:       config.GitTag,
		BuildVersion: config.BuildVersion,
	}

	return func(w http.ResponseWriter, req *http.Request) {
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
}
