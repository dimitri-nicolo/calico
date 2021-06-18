package gateway

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/tigera/es-gateway/pkg/clients/elastic"
	"github.com/tigera/es-gateway/pkg/middlewares"
	"github.com/tigera/es-gateway/pkg/proxy"
)

// GetProxyHandler generates an HTTP proxy handler based on the given Target.
func GetProxyHandler(t *proxy.Target) (func(http.ResponseWriter, *http.Request), error) {
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
				return nil, errors.Errorf("failed to create target handler for path %s: ca bundle was empty", t.Dest)
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

		// Use the modify response hook function to log the return value for response.
		// This is useful for troubleshooting and debugging.
		p.ModifyResponse = func(res *http.Response) error {
			log.Debugf("Response to request %s (proxied to %s): [HTTP %s]", res.Request.URL, t.Dest, res.Status)
			return nil
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(middlewares.ESUserKey)
		// User could be nil if this is a path that does not require authentication.
		if user != nil {
			user, ok := r.Context().Value(middlewares.ESUserKey).(*elastic.User)
			// This should never happen (logical bug somewhere else in the code). But we'll
			// leave this check here to help catch it.
			if !ok {
				log.Error("unable to authenticate user: ES user cannot be pulled from context (this is a logical bug)")
				http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
				return
			}
			log.Debugf("Received request %s from %s (with user %s), will proxy to %s", r.RequestURI, fmt.Sprintf("%s (%s)", r.RemoteAddr, r.Header), user.Username, t.Dest)
		}

		p.ServeHTTP(w, r)
	}, nil
}
