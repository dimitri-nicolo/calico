// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/clusters"
)

func returnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Error while encoding %#v", data), 500)
	}
}

type clusterHandler struct {
	clusters *clusters.Clusters
}

// Add a cluster
func (c clusterHandler) Add(target string, cluster *clusters.Cluster) {
	c.clusters.Add(target, cluster)
}

// List all clusters in sorted order by DisplayName field
func (c clusterHandler) List() []*clusters.Cluster {
	clusterList := make([]*clusters.Cluster, 0, len(c.clusters.List()))
	for _, c := range c.clusters.List() {
		// Only incldue non-sensitive fields
		clusterList = append(clusterList, &clusters.Cluster{
			ID:          c.ID,
			DisplayName: c.DisplayName,
		})
	}

	sort.Slice(clusterList, func(i, j int) bool {
		return clusterList[i].DisplayName < clusterList[j].DisplayName
	})

	return clusterList
}

// Handler to create a new or update an existing cluster
func (c *clusterHandler) updateCluster(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}
	// no validations... for now
	// WARNING: there's a race condition in the write
	decoder := json.NewDecoder(r.Body)
	cluster := &clusters.Cluster{}
	err := decoder.Decode(cluster)
	if err != nil {
		panic(err)
	}
	log.Debugf("cluster = %+v\n", cluster)
	c.Add(cluster.ID, cluster)
	returnJSON(w, cluster)
}

// Handler to return a listing of clusters
func (c *clusterHandler) listClusters(w http.ResponseWriter, r *http.Request) {
	returnJSON(w, c.List())
}

// Determine which handler to execute based on HTTP method.
func (c *clusterHandler) handle(w http.ResponseWriter, r *http.Request) {
	log.Debugf("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
	switch r.Method {
	// TODO We need a MethodPost handler as well, should use instead of Put for
	// creating a cluster entity (whcih will auto-generate the ID)
	// TODO This Put handler does not behaviour list a standard PUT endpoint
	// (since it doesn't retrieve the entity ID from the URI) ... fix it later
	case http.MethodPut:
		c.updateCluster(w, r)
	case http.MethodGet:
		c.listClusters(w, r)
	default:
		http.NotFound(w, r)
	}
}
