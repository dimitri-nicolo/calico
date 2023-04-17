package managedcluster_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestManagedcluster(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/managedcluster_suite.xml") // match this to the correct path of the package
	RunSpecsWithDefaultAndCustomReporters(t, "ManagedClusterController Suite", []Reporter{junitReporter})
}
