// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package helpers

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"
)

// ReadCredentials reads from an absolute path a certificate and key as byte arrays
func ReadCredentials(certPath, keyPath string) (cert []byte, key []byte, err error) {
	if len(certPath) == 0 || len(keyPath) == 0 {
		return nil, nil, errors.New("path provided for credentials is empty")
	}

	cert, err = os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}

	key, err = os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// DecodeCertAndKey decodes a x509 certificate and RSA PKCS#1 private from byte arrays
// An error will be returned in case either one cannot be decoded
func DecodeCertAndKey(caCert, key []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	keyPEM, _ := pem.Decode(key)
	if keyPEM == nil || keyPEM.Type != "RSA PRIVATE KEY" {
		return nil, nil, errors.New("provided key does not have PKCS#1 format")
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(keyPEM.Bytes)
	if err != nil {
		return nil, nil, err
	}

	block, _ := pem.Decode(caCert)
	if block == nil {
		return nil, nil, errors.New("provided cert is not in PEM format")
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return x509Cert, rsaKey, nil
}

// Generate generates a x509 client certificate and its private key
func Generate(caCert *x509.Certificate, caPrivateKey crypto.Signer, clusterName string) (*x509.Certificate, crypto.Signer, error) {
	if len(clusterName) == 0 {
		return nil, nil, errors.New("Cluster name cannot be empty")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: clusterName},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(1000000 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	bytes, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(bytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}
