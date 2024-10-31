package managedcluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManagedClusterReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mananged Cluster Reconciler Test Suite")
}
