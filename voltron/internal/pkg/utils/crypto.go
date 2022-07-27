// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package utils has a set of utility function to be used across components
package utils

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/pkg/errors"
)

// LoadX509Key reads private keys from file and returns the key as a crypto.Signer
func LoadX509Key(keyFile string) (crypto.Signer, error) {
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Could not read key %s", keyFile))
	}

	key, err := ssh.ParseRawPrivateKey(keyPEMBlock)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not parse key")
	}

	return key.(crypto.Signer), nil
}

// LoadX509Cert reads a certificate from file and returns the cert (as a crypto.Signer)
func LoadX509Cert(certFile string) (*x509.Certificate, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Could not read cert %s", certFile))
	}

	block, _ := pem.Decode(certPEMBlock)
	if block == nil {
		return nil, errors.WithMessage(err, "Could not decode cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not parse cert")
	}

	return cert, nil
}

// KeyPEMEncode encodes a crypto.Signer as a PEM block
func KeyPEMEncode(key crypto.Signer) ([]byte, error) {
	var block *pem.Block

	switch k := key.(type) {
	case *rsa.PrivateKey:
		block = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	default:
		return nil, errors.Errorf("unsupported key type")
	}

	return pem.EncodeToMemory(block), nil
}

// CertPEMEncode encodes a x509.Certificate as a PEM block
func CertPEMEncode(cert *x509.Certificate) []byte {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return pem.EncodeToMemory(block)
}

// GenerateFingerprint returns the hash for a x509 certificate printed as a hex number
func GenerateFingerprint(fipsMode bool, certificate *x509.Certificate) string {
	var fingerprint string
	if fipsMode {
		fingerprint = fmt.Sprintf("%x", sha256.Sum256(certificate.Raw))
	}
	fingerprint = fmt.Sprintf("%x", md5.Sum(certificate.Raw))
	log.Infof("Created fingerprint for cert with fipsModeEnabled: %t,  common name: %s and fingerprint: %s", fipsMode, certificate.Subject.CommonName, fingerprint)
	return fingerprint
}
