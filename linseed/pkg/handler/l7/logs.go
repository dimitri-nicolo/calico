// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package l7

import (
	"fmt"
	"net/http"

	"github.com/projectcalico/calico/linseed/pkg/handler"
)

type L7Logs struct {
	// TODO: Add storage
}

func (n L7Logs) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     fmt.Sprintf("%s/logs", baseURL),
			Handler: n.Serve(),
		},
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
