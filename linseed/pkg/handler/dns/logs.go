// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package dns

import (
	"fmt"
	"net/http"
)

type L7Logs struct {
	// TODO: Add storage
}

func (n L7Logs) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": n.Serve(),
	}
}

func (n L7Logs) URL() string {
	return fmt.Sprintf("%s/logs", baseURL)
}

func (n L7Logs) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write([]byte("net-logs"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
