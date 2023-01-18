// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package l3

import "net/http"

type NetworkLogs struct {
	//TODO: Add storage
}

func (n NetworkLogs) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": n.Post(),
	}
}

func (n NetworkLogs) URL() string {
	return "/flows/network/logs"
}

func (n NetworkLogs) Post() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write([]byte("net-logs"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
