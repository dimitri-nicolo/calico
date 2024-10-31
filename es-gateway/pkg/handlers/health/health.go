// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package health

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/es-gateway/pkg/clients/elastic"
	"github.com/projectcalico/calico/es-gateway/pkg/clients/kibana"
	"github.com/projectcalico/calico/es-gateway/pkg/clients/kubernetes"
	httpUtils "github.com/projectcalico/calico/es-gateway/pkg/handlers/internal/common/http"
)

// GetHealthHandler returns an HTTP handler to check whether Kube API is ready. This is the only
// dependency that ES gateway needs in order to perform it's responsibility (which is to proxy
// traffic to its configured destinations).
func GetHealthHandler(k8s kubernetes.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
		switch r.Method {
		case http.MethodGet:
			if k8sErr := k8s.GetK8sReadyz(); k8sErr != nil {
				log.Errorf("Kube API health check failed: [%s]", k8sErr)
				http.Error(w, `"unavailable"`, http.StatusServiceUnavailable)
				return
			}
			httpUtils.ReturnJSON(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}
}

// GetESHealthHandler returns an HTTP handler to check the health status of Elasticsearch API.
func GetESHealthHandler(es elastic.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
		switch r.Method {
		case http.MethodGet:
			if esErr := es.GetClusterHealth(); esErr != nil {
				log.Errorf("ES health check failed: [%s]", esErr)
				http.Error(w, `"unavailable"`, http.StatusServiceUnavailable)
				return
			}
			httpUtils.ReturnJSON(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}
}

// GetKBHealthHandler returns an HTTP handler to check the health status of Kibana API.
func GetKBHealthHandler(kb kibana.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
		switch r.Method {
		case http.MethodGet:
			if kbErr := kb.GetKibanaStatus(); kbErr != nil {
				log.Errorf("Kibana health check failed: [%s]", kbErr)
				http.Error(w, `"unavailable"`, http.StatusServiceUnavailable)
				return
			}
			httpUtils.ReturnJSON(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}
}
