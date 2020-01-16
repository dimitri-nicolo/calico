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
		{"../server/testdata/cert.pem", "../server/testdata/key.pem"},
		{"../server/testdata/self-signed-cert.pem", "../server/testdata/self-signed-key.pem"},
	}

	for _, entry := range data {
		var _, _, err = LoadX509Pair(entry.certFile, entry.keyFile)
		g.Expect(err).NotTo(HaveOccurred())
	}
}
