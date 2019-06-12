// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"

	jclust "github.com/tigera/voltron/internal/pkg/clusters"
)

type cluster struct {
	DisplayName string
	// TODO tunnel info will be here
}

type clusters struct {
	sync.RWMutex
	clusters map[string]*cluster
}

func returnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Error while encoding %#v", data), 500)
	}
}

// List all clusters in sorted order by DisplayName field
func (c *clusters) List() []*jclust.Cluster {
	c.RLock()
	defer c.RUnlock()

	clusterList := make([]*jclust.Cluster, 0, len(c.clusters))
	for id, c := range c.clusters {
		// Only incldue non-sensitive fields
		clusterList = append(clusterList, &jclust.Cluster{
			ID:          id,
			DisplayName: c.DisplayName,
		})
	}

	sort.Slice(clusterList, func(i, j int) bool {
		return clusterList[i].DisplayName < clusterList[j].DisplayName
	})

	return clusterList
}

// Handler to create a new or update an existing cluster
func (c *clusters) updateCluster(w http.ResponseWriter, r *http.Request) {
	c.Lock()
	defer c.Unlock()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}

	// no validations... for now
	decoder := json.NewDecoder(r.Body)
	jc := &jclust.Cluster{}
	err := decoder.Decode(jc)
	if err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	action := "Adding"
	if _, ok := c.clusters[jc.ID]; ok {
		//TODO when updating, we must take into account an existing tunnel to the cluster
		action = "Updating"
	}

	log.Infof("%s cluster ID: %q DisplayName: %q", action, jc.ID, jc.DisplayName)

	c.clusters[jc.ID] = &cluster{
		DisplayName: jc.DisplayName,
	}

	// TODO we will return clusters credentials
	returnJSON(w, jc)
}

// Determine which handler to execute based on HTTP method.
func (c *clusters) handle(w http.ResponseWriter, r *http.Request) {
	log.Debugf("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
	switch r.Method {
	// TODO We need a MethodPost handler as well, should use instead of Put for
	// creating a cluster entity (whcih will auto-generate the ID)
	// TODO This Put handler does not behaviour list a standard PUT endpoint
	// (since it doesn't retrieve the entity ID from the URI) ... fix it later
	case http.MethodPut:
		c.updateCluster(w, r)
	case http.MethodGet:
		returnJSON(w, c.List())
	default:
		http.NotFound(w, r)
	}
}
