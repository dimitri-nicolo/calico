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
	"k8s.io/client-go/kubernetes"

	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/voltron/internal/pkg/auth"
	jclust "github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/utils"
	"github.com/tigera/voltron/pkg/tunnel"
)

const (
	// ClusterHeaderField represents the request header key used to determine
	// which cluster to proxy for
	ClusterHeaderField = "x-cluster-id"
)

// ClusterHeaderFieldCanon represents the request header key used to determine which
// cluster to proxy for (Canonical)
var ClusterHeaderFieldCanon = textproto.CanonicalMIMEHeaderKey(ClusterHeaderField)

// K8sInterface represent the interface server needs to access all k8s resources
type K8sInterface interface {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

// Server is the voltron server that accepts tunnels from the app clusters. It
// serves HTTP requests and proxies them to the tunnels.
type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	http     *http.Server
	proxyMux *http.ServeMux

	k8s K8sInterface

	defaultProxy *proxy.Proxy

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

	tunnelEnableKeepAlive   bool
	tunnelKeepAliveInterval time.Duration

	template      string
	publicAddress string

	auth *auth.Identity

	toggles toggles
}

// toggles are the toggles that enable or disable a feature
type toggles struct {
	autoRegister bool
}

// New returns a new Server. k8s may be nil and options must check if it is nil
// or not if they set its user and return an error if it is nil
func New(k8s K8sInterface, opts ...Option) (*Server, error) {
	srv := &Server{
		k8s:  k8s,
		http: new(http.Server),
		clusters: &clusters{
			clusters: make(map[string]*cluster),
		},
		tunnelEnableKeepAlive:   true,
		tunnelKeepAliveInterval: 100 * time.Millisecond,
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
	// Special case: For POST request on ManagedCluster resource we want to intercept the response before
	// it gets sent back to client. The interception allows us to generate the manifest for Guardian that
	// corresponds to the ManagedCluster that was just created.
	// We accomplish this using a middlewares that wraps the clusterMux handler.
	srv.proxyMux.Handle(
		"/apis/projectcalico.org/v3/managedclusters",
		srv.clusters.managedClusterHandler(http.HandlerFunc(srv.clusterMuxer)),
	)

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
	go srv.acceptTunnels(
		tunnel.WithKeepAliveSettings(srv.tunnelEnableKeepAlive, srv.tunnelKeepAliveInterval),
	)
	srv.clusters.renderManifest, err = newRenderer(srv.template, srv.publicAddress, srv.tunnelCert)
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

func (s *Server) autoRegister(id string, ident tunnel.Identity) (*cluster, error) {
	cert, ok := ident.(*x509.Certificate)
	if !ok {
		return nil, errors.Errorf("unexpected identity type: %T", ident)
	}

	c := &cluster{
		Cluster: jclust.Cluster{
			ID:          id,
			DisplayName: id,
		},
		cert: cert,
	}

	s.clusters.Lock()
	err := s.clusters.add(id, c)
	s.clusters.Unlock()

	if err != nil {
		return nil, err
	}

	c.RLock()

	return c, nil
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

			if s.toggles.autoRegister {
				var err error
				c, err = s.autoRegister(clusterID, t.Identity())
				if err != nil {
					log.Errorf("auto-registration of %q failed: %s", clusterID, err)
					t.Close()
					continue
				}
			} else {
				log.Errorf("cluster %q does not exist", clusterID)
				t.Close()
				continue
			}
		}

		c.RLock()

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
	if _, ok := r.Header[ClusterHeaderFieldCanon]; !ok {
		if s.defaultProxy != nil {
			s.defaultProxy.ServeHTTP(w, r)
			return
		}

		msg := fmt.Sprintf("missing %q header", ClusterHeaderField)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	if len(r.Header[ClusterHeaderFieldCanon]) > 1 {
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

	user, err := s.auth.Authenticate(r)
	if err != nil {
		log.Errorf("Could not authenticate user from request: %v", err)
		http.Error(w, err.Error(), 401)
		return
	}
	addImpersonationHeaders(r, user)
	removeAuthHeaders(r)

	log.Debugf("tunneling %q from %q through %q", r.URL, r.RemoteAddr, clusterID)
	r.Header.Del(ClusterHeaderField)

	c.ServeHTTP(w, r)
}

func removeAuthHeaders(r *http.Request) {
	r.Header.Del("Authorization")
	r.Header.Del("Auth")
}

func addImpersonationHeaders(r *http.Request, user *auth.User) {
	r.Header.Add("Impersonate-User", user.Name)
	for _, group := range user.Groups {
		r.Header.Add("Impersonate-Group", group)
	}
	log.Debugf("Setting headers %v", r.Header)
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

	c.RLock()
	defer c.RUnlock()

	cPem := utils.CertPEMEncode(c.cert)

	pem, err := utils.KeyPEMEncode(c.key)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "generated key - NEVER HAPPENS")
	}

	return cPem, pem, nil
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

	return s.clusters.watchK8s(s.ctx, s.k8s, syncC)
}

// SimpleServer is a struct that allows a simple server & its tunnel to be spun up.
type SimpleServer struct {
	listenAddress string
	tunnelAddress string
}

// NewSimpleServer creates a simple Sender which listens on TCP Socket at listenAddress
// and forwards it to tunnel on tunnelAddress
func NewSimpleServer(listenAddress, tunnelAddress string) *SimpleServer {
	s := &SimpleServer{
		listenAddress: listenAddress,
		tunnelAddress: tunnelAddress,
	}

	log.Infof("Created Sender with Server Address %s and Tunnel Address %s", s.listenAddress, s.tunnelAddress)
	return s
}

// Start listens on TCP socket for incoming connections and connects to the other end of the tunnel.
func (s *SimpleServer) Start() {
	// Create Server
	lisServer, err := net.Listen("tcp", s.listenAddress)
	defer lisServer.Close()
	if err != nil {
		log.Fatalf("Main Fail to Listen, %s", err)
	}

	// Tunnel Starts Listening
	lisTunnel, err := net.Listen("tcp", s.tunnelAddress)
	if err != nil {
		log.Fatalf("Fail to Listen, %s", err)
	}

	// Create server for the tunnel
	srv, err := tunnel.NewServer()
	if err != nil {
		log.Fatal("Fail to Create tunnel")
	}

	go func() {
		log.Infof("Main Srv Listening on %s", lisServer.Addr())
		err := srv.Serve(lisTunnel)
		if err != nil {
			log.Fatalf("Error starting main server: %s", err)
		}
	}()

	// Tunnel Set Up. Voltron accepts Tunnel from Guardian (Guardian Dials)

	srvTunnel, err := srv.AcceptTunnel()
	if err != nil {
		log.Fatal("Fail to Establish Tunnel.")
	}

	log.Infof("Tunnel established & server listening on address: %s", lisTunnel.Addr())

	log.Info("Main Server listening for connections")
	for {
		conn, err := lisServer.Accept()
		if err != nil {
			log.Errorf("Error Accepting Connection %s", err.Error())
		}

		rwc, err := srvTunnel.OpenStream()
		if err != nil {
			log.Fatalf("Error opening stream %s", err)
		}

		go utils.SocketCopy(rwc, conn)
	}
}
