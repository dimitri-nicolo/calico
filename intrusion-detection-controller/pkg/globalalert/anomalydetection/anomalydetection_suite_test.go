package anomalydetection_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAnomalydetection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Anomalydetection Suite")
}
