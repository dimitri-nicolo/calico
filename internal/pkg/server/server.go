// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/apiserver/pkg/authentication"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/utils"
	"github.com/tigera/voltron/pkg/tunnel"
	"github.com/tigera/voltron/pkg/tunnelmgr"

	"k8s.io/apiserver/pkg/authentication/user"
)

const (
	// ClusterHeaderField represents the request header key used to determine
	// which cluster to proxy for
	ClusterHeaderField = "x-cluster-id"
	// This cluster name is the management cluster. No tunnel is necessary for
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
	authenticator authentication.Authenticator

	defaultProxy          *proxy.Proxy
	tunnelTargetWhitelist []regexp.Regexp
	kubernetesAPITargets  []regexp.Regexp

	clusters *clusters
	health   *health

	tunSrv *tunnel.Server

	externalCert tls.Certificate
	internalCert tls.Certificate

	addr string

	// Creds to be used for the tunnel endpoints and to generate creds for the
	// tunnel clients a.k.a guardians
	tunnelCert *x509.Certificate
	tunnelKey  crypto.Signer

	tunnelEnableKeepAlive   bool
	tunnelKeepAliveInterval time.Duration

	publicAddress string

	sniServiceMap map[string]string
}

// New returns a new Server. k8s may be nil and options must check if it is nil
// or not if they set its user and return an error if it is nil
func New(k8s bootstrap.K8sClient, config *rest.Config, authenticator authentication.Authenticator, opts ...Option) (*Server, error) {
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

	cfg := &tls.Config{}

	cfg.Certificates = append(cfg.Certificates, srv.externalCert)

	if len(srv.internalCert.Certificate) > 0 {
		cfg.Certificates = append(cfg.Certificates, srv.internalCert)
	}

	cfg.BuildNameToCertificate()

	srv.http = &http.Server{
		Addr:        srv.addr,
		Handler:     srv.proxyMux,
		TLSConfig:   cfg,
		ReadTimeout: DefaultReadTimeout,
	}

	srv.proxyMux.HandleFunc("/", srv.clusterMuxer)
	srv.proxyMux.HandleFunc("/voltron/api/health", srv.health.apiHandle)
	srv.proxyMux.HandleFunc("/voltron/api/clusters", srv.clusters.apiHandle)

	var tunOpts []tunnel.ServerOption

	if srv.tunnelCert != nil && srv.tunnelKey != nil {
		tunOpts = append(tunOpts, tunnel.WithCreds(srv.tunnelCert, srv.tunnelKey))
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

		var clusterID string
		var fingerprint string

		clusterID, fingerprint = s.extractIdentity(t, clusterID, fingerprint)

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
				log.Error("No fingerprint has been stored against the current connection")
				closeTunnel(t)
				return

			}
			if fingerprint != c.ActiveFingerprint {
				log.Error("Stored fingerprint does not match provided fingerprint")
				closeTunnel(t)
				return
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

func (s *Server) extractIdentity(t *tunnel.Tunnel, clusterID string, fingerprint string) (string, string) {
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
	return clusterID, fingerprint
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

	usr, status, err := s.authenticator.Authenticate(r.Header.Get(authentication.AuthorizationHeader))
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
