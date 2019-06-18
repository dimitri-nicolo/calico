// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/textproto"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	jclust "github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/utils"
	"github.com/tigera/voltron/pkg/tunnel"
)

const (
	// ClusterHeaderField represents the request header key used to determine
	// which cluster to proxy for
	ClusterHeaderField = "x-cluster-id"
)

var clusterHeaderFieldCanon = textproto.CanonicalMIMEHeaderKey(ClusterHeaderField)

// Server is the voltron server that accepts tunnels from the app clusters. It
// serves HTTP requests and proxies them to the tunnels.
type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	http     *http.Server
	proxyMux *http.ServeMux

	clusters *clusters
	health   *health

	tunSrv *tunnel.Server

	certFile string
	keyFile  string

	// Creds to be used for the tunnel endpoints and to generate creds for the
	// tunnel clients a.k.a guardians
	//
	// If not set, will be populated from certFile and keyFile
	tunnelCert *x509.Certificate
	tunnelKey  crypto.Signer

	template      string
	publicAddress string
}

// New returns a new Server
func New(opts ...Option) (*Server, error) {
	srv := &Server{
		http: new(http.Server),
		clusters: &clusters{
			clusters: make(map[string]*cluster),
		},
	}

	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	srv.clusters.generateCreds = srv.generateCreds

	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	srv.proxyMux = http.NewServeMux()
	srv.http.Handler = srv.proxyMux

	srv.proxyMux.HandleFunc("/", srv.clusterMuxer)
	srv.proxyMux.HandleFunc("/voltron/api/health", srv.health.apiHandle)
	srv.proxyMux.HandleFunc("/voltron/api/clusters", srv.clusters.apiHandle)

	var tunOpts []tunnel.ServerOption

	if srv.tunnelCert == nil || srv.tunnelKey == nil {
		// if at least one piece is missing, use the default creds
		certPEM, err := utils.LoadPEMFromFile(srv.certFile)
		if err != nil {
			return nil, errors.WithMessage(err, "cert")
		}

		keyPEM, err := utils.LoadPEMFromFile(srv.keyFile)
		if err != nil {
			return nil, errors.WithMessage(err, "key")
		}

		srv.tunnelCert, srv.tunnelKey, err = utils.LoadX509KeyPairFromPEM(certPEM, keyPEM)
		if err != nil {
			return nil, errors.WithMessage(err, "loading cert/key pair")
		}
	}

	if srv.tunnelCert != nil {
		tunOpts = append(tunOpts, tunnel.WithCreds(srv.tunnelCert, srv.tunnelKey))
	}

	var err error
	srv.tunSrv, err = tunnel.NewServer(tunOpts...)
	if err != nil {
		return nil, errors.WithMessage(err, "tunnel server")
	}
	go srv.acceptTunnels()
	srv.clusters.renderer, err = NewRenderer(srv.template, srv.publicAddress, srv.tunnelCert)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not create a template to render manifests")
	}

	return srv, nil
}

// ListenAndServeHTTP starts listening and serving HTTP requests
func (s *Server) ListenAndServeHTTP() error {
	return s.http.ListenAndServe()

}

// ServeHTTP starts serving HTTP requests
func (s *Server) ServeHTTP(lis net.Listener) error {
	return s.http.Serve(lis)
}

// ListenAndServeHTTPS starts listening and serving HTTPS requests
func (s *Server) ListenAndServeHTTPS() error {
	return s.http.ListenAndServeTLS(s.certFile, s.keyFile)
}

// ServeHTTPS starts serving HTTPS requests
func (s *Server) ServeHTTPS(lis net.Listener) error {
	return s.http.ServeTLS(lis, s.certFile, s.keyFile)
}

// Close stop the server
func (s *Server) Close() error {
	s.cancel()
	s.tunSrv.Stop()
	return s.http.Close()
}

// ServeTunnels starts serving tunnels using the provided listener
func (s *Server) ServeTunnels(lis net.Listener) error {
	err := s.tunSrv.Serve(lis)
	if err != nil {
		return errors.WithMessage(err, "ServeTunnels")
	}

	return nil
}

// ServeTunnelsTLS start serving TLS secured tunnels using the provided listener and
// the TLS configuration of the Server
func (s *Server) ServeTunnelsTLS(lis net.Listener) error {
	err := s.tunSrv.ServeTLS(lis)
	if err != nil {
		return errors.WithMessage(err, "ServeTunnels")
	}

	return nil
}

func (s *Server) acceptTunnels() {
	defer log.Debugf("acceptTunnels exited")

	for {
		t, err := s.tunSrv.AcceptTunnel()
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

		var idChecker func(c *cluster) error

		switch id := t.Identity().(type) {
		case *x509.Certificate:
			// N.B. By now, we know that we signed this certificate, that means,
			// it contains what we placed into that cert, therefore there is no
			// need to do any additional checks on that cert.
			clusterID = id.EmailAddresses[0]
			// However, the cert may be outdate (e.g. revoked, custer id
			// reused, etc.) so we need to double check the cert
			idChecker = func(c *cluster) error {
				if c.cert == nil {
					return errors.Errorf("no cert assigned to cluster")
				}
				if !c.cert.Equal(id) {
					return errors.Errorf("cert assigned to cluster does not match presented cert")
				}
				return nil
			}
		default:
			log.Errorf("unknown tunnel identity type %T", id)
		}

		c := s.clusters.get(clusterID)

		if c == nil {
			log.Errorf("cluster %q does not exist", clusterID)
			t.Close()
			continue
		}

		// we call this function so that we can return and unlock on any failed
		// check
		func() {
			defer c.RUnlock()

			if err := idChecker(c); err != nil {
				log.Errorf("id check error: %s", err)
				t.Close()
				return
			}

			if c.tunnel != nil {
				log.Infof("Openning a second tunnel ID %q rejected", clusterID)
				t.Close()
				return
			}

			c.assignTunnel(t)

			log.Debugf("Accepted a new tunnel from %s", clusterID)
		}()
	}
}

func (s *Server) clusterMuxer(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Header[clusterHeaderFieldCanon]; !ok {
		msg := fmt.Sprintf("missing %q header", ClusterHeaderField)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	if len(r.Header[clusterHeaderFieldCanon]) > 1 {
		msg := fmt.Sprintf("multiple %q headers", ClusterHeaderField)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	clusterID := r.Header.Get(ClusterHeaderField)

	c := s.clusters.get(clusterID)

	if c == nil {
		msg := fmt.Sprintf("Unknown target cluster %q", clusterID)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	r.URL.Scheme = "http"
	// N.B. Host is only set to make the ReverseProxy happy, DialContext ignores
	// this as the destinatination has been decided by choosing the tunnel.
	r.URL.Host = "voltron-tunnel"

	// TODO this is the place for the impersonation hook

	log.Debugf("tunneling %q from %q through %q", r.URL, r.RemoteAddr, clusterID)
	c.ServeHTTP(w, r)
	c.RUnlock()
}

func (s *Server) generateCreds(clusterInfo *jclust.Cluster) (*x509.Certificate, crypto.Signer, error) {
	if s.tunnelCert == nil || s.tunnelKey == nil {
		return nil, nil, errors.Errorf("no credential to sign generated cert")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return nil, nil, errors.Errorf("generating RSA key: %s", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:   big.NewInt(1),
		EmailAddresses: []string{clusterInfo.ID},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().Add(1000000 * time.Hour), // XXX TBD
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	bytes, err := x509.CreateCertificate(rand.Reader, tmpl, s.tunnelCert, &key.PublicKey, s.tunnelKey)
	if err != nil {
		return nil, nil, errors.Errorf("creating X509 cert: %s", err)
	}

	cert, err := x509.ParseCertificate(bytes)
	if err != nil {
		// should never happen, we just generated the key
		return nil, nil, errors.Errorf("parsing X509 cert: %s", err)
	}

	return cert, key, nil
}

// ClusterCreds returns credential assigned to a registered cluster as PEM blocks
func (s *Server) ClusterCreds(id string) ([]byte, []byte, error) {
	c := s.clusters.get(id)
	if c == nil {
		return nil, nil, errors.Errorf("cluster id %q does not exist", id)
	}

	defer c.RUnlock()

	cPem := utils.CertPEMEncode(c.cert)

	kPem, err := utils.KeyPEMEncode(c.key)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "generated key - NEVER HAPPENS")
	}

	return cPem, kPem, nil
}
