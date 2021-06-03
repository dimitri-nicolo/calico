package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Target defines a destination URL to which HTTP requests will be proxied. The path prefix dictates
// which requests will be proxied to this particular target.
type Target struct {
	PathPrefix string
	Dest       *url.URL

	// Provides the CA cert to use for TLS verification.
	CAPem string

	// Transport to use for this target. If nil, a transport will be provided. This is useful for testing.
	Transport http.RoundTripper

	// Allow TLS without the verify step. This is useful for testing.
	AllowInsecureTLS bool
}

// CreateTarget returns a Target instance based on the provided parameter values.
func CreateTarget(pathPrefix, dest, caBundlePath string, allowInsecureTLS bool) (*Target, error) {
	target := &Target{
		PathPrefix:       pathPrefix,
		AllowInsecureTLS: allowInsecureTLS,
	}

	if pathPrefix == "" {
		return nil, errors.New("proxy target path cannot be empty")
	}

	var err error
	target.Dest, err = url.Parse(dest)
	if err != nil {
		return nil, errors.Errorf("Incorrect URL %q for path %q: %s", dest, pathPrefix, err)
	}

	if target.Dest.Scheme == "https" && !allowInsecureTLS && caBundlePath == "" {
		return nil, errors.Errorf("target for path '%s' must specify the ca bundle if AllowInsecureTLS is false when the scheme is https", pathPrefix)
	}

	if caBundlePath != "" {
		target.CAPem = caBundlePath
	}

	return target, nil
}

// GetProxyHandler generates an HTTP proxy handler based on the given Target.
func (t *Target) GetProxyHandler() (func(http.ResponseWriter, *http.Request), error) {
	p := httputil.NewSingleHostReverseProxy(t.Dest)
	p.FlushInterval = -1

	if t.Transport != nil {
		p.Transport = t.Transport
	} else if t.Dest.Scheme == "https" {
		var tlsCfg *tls.Config

		if t.AllowInsecureTLS {
			tlsCfg = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			if len(t.CAPem) == 0 {
				return nil, errors.Errorf("failed to create target handler for path %s: ca bundle was empty", t.PathPrefix)
			}

			log.Debugf("Detected secure transport for %s. Will pick up system cert pool", t.Dest)
			var ca *x509.CertPool
			ca, err := x509.SystemCertPool()
			if err != nil {
				log.WithError(err).Warn("failed to get system cert pool, creating a new one")
				ca = x509.NewCertPool()
			}

			file, err := ioutil.ReadFile(t.CAPem)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not read cert from file %s", t.CAPem))
			}

			ca.AppendCertsFromPEM(file)
			tlsCfg = &tls.Config{
				RootCAs: ca,
			}
		}

		p.Transport = &http.Transport{
			TLSClientConfig: tlsCfg,
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Received request %s will proxy to %s", r.RequestURI, t.Dest)
		p.ServeHTTP(w, r)
	}, nil
}
