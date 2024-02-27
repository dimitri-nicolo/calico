package tls

import (
	"crypto/tls"

	log "github.com/sirupsen/logrus"
)

// Ciphers supported by TLS 1.2
var tls12Ciphers = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
}

// Ciphers supported by TLS 1.3
var tls13Ciphers = []uint16{
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_AES_128_GCM_SHA256,
}

// NewTLSConfig returns a tls.Config with the recommended default settings for Calico components. Based on build flags,
// boringCrypto may be used and fips strict mode may be enforced, which can override the parameters defined in this func.
func NewTLSConfig() *tls.Config {
	log.WithField("BuiltWithBoringCrypto", BuiltWithBoringCrypto).Debug("creating a TLS config")
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: append(tls12Ciphers, tls13Ciphers...),
	}
}
