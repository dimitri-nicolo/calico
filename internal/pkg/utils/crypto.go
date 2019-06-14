// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package utils has a set of utility function to be used across components
package utils

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"github.com/pkg/errors"
)

// LoadPEMFromFile returns decoded PEM from a file as []byte
func LoadPEMFromFile(f string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errors.Errorf("could not read file %s: %s", f, err)
	}

	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, errors.Errorf("no block in file %s", f)
	}

	return block.Bytes, nil
}

// LoadX509KeyPairFromPEM parse PEM blocks and returns the cert and key (as a
// crypto.Signer)
func LoadX509KeyPairFromPEM(cert []byte, key []byte) (*x509.Certificate, crypto.Signer, error) {
	xCert, err := x509.ParseCertificate(cert)
	if err != nil {
		return nil, nil, errors.Errorf("parsing cert PEM failed: %s", err)
	}

	signer, err := x509.ParsePKCS1PrivateKey(key)
	if err != nil {
		return nil, nil, errors.Errorf("parsing key PEM failed: %s", err)
	}

	return xCert, signer, nil
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
