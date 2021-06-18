package health

import (
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	httpUtils "github.com/tigera/es-gateway/pkg/handlers/internal/common/http"

	"github.com/tigera/es-gateway/pkg/clients/elastic"
	"github.com/tigera/es-gateway/pkg/clients/kibana"
	"github.com/tigera/es-gateway/pkg/clients/kubernetes"
)

func GetHealthHandler(es elastic.Client, kb kibana.Client, k8s kubernetes.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
		switch r.Method {
		case http.MethodGet:
			isHealthy := true
			var wg sync.WaitGroup

			// Make calls to check each dependency in parallel.
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				if esErr := es.GetClusterHealth(); esErr != nil {
					log.Errorf("ES health check failed: [%s]", esErr)
					isHealthy = false
				}
			}(&wg)

			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				if kbErr := kb.GetKibanaStatus(); kbErr != nil {
					log.Errorf("Kibana health check failed: [%s]", kbErr)
					isHealthy = false
				}
			}(&wg)

			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				if k8sErr := k8s.GetK8sReadyz(); k8sErr != nil {
					log.Errorf("Kube API health check failed: [%s]", k8sErr)
					isHealthy = false
				}
			}(&wg)

			// Ensure all separate checks have returned before proceeding.
			wg.Wait()
			if isHealthy {
				httpUtils.ReturnJSON(w, "ok")
			} else {
				http.Error(w, `"unavailable"`, http.StatusServiceUnavailable)
			}
		default:
			http.NotFound(w, r)
		}
	}
}
