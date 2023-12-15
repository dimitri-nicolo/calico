package specifications

import (
	"testing"

	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
)

// Test that FakeSecurityEventWebhook complies with the corresponding specification
func TestFakeSecurityEventWebhook(t *testing.T) {
	fakeSecurityEventWebhook := &testutils.FakeSecurityEventWebhook{}
	SecurityEventWebhookInterfaceSpecification(t, fakeSecurityEventWebhook)
}
