/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	stdjson "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	auditinternal "k8s.io/apiserver/pkg/apis/audit"
	auditv1alpha1 "k8s.io/apiserver/pkg/apis/audit/v1alpha1"
	"k8s.io/apiserver/pkg/audit"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

// newWebhookHandler returns a handler which recieves webhook events and decodes the
// request body. The caller passes a callback which is called on each webhook POST.
func newWebhookHandler(t *testing.T, cb func(events *auditv1alpha1.EventList)) http.Handler {
	s := json.NewSerializer(json.DefaultMetaFactory, audit.Scheme, audit.Scheme, false)
	return &testWebhookHandler{
		t:          t,
		onEvents:   cb,
		serializer: s,
	}
}

type testWebhookHandler struct {
	t *testing.T

	onEvents func(events *auditv1alpha1.EventList)

	serializer runtime.Serializer
}

func (t *testWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("read webhook request body: %v", err)
		}

		obj, _, err := t.serializer.Decode(body, nil, &auditv1alpha1.EventList{})
		if err != nil {
			return fmt.Errorf("decode request body: %v", err)
		}
		list, ok := obj.(*auditv1alpha1.EventList)
		if !ok {
			return fmt.Errorf("expected *v1alpha1.EventList got %T", obj)
		}
		t.onEvents(list)
		return nil
	}()

	if err == nil {
		io.WriteString(w, "{}")
		return
	}
	// In a goroutine, can't call Fatal.
	assert.NoError(t.t, err, "failed to read request body")
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func newTestBlockingWebhook(t *testing.T, endpoint string) *blockingBackend {
	return newWebhook(t, endpoint, ModeBlocking).(*blockingBackend)
}

func newTestBatchWebhook(t *testing.T, endpoint string) *batchBackend {
	return newWebhook(t, endpoint, ModeBatch).(*batchBackend)
}

func newWebhook(t *testing.T, endpoint string, mode string) audit.Backend {
	config := v1.Config{
		Clusters: []v1.NamedCluster{
			{Cluster: v1.Cluster{Server: endpoint, InsecureSkipTLSVerify: true}},
		},
	}
	f, err := ioutil.TempFile("", "k8s_audit_webhook_test_")
	require.NoError(t, err, "creating temp file")

	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	// NOTE(ericchiang): Do we need to use a proper serializer?
	require.NoError(t, stdjson.NewEncoder(f).Encode(config), "writing kubeconfig")

	backend, err := NewBackend(f.Name(), mode)
	require.NoError(t, err, "initializing backend")

	return backend
}

func TestWebhook(t *testing.T) {
	gotEvents := false
	defer func() { require.True(t, gotEvents, "no events received") }()

	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		gotEvents = true
	}))
	defer s.Close()

	backend := newTestBlockingWebhook(t, s.URL)

	// Ensure this doesn't return a serialization error.
	event := &auditinternal.Event{}
	require.NoError(t, backend.processEvents(event), "failed to send events")
}

// waitForEmptyBuffer indicates when the sendBatchEvents method has read from the
// existing buffer. This lets test coordinate closing a timer and stop channel
// until the for loop has read from the buffer.
func waitForEmptyBuffer(b *batchBackend) {
	for len(b.buffer) != 0 {
		time.Sleep(time.Millisecond)
	}
}

func TestBatchWebhookMaxEvents(t *testing.T) {
	nRest := 10
	events := make([]*auditinternal.Event, defaultBatchMaxSize+nRest) // greater than max size.
	for i := range events {
		events[i] = &auditinternal.Event{}
	}

	got := make(chan int, 2)
	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		got <- len(events.Items)
	}))
	defer s.Close()

	backend := newTestBatchWebhook(t, s.URL)

	backend.ProcessEvents(events...)

	stopCh := make(chan struct{})
	timer := make(chan time.Time, 1)

	backend.sendBatchEvents(stopCh, timer)
	require.Equal(t, defaultBatchMaxSize, <-got, "did not get batch max size")

	go func() {
		waitForEmptyBuffer(backend) // wait for the buffer to empty
		timer <- time.Now()         // Trigger the wait timeout
	}()

	backend.sendBatchEvents(stopCh, timer)
	require.Equal(t, nRest, <-got, "failed to get the rest of the events")
}

func TestBatchWebhookStopCh(t *testing.T) {
	events := make([]*auditinternal.Event, 1) // less than max size.
	for i := range events {
		events[i] = &auditinternal.Event{}
	}

	expected := len(events)
	got := make(chan int, 2)
	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		got <- len(events.Items)
	}))
	defer s.Close()

	backend := newTestBatchWebhook(t, s.URL)
	backend.ProcessEvents(events...)

	stopCh := make(chan struct{})
	timer := make(chan time.Time)

	go func() {
		waitForEmptyBuffer(backend)
		close(stopCh) // stop channel has stopped
	}()
	backend.sendBatchEvents(stopCh, timer)
	require.Equal(t, expected, <-got, "get queued events after timer expires")
}

func TestBatchWebhookEmptyBuffer(t *testing.T) {
	events := make([]*auditinternal.Event, 1) // less than max size.
	for i := range events {
		events[i] = &auditinternal.Event{}
	}

	expected := len(events)
	got := make(chan int, 2)
	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		got <- len(events.Items)
	}))
	defer s.Close()

	backend := newTestBatchWebhook(t, s.URL)

	stopCh := make(chan struct{})
	timer := make(chan time.Time, 1)

	timer <- time.Now() // Timer is done.

	// Buffer is empty, no events have been queued. This should exit but send no events.
	backend.sendBatchEvents(stopCh, timer)

	// Send additional events after the sendBatchEvents has been called.
	backend.ProcessEvents(events...)
	go func() {
		waitForEmptyBuffer(backend)
		timer <- time.Now()
	}()

	backend.sendBatchEvents(stopCh, timer)

	// Make sure we didn't get a POST with zero events.
	require.Equal(t, expected, <-got, "expected one event")
}

func TestBatchBufferFull(t *testing.T) {
	events := make([]*auditinternal.Event, defaultBatchBufferSize+1) // More than buffered size
	for i := range events {
		events[i] = &auditinternal.Event{}
	}
	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		// Do nothing.
	}))
	defer s.Close()

	backend := newTestBatchWebhook(t, s.URL)

	// Make sure this doesn't block.
	backend.ProcessEvents(events...)
}

func TestBatchRun(t *testing.T) {

	// Divisable by max batch size so we don't have to wait for a minute for
	// the test to finish.
	events := make([]*auditinternal.Event, defaultBatchMaxSize*3)
	for i := range events {
		events[i] = &auditinternal.Event{}
	}

	got := new(int64)
	want := len(events)

	wg := new(sync.WaitGroup)
	wg.Add(want)
	done := make(chan struct{})

	go func() {
		wg.Wait()
		// When the expected number of events have been received, close the channel.
		close(done)
	}()

	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		atomic.AddInt64(got, int64(len(events.Items)))
		wg.Add(-len(events.Items))
	}))
	defer s.Close()

	stopCh := make(chan struct{})
	defer close(stopCh)

	backend := newTestBatchWebhook(t, s.URL)

	// Test the Run codepath. E.g. that the spawned goroutines behave correctly.
	backend.Run(stopCh)

	backend.ProcessEvents(events...)

	select {
	case <-done:
		// Received all the events.
	case <-time.After(2 * time.Minute):
		t.Errorf("expected %d events got %d", want, atomic.LoadInt64(got))
	}
}

func TestBatchConcurrentRequests(t *testing.T) {
	events := make([]*auditinternal.Event, defaultBatchBufferSize) // Don't drop events
	for i := range events {
		events[i] = &auditinternal.Event{}
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(events))

	s := httptest.NewServer(newWebhookHandler(t, func(events *auditv1alpha1.EventList) {
		wg.Add(-len(events.Items))

		// Since the webhook makes concurrent requests, blocking on the webhook response
		// shouldn't block the webhook from sending more events.
		//
		// Wait for all responses to be received before sending the response.
		wg.Wait()
	}))
	defer s.Close()

	stopCh := make(chan struct{})
	defer close(stopCh)

	backend := newTestBatchWebhook(t, s.URL)
	backend.Run(stopCh)

	backend.ProcessEvents(events...)
	// Wait for the webhook to receive all events.
	wg.Wait()
}
