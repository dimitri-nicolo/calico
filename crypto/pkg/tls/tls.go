package tls

import (
	"crypto/tls"

	log "github.com/sirupsen/logrus"
)

// NewTLSConfig returns a tls.Config with the recommended default settings for Calico Enterprise components.
// Read more recommendations here in Chapter 3:
// https://www.gsa.gov/cdnstatic/SSL_TLS_Implementation_%5BCIO_IT_Security_14-69_Rev_6%5D_04-06-2021docx.pdf
func NewTLSConfig(_ bool) *tls.Config {
	log.WithField("BuiltWithBoringCrypto", BuiltWithBoringCrypto).Info("creating a TLS config")
	// When we build with GOEXPERIMENT and tag boringcrypto, the tls settings in the config will automatically
	// be overwritten and set to strict mode, due to the fipsonly import in fipstls.go.
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}
}
