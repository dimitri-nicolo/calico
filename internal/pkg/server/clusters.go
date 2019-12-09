// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/tigera/voltron/pkg/tunnelmgr"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	apiv3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	jclust "github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/pkg/tunnel"
)

// AppYaml is the content-type that will be returned when returning a yaml file
const AppYaml = "application/vnd.yaml"

type cluster struct {
	jclust.ManagedCluster

	sync.RWMutex

	tunnelManager tunnelmgr.Manager
	proxy         *httputil.ReverseProxy

	cert *x509.Certificate
	key  crypto.Signer

	// parameters for forwarding guardian requests to a default server
	forwardingEnabled               bool
	defaultForwardServerName        string
	defaultForwardDialRetryAttempts int
	defaultForwardDialRetryInterval time.Duration
}

type clusters struct {
	sync.RWMutex
	clusters      map[string]*cluster
	generateCreds func(*jclust.ManagedCluster) (*x509.Certificate, crypto.Signer, error)

	// keep the generated keys, only for testing and debugging
	keepKeys       bool
	renderManifest manifestRenderer

	watchAdded bool

	// parameters for forwarding guardian requests to a default server
	forwardingEnabled               bool
	defaultForwardServerName        string
	defaultForwardDialRetryAttempts int
	defaultForwardDialRetryInterval time.Duration
}

func returnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error("Error while encoding data for response")
		// TODO: We need named errors, with predefined
		// error codes and user-friendly error messages here
		http.Error(w, "\"An error occurred\"", 500)
	}
}

func returnManifests(w http.ResponseWriter, manifest io.Reader) error {
	w.Header().Set("Content-Type", AppYaml)
	_, err := io.Copy(w, manifest)
	return err
}

func (cs *clusters) add(id string, c *cluster) error {
	if cs.clusters[id] != nil {
		return errors.Errorf("cluster id %q already exists", id)
	}
	cs.clusters[id] = c
	return nil
}

// List all clusters in sorted order by ID field (which is the resource name)
func (cs *clusters) List() []jclust.ManagedCluster {
	cs.RLock()
	defer cs.RUnlock()

	clusterList := make([]jclust.ManagedCluster, 0, len(cs.clusters))
	for _, c := range cs.clusters {
		// Only include non-sensitive fields

		c.RLock()
		clusterList = append(clusterList, c.ManagedCluster)
		c.RUnlock()
	}

	sort.Slice(clusterList, func(i, j int) bool {
		return clusterList[i].ID < clusterList[j].ID
	})

	log.Debugf("Listing current %d clusters.", len(clusterList))
	for _, cluster := range clusterList {
		log.Debugf("ID = %s", cluster.ID)
	}
	return clusterList
}

func (cs *clusters) addNew(mc *jclust.ManagedCluster) (*bytes.Buffer, error) {
	log.Infof("Adding cluster ID: %q", mc.ID)

	resp := new(bytes.Buffer)

	cert, key, err := cs.generateCreds(mc)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to generate cluster credentials")
	}

	c := &cluster{
		ManagedCluster:                  *mc,
		cert:                            cert,
		forwardingEnabled:               cs.forwardingEnabled,
		defaultForwardServerName:        cs.defaultForwardServerName,
		defaultForwardDialRetryAttempts: cs.defaultForwardDialRetryAttempts,
		defaultForwardDialRetryInterval: cs.defaultForwardDialRetryInterval,
		tunnelManager:                   tunnelmgr.NewManager(),
	}
	if cs.keepKeys {
		c.key = key
	}

	cs.add(mc.ID, c)
	err = cs.renderManifest(resp, cert, key)
	if err != nil {
		return nil, errors.WithMessage(err, "could not renderer manifest")
	}

	return resp, nil
}

func (cs *clusters) addRecovered(mc *jclust.ManagedCluster) error {
	log.Infof("Recovering cluster ID: %q", mc.ID)
	c := &cluster{
		ManagedCluster:                  *mc,
		forwardingEnabled:               cs.forwardingEnabled,
		defaultForwardServerName:        cs.defaultForwardServerName,
		defaultForwardDialRetryAttempts: cs.defaultForwardDialRetryAttempts,
		defaultForwardDialRetryInterval: cs.defaultForwardDialRetryInterval,
		tunnelManager:                   tunnelmgr.NewManager(),
	}

	cs.add(mc.ID, c)

	return nil
}

func (cs *clusters) update(mc *jclust.ManagedCluster) (*bytes.Buffer, error) {
	cs.Lock()
	defer cs.Unlock()
	return cs.updateLocked(mc, false)
}

func (cs *clusters) updateLocked(mc *jclust.ManagedCluster, recovery bool) (*bytes.Buffer, error) {
	if c, ok := cs.clusters[mc.ID]; ok {
		c.Lock()
		log.Infof("Updating cluster ID: %q", c.ID)
		c.ManagedCluster = *mc
		log.Infof("New cluster ID: %q", c.ID)
		c.Unlock()
		return nil, nil
	}

	if recovery {
		return nil, cs.addRecovered(mc)
	}

	return cs.addNew(mc)
}

func (cs *clusters) remove(mc *jclust.ManagedCluster) error {
	cs.Lock()

	c, ok := cs.clusters[mc.ID]
	if !ok {
		cs.Unlock()
		msg := fmt.Sprintf("Cluster id %q does not exist", mc.ID)
		log.Debugf(msg)
		return errors.Errorf(msg)
	}

	// remove from the map so nobody can get it, but whoever uses it can
	// keep doing so
	delete(cs.clusters, mc.ID)
	cs.Unlock()
	c.stop()
	log.Infof("Cluster id %q removed", mc.ID)

	return nil
}

// Handler to create a new or update an existing cluster
func (cs *clusters) updateClusterREST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}

	// no validations... for now
	decoder := json.NewDecoder(r.Body)

	mc := new(jclust.ManagedCluster)
	err := decoder.Decode(mc)
	if err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	resp, err := cs.update(mc)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = returnManifests(w, resp)
	if err != nil {
		log.Errorf("Sending manifest to %q failed: %s", r.RemoteAddr, err)
	}
}

func (cs *clusters) deleteClusterREST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}

	decoder := json.NewDecoder(r.Body)

	mc := new(jclust.ManagedCluster)
	if err := decoder.Decode(mc); err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	err := cs.remove(mc)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	fmt.Fprintf(w, "Deleted")
}

// Determine which handler to execute based on HTTP method.
func (cs *clusters) apiHandle(w http.ResponseWriter, r *http.Request) {
	log.Debugf("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)

	// TODO auth the request

	switch r.Method {
	// TODO We need a MethodPost handler as well, should use instead of Put for
	// creating a cluster entity (which will auto-generate the ID)
	// TODO This Put handler does not behaviour list a standard PUT endpoint
	// (since it doesn't retrieve the entity ID from the URI) ... fix it later
	case http.MethodPut:
		cs.updateClusterREST(w, r)
	case http.MethodGet:
		returnJSON(w, cs.List())
	case http.MethodDelete:
		cs.deleteClusterREST(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Determine which handler to execute based on HTTP method.
func (cs *clusters) managedClusterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		// Special case: We intercept the handling for a create ManagedCluster request
		case http.MethodPost:
			cs.interceptCreateManagedCluster(h, w, r)
		// All other requests for ManagedCluster resource do not need interception
		default:
			h.ServeHTTP(w, r)
		}
	})
}

// Intercept the request and response for the given handler (assumed to be for create ManagedCluster),
// so that we can generate the Guardian manifest for the corresponding ManagedCluster and inject it
// into the response body.
func (cs *clusters) interceptCreateManagedCluster(h http.Handler, w http.ResponseWriter, r *http.Request) {
	// Use a recorder to capture the response from the proxied request
	recorder := httptest.NewRecorder()

	// Execute the actual handler
	h.ServeHTTP(recorder, r)

	recordedData := recorder.Body.Bytes()

	// Copy the recorded response headers first
	for k, v := range recorder.Header() {
		w.Header()[k] = v
	}

	// Do not modify the response for any response type other than HTTP 201 Created
	// (error related or otherwise).
	if recorder.Code != 201 {
		w.WriteHeader(recorder.Code)
		w.Write(recordedData)
		return
	}

	// Perform interception with manifest generation if the proxied request had
	// a HTTP 201 Created response.

	// Set up data structure and buffer to manipulate recorded response body
	var data map[string]interface{}
	var dataBuffer bytes.Buffer // A Buffer needs no initialization.

	if err := json.Unmarshal(recordedData, &data); err != nil {
		log.Errorf("managedClusterHandler: Could not decode JSON %s", recordedData)
		http.Error(w, "ManagedCluster was created, but unable to decode resource entity", 500)
		return
	}

	log.Debugf("ManagedClusterHandler: Successfully decoded received data")

	// New cluster used to generate manifest
	metadataObj := data["metadata"].(map[string]interface{})
	mc := jclust.ManagedCluster{
		ID: metadataObj["name"].(string),
	}

	renderedManifest, err := cs.update(&mc)
	if err != nil {
		log.Errorf("managedClusterHandler: Manifest generation failed %s", err.Error())
		http.Error(w, "managedCluster was created, but installation manifest could not be generated", 500)
		return
	}

	// Create object for the spec including the generated manifest
	data["spec"] = struct {
		InstallationManifest string `json:"installationManifest"`
	}{
		renderedManifest.String(),
	}

	if err := json.NewEncoder(&dataBuffer).Encode(data); err != nil {
		log.Errorf("managedClusterHandler: Error while encoding data for response: %s", err.Error())
		http.Error(w, "ManagedCluster was created, but unable to encode resource entity", 500)
		return
	}

	// Set the correct content length corresponding to our modified version of the response data.
	w.Header().Set("Content-Length", strconv.Itoa(dataBuffer.Len()))

	// Only write the status code after headers, as this call writes out the headers
	w.WriteHeader(recorder.Code)

	log.Debugf("managedClusterHandler: Encoded object %s", dataBuffer.String())

	// Finally write the modified response output
	w.Write(dataBuffer.Bytes())
}

// get returns the cluster
func (cs *clusters) get(id string) *cluster {
	cs.RLock()
	defer cs.RUnlock()
	return cs.clusters[id]
}

func (cs *clusters) watchK8sFrom(ctx context.Context, k8s K8sInterface,
	syncC chan<- error, last string) error {
	watcher, err := k8s.ManagedClusters().Watch(metav1.ListOptions{
		ResourceVersion: last,
	})
	if err != nil {
		return errors.Errorf("failed to create k8s watch: %s", err)
	}

	for {
		select {
		case r, ok := <-watcher.ResultChan():
			if !ok {
				return errors.Errorf("watcher stopped unexpectedly")
			}

			mcResource, ok := r.Object.(*apiv3.ManagedCluster)
			if !ok {
				log.Debugf("Unexpected object type %T", r.Object)
				continue
			}

			mc := &jclust.ManagedCluster{
				ID: mcResource.ObjectMeta.Name,
			}

			log.Debugf("Watching K8s resource type: %s for cluster %s", r.Type, mc.ID)

			var err error

			switch r.Type {
			case watch.Added:
				if !cs.watchAdded {
					break
				}
				fallthrough
			case watch.Modified:
				_, err = cs.update(mc)
			case watch.Deleted:
				err = cs.remove(mc)
			default:
				err = errors.Errorf("Watch event %s unsupported", r.Type)
			}

			if err != nil {
				log.Errorf("ManagedClusters watch event %s failed: %s", r.Type, err)
			}

			if syncC != nil {
				syncC <- err
			}
		case <-ctx.Done():
			watcher.Stop()
			return errors.Errorf("watcher exiting: %s", ctx.Err())
		}
	}
}

func (cs *clusters) resyncWithK8s(ctx context.Context, k8s K8sInterface) (string, error) {
	list, err := k8s.ManagedClusters().List(metav1.ListOptions{})
	if err != nil {
		return "", errors.Errorf("failed to get k8s list: %s", err)
	}

	known := make(map[string]struct{})

	cs.Lock()
	defer cs.Unlock()

	for _, mc := range list.Items {
		id := mc.ObjectMeta.Name

		mc := &jclust.ManagedCluster{
			ID: id,
		}

		known[id] = struct{}{}

		log.Debugf("Sync K8s watch for cluster : %s", mc.ID)
		_, err = cs.updateLocked(mc, true)
		if err != nil {
			log.Errorf("ManagedClusters listing failed: %s", err)
		}
	}

	// remove all the active clusters not in the list since we must have missed
	// the DELETE watch event
	for id, c := range cs.clusters {
		if _, ok := known[id]; ok {
			continue
		}
		delete(cs.clusters, id)
		c.stop()
		log.Infof("Cluster id %q removed", id)
	}

	return list.ListMeta.ResourceVersion, nil
}

func (cs *clusters) watchK8s(ctx context.Context, k8s K8sInterface, syncC chan<- error) error {
	for {
		last, err := cs.resyncWithK8s(ctx, k8s)
		if err == nil {
			err = cs.watchK8sFrom(ctx, k8s, syncC, last)
			if err != nil {
				err = errors.WithMessage(err, "k8s watch failed")
			}
		} else {
			err = errors.WithMessage(err, "k8s list failed")
		}
		log.Debugf("ManagedClusters: could not sync watch due to %s", err)
		select {
		case <-ctx.Done():
			return errors.Errorf("watcher exiting: %s", ctx.Err())
		default:
		}
	}
}

func (c *cluster) checkTunnelState() {
	err := <-c.tunnelManager.ListenForErrors()

	c.Lock()
	defer c.Unlock()

	c.proxy = nil
	if err := c.tunnelManager.CloseTunnel(); err != nil {
		log.WithError(err).Error("an error occurred closing the tunnel")
	}

	log.Errorf("Cluster %s tunnel is broken (%s), deleted", c.ID, err)
}

func (c *cluster) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c.tunnelManager.Open()
}

func (c *cluster) DialTLS2(network, addr string, cfg *tls.Config) (net.Conn, error) {
	return c.tunnelManager.OpenTLS(cfg)
}

func (c *cluster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.RLock()
	proxy := c.proxy
	c.RUnlock()

	if proxy == nil {
		log.Debugf("Cannot proxy to cluster %s, no tunnel", c.ID)
		writeHTTPError(w, clusterNotConnectedError(c.ID))
		return
	}

	proxy.ServeHTTP(w, r)
}

func (c *cluster) assignTunnel(t *tunnel.Tunnel) error {
	// called with RLock held
	if err := c.tunnelManager.RunWithTunnel(t); err != nil {
		return err
	}

	c.proxy = &httputil.ReverseProxy{
		Director:      proxyVoidDirector,
		FlushInterval: -1,
		// TODO set the error logger
		Transport: &http2.Transport{
			DialTLS:         c.DialTLS2,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			AllowHTTP:       true,
		},
	}

	if c.forwardingEnabled {
		go func() {
			// This loop only stops trying to listen if the tunnel or manager was closed
			for {
				shouldStop := false
				func() {
					listener, err := c.tunnelManager.Listener()
					if err != nil {
						if err == tunnel.ErrTunnelClosed || err == tunnelmgr.ErrManagerClosed {
							shouldStop = true
							return
						}
						log.WithError(err).Error("failed to listen over the tunnel")
					}
					defer listener.Close()
					if err = listenAndForward(
						listener,
						c.defaultForwardServerName,
						c.defaultForwardDialRetryAttempts,
						c.defaultForwardDialRetryInterval); err != nil {
						log.WithError(err).Error("failed to listen over the tunnel")
					}
				}()

				if shouldStop {
					break
				}
				time.Sleep(1 * time.Second)
			}

		}()
	}
	// will clean up the tunnel if it breaks, will exit once the tunnel is gone
	go c.checkTunnelState()
	return nil
}

func (c *cluster) stop() {
	// close the tunnel to disconnect. Closing is thread save, but we need
	// to hold the RLock to access the tunnel
	c.RLock()
	if c.tunnelManager != nil {
		if err := c.tunnelManager.Close(); err != nil {
			log.WithError(err).Error("an error occurred closing the tunnelManager")
		}
	}
	c.RUnlock()
}

func proxyVoidDirector(*http.Request) {
	// do nothing with the request, we pass it forward as is, the other side of
	// the tunnel should do whatever it needs to proxy it further
}
