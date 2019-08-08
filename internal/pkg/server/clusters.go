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
	jclust.Cluster

	sync.RWMutex

	tunnel *tunnel.Tunnel
	proxy  *httputil.ReverseProxy

	cert *x509.Certificate
	key  crypto.Signer
}

type clusters struct {
	sync.RWMutex
	clusters      map[string]*cluster
	generateCreds func(*jclust.Cluster) (*x509.Certificate, crypto.Signer, error)

	// keep the generated keys, only for testing and debugging
	keepKeys bool
	renderer *Renderer
}

func returnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf("Error while encoding data for response %#v", data)
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

// List all clusters in sorted order by DisplayName field
func (cs *clusters) List() []jclust.Cluster {
	cs.RLock()
	defer cs.RUnlock()

	clusterList := make([]jclust.Cluster, 0, len(cs.clusters))
	for _, c := range cs.clusters {
		// Only include non-sensitive fields

		c.RLock()
		clusterList = append(clusterList, c.Cluster)
		c.RUnlock()
	}

	sort.Slice(clusterList, func(i, j int) bool {
		return clusterList[i].DisplayName < clusterList[j].DisplayName
	})

	log.Debugf("clusterList = %+v", clusterList)
	return clusterList
}

func (cs *clusters) update(jc *jclust.Cluster) (*bytes.Buffer, error) {
	cs.Lock()
	defer cs.Unlock()

	resp := new(bytes.Buffer)

	if c, ok := cs.clusters[jc.ID]; ok {
		c.Lock()
		log.Infof("Updating cluster ID: %q DisplayName: %q", c.ID, c.DisplayName)
		c.Cluster = *jc
		log.Infof("New cluster ID: %q DisplayName: %q", c.ID, c.DisplayName)
		c.Unlock()
	} else {
		log.Infof("Adding cluster ID: %q DisplayName: %q", jc.ID, jc.DisplayName)

		cert, key, err := cs.generateCreds(jc)
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to generate cluster credentials")
		}

		c := &cluster{
			Cluster: *jc,
			cert:    cert,
		}
		if cs.keepKeys {
			c.key = key
		}

		cs.add(jc.ID, c)
		ok := cs.renderer.RenderManifest(resp, cert, key)
		if !ok {
			return nil, errors.Errorf("could not renderer manifest")
		}
	}

	return resp, nil
}

func (cs *clusters) remove(jc *jclust.Cluster) error {
	cs.Lock()

	c, ok := cs.clusters[jc.ID]
	if !ok {
		cs.Unlock()
		msg := fmt.Sprintf("Cluster id %q does not exist", jc.ID)
		log.Debugf(msg)
		return errors.Errorf(msg)
	}

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

	log.Infof("Cluster id %q removed", jc.ID)

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

	jc := new(jclust.Cluster)
	err := decoder.Decode(jc)
	if err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	resp, err := cs.update(jc)
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

	jc := new(jclust.Cluster)
	if err := decoder.Decode(jc); err != nil {
		http.Error(w, "Invalid JSON body", 400)
		return
	}

	err := cs.remove(jc)
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

	log.Debugf("managedClusterHandler: Decoded object %+v", data)

	// New cluster used to generate manifest
	metadataObj := data["metadata"].(map[string]interface{})
	jc := jclust.Cluster{
		ID:          metadataObj["uid"].(string),
		DisplayName: metadataObj["name"].(string),
	}

	renderedManifest, err := cs.update(&jc)
	if err != nil {
		log.Errorf("managedClusterHandler: Manifest generation failed %s", err.Error())
		http.Error(w, "ManagedCluster was created, but installation manifest could not be generated", 500)
		return
	}

	// Create object for the spec including the generated manifest
	data["spec"] = struct {
		InstallationManifest string `json:"installationManifest"`
	}{
		renderedManifest.String(),
	}

	if err := json.NewEncoder(&dataBuffer).Encode(data); err != nil {
		log.Errorf("managedClusterHandler: Error while encoding data for response %#v: err %s", data, err.Error())
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

func (cs *clusters) watchK8s(ctx context.Context, k8s K8sInterface, syncC chan<- error) error {
	watcher, err := k8s.ManagedClusters().Watch(metav1.ListOptions{})
	if err != nil {
		return errors.Errorf("failed to create k8s watch: %s", err)
	}

	for {
		select {
		case r, ok := <-watcher.ResultChan():
			if !ok {
				return errors.Errorf("watcher stopped unexpectedly")
			}

			mc, ok := r.Object.(*apiv3.ManagedCluster)
			if !ok {
				log.Errorf("Unexpected object type %T value %+v", r.Object, r.Object)
				continue
			}

			jc := &jclust.Cluster{
				ID:          string(mc.ObjectMeta.UID),
				DisplayName: mc.ObjectMeta.Name,
			}

			log.Debugf("WatchK8s: %s %+v", r.Type, jc)

			var err error

			switch r.Type {
			case watch.Added, watch.Modified:
				_, err = cs.update(jc)
			case watch.Deleted:
				err = cs.remove(jc)
			default:
				err = errors.Errorf("watch event %s unsupported", r.Type)
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

func (c *cluster) checkTunnelState() {
	err := c.tunnel.WaitForError()

	c.Lock()
	defer c.Unlock()

	c.proxy = nil
	c.tunnel.Close()
	c.tunnel = nil
	log.Errorf("Cluster %s tunnel is broken (%s), deleted", c.ID, err)
}

func (c *cluster) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c.tunnel.Open()
}

func (c *cluster) DialTLS2(network, addr string, cfg *tls.Config) (net.Conn, error) {
	conn, err := c.tunnel.Open()
	if err != nil {
		return nil, errors.WithMessage(err, "c.tunnel.Open")
	}

	return tls.Client(conn, cfg), nil
}

func (c *cluster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.proxy == nil {
		log.Debugf("Cannot proxy to cluster %s, no tunnel", c.DisplayName)
		writeHTTPError(w, clusterNotConnectedError(c.DisplayName))
		return
	}

	c.proxy.ServeHTTP(w, r)
}

func (c *cluster) assignTunnel(t *tunnel.Tunnel) {
	c.tunnel = t
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

	// will clean up the tunnel if it breaks, will exit once the tunnel is gone
	go c.checkTunnelState()
}

func proxyVoidDirector(*http.Request) {
	// do nothing with the request, we pass it forward as is, the other side of
	// the tunnel should do whatever it needs to proxy it further
}
