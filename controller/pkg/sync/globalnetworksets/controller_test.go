// Copyright 2019 Tigera Inc. All rights reserved.

package globalnetworksets

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/tigera/intrusion-detection/controller/pkg/mock"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

func TestNewController(t *testing.T) {
	g := NewWithT(t)

	client := &mock.GlobalNetworkSetInterface{}
	uut := NewController(client)
	g.Expect(uut).ToNot(BeNil())
}

func TestController_Add_Success(t *testing.T) {
	g := NewWithT(t)

	client := &mock.GlobalNetworkSetInterface{W: &mock.Watch{C: make(chan watch.Event)}}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	gns := util.NewGlobalNetworkSet("test")
	fail := func() {}
	stat := &mock.Statser{}
	// Set an error which we expect to clear.
	stat.Error(statser.GlobalNetworkSetSyncFailed, errors.New("test"))
	uut.Add(gns, fail, stat)
	g.Expect(q.Len()).Should(Equal(1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uut.Run(ctx)

	ex := gns.DeepCopy()
	// all created sets are labelled.
	ex.Labels = map[string]string{LabelKey: LabelValue}

	// Wait for queue to be processed
	g.Eventually(q.Len).Should(Equal(0))
	g.Expect(client.Calls()).To(ContainElement(mock.Call{Method: "Create", GNS: ex}))
	g.Expect(stat.Status().ErrorConditions).To(HaveLen(0))

	// The watch will send the GNS back to the informer
	client.W.C <- watch.Event{
		Type:   watch.Added,
		Object: client.GlobalNetworkSet,
	}

	// Expect not to create or update, since the GNS is identical
	g.Consistently(countMethod(client, "Create")).Should(Equal(1))
}

func TestController_Delete(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	gns.Labels = map[string]string{LabelKey: LabelValue}
	client := &mock.GlobalNetworkSetInterface{GlobalNetworkSet: gns}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	uut.NoGC(gns)
	g.Expect(q.Len()).To(Equal(0))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	// Don't GC
	g.Consistently(countMethod(client, "Delete")).Should(Equal(0))

	// Ensure all processing is done before triggering the delete, otherwise we
	// can sometimes get two calls to delete.
	g.Eventually(q.Len).Should(Equal(0))

	uut.Delete(gns)
	g.Eventually(countMethod(client, "Delete")).Should(Equal(1))
	g.Expect(client.Calls()).To(ContainElement(mock.Call{Method: "Delete", Name: gns.Name}))
}

func TestController_Update(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	gns.Labels = map[string]string{LabelKey: LabelValue}
	client := &mock.GlobalNetworkSetInterface{GlobalNetworkSet: gns}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	fail := func() {}
	stat := &mock.Statser{}
	uut.Add(gns, fail, stat)
	g.Expect(q.Len()).Should(Equal(1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uut.Run(ctx)

	// Wait for queue to be processed
	g.Eventually(q.Len).Should(Equal(0))

	// Add the GNS with different data
	gns1 := gns.DeepCopy()
	gns1.Spec.Nets = []string{"192.168.9.45"}
	uut.Add(gns1, fail, stat)

	g.Eventually(countMethod(client, "Update")).Should(Equal(1))
	g.Expect(client.Calls()).To(ContainElement(mock.Call{Method: "Update", GNS: gns1}))

	// Update labels
	gns2 := gns1.DeepCopy()
	gns2.Labels["mock"] = "yes"
	uut.Add(gns2, fail, stat)

	g.Eventually(countMethod(client, "Update")).Should(Equal(2))
	g.Expect(client.Calls()).To(ContainElement(mock.Call{Method: "Update", GNS: gns2}))
}

// Add and then delete a GNS before there is a chance to process it.
func TestController_AddDelete(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	client := &mock.GlobalNetworkSetInterface{}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	fail := func() {}
	stat := &mock.Statser{}
	uut.Add(gns, fail, stat)
	g.Expect(q.Len()).Should(Equal(1))
	uut.Delete(gns)
	g.Expect(q.Len()).Should(Equal(1), "More more on same key should not add to workqueue")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uut.Run(ctx)

	// Wait for queue to be processed
	g.Eventually(q.Len).Should(Equal(0))

	g.Expect(client.Calls()).To(HaveLen(0))
}

func TestController_AddRetry(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	client := &mock.GlobalNetworkSetInterface{CreateError: []error{errors.New("test")}}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	fail := func() {}
	stat := &mock.Statser{}
	uut.Add(gns, fail, stat)
	g.Expect(q.Len()).Should(Equal(1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uut.Run(ctx)

	// Should be retried.
	g.Eventually(countMethod(client, "Create")).Should(Equal(2))
}

func TestController_AddFail(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	//
	client := &mock.GlobalNetworkSetInterface{}
	for i := 0; i < DefaultClientRetries+1; i++ {
		client.CreateError = append(client.CreateError, errors.New("test"))
	}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	var failed bool
	fail := func() { failed = true }
	stat := &mock.Statser{}
	uut.Add(gns, fail, stat)
	g.Expect(q.Len()).Should(Equal(1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uut.Run(ctx)

	// Should be retried.
	g.Eventually(countMethod(client, "Create")).Should(Equal(DefaultClientRetries + 1))
	g.Expect(failed).To(BeTrue())
	g.Expect(stat.Status().ErrorConditions).To(HaveLen(1))
	g.Expect(stat.Status().ErrorConditions[0].Type).To(Equal(statser.GlobalNetworkSetSyncFailed))
}

func TestController_ResourceEventHandlerFuncs(t *testing.T) {
	g := NewWithT(t)

	client := &mock.GlobalNetworkSetInterface{W: &mock.Watch{C: make(chan watch.Event)}}
	uut := NewController(client)

	// Grab a ref to the workqueue, which we'll use to measure progress.
	q := uut.(*controller).queue

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uut.Run(ctx)
	g.Expect(q.Len()).To(Equal(0))

	gns := util.NewGlobalNetworkSet("test")
	client.W.C <- watch.Event{
		Type:   watch.Added,
		Object: gns,
	}

	gnsUp := gns.DeepCopy()
	gnsUp.Spec.Nets = []string{"10.1.10.1"}
	client.W.C <- watch.Event{
		Type:   watch.Modified,
		Object: gns,
	}

	gnsDel := gnsUp.DeepCopy()
	client.W.C <- watch.Event{
		Type:   watch.Deleted,
		Object: gnsDel,
	}

	g.Eventually(q.Len).Should(Equal(0))
}

// Test the code that handles failing to sync. Very little to assert, but making
// sure it doesn't panic or lock.
func TestController_FailToSync(t *testing.T) {
	g := NewWithT(t)

	client := &mock.GlobalNetworkSetInterface{Error: errors.New("test")}
	uut := NewController(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	informer := uut.(*controller).informer

	uut.Run(ctx)
	cancel()
	g.Consistently(informer.HasSynced).Should(BeFalse())
}

// Test the code that handles failing to sync. Very little to assert, but making
// sure it doesn't panic or lock.
func TestController_ShutDown(t *testing.T) {
	g := NewWithT(t)

	client := &mock.GlobalNetworkSetInterface{}
	uut := NewController(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	informer := uut.(*controller).informer
	q := uut.(*controller).queue

	uut.Run(ctx)
	g.Eventually(informer.HasSynced).Should(BeTrue())
	cancel()

	g.Eventually(q.ShuttingDown).Should(BeTrue())
	g.Eventually(q.Len).Should(Equal(0))
}

func TestController_DeleteFailure(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	client := &mock.GlobalNetworkSetInterface{
		GlobalNetworkSet: gns,
		DeleteError:      errors.New("test"),
	}
	uut := NewController(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	g.Eventually(countMethod(client, "Delete")).Should(Equal(DefaultClientRetries + 1))
}

func TestController_UpdateFailure(t *testing.T) {
	g := NewWithT(t)

	gns := util.NewGlobalNetworkSet("test")
	client := &mock.GlobalNetworkSetInterface{
		GlobalNetworkSet: gns,
		UpdateError:      errors.New("test"),
	}
	uut := NewController(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gnsUp := gns.DeepCopy()
	gnsUp.Spec.Nets = []string{"4.5.6.7"}
	fail := func() {}
	stat := &mock.Statser{}
	uut.Add(gnsUp, fail, stat)

	uut.Run(ctx)

	g.Eventually(countMethod(client, "Update")).Should(Equal(DefaultClientRetries + 1))
}

func countMethod(client *mock.GlobalNetworkSetInterface, method string) func() int {
	return func() int {
		n := 0
		for _, c := range client.Calls() {
			if c.Method == method {
				n++
			}
		}
		return n
	}
}
