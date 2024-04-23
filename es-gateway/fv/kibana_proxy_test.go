// Copyright (c) 2024 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"net/http"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestFV_KibanaProxy(t *testing.T) {
	t.Run("Ensure Kibana Proxy sends connections to Elastic", func(t *testing.T) {
		defer setupAndTeardown(t, DefaultKibanaProxyArgs(), nil)()

		responseProxy, elasticBody, err := doRequest("GET", "http://localhost:5555/", nil, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, responseProxy.StatusCode)
		// Response sample from Elastic
		//{
		//  "name" : "asincu-Precision-5540",
		//  "cluster_name" : "docker-cluster",
		//  "cluster_uuid" : "5lIuJ_FXSBakaIOJJBl4Zw",
		//  "version" : {
		//    "number" : "7.17.18",
		//    "build_flavor" : "default",
		//    "build_type" : "docker",
		//    "build_hash" : "8682172c2130b9a411b1bd5ff37c9792367de6b0",
		//    "build_date" : "2024-02-02T12:04:59.691750271Z",
		//    "build_snapshot" : false,
		//    "lucene_version" : "8.11.1",
		//    "minimum_wire_compatibility_version" : "6.8.0",
		//    "minimum_index_compatibility_version" : "6.0.0-beta1"
		//  },
		//  "tagline" : "You Know, for Search"
		//}
		require.Contains(t, string(elasticBody), "You Know, for Search")
	})

	t.Run("Ensure Kibana connects to Elastic via Kibana Proxy", func(t *testing.T) {
		kibanaArgs := &RunKibanaArgs{
			Image: "docker.elastic.co/kibana/kibana:7.17.18",
			// We are setting the proxy endpoint as elastic backend
			ElasticHosts: "http://localhost:5555",
		}
		defer setupAndTeardown(t, DefaultKibanaProxyArgs(), kibanaArgs)()

		kibanaReady := func() bool {
			log.Debugf("Making requests to see if Kibana is up and ready")
			response, _, err := doRequest("GET", "http://localhost:5601/", nil, nil)
			if err != nil {
				log.Warnf("Received error %s", err)
				return false
			}
			if response.StatusCode != http.StatusOK {
				log.Warnf("Received status %s", response.Status)
				return false
			}
			return true
		}
		require.Eventually(t, kibanaReady, 30*time.Second, 100*time.Millisecond)

		responseKibana, _, err := doRequest("GET", "http://localhost:5601/api/features", nil, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, responseKibana.StatusCode)
	})

	// TODO: Alina run dashboards installer
	// TODO: Alina run namespace creation
	// TODO: Alina run a custom tenancy check using discover
}
