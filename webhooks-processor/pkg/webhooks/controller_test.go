package webhooks

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	calicoWatch "github.com/projectcalico/calico/libcalico-go/lib/watch"
)

func cwEvent(t calicoWatch.EventType, o *api.SecurityEventWebhook) calicoWatch.Event {
	return calicoWatch.Event{
		Type:     t,
		Object:   o,
		Previous: o,
	}
}

func wEvent(t watch.EventType, o runtime.Object) watch.Event {
	return watch.Event{
		Type:   t,
		Object: o,
	}
}

func TestControllerWithMocks(t *testing.T) {
	testOrBenchControllerWithMocks(t)
}

func BenchmarkControllerWithMocks(b *testing.B) {
	testOrBenchControllerWithMocks(b)
}

func testOrBenchControllerWithMocks(t testing.TB) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testSecurityEvent := api.NewSecurityEventWebhook()
	testRuntimeObject := &corev1.ConfigMap{}
	testState := &mockStateInterface{}
	testUpdater := &mockWebhookUpdaterInterface{}

	// ensure all state interface functions are called
	testState.On("IncomingWebhookUpdate", mock.Anything, testSecurityEvent)
	testState.On("CheckDependencies", testRuntimeObject)
	stopCall := testState.On("Stop", mock.Anything, testSecurityEvent)
	testState.On("StopAll").NotBefore(stopCall)

	runController(
		ctx, t,
		testState,
		testUpdater,
		testRuntimeObject,
		testSecurityEvent,
	)

	cancel()
	<-ctx.Done()
	// this is somehow needed to observe StopAll()
	time.Sleep(time.Millisecond)
	testState.AssertExpectations(t)
}

func runController(ctx context.Context, _ testing.TB, state StateInterface, updater WebhookUpdaterInterface, testRuntimeObject runtime.Object, testSecurityEvent *api.SecurityEventWebhook) {
	var wg sync.WaitGroup
	(&wg).Add(1)

	ct := NewWebhookController().
		WithState(state).
		WithUpdater(updater)

	if v, ok := updater.(*WebhookWatcherUpdater); ok {
		v.WithController(ct)
	}

	go ct.Run(ctx, &wg)

	//send events
	// handled events
	ct.WebhookEventsChan() <- cwEvent(calicoWatch.Added, testSecurityEvent)
	ct.WebhookEventsChan() <- cwEvent(calicoWatch.Modified, testSecurityEvent)
	ct.WebhookEventsChan() <- cwEvent(calicoWatch.Deleted, testSecurityEvent)

	// 'unhandled' events
	ct.WebhookEventsChan() <- cwEvent(calicoWatch.Error, testSecurityEvent)

	// handled events
	ct.K8sEventsChan() <- wEvent(watch.Modified, testRuntimeObject)
	ct.K8sEventsChan() <- wEvent(watch.Deleted, testRuntimeObject)

	// 'unhandled' events
	ct.K8sEventsChan() <- wEvent(watch.Added, testRuntimeObject)
	ct.K8sEventsChan() <- wEvent(watch.Error, testRuntimeObject)
}

type mockWebhookUpdaterInterface struct{}

func (*mockWebhookUpdaterInterface) UpdatesChan() chan<- *api.SecurityEventWebhook {
	return make(chan<- *api.SecurityEventWebhook)
}

type mockStateInterface struct {
	mock.Mock
}

func (s *mockStateInterface) OutgoingWebhookUpdates() <-chan *api.SecurityEventWebhook {
	return make(<-chan *api.SecurityEventWebhook)
}
func (s *mockStateInterface) IncomingWebhookUpdate(ctx context.Context, evt *api.SecurityEventWebhook) {
	s.Called(ctx, evt)
}
func (s *mockStateInterface) CheckDependencies(o runtime.Object) {
	s.Called(o)
}
func (s *mockStateInterface) Stop(ctx context.Context, evt *api.SecurityEventWebhook) {
	s.Called(ctx, evt)
}
func (s *mockStateInterface) StopAll() {
	s.Called()
}

var (
	_ WebhookUpdaterInterface = (*mockWebhookUpdaterInterface)(nil)
	_ StateInterface          = (*mockStateInterface)(nil)
)
