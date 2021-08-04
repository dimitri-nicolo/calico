package middlewares

import (
	"net/http"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-gateway/pkg/metrics"

)

func MetricsCollectionHandler(collector metrics.Collector) mux.MiddlewareFunc  {
	return func(handler http.Handler)http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
			log.Warnf("method: %v, url: %v, response: %v, contentlength: %v", r.Method, r.URL.String(), r.Response, float64(r.ContentLength)) //todo: remove
			if r.Context().Value(ESUserKey) != nil {
				log.Debug("User found: %v", r.Context().Value(ESUserKey))
				if r.ContentLength > 0 && r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
					collector.CollectLogBytes("my_tenant", "my_cluster", float64(r.ContentLength))
				}
			}
		})
	}
}
