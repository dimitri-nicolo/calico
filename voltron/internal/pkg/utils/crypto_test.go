package utils

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestLoadingCertificates(t *testing.T) {
	g := NewGomegaWithT(t)

	data := []struct {
		certFile string
		keyFile  string
	}{
		{"../server/testdata/cert-pkcs8-format.pem", "../server/testdata/key-pkcs8-format.pem"},
	}

	for _, entry := range data {
		_, err := LoadX509Key(entry.keyFile)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = LoadX509Cert(entry.certFile)
		g.Expect(err).NotTo(HaveOccurred())
	}
}
