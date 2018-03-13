// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var VERSION, BUILD_DATE, GIT_DESCRIPTION, GIT_REVISION string

type version struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitTagRef string `json:"gitTagRef"`
	GitCommit string `json:"gitCommit"`
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Version:     ", VERSION)
	fmt.Println("Build date:  ", BUILD_DATE)
	fmt.Println("Git tag ref: ", GIT_DESCRIPTION)
	fmt.Println("Git commit:  ", GIT_REVISION)
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
	w.Write(js)
	w.Write([]byte{'\n'})
}
