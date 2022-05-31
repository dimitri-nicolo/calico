// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package helpers

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	. "github.com/onsi/gomega"
)

func TestInstallationManifest(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		description      string
		caCert           []byte
		cert             []byte
		key              []byte
		expectedManifest string
	}{
		{
			"valid certificate and client",
			[]byte(CACert),
			[]byte(ClientCert),
			[]byte(ClientKey),
			ExpectedManifest,
		},
	}

	for _, test := range tests {
		t.Log(test.description)

		// Transform ca cert, client cert and key into parameters needed by InstallationManifest
		block, _ := pem.Decode(test.caCert)
		ca, err := x509.ParseCertificate(block.Bytes)
		g.Expect(err).NotTo(HaveOccurred())
		clientCert, clientKey, err := DecodeCertAndKey(test.cert, test.key)
		g.Expect(err).NotTo(HaveOccurred())

		// Invoke InstallationManifest
		manifest := InstallationManifest(ca, clientCert, clientKey, "example.org:1234", "operator-ns")
		g.Expect(manifest).NotTo(BeNil())
		g.Expect(manifest).To(Equal(test.expectedManifest))
	}
}
