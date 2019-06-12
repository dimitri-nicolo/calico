// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package test provides utilities for writing tests
package test

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

const pubRSA = `
-----BEGIN RSA PUBLIC KEY-----
MIIBCgKCAQEAutvmpHMbizCMfqbA5BmbpkNDZofdMXsXfL+3zas5SIkfaeaIK9BV
A5RPmYSO1wZstdbjdzq+zRw/Ot1SCZz/RcQKFCI3QYllgvcsh/x0RT0eGNYUUHQ1
jCGHPoMjaEeeIXVz7yr2xRnlCbHWvnmgEC8cuMkunSwsY3pZfAmURDMAEN/uA2HK
Y5dKcJ4VJ8XIpd4gyjyT3aRQk+kHvKkoippShRW1jF/j7tF5sjKW4w9bOhY9vC9l
UrfLZqwU/rkCTTBiorFn/de9/l7lt4AGA6KAYBe6aNV7MmKOUy/BDQKstU1B1QNi
c5J88YcvVRHr3lrMldlFqeCd6IHj61K1AQIDAQAB
-----END RSA PUBLIC KEY-----
`

// PrivateRSA is the private key used to sign the certificates
const PrivateRSA = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAutvmpHMbizCMfqbA5BmbpkNDZofdMXsXfL+3zas5SIkfaeaI
K9BVA5RPmYSO1wZstdbjdzq+zRw/Ot1SCZz/RcQKFCI3QYllgvcsh/x0RT0eGNYU
UHQ1jCGHPoMjaEeeIXVz7yr2xRnlCbHWvnmgEC8cuMkunSwsY3pZfAmURDMAEN/u
A2HKY5dKcJ4VJ8XIpd4gyjyT3aRQk+kHvKkoippShRW1jF/j7tF5sjKW4w9bOhY9
vC9lUrfLZqwU/rkCTTBiorFn/de9/l7lt4AGA6KAYBe6aNV7MmKOUy/BDQKstU1B
1QNic5J88YcvVRHr3lrMldlFqeCd6IHj61K1AQIDAQABAoIBAGrrgSIAK3aNpRaj
XCQo8wND4cE9ZLf3cw0Stp2cp/51V+BE5Q4M+1g8+P8i9ojbSEEUYLvMhXjf/N41
3cdaakcFUa8LlQqPD+LMhFKbhfxIaHxVovIWTL2OQdDnQM9ei4Ehr+DeeK13j7Lo
a7Q56/jWvFyP4XhV2mBhlep/oLMUcSO2Nj4KSjscMeg4ED4VPM0N8iQ6eaWNe/+6
aciRdHMmiNSR4SobK2+rskEZxkYKnzQeQT8dblggxN0uNlrhWEhaRFHnLYv41RMm
4ZrMkAMsTex2UUCYt5MUcaJfiafkRt1CbPVDkNKqKiHYTgn3pEIplEOWp7DDlMzT
8kHue0ECgYEA5CR+51SMIC+hY2FfP1oTxnzk+WHHhDTIjlEKNkyuJRUkiFloisi6
zI5001qbP4ufE9gk/AFsbRR4yhLPufgZPwccupAVzFKnsJI7MwpMvtgsJEEfIg5h
oToOE32/oCDGn3AMmwl7ob1pz3C/Jo8QZIpXCVPOAzVx1d1QNHBWjM0CgYEA0azs
ON7yAeAH1gtvtgD+lX294GxUoqa1BwLY2t18Rr0CsTfrKyyHCx+mqj3XPjh+eEUm
tsGt4QWXQOlYANoPx8uIzCvCvga7EEMuA8QRiPsLo03h3UFmHXI6EcQJIK5RMhjC
3KVaG+2LMdvAJLhQWQfz/X7BC4SMc/2zEhlpyQUCgYAwdfQi7VmqiJOOiaNy0I58
zhDRTEzWL2QenuY9bIJdTCVrdRp4yHSteOEl+AwcLmtHCtWoViES9pNF0UMgrKuo
MLmQg4St1yzZm+ZJTDnLHB4cQVz8nfNtDOjqiP6IZA3s1h9HW3dQfuyX7Modxavk
v2IHkC6ljde1ZwJfcTFhTQKBgCgOSPJ0ZPdGvTh+5tB2UCxu4R9GksSf5GV6fcMS
HPPGmAUTEbIlx4awfT54oe4ZDNAdJdA0H+ulDcgwy8cd4XXhxDh9A68ZyhLJQrkl
c9QfYZHJByUloURu1fke4j+EDa7sXA2a6SP8tWLJAGQDchYQFuSOmoKAx/RAuzzx
7euhAoGAZ1yQgvj12oF2bXTUTC64OgpaikmSc5G9xtstA2V4KlxSOu2jS4j4gsov
Vmy/ivvlEE9JkNBLRMxur/WEhE7Udx2JbveDWqe+T5UaG6IdaeE2HNqjPw8fbhE/
Gbs6cLS+CkglnRCvTeWtkqf7SawqfH4eKPu6k6xO1yuL2ylbFp0=
-----END RSA PRIVATE KEY-----
`

func loadKeys() (interface{}, interface{}, error) {
	block, _ := pem.Decode([]byte(pubRSA))
	if block == nil {
		return nil, nil, errors.Errorf("no block in public key")
	}

	pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, nil, errors.Errorf("parsing public failed: %s", err)
	}

	block, _ = pem.Decode([]byte(PrivateRSA))
	if block == nil {
		return nil, nil, errors.Errorf("no block in private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, errors.Errorf("parsing private failed: %s", err)
	}

	return pubKey, privKey, nil
}

func createCert(email string, parent *x509.Certificate, isCA bool) (*x509.Certificate, error) {
	templ := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		EmailAddresses:        []string{email},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: isCA,
		IsCA:                  isCA,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	if isCA {
		templ.KeyUsage |= x509.KeyUsageCertSign
	}

	if parent == nil {
		parent = templ
	}

	pubKey, privKey, err := loadKeys()
	if err != nil {
		return nil, err
	}

	bytes, err := x509.CreateCertificate(rand.Reader, templ, templ, pubKey, privKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(bytes)
}

// CreateSelfSignedX509Cert creates a self-signed certificate using predefined
// keys that includes the given email
func CreateSelfSignedX509Cert(email string, isCA bool) (*x509.Certificate, error) {
	return createCert(email, nil, isCA)
}

// CreateSignedX509Cert creates a cert signed by a parent cert using predefined
// keys that includes the given email
func CreateSignedX509Cert(email string, parent *x509.Certificate) (*x509.Certificate, error) {
	return createCert(email, parent, false)
}

// PemEncodeCert encde a cert as PEM
func PemEncodeCert(cert *x509.Certificate) []byte {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return pem.EncodeToMemory(block)
}
