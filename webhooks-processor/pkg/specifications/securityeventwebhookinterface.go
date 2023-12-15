package specifications

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
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

	t.Run("watcher propagates webhook updates", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		running := false
		var watcher watch.Interface
		var err error
		updatedEvents := []watch.Event{}
		go func() {
			running = true
			watcher, err = v3Client.Watch(ctx, options.ListOptions{})
			require.NoError(t, err)
			// We're just blocking until the channel closes
			for event := range watcher.ResultChan() {
				updatedEvents = append(updatedEvents, event)
			}
			running = false
		}()
		require.Eventually(t, func() bool { return running }, time.Second, 10*time.Millisecond)
		wh := testutils.NewTestWebhook("test")

		// First update notifies us that wh has been added
		_, updateErr := v3Client.Update(ctx, wh, options.SetOptions{})
		require.NoError(t, updateErr)
		require.Eventually(t, func() bool { return len(updatedEvents) > 0 }, time.Second, 10*time.Millisecond)
		require.Equal(t, watch.Added, updatedEvents[0].Type)

		// Second update notifies us that wh has been modified
		wh.Spec.Query = "type = waf"
		_, updateErr = v3Client.Update(ctx, wh, options.SetOptions{})
		require.NoError(t, updateErr)
		require.Eventually(t, func() bool { return len(updatedEvents) > 1 }, time.Second, 10*time.Millisecond)
		require.Equal(t, watch.Modified, updatedEvents[1].Type)

		cancel()
		require.Eventually(t, func() bool { return !running }, time.Second, 10*time.Millisecond)
	})
}
