/*
Copyright 2019-2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateCAKeyPair(cn string, altNames []string) ([]byte, []byte, error) {
	// Create a x509 template for the CreateCAKeyPair
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName: cn,
		},
		DNSNames:              altNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	// Generate a private key
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	caBytes, err := signAndEncodeCert(template, key, template, key)
	if err != nil {
		return nil, nil, err
	}

	caKeyBytes, err := encodeKey(key)
	if err != nil {
		return nil, nil, err
	}

	return caBytes, caKeyBytes, nil
}

func signAndEncodeCert(ca *x509.Certificate, caPrivateKey *rsa.PrivateKey, cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
	// Sign the certificate with the provided CA
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	// Encode the certificate
	certPEM := bytes.Buffer{}
	err = pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return nil, err
	}

	return certPEM.Bytes(), nil
}

func encodeKey(key *rsa.PrivateKey) ([]byte, error) {
	// Encode the private key
	keyPEM := bytes.Buffer{}
	privateBytes := x509.MarshalPKCS1PrivateKey(key)
	err := pem.Encode(&keyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateBytes})
	if err != nil {
		return nil, err
	}

	return keyPEM.Bytes(), nil
}

func ToSecret(name, namespace string, certPem, keyPem []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPem,
			corev1.TLSPrivateKeyKey: keyPem,
		},
	}
}
