package managedcluster_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestManagedcluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Managedcluster Suite")
}
