/*
Copyright 2016 Tigera Inc.
*/

package calico

import (
	"reflect"
	"testing"
	"time"

	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/apis/example"
	"k8s.io/apiserver/pkg/storage"
)

func TestWatch(t *testing.T) {
	testWatch(t, false)
}

func TestWatchList(t *testing.T) {
	testWatch(t, true)
}

// It tests that
// - first occurrence of objects should notify Add event
// - update should trigger Modified event
// - update that gets filtered should trigger Deleted event
func testWatch(t *testing.T, recursive bool) {
	ctx, store := testSetup(t)
	defer func() {
		testCleanup(t, ctx, store)
		store.client.NetworkPolicies().Delete(ctx, "default", "foo", options.DeleteOptions{})
		store.client.NetworkPolicies().Delete(ctx, "default", "bar", options.DeleteOptions{})
		store.client.NetworkPolicies().Delete(ctx, "default", "foo1", options.DeleteOptions{})
		store.client.NetworkPolicies().Delete(ctx, "default", "foo2", options.DeleteOptions{})
	}()

	policyFoo := &calico.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "foo"}}
	policyBar := &calico.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bar"}}
	policyFoo1 := &calico.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "foo1"}}
	policyFoo2 := &calico.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "foo2"}}

	tests := []struct {
		pred       storage.SelectionPredicate
		watchTests []*testWatchStruct
	}{{ // create a key
		watchTests: []*testWatchStruct{
			{
				key:         "projectcalico.org/networkpolicies/default/foo",
				obj:         policyFoo,
				expectEvent: true,
				watchType:   watch.Added,
			},
		},
		pred: storage.Everything,
	}, { // create a key but obj gets filtered. Then update it with unfiltered obj
		watchTests: []*testWatchStruct{
			{
				key:         "projectcalico.org/networkpolicies/default/foo",
				obj:         policyFoo,
				expectEvent: false,
				watchType:   "",
			},
			{
				key:         "projectcalico.org/networkpolicies/default/bar",
				obj:         policyBar,
				expectEvent: true,
				watchType:   watch.Added,
			},
		},
		pred: storage.SelectionPredicate{
			Label: labels.Everything(),
			Field: fields.ParseSelectorOrDie("metadata.name=bar"),
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
				policy := obj.(*calico.NetworkPolicy)
				return nil, fields.Set{"metadata.name": policy.Name}, policy.Initializers != nil, nil
			},
		},
	}, { // update
		watchTests: []*testWatchStruct{
			{
				key:         "projectcalico.org/networkpolicies/default/foo1",
				obj:         policyFoo1,
				expectEvent: true,
				watchType:   watch.Added,
			},
			{
				key:         "projectcalico.org/networkpolicies/default/bar",
				obj:         policyBar,
				expectEvent: true,
				watchType:   watch.Modified,
			},
		},
		pred: storage.Everything,
	}, { // delete because of being filtered
		watchTests: []*testWatchStruct{
			{
				key:         "projectcalico.org/networkpolicies/default/foo2",
				obj:         policyFoo2,
				expectEvent: true,
				watchType:   watch.Added,
			},
			{
				key:         "projectcalico.org/networkpolicies/default/bar",
				obj:         policyBar,
				expectEvent: true,
				watchType:   watch.Modified,
			},
		},
		pred: storage.SelectionPredicate{
			Label: labels.Everything(),
			Field: fields.ParseSelectorOrDie("metadata.name!=bar"),
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
				policy := obj.(*calico.NetworkPolicy)
				return nil, fields.Set{"metadata.name": policy.Name}, policy.Initializers != nil, nil
			},
		},
	}}
	for i, tt := range tests {
		var prevObj *calico.NetworkPolicy
		for _, watchTest := range tt.watchTests {
			w, err := store.Watch(ctx, watchTest.key, "0", tt.pred)
			if err != nil {
				t.Fatalf("Watch failed: %v", err)
			}
			out := &calico.NetworkPolicy{}
			key := watchTest.key
			err = store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
				func(runtime.Object) (runtime.Object, error) {
					return watchTest.obj, nil
				}))
			if err != nil {
				t.Fatalf("GuaranteedUpdate failed: %v", err)
			}
			if watchTest.expectEvent {
				expectObj := out
				if watchTest.watchType == watch.Deleted {
					expectObj = prevObj
					expectObj.ResourceVersion = out.ResourceVersion
				}
				testCheckResult(t, i, watchTest.watchType, w, expectObj)
			}
			prevObj = out
			w.Stop()
			testCheckStop(t, i, w)
		}
	}
}

/*
func TestDeleteTriggerWatch(t *testing.T) {
	ctx, store, cluster := testSetup(t)
	defer cluster.Terminate(t)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	w, err := store.Watch(ctx, key, storedObj.ResourceVersion, storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	if err := store.Delete(ctx, key, &example.Pod{}, nil); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	testCheckEventType(t, watch.Deleted, w)
}

// TestWatchFromZero tests that
// - watch from 0 should sync up and grab the object added before
// - watch from 0 is able to return events for objects whose previous version has been compacted
func TestWatchFromZero(t *testing.T) {
	ctx, store, cluster := testSetup(t)
	defer cluster.Terminate(t)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns"}})

	w, err := store.Watch(ctx, key, "0", storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, 0, watch.Added, w, storedObj)
	w.Stop()

	// Update
	out := &example.Pod{}
	err = store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns", Annotations: map[string]string{"a": "1"}}}, nil
		}))
	if err != nil {
		t.Fatalf("GuaranteedUpdate failed: %v", err)
	}

	// Make sure when we watch from 0 we receive an ADDED event
	w, err = store.Watch(ctx, key, "0", storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, 1, watch.Added, w, out)
	w.Stop()

	// Update again
	out = &example.Pod{}
	err = store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns"}}, nil
		}))
	if err != nil {
		t.Fatalf("GuaranteedUpdate failed: %v", err)
	}

	// Compact previous versions
	revToCompact, err := strconv.Atoi(out.ResourceVersion)
	if err != nil {
		t.Fatalf("Error converting %q to an int: %v", storedObj.ResourceVersion, err)
	}
	_, err = cluster.RandClient().Compact(ctx, int64(revToCompact), clientv3.WithCompactPhysical())
	if err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Make sure we can still watch from 0 and receive an ADDED event
	w, err = store.Watch(ctx, key, "0", storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, 2, watch.Added, w, out)
}

// TestWatchFromNoneZero tests that
// - watch from non-0 should just watch changes after given version
func TestWatchFromNoneZero(t *testing.T) {
	ctx, store, cluster := testSetup(t)
	defer cluster.Terminate(t)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	w, err := store.Watch(ctx, key, storedObj.ResourceVersion, storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	out := &example.Pod{}
	store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}, err
		}))
	testCheckResult(t, 0, watch.Modified, w, out)
}

func TestWatchError(t *testing.T) {
	codec := &testCodec{apitesting.TestCodec(codecs, examplev1.SchemeGroupVersion)}
	cluster := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 1})
	defer cluster.Terminate(t)
	invalidStore := newStore(cluster.RandClient(), false, codec, "", prefixTransformer{prefix: []byte("test!")})
	ctx := context.Background()
	w, err := invalidStore.Watch(ctx, "/abc", "0", storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	validStore := newStore(cluster.RandClient(), false, codec, "", prefixTransformer{prefix: []byte("test!")})
	validStore.GuaranteedUpdate(ctx, "/abc", &example.Pod{}, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}, nil
		}))
	testCheckEventType(t, watch.Error, w)
}

func TestWatchContextCancel(t *testing.T) {
	ctx, store, cluster := testSetup(t)
	defer cluster.Terminate(t)
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	// When we watch with a canceled context, we should detect that it's context canceled.
	// We won't take it as error and also close the watcher.
	w, err := store.watcher.Watch(canceledCtx, "/abc", 0, false, storage.Everything)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-w.ResultChan():
		if ok {
			t.Error("ResultChan() should be closed")
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("timeout after %v", wait.ForeverTestTimeout)
	}
}

func TestWatchErrResultNotBlockAfterCancel(t *testing.T) {
	origCtx, store, cluster := testSetup(t)
	defer cluster.Terminate(t)
	ctx, cancel := context.WithCancel(origCtx)
	w := store.watcher.createWatchChan(ctx, "/abc", 0, false, storage.Everything)
	// make resutlChan and errChan blocking to ensure ordering.
	w.resultChan = make(chan watch.Event)
	w.errChan = make(chan error)
	// The event flow goes like:
	// - first we send an error, it should block on resultChan.
	// - Then we cancel ctx. The blocking on resultChan should be freed up
	//   and run() goroutine should return.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		w.run()
		wg.Done()
	}()
	w.errChan <- fmt.Errorf("some error")
	cancel()
	wg.Wait()
}

func TestWatchDeleteEventObjectHaveLatestRV(t *testing.T) {
	ctx, store, cluster := testSetup(t)
	defer cluster.Terminate(t)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	w, err := store.Watch(ctx, key, storedObj.ResourceVersion, storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	etcdW := cluster.RandClient().Watch(ctx, "/", clientv3.WithPrefix())

	if err := store.Delete(ctx, key, &example.Pod{}, &storage.Preconditions{}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	e := <-w.ResultChan()
	watchedDeleteObj := e.Object.(*example.Pod)
	var wres clientv3.WatchResponse
	wres = <-etcdW

	watchedDeleteRev, err := storage.ParseWatchResourceVersion(watchedDeleteObj.ResourceVersion)
	if err != nil {
		t.Fatalf("ParseWatchResourceVersion failed: %v", err)
	}
	if int64(watchedDeleteRev) != wres.Events[0].Kv.ModRevision {
		t.Errorf("Object from delete event have version: %v, should be the same as etcd delete's mod rev: %d",
			watchedDeleteRev, wres.Events[0].Kv.ModRevision)
	}
}
*/
type testWatchStruct struct {
	key         string
	obj         *calico.NetworkPolicy
	expectEvent bool
	watchType   watch.EventType
}

/*
type testCodec struct {
	runtime.Codec
}

func (c *testCodec) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	return nil, nil, errTestingDecode
}*/

func testCheckEventType(t *testing.T, expectEventType watch.EventType, w watch.Interface) {
	select {
	case res := <-w.ResultChan():
		if res.Type != expectEventType {
			t.Errorf("event type want=%v, get=%v", expectEventType, res.Type)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("time out after waiting %v on ResultChan", wait.ForeverTestTimeout)
	}
}

func testCheckResult(t *testing.T, i int, expectEventType watch.EventType, w watch.Interface, expectObj *calico.NetworkPolicy) {
	select {
	case res := <-w.ResultChan():
		if res.Type != expectEventType {
			t.Errorf("#%d: event type want=%v, get=%v", i, expectEventType, res.Type)
			return
		}
		if !reflect.DeepEqual(expectObj, res.Object) {
			t.Errorf("#%d: obj want=\n%#v\nget=\n%#v", i, expectObj, res.Object)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("#%d: time out after waiting %v on ResultChan", i, wait.ForeverTestTimeout)
	}
}

func testCheckStop(t *testing.T, i int, w watch.Interface) {
	select {
	case e, ok := <-w.ResultChan():
		if ok {
			var obj string
			switch e.Object.(type) {
			case *example.Pod:
				obj = e.Object.(*example.Pod).Name
			case *metav1.Status:
				obj = e.Object.(*metav1.Status).Message
			}
			t.Errorf("#%d: ResultChan should have been closed. Event: %s. Object: %s", i, e.Type, obj)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("#%d: time out after waiting 1s on ResultChan", i)
	}
}
