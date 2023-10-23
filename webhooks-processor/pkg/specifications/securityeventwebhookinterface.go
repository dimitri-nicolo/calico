package specifications

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

// The specification specifies behavior that must be true for all clientv3.SecurityEventWebhookInterface.
// The intention is to to run these test in an e2e setup against a real API, and also against its fake
// counterpart in out tests to validate that the fake implementation has the same behavior as the real one.
// Inspired by https://quii.gitbook.io/learn-go-with-tests/testing-fundamentals/scaling-acceptance-tests
func SecurityEventWebhookInterfaceSpecification(t *testing.T, v3Client clientv3.SecurityEventWebhookInterface) {
	t.Run("watcher stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		running := false
		go func() {
			running = true
			watcher, err := v3Client.Watch(ctx, options.ListOptions{})
			require.NoError(t, err)
			// We're just blocking until the channel closes
			for range watcher.ResultChan() {
			}
			running = false
		}()
		require.Eventually(t, func() bool { return running }, time.Second, 10*time.Millisecond)
		cancel()
		require.Eventually(t, func() bool { return !running }, time.Second, 10*time.Millisecond)
	})
}
