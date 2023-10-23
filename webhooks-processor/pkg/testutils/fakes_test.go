// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
	"testing"

	"github.com/projectcalico/calico/webhooks-processor/pkg/specifications"
)

// Test that FakeSecurityEventWebhook complies with the corresponding specification
func TestFakeSecurityEventWebhook(t *testing.T) {
	fakeSecurityEventWebhook := &FakeSecurityEventWebhook{}
	specifications.SecurityEventWebhookInterfaceSpecification(t, fakeSecurityEventWebhook)
}
