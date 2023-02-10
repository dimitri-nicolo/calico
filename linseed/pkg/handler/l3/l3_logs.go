// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package l3

import (
	"net/http"

	"github.com/projectcalico/calico/linseed/pkg/handler"
)

type NetworkLogs struct {
	// TODO: Add storage
}

func (n NetworkLogs) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     "/flows/network/logs",
			Handler: n.Serve(),
		},
	}
}

func (n NetworkLogs) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write([]byte("net-logs"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
