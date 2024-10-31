// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package handler

import (
	"net/http"

	"github.com/projectcalico/calico/linseed/pkg/config"
	"github.com/projectcalico/calico/lma/pkg/httputils"
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
		httputils.Encode(w, v)
	}
}
