package tls

import (
	"errors"
	"time"
)

// ProxyOption is used for setting options for the Proxy.
type ProxyOption func(*proxy) error

// WithDefaultServiceURL sets the default URL that the Proxy will send traffic to if proxy on SNI is disabled, there's no
// SNI found, or the SNI is found but there's no entry for the SNI in the SNI service map (provided by WithSNIServiceMap
// option).
func WithDefaultServiceURL(defaultURL string) ProxyOption {
	return func(p *proxy) error {
		p.defaultURL = defaultURL
		return nil
	}
}

// WithProxyOnSNI sets whether try to route based on the SNI. This must be used in conjunction with WithSNIServiceMap to
// define what server names the proxy accepts and where the requests should be sent to.
func WithProxyOnSNI(proxyOnSNI bool) ProxyOption {
	return func(p *proxy) error {
		p.proxyOnSNI = proxyOnSNI
		return nil
	}
}

// WithSNIServiceMap sets what SNIs this proxy will proxy for and where to send the requests when we receive a tls connection
// with a SNI in this map. This must be used in conjunction with WithProxyOnSNI set to true.
func WithSNIServiceMap(sniServiceMap map[string]string) ProxyOption {
	return func(p *proxy) error {
		for key, value := range sniServiceMap {
			if key == "" || value == "" {
				return errors.New("neither the key nor the value can be empty")
			}
		}

		p.sniServiceMap = sniServiceMap
		return nil
	}
}

// WithConnectionRetryAttempts sets the number of times to retry connecting downstream when it fails to connect.
func WithConnectionRetryAttempts(retryAttempts int) ProxyOption {
	return func(p *proxy) error {
		if retryAttempts < 0 {
			return errors.New("attempts must be non negative")
		}

		p.retryAttempts = retryAttempts
		return nil
	}
}

// WithConnectionRetryInterval sets the interval between retrying to connect to the downstream when the proxy fails to connect.
func WithConnectionRetryInterval(retryInterval time.Duration) ProxyOption {
	return func(p *proxy) error {
		p.retryInterval = retryInterval
		return nil
	}
}

// WithConnectionTimeout sets the duration the proxy will wait to connect to the downstream.
func WithConnectionTimeout(connectTimeout time.Duration) ProxyOption {
	return func(p *proxy) error {
		p.connectTimeout = connectTimeout
		return nil
	}
}

// WithMaxConcurrentConnections sets the number of connections that can be handled concurrently.
// Defaults to 100.
func WithMaxConcurrentConnections(maxConcurrency int) ProxyOption {
	return func(p *proxy) error {
		if maxConcurrency < 1 {
			return errors.New("max concurrency must be greater than or equal to 1")
		}
		p.maxConcurrentConnections = maxConcurrency
		return nil
	}
}

// WithFipsModeEnabled Enables FIPS 140-2 verified crypto mode.
func WithFipsModeEnabled(fipsModeEnabled bool) ProxyOption {
	return func(p *proxy) error {
		p.fipsModeEnabled = fipsModeEnabled
		return nil
	}
}
