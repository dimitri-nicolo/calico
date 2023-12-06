package managedcluster_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestManagedClusterReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mananged Cluster Reconciler Test Suite")
}
