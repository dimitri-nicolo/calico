package tls

import (
	"crypto/tls"
)

// NewTLSConfig returns a tls.Config with the recommended default settings for Calico Enterprise components.
// Read more recommendations here in Chapter 3:
// https://www.gsa.gov/cdnstatic/SSL_TLS_Implementation_%5BCIO_IT_Security_14-69_Rev_6%5D_04-06-2021docx.pdf
func NewTLSConfig(fipsMode bool) *tls.Config {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	if fipsMode {
		cfg.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		}
		cfg.CurvePreferences = []tls.CurveID{tls.CurveP384, tls.CurveP256}
		cfg.MinVersion = tls.VersionTLS12
		// Our certificate for FIPS validation not mention validation for v1.3.
		cfg.MaxVersion = tls.VersionTLS12
		cfg.Renegotiation = tls.RenegotiateNever
	}
	return cfg
}
