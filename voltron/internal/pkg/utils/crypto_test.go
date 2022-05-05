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
		{"../server/testdata/rootCA-tunnel-generation.pem", "../server/testdata/rootCA-tunnel-generation.key"},
	}

	for _, entry := range data {
		var _, _, err = LoadX509Pair(entry.certFile, entry.keyFile)
		g.Expect(err).NotTo(HaveOccurred())
	}
}
