// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"regexp"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/rest"

	"github.com/projectcalico/calico/apiserver/pkg/authentication"
	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/voltron/internal/pkg/bootstrap"
	"github.com/projectcalico/calico/voltron/internal/pkg/proxy"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"
	"github.com/projectcalico/calico/voltron/pkg/tunnel"
	"github.com/projectcalico/calico/voltron/pkg/tunnelmgr"
)

const (
	// ClusterHeaderField represents the request header key used to determine
	// which cluster to proxy for
	ClusterHeaderField = "x-cluster-id"
	// DefaultClusterID is the name of the management cluster. No tunnel is necessary for
	// requests with this value in the ClusterHeaderField.
	DefaultClusterID   = "cluster"
	DefaultReadTimeout = 45 * time.Second
)

// ClusterHeaderFieldCanon represents the request header key used to determine which
// cluster to proxy for (Canonical)
var ClusterHeaderFieldCanon = textproto.CanonicalMIMEHeaderKey(ClusterHeaderField)

// Server is the voltron server that accepts tunnels from the app clusters. It
// serves HTTP requests and proxies them to the tunnels.
type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	http     *http.Server
	proxyMux *http.ServeMux

	k8s bootstrap.K8sClient
	// When impersonating a user we use the tigera-manager sa bearer token from this config.
	config        *rest.Config
	authenticator auth.JWTAuth

	defaultProxy          *proxy.Proxy
	tunnelTargetWhitelist []regexp.Regexp
	kubernetesAPITargets  []regexp.Regexp

	clusters *clusters
	health   *health

	tunSrv *tunnel.Server

	externalCert tls.Certificate
	internalCert tls.Certificate

	addr string

	// tunnelSigningCert is the cert that was used to generate creds for the tunnel clients a.k.a guardians and
	// thus the cert that can be used to verify its identity
	tunnelSigningCert *x509.Certificate

	// tunnelCert is the cert to be used for the tunnel endpoint
	tunnelCert tls.Certificate

	tunnelEnableKeepAlive   bool
	tunnelKeepAliveInterval time.Duration

	sniServiceMap map[string]string

	// Enable FIPS 140-2 verified mode.
	fipsModeEnabled bool

	// checkManagedClusterAuthorizationBeforeProxy
	checkManagedClusterAuthorizationBeforeProxy bool
}

// New returns a new Server. k8s may be nil and options must check if it is nil
// or not if they set its user and return an error if it is nil
func New(k8s bootstrap.K8sClient, config *rest.Config, authenticator auth.JWTAuth, opts ...Option) (*Server, error) {
	srv := &Server{
		k8s:           k8s,
		config:        config,
		authenticator: authenticator,
		clusters: &clusters{
			clusters: make(map[string]*cluster),
		},
		tunnelEnableKeepAlive:   true,
		tunnelKeepAliveInterval: 100 * time.Millisecond,
	}

	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	srv.clusters.k8sCLI = srv.k8s
	srv.clusters.sniServiceMap = srv.sniServiceMap
	srv.proxyMux = http.NewServeMux()

	cfg := calicotls.NewTLSConfig(srv.fipsModeEnabled)
	cfg.Certificates = append(cfg.Certificates, srv.externalCert)

	if len(srv.internalCert.Certificate) > 0 {
		cfg.Certificates = append(cfg.Certificates, srv.internalCert)
	}

	srv.http = &http.Server{
		Addr:        srv.addr,
		Handler:     srv.proxyMux,
		TLSConfig:   cfg,
		ReadTimeout: DefaultReadTimeout,
	}

	srv.proxyMux.HandleFunc("/", srv.clusterMuxer)
	srv.proxyMux.HandleFunc("/voltron/api/health", srv.health.apiHandle)

	var tunOpts []tunnel.ServerOption

	if srv.tunnelSigningCert != nil {
		tunOpts = append(tunOpts,
			tunnel.WithClientCert(srv.tunnelSigningCert),
			tunnel.WithServerCert(srv.tunnelCert),
			tunnel.WithFIPSModeEnabled(srv.fipsModeEnabled),
		)

		var err error
		srv.tunSrv, err = tunnel.NewServer(tunOpts...)
		if err != nil {
			return nil, errors.WithMessage(err, "tunnel server")
		}
		go srv.acceptTunnels(
			tunnel.WithKeepAliveSettings(srv.tunnelEnableKeepAlive, srv.tunnelKeepAliveInterval),
		)
		if err != nil {
			return nil, errors.WithMessage(err, "Could not create a template to render manifests")
		}
	}

	return srv, nil
}

// ServeHTTPS starts serving HTTPS requests
func (s *Server) ServeHTTPS(lis net.Listener, certFile, keyFile string) error {
	return s.http.ServeTLS(lis, certFile, keyFile)
}

// ListenAndServeHTTPS starts listening and serving HTTPS requests
func (s *Server) ListenAndServeHTTPS() error {
	return s.http.ListenAndServeTLS("", "")
}

// Close stop the server
func (s *Server) Close() error {
	s.cancel()
	if s.tunSrv != nil {
		s.tunSrv.Stop()
	}
	return s.http.Close()
}

// ServeTunnelsTLS start serving TLS secured tunnels using the provided listener and
// the TLS configuration of the Server
func (s *Server) ServeTunnelsTLS(lis net.Listener) error {
	if s.tunSrv == nil {
		return errors.Errorf("No tunnel server was initiated")
	}
	err := s.tunSrv.ServeTLS(lis)
	if err != nil {
		return errors.WithMessage(err, "ServeTunnels")
	}

	return nil
}

func (s *Server) acceptTunnels(opts ...tunnel.Option) {
	defer log.Debugf("acceptTunnels exited")

	for {
		t, err := s.tunSrv.AcceptTunnel(opts...)
		if err != nil {
			select {
			case <-s.ctx.Done():
				// N.B. When s.ctx.Done() AcceptTunnel will return with an
				// error, will not block
				return
			default:
				log.Errorf("accepting tunnel failed: %s", err)
				continue
			}
		}

		clusterID, fingerprint := s.extractIdentity(t)

		c := s.clusters.get(clusterID)
		if c == nil {
			log.Errorf("cluster %q does not exist", clusterID)
			t.Close()
			continue
		}

		c.RLock()

		// we call this function so that we can return and unlock on any failed
		// check
		func() {
			defer c.RUnlock()

			if len(c.ActiveFingerprint) == 0 {
				log.Error("no fingerprint has been stored against the current connection")
				closeTunnel(t)
				return

			}
			// Before Calico Enterprise v3.15, we use md5 hash algorithm for the managed cluster
			// certificate fingerprint. md5 is known to cause collisions and it is not approved in
			// FIPS mode. From v3.15, we are upgrading the active fingerprint to use sha256 hash algorithm.
			if hex.DecodedLen(len(c.ActiveFingerprint)) == sha256.Size {
				if fingerprint != c.ActiveFingerprint {
					log.Error("stored fingerprint does not match provided fingerprint")
					closeTunnel(t)
					return
				}
			} else {
				// md5 is not approved in FIPS mode so not upgrading from md5 to sha256
				if s.fipsModeEnabled {
					log.Errorf("cluster %s stored fingerprint can not be updated in FIPS mode", clusterID)
					closeTunnel(t)
					return
				}

				// check pre-v3.15 fingerprint (md5)
				if s.extractMD5Identity(t) != c.ActiveFingerprint {
					log.Error("stored fingerprint does not match provided fingerprint")
					closeTunnel(t)
					return
				}

				// update to v3.15 fingerprint hash (sha256) when matched
				if err := c.updateActiveFingerprint(fingerprint); err != nil {
					log.WithError(err).Errorf("failed to update cluster %s stored fingerprint", clusterID)
					closeTunnel(t)
					return
				}

				log.Infof("Cluster %s stored fingerprint is successfully updated", clusterID)
			}

			if err := c.assignTunnel(t); err != nil {
				if err == tunnelmgr.ErrTunnelSet {
					log.Errorf("opening a second tunnel ID %s rejected", clusterID)
				} else {
					log.WithError(err).Errorf("failed to open the tunnel for cluster %s", clusterID)
				}

				if err := t.Close(); err != nil {
					log.WithError(err).Errorf("failed closed tunnel after failing to assign it to cluster %s", clusterID)
				}
			}

			log.Debugf("Accepted a new tunnel from %s", clusterID)
		}()
	}
}

func closeTunnel(t *tunnel.Tunnel) {
	var err = t.Close()
	if err != nil {
		log.WithError(err).Error("Could not close tunnel")
	}
}

func (s *Server) extractIdentity(t *tunnel.Tunnel) (clusterID, fingerprint string) {
	switch id := t.Identity().(type) {
	case *x509.Certificate:
		// N.B. By now, we know that we signed this certificate as these checks
		// are performed during TLS handshake. We need to extract the common name
		// and fingerprint of the certificate to check against our internal records
		// We expect to have a cluster registered with this ID and matching fingerprint
		// for the cert.
		clusterID = id.Subject.CommonName
		fingerprint = utils.GenerateFingerprint(id)
	default:
		log.Errorf("unknown tunnel identity type %T", id)
	}
	return
}

func (s *Server) extractMD5Identity(t *tunnel.Tunnel) (fingerprint string) {
	switch id := t.Identity().(type) {
	case *x509.Certificate:
		fingerprint = fmt.Sprintf("%x", md5.Sum(id.Raw))
	default:
		log.Errorf("unknown tunnel identity type %T", id)
	}
	return
}

func (s *Server) clusterMuxer(w http.ResponseWriter, r *http.Request) {
	chdr, hasClusterHeader := r.Header[ClusterHeaderFieldCanon]
	isK8sRequest := requestPathMatches(r, s.kubernetesAPITargets)
	shouldUseTunnel := requestPathMatches(r, s.tunnelTargetWhitelist) && hasClusterHeader

	// If shouldUseTunnel=true, we do authn checks & impersonation and the request will be sent to guardian.
	// If isK8sRequest=true, we also do authn checks & impersonation.
	// If neither is true, we just proxy the request. Authn will be handled there.
	if (!shouldUseTunnel || !hasClusterHeader) && !isK8sRequest {
		// This is a request for the backend servers in the management cluster, like es-proxy or compliance.
		s.defaultProxy.ServeHTTP(w, r)
		return
	}

	if len(chdr) > 1 {
		msg := fmt.Sprintf("multiple %q headers", ClusterHeaderField)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	usr, status, err := s.authenticator.Authenticate(r)
	if err != nil {
		log.Errorf("Could not authenticate user from request: %s", err)
		http.Error(w, err.Error(), status)
		return
	}
	addImpersonationHeaders(r, usr)
	removeAuthHeaders(r)

	// Note, we expect the value passed in the request header field to be the resource
	// name for a ManagedCluster resource (which will be human-friendly and unique)
	clusterID := r.Header.Get(ClusterHeaderField)

	if isK8sRequest && (!hasClusterHeader || clusterID == DefaultClusterID) {
		r.Header.Set(authentication.AuthorizationHeader, fmt.Sprintf("Bearer %s", s.config.BearerToken))
		s.defaultProxy.ServeHTTP(w, r)
		return
	}

	c := s.clusters.get(clusterID)

	if c == nil {
		msg := fmt.Sprintf("Unknown target cluster %q", clusterID)
		log.Errorf("clusterMuxer: %s", msg)
		writeHTTPError(w, clusterNotFoundError(clusterID))
		return
	}

	// perform an authorization to make sure this user can get this cluster
	if s.checkManagedClusterAuthorizationBeforeProxy {
		ok, err := s.authenticator.Authorize(usr, &authorizationv1.ResourceAttributes{
			Verb:     "get",
			Group:    "projectcalico.org",
			Version:  "v3",
			Resource: "managedclusters",
			Name:     clusterID,
		}, nil)
		if err != nil {
			log.Errorf("Could not authenticate user for cluster: %s", err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !ok {
			http.Error(w, "not authorized for managed cluster", http.StatusUnauthorized)
			return
		}
	}

	// We proxy through a secure tunnel, therefore we only enforce https for HTTP/2
	// XXX What if we set http2.Transport.AllowHTTP = true ?
	r.URL.Scheme = "http"
	if r.ProtoMajor == 2 {
		r.URL.Scheme = "https"
	}
	// N.B. Host is only set to make the ReverseProxy happy, DialContext ignores
	// this as the destinatination has been decided by choosing the tunnel.
	r.URL.Host = "voltron-tunnel"
	r.Header.Del(ClusterHeaderField)
	c.ServeHTTP(w, r)
}

// Determine whether or not the given request should use the tunnel proxying
// by comparing its URL path against the provide list of regex expressions
// (representing paths for targets that the request might be going to).
func requestPathMatches(r *http.Request, targetPaths []regexp.Regexp) bool {
	for _, p := range targetPaths {
		if p.MatchString(r.URL.Path) {
			return true
		}
	}
	return false
}

func removeAuthHeaders(r *http.Request) {
	r.Header.Del("Authorization")
	r.Header.Del("Auth")
}

func addImpersonationHeaders(r *http.Request, user user.Info) {
	r.Header.Add("Impersonate-User", user.GetName())
	for _, group := range user.GetGroups() {
		r.Header.Add("Impersonate-Group", group)
	}
	log.Debugf("Adding impersonation headers")
}

// WatchK8s starts watching k8s resources, always exits with an error
func (s *Server) WatchK8s() error {
	return s.WatchK8sWithSync(nil)
}

// WatchK8sWithSync is a variant of WatchK8s for testing. Every time a watch
// event is handled its result is posted on the syncC channel
func (s *Server) WatchK8sWithSync(syncC chan<- error) error {
	if s.k8s == nil {
		return errors.Errorf("no k8s interface")
	}

	return s.clusters.watchK8s(s.ctx, syncC)
}
