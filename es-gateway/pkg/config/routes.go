// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package config

import "github.com/projectcalico/calico/es-gateway/pkg/proxy"

var (
	// ElasticsearchRoutes stores a listing of Routes that should be configured for the Elasticsearch Target.
	// These are intentionally not configurable from outside ES Gateway.
	ElasticsearchRoutes = proxy.Routes{
		// -------------------------------------------------------------------------------------------------
		// Routes for creating index log data.
		// -------------------------------------------------------------------------------------------------
		proxy.Route{
			Name:                          "es-create-new-doc-by-id",
			Path:                          "/{target}/_create/{id}",
			HTTPMethods:                   []string{"PUT", "POST"},
			RequireAuth:                   true,
			RejectUnacceptableContentType: true,
		},
		proxy.Route{
			Name:                          "es-create-new-doc-by-id",
			Path:                          "/{target}/_doc/{id}",
			HTTPMethods:                   []string{"PUT", "POST"},
			RequireAuth:                   true,
			RejectUnacceptableContentType: true,
		},
		// Fluentd component uses the Elasticsearch bulk API by default to create log data.
		// https://docs.fluentd.org/output/elasticsearch
		proxy.Route{
			Name:                          "es-create-new-doc-bulk",
			Path:                          "/_bulk",
			HTTPMethods:                   []string{"POST"},
			RequireAuth:                   true,
			RejectUnacceptableContentType: true,
		},
		proxy.Route{
			Name:                          "es-create-new-doc-bulk-by-index",
			Path:                          "/{target}/_bulk",
			HTTPMethods:                   []string{"POST"},
			RequireAuth:                   true,
			RejectUnacceptableContentType: true,
		},
		// -------------------------------------------------------------------------------------------------
	}

	// KibanaRoutes stores a listing of Routes that should be configured for the Kibana Target.
	// These are intentionally not configurable from outside ES Gateway.
	KibanaRoutes = proxy.Routes{
		// -------------------------------------------------------------------------------------------------
		// Routes for creating saved objects (dashboards).
		// https://www.elastic.co/guide/en/kibana/current/saved-objects-api-create.html
		// -------------------------------------------------------------------------------------------------
		proxy.Route{
			Name:        "kb-create-new-saved-obj-bulk-default",
			Path:        "/tigera-kibana/api/saved_objects/_bulk_create",
			HTTPMethods: []string{"POST", "PUT"},
			RequireAuth: true,
		},
		proxy.Route{
			Name:        "kb-create-new-saved-obj-config-default",
			Path:        "/tigera-kibana/api/saved_objects/config/7.6.2",
			HTTPMethods: []string{"POST", "PUT"},
			RequireAuth: true,
		},
		proxy.Route{
			Name:        "kb-create-new-saved-obj-bulk-space",
			Path:        "/tigera-kibana/s/{space_id}/api/saved_objects/_bulk_create",
			HTTPMethods: []string{"POST", "PUT"},
			RequireAuth: true,
		},
		proxy.Route{
			Name:        "kb-create-new-saved-obj-config-space",
			Path:        "/tigera-kibana/s/{space_id}/api/saved_objects/config/7.6.2",
			HTTPMethods: []string{"POST", "PUT"},
			RequireAuth: true,
		},
		// -------------------------------------------------------------------------------------------------
	}
)
