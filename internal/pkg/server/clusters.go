// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	jclust "github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/pkg/tunnel"
)

type cluster struct {
	jclust.Cluster

	sync.RWMutex

	tunnel *tunnel.Tunnel
	proxy  *httputil.ReverseProxy
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

func (cs *clusters) add(id string, c *cluster) {
	cs.clusters[id] = c
}

// List all clusters in sorted order by DisplayName field
func (cs *clusters) List() []jclust.Cluster {
	cs.RLock()
	defer cs.RUnlock()

	clusterList := make([]jclust.Cluster, 0, len(cs.clusters))
	for _, c := range cs.clusters {
		// Only incldue non-sensitive fields

		c.RLock()
		clusterList = append(clusterList, c.Cluster)
		c.RUnlock()
	}

	sort.Slice(clusterList, func(i, j int) bool {
		return clusterList[i].DisplayName < clusterList[j].DisplayName
	})

	log.Debugf("clusterList = %+v\n", clusterList)
	return clusterList
}

// Handler to create a new or update an existing cluster
func (cs *clusters) updateCluster(w http.ResponseWriter, r *http.Request) {
	cs.Lock()
	defer cs.Unlock()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}

	// no validations... for now
	decoder := json.NewDecoder(r.Body)

	jc := new(jclust.Cluster)
	err := decoder.Decode(jc)
	if err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	if c, ok := cs.clusters[jc.ID]; ok {
		c.Lock()
		log.Infof("Updating cluster ID: %q DisplayName: %q", c.ID, c.DisplayName)
		c.Cluster = *jc
		log.Infof("New cluster ID: %q DisplayName: %q", c.ID, c.DisplayName)
		c.Unlock()
	} else {
		log.Infof("Adding cluster ID: %q DisplayName: %q", jc.ID, jc.DisplayName)
		cs.add(jc.ID,
			&cluster{
				Cluster: *jc,
			})
	}

	// TODO we will return clusters credentials
	returnJSON(w, jc)
}

func (cs *clusters) deleteCluster(w http.ResponseWriter, r *http.Request) {
	cs.Lock()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}

	decoder := json.NewDecoder(r.Body)

	jc := new(jclust.Cluster)
	if err := decoder.Decode(jc); err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	if c, ok := cs.clusters[jc.ID]; ok {
		// remove from the map so nobody can get it, but whoever uses it can
		// keep doing so
		delete(cs.clusters, jc.ID)
		cs.Unlock()
		// close the tunnel to disconnect. Closing is thread save, but we need
		// to hold the RLock to access the tunnel
		c.RLock()
		if c.tunnel != nil {
			c.tunnel.Close()
		}
		c.RUnlock()
		fmt.Fprintf(w, "Deleted")
	} else {
		cs.Unlock()
		msg := fmt.Sprintf("Cluster id %q does not exist", jc.ID)
		log.Debugf(msg)
		http.Error(w, msg, 404)
	}
}

// Determine which handler to execute based on HTTP method.
func (cs *clusters) apiHandle(w http.ResponseWriter, r *http.Request) {
	log.Debugf("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)

	// TODO auth the request

	switch r.Method {
	// TODO We need a MethodPost handler as well, should use instead of Put for
	// creating a cluster entity (whcih will auto-generate the ID)
	// TODO This Put handler does not behaviour list a standard PUT endpoint
	// (since it doesn't retrieve the entity ID from the URI) ... fix it later
	case http.MethodPut:
		cs.updateCluster(w, r)
	case http.MethodGet:
		returnJSON(w, cs.List())
	case http.MethodDelete:
		cs.deleteCluster(w, r)
	default:
		http.NotFound(w, r)
	}
}

// get returns the cluster read-locked so that nobody can modify it while its
// being used.
func (cs *clusters) get(id string) *cluster {
	cs.RLock()
	defer cs.RUnlock()

	// lock it while the cs.Lock is held to avoid a race with whoever would like
	// to remove the cluster from the list
	c := cs.clusters[id]
	if c != nil {
		c.RLock()
	}

	return c
}

func (c *cluster) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c.tunnel.Open()
}

func (c *cluster) Dial(network, addr string) (net.Conn, error) {
	return c.tunnel.Open()
}

func (c *cluster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.proxy == nil {
		http.Error(w, fmt.Sprintf("Cluster %s unreachable, no connection", c.DisplayName), 503)
		return
	}
	c.proxy.ServeHTTP(w, r)
}

func (c *cluster) assignTunnel(t *tunnel.Tunnel) {
	c.tunnel = t
	c.proxy = &httputil.ReverseProxy{
		Director:      proxyVoidDirector,
		FlushInterval: 100 * time.Millisecond,
		// TODO set the error logger
		Transport: &http.Transport{
			DialContext: c.DialContext,
			DialTLS:     c.Dial, // to avoid TLS in a TLSed tunnel
		},
	}
}

func proxyVoidDirector(*http.Request) {
	// do nothing with the request, we pass it forward as is, the other side of
	// the tunnel should do whatever it needs to proxy it further
}
