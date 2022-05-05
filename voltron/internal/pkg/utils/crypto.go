// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package utils has a set of utility function to be used across components
package utils

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"

	"github.com/pkg/errors"
)

// LoadX509Pair reads certificates and private keys from file and returns the cert and key (as a
// crypto.Signer)
func LoadX509Pair(certFile, keyFile string) (*x509.Certificate, crypto.Signer, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, nil, errors.WithMessage(err, fmt.Sprintf("Could not read cert %s", certFile))
	}
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, nil, errors.WithMessage(err, fmt.Sprintf("Could not read key %s", keyFile))
	}

	key, err := ssh.ParseRawPrivateKey(keyPEMBlock)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "Could not parse key")
	}

	block, _ := pem.Decode(certPEMBlock)
	if block == nil {
		return nil, nil, errors.WithMessage(err, "Could not decode cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "Could not parse cert")
	}

	return cert, key.(crypto.Signer), nil

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

// GenerateFingerprint returns the MD5 hash for a x509 certificate printed as a hex number
func GenerateFingerprint(certificate *x509.Certificate) string {
	return fmt.Sprintf("%x", md5.Sum(certificate.Raw))
}
