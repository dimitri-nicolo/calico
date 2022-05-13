// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package version

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BuildVersion stores the SemVer for the given build
var BuildVersion string

// BuildDate stores the date of the build
var BuildDate string

// GitTag stores the tag description
var GitTag string

// GitCommit stores git commit hash for the given build
var GitCommit string

// Version prints version and build information.
func Version() {
	fmt.Println("Version:     ", BuildVersion)
	fmt.Println("Build date:  ", BuildDate)
	fmt.Println("Git tag ref: ", GitTag)
	fmt.Println("Git commit:  ", GitCommit)
}

type version struct {
	BuildDate    string `json:"buildDate"`
	GitCommit    string `json:"gitCommit"`
	GitTag       string `json:"gitTag"`
	BuildVersion string `json:"buildVersion"`
}

// Handler is an HTTP handler that returns the version in json format
func Handler(w http.ResponseWriter, r *http.Request) {
	v := version{
		BuildDate:    BuildDate,
		GitCommit:    GitCommit,
		GitTag:       GitTag,
		BuildVersion: BuildVersion,
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
