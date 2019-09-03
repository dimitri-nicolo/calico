// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	calicov3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	apitesting "k8s.io/apimachinery/pkg/api/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"

	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"

	"golang.org/x/net/context"
)

func init() {
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)
	calico.AddToScheme(scheme)
	calicov3.AddToScheme(scheme)
}

func TestTierCreate(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	key := "projectcalico.org/tiers/foo"
	out := &calico.Tier{}
	obj := &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}

	// verify that kv pair is empty before set
	libcTier, err := store.client.Tiers().Get(ctx, "foo", options.GetOptions{})
	if libcTier != nil {
		t.Fatalf("expecting empty result on key: %s", key)
	}

	err = store.Create(ctx, key, obj, out, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	// basic tests of the output
	if obj.ObjectMeta.Name != out.ObjectMeta.Name {
		t.Errorf("pod name want=%s, get=%s", obj.ObjectMeta.Name, out.ObjectMeta.Name)
	}
	if out.ResourceVersion == "" {
		t.Errorf("output should have non-empty resource version")
	}

	// verify that kv pair is not empty after set
	libcTier, err = store.client.Tiers().Get(ctx, "foo", options.GetOptions{})
	if err != nil {
		t.Fatalf("libcalico Tier client get failed: %v", err)
	}
	if libcTier == nil {
		t.Fatalf("expecting empty result on key: %s", key)
	}
}

func TestTierCreateWithTTL(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	input := &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	key := "projectcalico.org/tiers/foo"

	out := &calico.Tier{}
	if err := store.Create(ctx, key, input, out, 1); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	w, err := store.Watch(ctx, key, out.ResourceVersion, storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckEventType(t, watch.Deleted, w)
}

func TestTierCreateWithKeyExist(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	obj := &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	key, _ := testTierPropogateStore(ctx, t, store, obj)
	out := &calico.Tier{}
	err := store.Create(ctx, key, obj, out, 0)
	if err == nil || !storage.IsNodeExist(err) {
		t.Errorf("expecting key exists error, but get: %s", err)
	}
}

func TestTierGet(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	key, storedObj := testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	tests := []struct {
		key               string
		ignoreNotFound    bool
		expectNotFoundErr bool
		expectedOut       *calico.Tier
	}{{ // test get on existing item
		key:               key,
		ignoreNotFound:    false,
		expectNotFoundErr: false,
		expectedOut:       storedObj,
	}, { // test get on non-existing item with ignoreNotFound=false
		key:               "projectcalico.org/tiers/non-existing",
		ignoreNotFound:    false,
		expectNotFoundErr: true,
	}, { // test get on non-existing item with ignoreNotFound=true
		key:               "projectcalico.org/tiers/non-existing",
		ignoreNotFound:    true,
		expectNotFoundErr: false,
		expectedOut:       &calico.Tier{},
	}}

	for i, tt := range tests {
		out := &calico.Tier{}
		err := store.Get(ctx, tt.key, "", out, tt.ignoreNotFound)
		if tt.expectNotFoundErr {
			if err == nil || !storage.IsNotFound(err) {
				t.Errorf("#%d: expecting not found error, but get: %s", i, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !reflect.DeepEqual(tt.expectedOut, out) {
			t.Errorf("#%d: pod want=%#v, get=%#v", i, tt.expectedOut, out)
		}
	}
}

func TestTierUnconditionalDelete(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	key, storedObj := testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	tests := []struct {
		key               string
		expectedObj       *calico.Tier
		expectNotFoundErr bool
	}{{ // test unconditional delete on existing key
		key:               key,
		expectedObj:       storedObj,
		expectNotFoundErr: false,
	}, { // test unconditional delete on non-existing key
		key:               "projectcalico.org/tiers/non-existing",
		expectedObj:       nil,
		expectNotFoundErr: true,
	}}

	for i, tt := range tests {
		out := &calico.Tier{} // reset
		err := store.Delete(ctx, tt.key, out, nil)
		if tt.expectNotFoundErr {
			if err == nil || !storage.IsNotFound(err) {
				t.Errorf("#%d: expecting not found error, but get: %s", i, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if !reflect.DeepEqual(tt.expectedObj, out) {
			t.Errorf("#%d: pod want=%#v, get=%#v", i, tt.expectedObj, out)
		}
	}
}

func TestTierConditionalDelete(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	key, storedObj := testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo", UID: "A"}})

	tests := []struct {
		precondition        *storage.Preconditions
		expectInvalidObjErr bool
	}{{ // test conditional delete with UID match
		precondition:        storage.NewUIDPreconditions("A"),
		expectInvalidObjErr: false,
	}, { // test conditional delete with UID mismatch
		precondition:        storage.NewUIDPreconditions("B"),
		expectInvalidObjErr: true,
	}}

	for i, tt := range tests {
		out := &calico.Tier{}
		err := store.Delete(ctx, key, out, tt.precondition)
		if tt.expectInvalidObjErr {
			if err == nil || !storage.IsInvalidObj(err) {
				t.Errorf("#%d: expecting invalid UID error, but get: %s", i, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if !reflect.DeepEqual(storedObj, out) {
			t.Errorf("#%d: pod want=%#v, get=%#v", i, storedObj, out)
		}
		key, storedObj = testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo", UID: "A"}})
	}
}

func TestTierGetToList(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	key, storedObj := testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	tests := []struct {
		key         string
		pred        storage.SelectionPredicate
		expectedOut []*calico.Tier
	}{{ // test GetToList on existing key
		key:         key,
		pred:        storage.Everything,
		expectedOut: []*calico.Tier{storedObj},
	}, { // test GetToList on non-existing key
		key:         "projectcalico.org/tiers/non-existing",
		pred:        storage.Everything,
		expectedOut: nil,
	}, { // test GetToList with matching tier name
		key: "projectcalico.org/tiers/non-existing",
		pred: storage.SelectionPredicate{
			Label: labels.Everything(),
			Field: fields.ParseSelectorOrDie("metadata.name!=" + storedObj.Name),
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
				tier := obj.(*calico.Tier)
				return nil, fields.Set{"metadata.name": tier.Name}, tier.Initializers != nil, nil
			},
		},
		expectedOut: nil,
	}}

	for i, tt := range tests {
		out := &calico.TierList{}
		err := store.GetToList(ctx, tt.key, "", tt.pred, out)
		if err != nil {
			t.Fatalf("GetToList failed: %v", err)
		}
		if len(out.Items) != len(tt.expectedOut) {
			t.Errorf("#%d: length of list want=%d, get=%d", i, len(tt.expectedOut), len(out.Items))
			continue
		}
		for j, wantTier := range tt.expectedOut {
			getTier := &out.Items[j]
			if !reflect.DeepEqual(wantTier, getTier) {
				t.Errorf("#%d: pod want=%#v, get=%#v", i, wantTier, getTier)
			}
		}
	}
}

func TestTierGuaranteedUpdate(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer func() {
		testTierCleanup(t, ctx, store)
		store.client.Tiers().Delete(ctx, "non-existing", options.DeleteOptions{})
	}()
	key, storeObj := testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo", UID: "A"}})

	tests := []struct {
		key                 string
		ignoreNotFound      bool
		precondition        *storage.Preconditions
		expectNotFoundErr   bool
		expectInvalidObjErr bool
		expectNoUpdate      bool
		transformStale      bool
	}{{ // GuaranteedUpdate on non-existing key with ignoreNotFound=false
		key:                 "projectcalico.org/tiers/non-existing",
		ignoreNotFound:      false,
		precondition:        nil,
		expectNotFoundErr:   true,
		expectInvalidObjErr: false,
		expectNoUpdate:      false,
	}, { // GuaranteedUpdate on non-existing key with ignoreNotFound=true
		key:                 "projectcalico.org/tiers/non-existing",
		ignoreNotFound:      true,
		precondition:        nil,
		expectNotFoundErr:   false,
		expectInvalidObjErr: false,
		expectNoUpdate:      false,
	}, { // GuaranteedUpdate on existing key
		key:                 key,
		ignoreNotFound:      false,
		precondition:        nil,
		expectNotFoundErr:   false,
		expectInvalidObjErr: false,
		expectNoUpdate:      false,
	}, { // GuaranteedUpdate with same data
		key:                 key,
		ignoreNotFound:      false,
		precondition:        nil,
		expectNotFoundErr:   false,
		expectInvalidObjErr: false,
		expectNoUpdate:      true,
	}, { // GuaranteedUpdate with same data but stale
		key:                 key,
		ignoreNotFound:      false,
		precondition:        nil,
		expectNotFoundErr:   false,
		expectInvalidObjErr: false,
		expectNoUpdate:      false,
		transformStale:      true,
	}, { // GuaranteedUpdate with UID match
		key:                 key,
		ignoreNotFound:      false,
		precondition:        storage.NewUIDPreconditions("A"),
		expectNotFoundErr:   false,
		expectInvalidObjErr: false,
		expectNoUpdate:      true,
	}, { // GuaranteedUpdate with UID mismatch
		key:                 key,
		ignoreNotFound:      false,
		precondition:        storage.NewUIDPreconditions("B"),
		expectNotFoundErr:   false,
		expectInvalidObjErr: true,
		expectNoUpdate:      true,
	}}

	for i, tt := range tests {
		out := &calico.Tier{}
		selector := fmt.Sprintf("foo-%d", i)
		if tt.expectNoUpdate {
			selector = ""
		}
		version := storeObj.ResourceVersion
		err := store.GuaranteedUpdate(ctx, tt.key, out, tt.ignoreNotFound, tt.precondition,
			storage.SimpleUpdate(func(obj runtime.Object) (runtime.Object, error) {
				if tt.expectNotFoundErr && tt.ignoreNotFound {
					if tier := obj.(*calico.Tier); tier.GenerateName != "" {
						t.Errorf("#%d: expecting zero value, but get=%#v", i, tier)
					}
				}
				tier := *storeObj
				if !tt.expectNoUpdate {
					tier.GenerateName = selector
				}
				return &tier, nil
			}))

		if tt.expectNotFoundErr {
			if err == nil || !storage.IsNotFound(err) {
				t.Errorf("#%d: expecting not found error, but get: %v", i, err)
			}
			continue
		}
		if tt.expectInvalidObjErr {
			if err == nil || !storage.IsInvalidObj(err) {
				t.Errorf("#%d: expecting invalid UID error, but get: %s", i, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("GuaranteedUpdate failed: %v", err)
		}
		if !tt.expectNoUpdate {
			if out.GenerateName != selector {
				t.Errorf("#%d: tier selector want=%s, get=%s", i, selector, out.GenerateName)
			}
		}
		switch tt.expectNoUpdate {
		case true:
			if version != out.ResourceVersion {
				t.Errorf("#%d: expect no version change, before=%s, after=%s", i, version, out.ResourceVersion)
			}
		case false:
			if version == out.ResourceVersion {
				t.Errorf("#%d: expect version change, but get the same version=%s", i, version)
			}
		}
		storeObj = out
	}
}

func TestTierGuaranteedUpdateWithTTL(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)

	input := &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	input.SetCreationTimestamp(metav1.Time{time.Now()})
	input.SetUID("test_uid")
	key := "projectcalico.org/tiers/foo"

	out := &calico.Tier{}
	err := store.GuaranteedUpdate(ctx, key, out, true, nil,
		func(_ runtime.Object, _ storage.ResponseMeta) (runtime.Object, *uint64, error) {
			ttl := uint64(1)
			return input, &ttl, nil
		})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	w, err := store.Watch(ctx, key, out.ResourceVersion, storage.Everything)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckEventType(t, watch.Deleted, w)
}

func TestTierGuaranteedUpdateWithConflict(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer testTierCleanup(t, ctx, store)
	key, _ := testTierPropogateStore(ctx, t, store, &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	errChan := make(chan error, 1)
	var firstToFinish sync.WaitGroup
	var secondToEnter sync.WaitGroup
	firstToFinish.Add(1)
	secondToEnter.Add(1)

	go func() {
		err := store.GuaranteedUpdate(ctx, key, &calico.Tier{}, false, nil,
			storage.SimpleUpdate(func(obj runtime.Object) (runtime.Object, error) {
				tier := obj.(*calico.Tier)
				tier.GenerateName = "foo-1"
				secondToEnter.Wait()
				return tier, nil
			}))
		firstToFinish.Done()
		errChan <- err
	}()

	updateCount := 0
	err := store.GuaranteedUpdate(ctx, key, &calico.Tier{}, false, nil,
		storage.SimpleUpdate(func(obj runtime.Object) (runtime.Object, error) {
			if updateCount == 0 {
				secondToEnter.Done()
				firstToFinish.Wait()
			}
			updateCount++
			tier := obj.(*calico.Tier)
			tier.GenerateName = "foo-2"
			return tier, nil
		}))
	if err != nil {
		t.Fatalf("Second GuaranteedUpdate error %#v", err)
	}
	if err := <-errChan; err != nil {
		t.Fatalf("First GuaranteedUpdate error %#v", err)
	}

	if updateCount != 2 {
		t.Errorf("Should have conflict and called update func twice")
	}
}

func TestTierList(t *testing.T) {
	ctx, store := testTierSetup(t)
	defer func() {
		store.client.Tiers().Delete(ctx, "foo", options.DeleteOptions{})
		store.client.Tiers().Delete(ctx, "bar", options.DeleteOptions{})
	}()

	preset := []struct {
		key       string
		obj       *calico.Tier
		storedObj *calico.Tier
	}{{
		key: "projectcalico.org/tiers/foo",
		obj: &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
	}, {
		key: "projectcalico.org/tiers/bar",
		obj: &calico.Tier{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
	}}

	for i, ps := range preset {
		preset[i].storedObj = &calico.Tier{}
		err := store.Create(ctx, ps.key, ps.obj, preset[i].storedObj, 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	defaultTier := &calico.Tier{}
	store.Get(ctx, "projectcalico.org/tiers/default", "", defaultTier, false)

	tests := []struct {
		prefix      string
		pred        storage.SelectionPredicate
		expectedOut []*calico.Tier
	}{{ // test List at cluster scope
		prefix:      "projectcalico.org/tiers/foo",
		pred:        storage.Everything,
		expectedOut: []*calico.Tier{preset[0].storedObj},
	}, { // test List with tier name matching
		prefix: "projectcalico.org/tiers/",
		pred: storage.SelectionPredicate{
			Label: labels.Everything(),
			Field: fields.ParseSelectorOrDie("metadata.name!=" + preset[0].storedObj.Name),
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
				tier := obj.(*calico.Tier)
				return nil, fields.Set{"metadata.name": tier.Name}, tier.Initializers != nil, nil
			},
		},
		expectedOut: []*calico.Tier{preset[1].storedObj, defaultTier},
	}}

	for i, tt := range tests {
		out := &calico.TierList{}
		err := store.List(ctx, tt.prefix, "0", tt.pred, out)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(tt.expectedOut) != len(out.Items) {
			t.Errorf("#%d: length of list want=%d, get=%d", i, len(tt.expectedOut), len(out.Items))
			continue
		}
		for j, wantTier := range tt.expectedOut {
			getTier := &out.Items[j]
			if !reflect.DeepEqual(wantTier, getTier) {
				t.Errorf("#%d: tier want=%#v, get=%#v", i, wantTier, getTier)
			}
		}
	}
}

func testTierSetup(t *testing.T) (context.Context, *resourceStore) {
	codec := apitesting.TestCodec(codecs, calicov3.SchemeGroupVersion)
	cfg, err := apiconfig.LoadClientConfig("")
	if err != nil {
		glog.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}
	cfg.Spec.DatastoreType = "etcdv3"
	cfg.Spec.EtcdEndpoints = "http://localhost:2379"
	c, err := clientv3.New(*cfg)
	if err != nil {
		glog.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	glog.Infof("Client: %v", c)
	opts := Options{
		RESTOptions: generic.RESTOptions{
			StorageConfig: &storagebackend.Config{
				Codec: codec,
			},
		},
	}
	store, _ := NewTierStorage(opts)
	ctx := context.Background()

	return ctx, store.(*resourceStore)
}

func testTierCleanup(t *testing.T, ctx context.Context, store *resourceStore) {
	tr, _ := store.client.Tiers().Get(ctx, "default", options.GetOptions{})
	if tr != nil {
		store.client.Tiers().Delete(ctx, "foo", options.DeleteOptions{})
	}
}

// testTierPropogateStore helps propogates store with objects, automates key generation, and returns
// keys and stored objects.
func testTierPropogateStore(ctx context.Context, t *testing.T, store *resourceStore, obj *calico.Tier) (string, *calico.Tier) {
	// Setup store with a key and grab the output for returning.
	key := "projectcalico.org/tiers/foo"
	setOutput := &calico.Tier{}
	err := store.Create(ctx, key, obj, setOutput, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	return key, setOutput
}
