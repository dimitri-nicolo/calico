package pip_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"testing"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.InfoLevel)
}

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PIP Suite")
}
