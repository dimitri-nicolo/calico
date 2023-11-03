// Copyright (c) 2019-2023 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/voltron/internal/pkg/bootstrap"
	jclust "github.com/projectcalico/calico/voltron/internal/pkg/clusters"
	"github.com/projectcalico/calico/voltron/internal/pkg/config"
	"github.com/projectcalico/calico/voltron/internal/pkg/proxy"
	vtls "github.com/projectcalico/calico/voltron/pkg/tls"
	"github.com/projectcalico/calico/voltron/pkg/tunnel"
	"github.com/projectcalico/calico/voltron/pkg/tunnelmgr"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// AnnotationActiveCertificateFingerprint is an annotation that is used to store the fingerprint for
// managed cluster certificate that is allowed to initiate connections.
const AnnotationActiveCertificateFingerprint = "certs.tigera.io/active-fingerprint"
const contextTimeout = 30 * time.Second

type cluster struct {
	jclust.ManagedCluster

	sync.RWMutex

	tunnelManager tunnelmgr.Manager

	// proxy is a reverse proxy for handling connections to Voltron from the management cluster
	// that should be directed down the tunnel to the managed cluster.
	proxy *httputil.ReverseProxy

	// Kubernetes client used for querying and watching ManagedCluster resources.
	k8sCLI bootstrap.K8sClient

	client ctrlclient.WithWatch

	// tlsProxy is the proxy that handles incoming TLS connections from the managed cluster.
	// These connections are routed via the server field in the TLS header. Connections via this proxy that
	// target Voltron itself will be handled by the proxy's inner TLS server.
	tlsProxy vtls.Proxy

	// Pointer to general Voltron configuration.
	voltronCfg *config.Config
}

// updateActiveFingerprint updates the active fingerprint annotation for a ManagedCluster resource
// in the management cluster.
func (c *cluster) updateActiveFingerprint(fingerprint string) error {

	mc := &v3.ManagedCluster{}
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	// Client Get act as single tenant when the TenantNamespace is empty
	err := c.client.Get(ctx, types.NamespacedName{Name: c.ID, Namespace: c.voltronCfg.TenantNamespace}, mc)
	if err != nil {
		return err
	}

	mc.Annotations[AnnotationActiveCertificateFingerprint] = fingerprint

	err = c.client.Update(ctx, mc)
	if err != nil {
		return err
	}

	c.ActiveFingerprint = fingerprint

	return nil
}

type clusters struct {
	sync.RWMutex
	clusters        map[string]*cluster
	sniServiceMap   map[string]string
	k8sCLI          bootstrap.K8sClient
	client          ctrlclient.WithWatch
	fipsModeEnabled bool

	// parameters for forwarding guardian requests to a default server
	forwardingEnabled               bool
	defaultForwardServerName        string
	defaultForwardDialRetryAttempts int
	defaultForwardDialRetryInterval time.Duration

	// Pointer to general Voltron config.
	voltronCfg *config.Config

	// TLS configuration to use for inner tunnel HTTPS servers.
	tlsConfig *tls.Config

	// Proxier to use for connections from managed clusters.
	innerProxy *proxy.Proxy

	// Pool used for client certificate verification.
	clientCertificatePool *x509.CertPool
}

func (cs *clusters) makeInnerTLSConfig() error {
	cfg := calicotls.NewTLSConfig(cs.voltronCfg.FIPSModeEnabled)
	if cs.voltronCfg.LinseedServerKey != "" && cs.voltronCfg.LinseedServerCert != "" {
		certBytes, err := os.ReadFile(cs.voltronCfg.LinseedServerCert)
		if err != nil {
			return err
		}
		keyBytes, err := os.ReadFile(cs.voltronCfg.LinseedServerKey)
		if err != nil {
			return err
		}
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return err
		}
		cfg.Certificates = append(cfg.Certificates, cert)
	}
	cs.tlsConfig = cfg
	return nil
}

func (cs *clusters) add(mc *jclust.ManagedCluster) (*cluster, error) {
	if cs.clusters[mc.ID] != nil {
		return nil, errors.Errorf("cluster id %q already exists", mc.ID)
	}

	c := &cluster{
		ManagedCluster: *mc,
		tunnelManager:  tunnelmgr.NewManager(),
		k8sCLI:         cs.k8sCLI,
		client:         cs.client,
		voltronCfg:     cs.voltronCfg,
	}

	// Append the new certificate to the client certificate pool.
	cs.clientCertificatePool.AppendCertsFromPEM(mc.Certificate)
	log.Infof("Appended certificate for cluster %s to client certificate pool", mc.ID)

	// Create a proxy to use for connections received over the tunnel that aren't
	// directed via SNI. This is just used for Linseed connections from managed clusters.
	// We use the same TLS configuration as the main tunnel. We will proxy the connection
	// presenting Voltron's internal management cluster certificate, as Linseed requires mTLS.
	//
	// This handler will only be used for requests from managed clusters over the mTLS tunnel
	// with a server name of "tigera-linseed.tigera-elasticsearch".
	innerServer := &http.Server{
		Handler:     NewInnerHandler(cs.voltronCfg.TenantID, mc, cs.innerProxy).Handler(),
		TLSConfig:   cs.tlsConfig,
		ReadTimeout: DefaultReadTimeout,
	}

	if cs.forwardingEnabled {
		tlsProxy, err := vtls.NewProxy(
			vtls.WithDefaultServiceURL(cs.defaultForwardServerName),
			vtls.WithProxyOnSNI(true),
			vtls.WithSNIServiceMap(cs.sniServiceMap),
			vtls.WithConnectionRetryAttempts(cs.defaultForwardDialRetryAttempts),
			vtls.WithConnectionRetryInterval(cs.defaultForwardDialRetryInterval),
			vtls.WithFipsModeEnabled(cs.fipsModeEnabled),
			vtls.WithInnerServer(innerServer),
		)
		if err != nil {
			return nil, err
		}

		c.tlsProxy = tlsProxy
	}

	cs.clusters[mc.ID] = c
	return c, nil
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

func (cs *clusters) addNew(mc *jclust.ManagedCluster) error {
	log.Infof("Adding cluster ID: %q", mc.ID)

	_, err := cs.add(mc)
	if err != nil {
		return err
	}

	return nil
}

func (cs *clusters) addRecovered(mc *jclust.ManagedCluster) error {
	log.Infof("Recovering cluster ID: %q", mc.ID)

	_, err := cs.add(mc)
	return err
}

func (cs *clusters) update(mc *jclust.ManagedCluster) error {
	cs.Lock()
	defer cs.Unlock()
	return cs.updateLocked(mc, false)
}

func (cs *clusters) updateLocked(mc *jclust.ManagedCluster, recovery bool) error {
	if c, ok := cs.clusters[mc.ID]; ok {
		c.Lock()
		log.Infof("Updating cluster ID: %q", c.ID)
		// Update the certificate pool if the certificate has changed.
		err, updated := cs.updateCertPool(mc.Certificate, c.Certificate)
		if err != nil {
			c.Unlock()
			return err
		}
		c.ManagedCluster = *mc
		log.Infof("New cluster ID: %q", c.ID)
		if updated {
			if err := c.tunnelManager.CloseTunnel(); err != nil {
				log.Error("failed to close tunnel")
			}
		}
		c.Unlock()
		return nil
	}

	if recovery {
		return cs.addRecovered(mc)
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

// get returns the cluster
func (cs *clusters) get(id string) *cluster {
	cs.RLock()
	defer cs.RUnlock()
	return cs.clusters[id]
}

func (cs *clusters) watchK8sFrom(ctx context.Context, syncC chan<- error, last string) error {

	watcher, err := cs.client.Watch(ctx, &v3.ManagedClusterList{}, &ctrlclient.ListOptions{Namespace: cs.voltronCfg.TenantNamespace})
	if err != nil {
		return errors.Errorf("failed to create k8s watch: %s", err)
	}

	for {
		select {
		case r, ok := <-watcher.ResultChan():
			if !ok {
				return errors.Errorf("watcher stopped unexpectedly")
			}
			mcResource, ok := r.Object.(*v3.ManagedCluster)
			if !ok {
				log.Debugf("Unexpected object type %T", r.Object)
				continue
			}

			mc := &jclust.ManagedCluster{
				ID:                mcResource.ObjectMeta.Name,
				ActiveFingerprint: mcResource.ObjectMeta.Annotations[AnnotationActiveCertificateFingerprint],
				Certificate:       mcResource.Spec.Certificate,
				FIPSModeEnabled:   cs.fipsModeEnabled,
			}

			log.Debugf("Watching K8s resource type: %s for cluster %s", r.Type, mc.ID)

			var err error

			switch r.Type {
			case watch.Added, watch.Modified:
				log.Infof("Adding/Updating %s", mc.ID)
				err = cs.update(mc)
			case watch.Deleted:
				log.Infof("Deleting %s", mc.ID)
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

func (cs *clusters) resyncWithK8s(ctx context.Context, startupSync bool) (string, error) {

	list := &v3.ManagedClusterList{}
	err := cs.client.List(ctx, list, &ctrlclient.ListOptions{Namespace: cs.voltronCfg.TenantNamespace})
	if err != nil {
		return "", errors.Errorf("failed to get k8s list: %s", err)
	}

	known := make(map[string]struct{})

	cs.Lock()
	defer cs.Unlock()

	for _, mc := range list.Items {
		id := mc.ObjectMeta.Name

		mc := &jclust.ManagedCluster{
			ID:                id,
			ActiveFingerprint: mc.ObjectMeta.Annotations[AnnotationActiveCertificateFingerprint],
			Certificate:       mc.Spec.Certificate,
		}

		known[id] = struct{}{}

		log.Debugf("Sync K8s watch for cluster : %s", mc.ID)
		err = cs.updateLocked(mc, true)
		if err != nil {
			log.Errorf("ManagedClusters listing failed: %s", err)
		}

		if c, ok := cs.clusters[id]; ok {
			if startupSync {
				c.Lock()

				// Just update the cluster status even if it's already set to false, we just do this on startup.
				if err := c.setConnectedStatus(v3.ManagedClusterStatusValueFalse); err != nil {
					c.Unlock()
					return "", errors.Errorf("failed to update the connection status for cluster %s during startup.", c.ID)
				}
				c.Unlock()
			}
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

func (cs *clusters) watchK8s(ctx context.Context, syncC chan<- error) error {
	// Initial sync for new server
	startupSync := true
	for {
		last, err := cs.resyncWithK8s(ctx, startupSync)
		if err == nil {
			startupSync = false
			err = cs.watchK8sFrom(ctx, syncC, last)
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

// updateCertPool updates the client cert pool if the new (non-empty) certificate is different from
// the old one.
func (cs *clusters) updateCertPool(newCertPEM, oldCertPEM []byte) (error, bool) {
	updated := false
	if len(newCertPEM) == 0 {
		// No pool update necessary for an empty certificate.
		log.Debugf("No pool update necessary for an empty certificate.")
		return nil, updated
	}

	newCert, err := parseCertificatePEMBlock(newCertPEM)
	if err != nil {
		return err, updated
	}

	if len(oldCertPEM) != 0 {
		oldCert, err := parseCertificatePEMBlock(oldCertPEM)
		if err != nil {
			return err, updated
		}

		if oldCert.Equal(newCert) {
			// No pool update necessary if the certificates are the same.
			log.Debugf("No pool update necessary if the certificates are the same.")
			return nil, updated
		}
	}

	cs.clientCertificatePool.AddCert(newCert)
	updated = true
	log.Infof("Updated client cert pool with new value.")

	return nil, updated
}

func (c *cluster) checkTunnelState() {
	err := <-c.tunnelManager.ListenForErrors()

	c.Lock()
	defer c.Unlock()

	c.proxy = nil
	if err := c.tunnelManager.CloseTunnel(); err != nil {
		log.WithError(err).Error("an error occurred closing the tunnel")
	}

	if err := c.setConnectedStatus(v3.ManagedClusterStatusValueFalse); err != nil {
		log.WithError(err).Errorf("failed to update the connection status for cluster %s", c.ID)
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
	if err := c.tunnelManager.SetTunnel(t); err != nil {
		return err
	}

	tlsConfig := calicotls.NewTLSConfig(c.FIPSModeEnabled)
	tlsConfig.InsecureSkipVerify = true // todo: not sure where this comes from, but this should be dealt with.
	c.proxy = &httputil.ReverseProxy{
		Director:      proxyVoidDirector,
		FlushInterval: -1,
		// TODO set the error logger
		Transport: &http2.Transport{
			DialTLS:         c.DialTLS2,
			TLSClientConfig: tlsConfig,
			AllowHTTP:       true,
		},
	}

	if c.tlsProxy != nil {
		go func() {
			log.Debugf("server has started listening for connections from cluster %s", c.ID)
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
						return
					}
					defer listener.Close()

					if err := c.tlsProxy.ListenAndProxy(listener); err != nil {
						log.WithError(err).Error("failed to listen for incoming requests through the tunnel")
					}
				}()

				if shouldStop {
					break
				}
				time.Sleep(1 * time.Second)
			}
			log.Debugf("server has stopped listening for connections from %s", c.ID)
		}()
	}
	if err := c.setConnectedStatus(v3.ManagedClusterStatusValueTrue); err != nil {
		log.WithError(err).Errorf("failed to update the connection status for cluster %s", c.ID)
	}
	// will clean up the tunnel if it breaks, will exit once the tunnel is gone
	go c.checkTunnelState()

	return nil
}

// setConnectedStatus updates the MangedClusterConnected condition of this cluster's ManagedCluster CR.
func (c *cluster) setConnectedStatus(status v3.ManagedClusterStatusValue) error {

	var mc = &v3.ManagedCluster{}
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	err := c.client.Get(ctx, types.NamespacedName{Name: c.ID, Namespace: c.voltronCfg.TenantNamespace}, mc)
	if err != nil {
		return err
	}

	var updatedConditions []v3.ManagedClusterStatusCondition

	connectedConditionFound := false
	for _, c := range mc.Status.Conditions {
		if c.Type == v3.ManagedClusterStatusTypeConnected {
			c.Status = status
			connectedConditionFound = true
		}
		updatedConditions = append(updatedConditions, c)
	}

	if !connectedConditionFound {
		updatedConditions = append(updatedConditions, v3.ManagedClusterStatusCondition{
			Type:   v3.ManagedClusterStatusTypeConnected,
			Status: status,
		})
	}

	mc.Status.Conditions = updatedConditions

	err = c.client.Update(ctx, mc)
	if err != nil {
		return err
	}

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

// parseCertificatePEMBlock decodes a PEM encoded certificate and returns the parsed x509 certificate.
// The PEM cert is assumed to be a single block.
func parseCertificatePEMBlock(certPEM []byte) (*x509.Certificate, error) {
	// Decode PEM content
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM block containing certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
