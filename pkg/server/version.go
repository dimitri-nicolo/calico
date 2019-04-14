// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/version"
)

// handleVersion implements the version endpoint which returns JSON encapsulated version info.
func (_ *server) handleVersion(response http.ResponseWriter, _ *http.Request) {
	log.WithFields(log.Fields{
		"Version":   version.VERSION,
		"BuildDate": version.BUILD_DATE,
		"GitTagRef": version.GIT_DESCRIPTION,
		"GitCommit": version.GIT_REVISION,
	}).Debug("Handling version request")

	v := VersionData{
		Version:   version.VERSION,
		BuildDate: version.BUILD_DATE,
		GitTagRef: version.GIT_DESCRIPTION,
		GitCommit: version.GIT_REVISION,
	}

	writeJSON(response, v, true)
}
