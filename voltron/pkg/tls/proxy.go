package tls

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/projectcalico/calico/voltron/pkg/conn"

	log "github.com/sirupsen/logrus"
)

// Proxy allows you to proxy https connections with redirection based on the SNI in the client hello
type Proxy interface {
	ListenAndProxy(listener net.Listener) error
}

type proxy struct {
	defaultURL string
	proxyOnSNI bool
	// TODO Consider just having a list of urls and try to match the host of the incoming traffic to the host of a url in
	// TODO in the list.
	sniServiceMap            map[string]string
	retryAttempts            int
	retryInterval            time.Duration
	connectTimeout           time.Duration
	maxConcurrentConnections int
}

const (
	defaultRetryAttempts            = 5
	defaultRetryInterval            = 2 * time.Second
	defaultConnectTimeout           = 30 * time.Second
	defaultMaxConcurrentConnections = 100
)

// NewProxy creates and returns a new Proxy instance
func NewProxy(options ...ProxyOption) (Proxy, error) {
	p := &proxy{
		retryAttempts:            defaultRetryAttempts,
		retryInterval:            defaultRetryInterval,
		connectTimeout:           defaultConnectTimeout,
		maxConcurrentConnections: defaultMaxConcurrentConnections,
	}

	for _, option := range options {
		if err := option(p); err != nil {
			return nil, err
		}
	}

	if p.defaultURL == "" && !p.proxyOnSNI {
		return nil, errors.New("either a default url must be provided or ProxyOnSNI must be enabled")
	}

	if p.proxyOnSNI && len(p.sniServiceMap) == 0 {
		return nil, errors.New("proxyOnSNI has been set but no SNI service map has been provided")
	}

	return p, nil
}

// ListenAndProxy listens for connections on the given listener and proxies the data sent on them. If proxyOnSNI is enabled
// then this proxy will attempt to extract out the SNI from the TLS request and proxy it.
func (p *proxy) ListenAndProxy(listener net.Listener) error {
	// tokenPool is generated here so that it can be closed before we return. This stops us from needing a "Close" function
	// for the proxy, and allows us to use the same proxy multiple times.
	tokenPool := make(chan struct{}, p.maxConcurrentConnections)
	for i := 0; i < p.maxConcurrentConnections; i++ {
		tokenPool <- struct{}{}
	}
	defer close(tokenPool)

	var wg sync.WaitGroup

	err := p.acceptConnections(listener, tokenPool, &wg)

	wg.Wait()

	return err
}

// acceptConnections accepts connections from the given listener. Before accepting a connection a token is taken off the
// the given token pool, and put back into the pool when we're finished with the connection. This ensures we don't go past
// our maximum connection concurrency limit.
// The WaitGroup should be waited on, and Wait will return once all the go routines have finished
func (p *proxy) acceptConnections(listener net.Listener, tokenPool chan struct{}, wg *sync.WaitGroup) error {
	shutDown := make(chan struct{})

	// ensure that we close any connections that are open when returning from this function (this triggers a close of all
	// outstanding connections)
	defer close(shutDown)

	for {
		token := <-tokenPool
		srcConn, err := listener.Accept()
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(conn net.Conn, token struct{}) {
			defer wg.Done()
			defer func() { tokenPool <- token }()

			if err := p.proxyConnectionWithConfirmedShutdown(conn, shutDown); err != nil {
				log.WithError(err).Error("failed to proxy the connection")
				// If an error was returned, then the source connection wasn't closed, so close it
				if err := conn.Close(); err != nil {
					log.WithError(err).Debug("failed to close source connection")
				}
			}
		}(srcConn, token)
	}
}

// proxyConnectionWithConfirmedShutdown makes sure that the connection is closed when the shutDown channel is closed
func (p *proxy) proxyConnectionWithConfirmedShutdown(srcConn net.Conn, shutDown chan struct{}) error {
	done := make(chan struct{})
	defer close(done)

	go func(srcConn net.Conn, done chan struct{}, shutDown chan struct{}) {
		select {
		case <-shutDown:
			if err := srcConn.Close(); err != nil {
				log.WithError(err).Error("failed to close connection after shutdown")
			}
		case <-done:
		}
	}(srcConn, done, shutDown)

	return p.proxyConnection(srcConn)
}

// proxyConnection proxies the data from the given connection to the downstream (downstream is determined by the SNI settings
// / the default URL). If this connection is not a tls connection then it will return an error.
func (p *proxy) proxyConnection(srcConn net.Conn) error {
	url := p.defaultURL
	var bytesRead []byte

	// we try to extract the SNI so that we can verify this is a tls connection
	serverName, bytesRead, err := extractSNI(srcConn)
	if err != nil {
		return err
	}

	if p.proxyOnSNI {
		if serverName != "" {
			if serverNameURL, ok := p.sniServiceMap[serverName]; ok {
				log.Debugf("Extracted SNI '%s' from client hello", serverName)

				url = serverNameURL
			}
		}
	}

	if url == "" {
		return errors.New("couldn't figure out where to send the request")
	}

	dstConn, err := p.dial(url)
	if err != nil {
		log.WithError(err).Errorf("failed to open a connection to %s", url)
		if err := srcConn.Close(); err != nil {
			log.WithError(err).Error("failed to close source connection")
		}
		return nil
	}

	if len(bytesRead) > 0 {
		if err := writeBytesToConn(bytesRead, dstConn); err != nil {
			if err := dstConn.Close(); err != nil {
				log.WithError(err).Debug("failed to close destination connection")
			}

			return err
		}
	}

	conn.Forward(srcConn, dstConn)

	return nil
}

func writeBytesToConn(bytes []byte, conn net.Conn) error {
	bytesWritten := 0
	for bytesWritten < len(bytes) {
		i, err := conn.Write(bytes[bytesWritten:])
		if err != nil {
			return err
		}

		bytesWritten += i
	}

	return nil
}

func (p *proxy) dial(url string) (net.Conn, error) {
	var dstConn net.Conn
	var err error

	// retryAttempts+1 for the initial dial
	for i := 1; i <= p.retryAttempts+1; i++ {
		dstConn, err = net.DialTimeout("tcp", url, p.connectTimeout)
		if err == nil {
			return dstConn, nil
		}

		log.WithError(err).Errorf("failed to open a connection to %s, will retry in %d seconds (attempt %d of %d)", url, p.retryInterval, i, p.retryAttempts+1)
		time.Sleep(p.retryInterval * time.Second)
	}

	return nil, err
}
