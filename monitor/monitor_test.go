package monitor_test

import (
	"testing"

	. "github.com/onsi/gomega"

	lclient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/monitor"
)

func init() {}

func TestBasicFunction(t *testing.T) {

	t.Run("With no datastore connection", func(t *testing.T) {
		RegisterTestingT(t)

		// Create a monitor, passing in a nil client.
		m := monitor.New(nil)

		// Expect that feature status is returned as false.
		status := m.GetFeatureStatus("foo")
		Expect(status).To(BeFalse())

		// Expect the license to be listed as not loaded.
		Expect(m.GetLicenseStatus()).To(Equal(lclient.NoLicenseLoaded))
	})
}
