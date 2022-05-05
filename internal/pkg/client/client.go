package client

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/tigera/voltron/pkg/conn"

	"github.com/tigera/voltron/pkg/tunnelmgr"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/pkg/tunnel"
)

// Client is the voltron client. It accepts requests from voltron server
// and redirects it to different parts in the cluster.
type Client struct {
	http      *http.Server
	proxyMux  *http.ServeMux
	targets   []proxy.Target
	closeOnce sync.Once

	tunnelAddr    string
	tunnelCert    *tls.Certificate
	tunnelRootCAs *x509.CertPool

	tunnelEnableKeepAlive   bool
	tunnelKeepAliveInterval time.Duration

	tunnelManager tunnelmgr.Manager
	tunnelDialer  tunnel.Dialer

	tunnelDialRetryAttempts int
	tunnelDialTimeout       time.Duration
	tunnelDialRetryInterval time.Duration

	connRetryAttempts int
	connRetryInterval time.Duration
}

// New returns a new Client
func New(addr string, opts ...Option) (*Client, error) {
	var err error
	client := &Client{
		http:                    new(http.Server),
		tunnelEnableKeepAlive:   true,
		tunnelKeepAliveInterval: 100 * time.Millisecond,

		tunnelDialRetryAttempts: 5,
		tunnelDialRetryInterval: 2 * time.Second,
		tunnelDialTimeout:       60 * time.Second,

		connRetryAttempts: 5,
		connRetryInterval: 2 * time.Second,
	}

	client.tunnelAddr = addr
	log.Infof("Tunnel Address: %s", client.tunnelAddr)

	for _, o := range opts {
		if err := o(client); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	// set the dialer for the tunnel manager if one hasn't been specified
	tunnelAddress := client.tunnelAddr
	tunnelKeepAlive := client.tunnelEnableKeepAlive
	tunnelKeepAliveInterval := client.tunnelKeepAliveInterval
	if client.tunnelDialer == nil {
		var dialerFunc tunnel.DialerFunc
		if client.tunnelCert == nil {
			log.Warnf("No tunnel creds, using unsecured tunnel")
			dialerFunc = func() (*tunnel.Tunnel, error) {
				return tunnel.Dial(
					tunnelAddress,
					tunnel.WithKeepAliveSettings(tunnelKeepAlive, tunnelKeepAliveInterval),
				)
			}
		} else {
			tunnelCert := client.tunnelCert
			tunnelRootCAs := client.tunnelRootCAs
			dialerFunc = func() (*tunnel.Tunnel, error) {
				log.Debug("Dialing tunnel...")
				return tunnel.DialTLS(
					tunnelAddress,
					&tls.Config{
						Certificates: []tls.Certificate{*tunnelCert},
						RootCAs:      tunnelRootCAs,
						ServerName:   "voltron",
					},
					client.tunnelDialTimeout,
					tunnel.WithKeepAliveSettings(tunnelKeepAlive, tunnelKeepAliveInterval),
				)
			}
		}
		client.tunnelDialer = tunnel.NewDialer(
			dialerFunc,
			client.tunnelDialRetryAttempts,
			client.tunnelDialRetryInterval,
			client.tunnelDialTimeout,
		)
	}

	client.tunnelManager = tunnelmgr.NewManagerWithDialer(client.tunnelDialer)

	for _, target := range client.targets {
		log.Infof("Will route traffic to %s for requests matching %s", target.Dest, target.Path)
	}

	client.proxyMux = http.NewServeMux()
	client.http.Handler = client.proxyMux

	handler, err := proxy.New(client.targets)
	if err != nil {
		return nil, errors.WithMessage(err, "proxy.New")
	}
	client.proxyMux.Handle("/", handler)

	return client, nil
}

// ServeTunnelHTTP starts serving HTTP requests through the tunnel
func (c *Client) ServeTunnelHTTP() error {
	log.Debug("Getting listener for tunnel.")

	var listener net.Listener
	var err error

	for i := 1; i <= c.connRetryAttempts; i++ {
		listener, err = c.tunnelManager.Listener()
		if err == nil || err != tunnelmgr.ErrStillDialing {
			break
		}

		time.Sleep(c.connRetryInterval)
	}

	if err != nil {
		return err
	}

	if c.tunnelCert != nil {
		// we need to upgrade the tunnel to a TLS listener to support HTTP2
		// on this side.
		listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{*c.tunnelCert},
			NextProtos:   []string{"h2"},
		})
		log.Infof("serving HTTP/2 enabled")
	}

	log.Infof("starting to server tunneled HTTP")
	return c.http.Serve(listener)
}

// AcceptAndProxy accepts connections on the given listener and sends them down the tunnel
func (c *Client) AcceptAndProxy(listener net.Listener) error {
	defer listener.Close()

	for {
		srcConn, err := listener.Accept()
		if err != nil {
			return err
		}

		var dstConn net.Conn

		for i := 1; i <= c.connRetryAttempts; i++ {
			dstConn, err = c.tunnelManager.Open()
			if err == nil || err != tunnelmgr.ErrStillDialing {
				break
			}

			time.Sleep(c.connRetryInterval)
		}

		if err != nil {
			if err := srcConn.Close(); err != nil {
				log.WithError(err).Error("failed to close source connection")
			}

			log.WithError(err).Error("failed to open connection to the tunnel")
			return err
		}

		//TODO I think we want to throttle the connections
		go conn.Forward(srcConn, dstConn)
	}
}

// Close stops the server.
func (c *Client) Close() error {
	var retErr error

	c.closeOnce.Do(func() {
		if c.tunnelManager != nil {
			if err := c.tunnelManager.Close(); err != nil {
				retErr = err
			}
		}
		if err := c.http.Close(); err != nil && retErr == nil {
			retErr = err
		}
	})

	return retErr
}
