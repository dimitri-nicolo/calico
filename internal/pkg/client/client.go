package client

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/targets"
	"github.com/tigera/voltron/pkg/tunnel"
)

// Client is the voltron client. It accepts requests from voltron server
// and redirects it to different parts in the cluster.
type Client struct {
	http      *http.Server
	proxyMux  *http.ServeMux
	targets   *targets.Targets
	tunnel    *tunnel.Tunnel
	closeOnce sync.Once

	tunnelAddr    string
	tunnelCertPEM []byte
	tunnelKeyPEM  []byte
	tunnelRootCAs *x509.CertPool

	tunnelReady chan error
}

// New returns a new Client
func New(addr string, opts ...Option) (*Client, error) {
	client := &Client{
		http:        new(http.Server),
		targets:     targets.NewEmpty(),
		tunnelReady: make(chan error, 1),
	}

	client.tunnelAddr = addr
	log.Infof("Tunnel Address: %v", client.tunnelAddr)

	for _, o := range opts {
		if err := o(client); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	log.Infof("Targets are: %v", client.targets)

	client.proxyMux = http.NewServeMux()
	client.http.Handler = client.proxyMux

	handler := proxy.New(proxy.NewPathMatcher(client.targets))
	client.proxyMux.Handle("/", handler)

	return client, nil
}

// WaitForTunnel waits until a tunnel is established or until an error happens
func (c *Client) WaitForTunnel() error {
	return <-c.tunnelReady
}

// ServeTunnelHTTP starts serving HTTP requests through a tunnel
func (c *Client) ServeTunnelHTTP() error {
	var lis net.Listener

	log.Infof("Dialing tunnel to %s ...", c.tunnelAddr)
	err := func() error {
		var err error

		if c.tunnelCertPEM == nil || c.tunnelKeyPEM == nil {
			log.Warnf("no tunnel creds, using unsecured tunnel")
			c.tunnel, err = tunnel.Dial(c.tunnelAddr)
			if err != nil {
				return err
			}

			lis = c.tunnel
		} else {
			cert, err := tls.X509KeyPair(c.tunnelCertPEM, c.tunnelKeyPEM)
			if err != nil {
				return errors.Errorf("tls.X509KeyPair: %s", err)
			}

			c.tunnel, err = tunnel.DialTLS(
				c.tunnelAddr,
				&tls.Config{
					Certificates: []tls.Certificate{cert},
					RootCAs:      c.tunnelRootCAs,
					ServerName:   "voltron",
				},
			)
			if err != nil {
				return err
			}

			// we need to upgrade the tunnel to a TLS listener to support HTTP2
			// on this side.
			lis = tls.NewListener(c.tunnel, &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2"},
			})
			log.Infof("serving HTTP/2 enabled")
		}

		return nil
	}()

	c.tunnelReady <- err
	close(c.tunnelReady)
	if err != nil {
		log.Errorf("Failed to dial tunnel: %s", err)
		return err
	}

	log.Infof("Tunnel established, starting to server tunneled HTTP")
	return c.http.Serve(lis)
}

// Close stops the server.
func (c *Client) Close() error {
	var retErr error

	c.closeOnce.Do(func() {
		if c.tunnel != nil {
			if err := c.tunnel.Close(); err != nil {
				retErr = err
			}
		}
		if err := c.http.Close(); err != nil && retErr == nil {
			retErr = err
		}
	})

	return retErr
}
