// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	apitesting "k8s.io/apimachinery/pkg/api/apitesting"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog"

	calico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
	calicov3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

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

	opts := storage.ListOptions{ResourceVersion: out.ResourceVersion, Predicate: storage.Everything}
	w, err := store.Watch(ctx, key, opts)
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
		opts := storage.GetOptions{IgnoreNotFound: tt.ignoreNotFound}
		err := store.Get(ctx, tt.key, opts, out)
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
		err := store.Delete(ctx, tt.key, out, nil, nil, nil)
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
		err := store.Delete(ctx, key, out, tt.precondition, nil, nil)
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
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
				tier := obj.(*calico.Tier)
				return nil, fields.Set{"metadata.name": tier.Name}, nil
			},
		},
		expectedOut: nil,
	}}

	for i, tt := range tests {
		out := &calico.TierList{}
		opts := storage.ListOptions{Predicate: tt.pred}
		err := store.GetToList(ctx, tt.key, opts, out)
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
			}), nil)

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
		}, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	opts := storage.ListOptions{ResourceVersion: out.ResourceVersion, Predicate: storage.Everything}
	w, err := store.Watch(ctx, key, opts)
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
			}), nil)
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
		}), nil)
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
		store.client.LicenseKey().Delete(ctx, "default", options.DeleteOptions{})
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
	opts := storage.GetOptions{IgnoreNotFound: false}
	store.Get(ctx, "projectcalico.org/tiers/default", opts, defaultTier)

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
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
				tier := obj.(*calico.Tier)
				return nil, fields.Set{"metadata.name": tier.Name}, nil
			},
		},
		expectedOut: []*calico.Tier{preset[1].storedObj, defaultTier},
	}}

	for i, tt := range tests {
		out := &calico.TierList{}
		opts := storage.ListOptions{ResourceVersion: "0", Predicate: tt.pred}
		err := store.List(ctx, tt.prefix, opts, out)
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
		klog.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}
	cfg.Spec.DatastoreType = "etcdv3"
	cfg.Spec.EtcdEndpoints = "http://localhost:2379"
	c, err := clientv3.New(*cfg)
	if err != nil {
		klog.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	klog.Infof("Client: %v", c)
	cache := NewLicenseCache()
	cache.Store(*getLicenseKey("default", validLicenseCertificate, validLicenseToken))

	opts := Options{
		RESTOptions: generic.RESTOptions{
			StorageConfig: &storagebackend.Config{
				Codec: codec,
			},
		},
		LicenseCache: cache,
	}
	store, _ := NewTierStorage(opts)
	ctx := context.Background()

	return ctx, store.Storage.(*resourceStore)
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

func getLicenseKey(name, cert, token string) *apiv3.LicenseKey {

	validLicenseKey := &apiv3.LicenseKey{ObjectMeta: metav1.ObjectMeta{Name: name}}

	validLicenseKey.Spec.Certificate = cert
	validLicenseKey.Spec.Token = token

	return validLicenseKey
}

const validLicenseCertificate = `-----BEGIN CERTIFICATE-----
MIIFxjCCA66gAwIBAgIQVq3rz5D4nQF1fIgMEh71DzANBgkqhkiG9w0BAQsFADCB
tTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1
cml0eSA8c2lydEB0aWdlcmEuaW8+MT8wPQYDVQQDEzZUaWdlcmEgRW50aXRsZW1l
bnRzIEludGVybWVkaWF0ZSBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwHhcNMTgwNDA1
MjEzMDI5WhcNMjAxMDA2MjEzMDI5WjCBnjELMAkGA1UEBhMCVVMxEzARBgNVBAgT
CkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xFDASBgNVBAoTC1Rp
Z2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1cml0eSA8c2lydEB0aWdlcmEuaW8+MSgw
JgYDVQQDEx9UaWdlcmEgRW50aXRsZW1lbnRzIENlcnRpZmljYXRlMIIBojANBgkq
hkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAwg3LkeHTwMi651af/HEXi1tpM4K0LVqb
5oUxX5b5jjgi+LHMPzMI6oU+NoGPHNqirhAQqK/k7W7r0oaMe1APWzaCAZpHiMxE
MlsAXmLVUrKg/g+hgrqeije3JDQutnN9h5oZnsg1IneBArnE/AKIHH8XE79yMG49
LaKpPGhpF8NoG2yoWFp2ekihSohvqKxa3m6pxoBVdwNxN0AfWxb60p2SF0lOi6B3
hgK6+ILy08ZqXefiUs+GC1Af4qI1jRhPkjv3qv+H1aQVrq6BqKFXwWIlXCXF57CR
hvUaTOG3fGtlVyiPE4+wi7QDo0cU/+Gx4mNzvmc6lRjz1c5yKxdYvgwXajSBx2pw
kTP0iJxI64zv7u3BZEEII6ak9mgUU1CeGZ1KR2Xu80JiWHAYNOiUKCBYHNKDCUYl
RBErYcAWz2mBpkKyP6hbH16GjXHTTdq5xENmRDHabpHw5o+21LkWBY25EaxjwcZa
Y3qMIOllTZ2iRrXu7fSP6iDjtFCcE2bFAgMBAAGjZzBlMA4GA1UdDwEB/wQEAwIF
oDATBgNVHSUEDDAKBggrBgEFBQcDAjAdBgNVHQ4EFgQUIY7LzqNTzgyTBE5efHb5
kZ71BUEwHwYDVR0jBBgwFoAUxZA5kifzo4NniQfGKb+4wruTIFowDQYJKoZIhvcN
AQELBQADggIBAAK207LaqMrnphF6CFQnkMLbskSpDZsKfqqNB52poRvUrNVUOB1w
3dSEaBUjhFgUU6yzF+xnuH84XVbjD7qlM3YbdiKvJS9jrm71saCKMNc+b9HSeQAU
DGY7GPb7Y/LG0GKYawYJcPpvRCNnDLsSVn5N4J1foWAWnxuQ6k57ymWwcddibYHD
OPakOvO4beAnvax3+K5dqF0bh2Np79YolKdIgUVzf4KSBRN4ZE3AOKlBfiKUvWy6
nRGvu8O/8VaI0vGaOdXvWA5b61H0o5cm50A88tTm2LHxTXynE3AYriHxsWBbRpoM
oFnmDaQtGY67S6xGfQbwxrwCFd1l7rGsyBQ17cuusOvMNZEEWraLY/738yWKw3qX
U7KBxdPWPIPd6iDzVjcZrS8AehUEfNQ5yd26gDgW+rZYJoAFYv0vydMEyoI53xXs
cpY84qV37ZC8wYicugidg9cFtD+1E0nVgOLXPkHnmc7lIDHFiWQKfOieH+KoVCbb
zdFu3rhW31ygphRmgszkHwApllCTBBMOqMaBpS8eHCnetOITvyB4Kiu1/nKvVxhY
exit11KQv8F3kTIUQRm0qw00TSBjuQHKoG83yfimlQ8OazciT+aLpVaY8SOrrNnL
IJ8dHgTpF9WWHxx04DDzqrT7Xq99F9RzDzM7dSizGxIxonoWcBjiF6n5
-----END CERTIFICATE-----`

const validLicenseToken = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJTZGUtWW1KV0pIcTNWREJ3IiwidGFnIjoiOGF2UVZEVHptME9mRGlVM2hfRlhYQSIsInR5cCI6IkpXVCJ9.nZK7QAqo3Jfa3LjUPtFHmw.Y_QN4NvAH0GmSMO9.bMxJ4AtoIF7uLShaSRXDL6cGXUq4kPVQjsh_dFndWud3fjSn1S7q09HcnTHKNmTupCsmStSB_lV363Ar9ShrV8WRebZeKZYqB4OOMzbj89fiTPPPA0AlqxrlEMnHyQYefyp_Kjy_eymHoaiZBzIiHZgKBDP4Dh6lhrMThUMaer6iKo_iMjtI-zRlAQ0_eMAcxRyiyFFIbUdUcy3uMz1UBQFLlm7YMslRBRzvf8gT__Ptihjll0KsxyGtivzYEwgOZ4lheWr1Af5nmslNQP9mR6MOF4TeSik3_yzq6TP3mgUol5HNCWyNB9-o-uqk9Wn0mQG3uy1ERJCMHNPKoUrvSTA5DiF7QeN8YR2h1C36ehcGLYi9L9jj1nT2JOO-uFagTdJeGH3lRQnF6RYkyfw-kitHuac8Ghte-YZNvXTmRBp7wT_L-X89-FcT4XveW5va0ChVOdl7aKAlkf8GDl3gZEkz22eVtZAnFEp6N-ApSasFA-3clqTulSlsLL4WkQ_Vin3lMEr11cYl2VFnQovLw3F30vrB2XEyjEiGRw86R4PRfxlYkHDgK7FhGgFb1UM4lmZUCycExzSYYpDd3oQBFEDR_fhZ0oq6Fp7SUeA6ypFL_Hph1NB0kf5emGnq4R2vr-T4BuM8YYe9Qa6OuVtf2U3o3ipCqdsAAHII0GhlLJWCs5ovNPOEbS_ky_0mLW8mvzfHnPqGL3HjZA2DZb0pZlqI7qbmwiO8N9iU5uZA0RsHJX_ClDF971m2LoUQAbe2I0rCtrhVhW5ljQPuJSTv0chLSDCPxk0-jEsTpA12dqK3eiyT-hWyTTXb2ZsivBdCIpOpVbZM2z2EvvEMvsN3lLCHGP61i0C0KPlze9DJE6vZVxAW1nzqRqi1IqU5mfZuoX8McbQiAEzBQ096hvypIygBmVTr17N8sXmHwJPNEdiLQ3pTLfyHBGZDyZlpy2Ej-4mG-Iegg8hjTkEm3q7QHzRL8hWTP0ff7MHT1NOXkSbN_bIpLmtjb75-we3Mc2cBPyyV96D89G16UUGkh0lzy0pLMMbz_ejSbKlULFkJJWRGn_58Hkw1ROBeREccg_F5B0wqLKY__jyq1OqrzcIZrxhUPLaWfoDKzSykDw.yeAEkIEd1wSwvuwgHs_6dw`
