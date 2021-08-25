package middlewares

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-gateway/pkg/metrics"

)

var legacyURLPath, extractIndexPrefixPattern, bulkInsert *regexp.Regexp

func init() {
	// This regexp matches legacy queries, for example: "/tigera-elasticsearch/tigera_secure_ee_flows.cluster.*/_search"
	legacyURLPath = regexp.MustCompile(`^.*/tigera_secure_ee_.*?\.(.*)\..*?$`)

	// This regexp extracts the index prefix from a legacy query URL path (up to first '.').
	extractIndexPrefixPattern = regexp.MustCompile(`/(tigera_secure_ee_[_a-z0-9*]*)(?:\..*)?/_search`)

	bulkInsert = regexp.MustCompile(`^(/_bulk).*$`)
}

func MetricsCollectionHandler(collector metrics.Collector) mux.MiddlewareFunc  {
	return func(handler http.Handler)http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
			log.Warnf("method: %v, url: %v, response: %v, contentlength: %v", r.Method, r.URL.String(), r.Response, float64(r.ContentLength)) //todo: remove
			if r.Context().Value(ESUserKey) != nil {
				log.Warn("User found: %v", r.Context().Value(ESUserKey))
				if r.ContentLength > 0 && r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
					tenant, cluster, err := extractTenantClusterFromRequest(r)
					if err != nil {
						log.Warnf("error occurred while extracting tenant and clusterID: %v", err)
					}
					if len(cluster) != 0 {
						collector.CollectLogBytes(tenant, cluster, float64(r.ContentLength))
					}
				}
			}
		})
	}
}

// extractTenantClusterFromRequest extracts the tenant and cluster from a URI. Example URIs:
// - /_all/_stats?level=shards                        -> returns "", "", false
// - /_cluster/settings                               -> returns "", "", false
// - /.tigera.domainnameset.cluster/_search?scroll=5m -> returns "", "", false
// - /tigera-kibana/api/index_management/indices      -> returns "", "", false
// - /tigera-secure
// - /_bulk
func extractTenantClusterFromRequest(r *http.Request) (tenant string, clusterID string, err error) {
	if r == nil {
		err = fmt.Errorf("request is nil, this is unexpected")
		return
	}
	if bulkInsert.MatchString(r.RequestURI) && r.Body != nil{
		buf, bodyErr := ioutil.ReadAll(r.Body)
		if bodyErr != nil {
			log.Print("bodyErr ", bodyErr.Error())
			err = bodyErr
			return
		}

		rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
		rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
		log.Printf("BODY: %q", rdr1)
		r.Body = rdr2
	}
    tenant = "tenant"
	clusterID = "cluster"

	return
}
