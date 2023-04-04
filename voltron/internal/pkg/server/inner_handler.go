package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	jclust "github.com/projectcalico/calico/voltron/internal/pkg/clusters"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"
)

type InnerHandler interface {
	Handler() http.Handler
}

func NewInnerHandler(t string, c *jclust.ManagedCluster, proxy http.Handler) InnerHandler {
	return &handlerHelper{
		ManagedCluster: c,
		proxy:          proxy,
		tenantID:       t,
	}
}

type handlerHelper struct {
	ManagedCluster *jclust.ManagedCluster
	proxy          http.Handler
	tenantID       string
}

func (h *handlerHelper) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the cluster and tenant ID headers here. If they are already set,
		// but don't match the expected value for this cluster, return an error.
		clusterID := r.Header.Get(utils.ClusterHeaderField)
		tenantID := r.Header.Get(utils.TenantHeaderField)
		fields := log.Fields{
			"url":                    r.URL,
			utils.ClusterHeaderField: clusterID,
			utils.TenantHeaderField:  tenantID,
		}
		logCtx := log.WithFields(fields)

		if clusterID != "" {
			if clusterID != h.ManagedCluster.ID {
				// Cluster ID is set, and it doesn't match what we expect.
				logCtx.Warn("Unexpected cluster ID")
				writeHTTPError(w, unexpectedClusterIDError(clusterID))
				return
			}
		}

		// Set the cluster ID header before forwarding to indicate the originating cluster.
		r.Header.Set(utils.ClusterHeaderField, h.ManagedCluster.ID)

		if h.tenantID != "" {
			// Running in multi-tenant mode. We need to set the tenant ID on
			// any requests received over the tunnel.
			if tenantID != "" && tenantID != h.tenantID {
				// Tenant ID is set, and it doesn't match what we expect.
				logCtx.Warn("Unexpected tenant ID")
				writeHTTPError(w, unexpectedTenantIDError(tenantID))
				return

			}

			// Set the tenant ID before forwarding to indicate the originating tenant.
			r.Header.Set(utils.TenantHeaderField, h.tenantID)
		}

		// Headers have been set properly. Now, proxy the connection
		// using Voltron's own key / cert for mTLS with Linseed.
		logCtx.Debug("Handling connection received over the tunnel")
		h.proxy.ServeHTTP(w, r)
	})
}
