// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package dns

import (
	"fmt"
	"net/http"
)

type Logs struct {
	// TODO: Add storage
}

func (n Logs) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": n.Serve(),
	}
}

func (n Logs) URL() string {
	return fmt.Sprintf("%s/logs", baseURL)
}

func (n Logs) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write([]byte("net-logs"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
