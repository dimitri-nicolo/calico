// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
	"context"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// My typical approach is to use simple solutions while acceptable and reach for proper mocking tools
// once their advantages become more obvious.
// So this approach is likely to evolve as test coverage increases...
type FakeSecurityEventWebhook struct {
	ExpectedWebhook *api.SecurityEventWebhook
	Watcher         *FakeWatcher
}

type FakeWatcher struct {
	Results chan watch.Event
}

func (fw *FakeWatcher) Stop() {
	close(fw.Results)
}
func (fw *FakeWatcher) ResultChan() <-chan watch.Event {
	return fw.Results
}

// Returned event is never used, we may care about simulating an error later
func (w *FakeSecurityEventWebhook) Update(ctx context.Context, res *api.SecurityEventWebhook, opts options.SetOptions) (*api.SecurityEventWebhook, error) {
	return nil, nil
}
func (w *FakeSecurityEventWebhook) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	w.Watcher = &FakeWatcher{
		Results: make(chan watch.Event),
	}

	// Close on context cancellation
	go func() {
		<-ctx.Done()
		w.Watcher.Stop()
	}()
	return w.Watcher, nil
}

// We only care about list and watch, the other ones are only here to please the compiler
func (w *FakeSecurityEventWebhook) Create(ctx context.Context, res *api.SecurityEventWebhook, opts options.SetOptions) (*api.SecurityEventWebhook, error) {
	return nil, nil
}
func (w *FakeSecurityEventWebhook) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*api.SecurityEventWebhook, error) {
	return nil, nil
}
func (w *FakeSecurityEventWebhook) Get(ctx context.Context, name string, opts options.GetOptions) (*api.SecurityEventWebhook, error) {
	return nil, nil
}
func (w *FakeSecurityEventWebhook) List(ctx context.Context, opts options.ListOptions) (*api.SecurityEventWebhookList, error) {
	return nil, nil
}
