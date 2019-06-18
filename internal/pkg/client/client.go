package client

import (
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/targets"
	"github.com/tigera/voltron/pkg/tunnel"
)

// Client is the voltron client. It accepts requests from voltron server
// and redirects it to different parts in the cluster.
type Client struct {
	http     *http.Server
	proxyMux *http.ServeMux
	targets  *targets.Targets
	tunnel   *tunnel.Tunnel

	tunnelAddr string
	certFile   string
	keyFile    string
}

// New returns a new Client
func New(addr string, opts ...Option) (*Client, error) {
	client := &Client{
		http:    new(http.Server),
		targets: targets.NewEmpty(),
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

// ListenAndServeTLS starts listening and serving HTTPS requests
func (c *Client) ListenAndServeTLS() error {
	log.Infof("URL is: %s", c.http.Addr)
	return http.ListenAndServeTLS(c.http.Addr, c.certFile, c.keyFile, nil)
}

// ServeHTTP starts serving HTTP requests
func (c *Client) ServeHTTP() error {
	var err error
	c.tunnel, err = tunnel.Dial(c.tunnelAddr)
	if err != nil {
		return err
	}

	return c.http.Serve(c.tunnel)
}

// Close stops the server.
func (c *Client) Close() error {
	if err := c.tunnel.Close(); err != nil {
		return err
	}

	return c.http.Close()
}
