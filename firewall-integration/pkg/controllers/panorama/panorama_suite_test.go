package panorama_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPanorama(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Panorama Suite")
}
