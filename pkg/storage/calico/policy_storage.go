package calico

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apiv2"
	"github.com/projectcalico/libcalico-go/lib/clientv2"
	"github.com/projectcalico/libcalico-go/lib/options"
	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

type policyStore struct {
	client    clientv2.Interface
	codec     runtime.Codec
	versioner storage.Versioner
}

// NewNetworkPolicyStorage creates a new libcalico-based storage.Interface implementation for Policy
func NewNetworkPolicyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	var err error

	cfg, err := apiconfig.LoadClientConfig("")
	if err != nil {
		glog.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}

	c, err := clientv2.New(*cfg)
	if err != nil {
		glog.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	glog.Infof("Client: %v", c)

	return &policyStore{
		client:    c,
		codec:     opts.RESTOptions.StorageConfig.Codec,
		versioner: etcd.APIObjectVersioner{},
	}, func() {}
}

// Versioned returns the versioned associated with this interface
func (ps *policyStore) Versioner() storage.Versioner {
	return ps.versioner
}

// Create adds a new object at a key unless it already exists. 'ttl' is time-to-live
// in seconds (0 means forever). If no error is returned and out is not nil, out will be
// set to the read value from database.
func (ps *policyStore) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	networkPolicy := obj.(*aapi.NetworkPolicy)
	libcalicoPolicy := &libcalicoapi.NetworkPolicy{}
	libcalicoPolicy.TypeMeta = networkPolicy.TypeMeta
	libcalicoPolicy.ObjectMeta = networkPolicy.ObjectMeta
	libcalicoPolicy.Spec = networkPolicy.Spec

	pHandler := ps.client.NetworkPolicies()
	opts := options.SetOptions{TTL: time.Duration(ttl) * time.Second}
	createdLibcalicoPolicy, err := pHandler.Create(ctx, libcalicoPolicy, opts)
	if err != nil {
		return aapiError(err, key)
	}
	networkPolicy = out.(*aapi.NetworkPolicy)
	networkPolicy.Spec = createdLibcalicoPolicy.Spec
	networkPolicy.TypeMeta = createdLibcalicoPolicy.TypeMeta
	networkPolicy.ObjectMeta = createdLibcalicoPolicy.ObjectMeta
	return nil
}

// Delete removes the specified key and returns the value that existed at that spot.
// If key didn't exist, it will return NotFound storage error.
func (ps *policyStore) Delete(ctx context.Context, key string, out runtime.Object,
	preconditions *storage.Preconditions) error {
	pHandler := ps.client.NetworkPolicies()
	ns, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return err
	}

	delOpts := options.DeleteOptions{}
	if preconditions != nil {
		// Get the object to check for validity of UID
		opts := options.GetOptions{}
		libcalicoPolicy, err := pHandler.Get(ctx, ns, name, opts)
		if err != nil {
			return aapiError(err, key)
		}
		networkPolicy := &aapi.NetworkPolicy{}
		networkPolicy.Spec = libcalicoPolicy.Spec
		networkPolicy.TypeMeta = libcalicoPolicy.TypeMeta
		networkPolicy.ObjectMeta = libcalicoPolicy.ObjectMeta
		if err := checkPreconditions(key, preconditions, networkPolicy); err != nil {
			return err
		}
		// Set the Resource Version for Deletion
		delOpts.ResourceVersion = networkPolicy.ResourceVersion
	}

	libcalicoPolicy, err := pHandler.Delete(ctx, ns, name, delOpts)
	if err != nil {
		return aapiError(err, key)
	}
	networkPolicy := out.(*aapi.NetworkPolicy)
	networkPolicy.Spec = libcalicoPolicy.Spec
	networkPolicy.TypeMeta = libcalicoPolicy.TypeMeta
	networkPolicy.ObjectMeta = libcalicoPolicy.ObjectMeta
	return nil
}

func checkPreconditions(key string, preconditions *storage.Preconditions, out runtime.Object) error {
	if preconditions == nil {
		return nil
	}
	objMeta, err := meta.Accessor(out)
	if err != nil {
		return storage.NewInternalErrorf("can't enforce preconditions %v on un-introspectable object %v, got error: %v", *preconditions, out, err)
	}
	if preconditions.UID != nil && *preconditions.UID != objMeta.GetUID() {
		errMsg := fmt.Sprintf("Precondition failed: UID in precondition: %v, UID in object meta: %v", *preconditions.UID, objMeta.GetUID())
		return storage.NewInvalidObjError(key, errMsg)
	}
	return nil
}

// Watch begins watching the specified key. Events are decoded into API objects,
// and any items selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will get current object at given key
// and send it in an "ADDED" event, before watch starts.
func (ps *policyStore) Watch(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate) (watch.Interface, error) {
	ns, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return nil, err
	}
	return ps.watch(ctx, resourceVersion, p, name, ns)
}

// WatchList begins watching the specified key's items. Items are decoded into API
// objects and any item selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will list current objects directory defined by key
// and send them in "ADDED" events, before watch starts.
func (ps *policyStore) WatchList(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate) (watch.Interface, error) {
	ns, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return nil, err
	}
	return ps.watch(ctx, resourceVersion, p, name, ns)
}

func (ps *policyStore) watch(ctx context.Context, resourceVersion string,
	p storage.SelectionPredicate, name, namespace string) (watch.Interface, error) {
	pHandler := ps.client.NetworkPolicies()
	opts := options.ListOptions{Name: name, Namespace: namespace, ResourceVersion: resourceVersion}
	lWatch, err := pHandler.Watch(ctx, opts)
	if err != nil {
		return nil, err
	}
	wc := createWatchChan(ctx, lWatch, p)
	go wc.run()
	return wc, nil
}

// Get unmarshals json found at key into objPtr. On a not found error, will either
// return a zero object of the requested type, or an error, depending on ignoreNotFound.
// Treats empty responses and nil response nodes exactly like a not found error.
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ps *policyStore) Get(ctx context.Context, key string, resourceVersion string,
	out runtime.Object, ignoreNotFound bool) error {
	ns, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return err
	}
	pHandler := ps.client.NetworkPolicies()
	opts := options.GetOptions{ResourceVersion: resourceVersion}
	libcalicoPolicy, err := pHandler.Get(ctx, ns, name, opts)
	if err != nil {
		e := aapiError(err, key)
		if storage.IsNotFound(e) && ignoreNotFound {
			return runtime.SetZeroValue(out)
		}
		return e
	}
	networkPolicy := out.(*aapi.NetworkPolicy)
	networkPolicy.Spec = libcalicoPolicy.Spec
	networkPolicy.TypeMeta = libcalicoPolicy.TypeMeta
	networkPolicy.ObjectMeta = libcalicoPolicy.ObjectMeta
	return nil
}

// GetToList unmarshals json found at key and opaque it into *List api object
// (an object that satisfies the runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ps *policyStore) GetToList(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate, listObj runtime.Object) error {
	return ps.List(ctx, key, resourceVersion, p, listObj)
}

// List unmarshalls jsons found at directory defined by key and opaque them
// into *List api object (an object that satisfies runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ps *policyStore) List(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate, listObj runtime.Object) error {
	ns, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return err
	}
	pHandler := ps.client.NetworkPolicies()
	opts := options.ListOptions{Namespace: ns, Name: name, ResourceVersion: resourceVersion}
	libcalicoPolicyList, err := pHandler.List(ctx, opts)
	if err != nil {
		e := aapiError(err, key)
		if storage.IsNotFound(e) {
			networkPolicyList := listObj.(*aapi.NetworkPolicyList)
			networkPolicyList.Items = []aapi.NetworkPolicy{}
			return nil
		}
		return e
	}
	filterFunc := storage.SimpleFilter(p)
	networkPolicyList := listObj.(*aapi.NetworkPolicyList)
	networkPolicyList.Items = []aapi.NetworkPolicy{}
	for _, item := range libcalicoPolicyList.Items {
		networkPolicy := aapi.NetworkPolicy{}
		networkPolicy.TypeMeta = item.TypeMeta
		networkPolicy.ObjectMeta = item.ObjectMeta
		networkPolicy.Spec = item.Spec
		if filterFunc(&networkPolicy) {
			networkPolicyList.Items = append(networkPolicyList.Items, networkPolicy)
		}
	}
	networkPolicyList.TypeMeta = libcalicoPolicyList.TypeMeta
	networkPolicyList.ListMeta = libcalicoPolicyList.ListMeta
	return nil
}

type objState struct {
	obj  runtime.Object
	meta *storage.ResponseMeta
	rev  int64
	data []byte
}

func (ps *policyStore) getStateFromObject(obj runtime.Object) (*objState, error) {
	state := &objState{
		obj:  obj,
		meta: &storage.ResponseMeta{},
	}

	rv, err := ps.versioner.ObjectResourceVersion(obj)
	if err != nil {
		return nil, fmt.Errorf("couldn't get resource version: %v", err)
	}
	state.rev = int64(rv)
	state.meta.ResourceVersion = uint64(state.rev)

	state.data, err = runtime.Encode(ps.codec, obj)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func decode(
	codec runtime.Codec,
	value []byte,
	objPtr runtime.Object,
) error {
	if _, err := conversion.EnforcePtr(objPtr); err != nil {
		panic("unable to convert output object to pointer")
	}
	_, _, err := codec.Decode(value, nil, objPtr)
	if err != nil {
		return err
	}
	return nil
}

// GuaranteedUpdate keeps calling 'tryUpdate()' to update key 'key' (of type 'ptrToType')
// retrying the update until success if there is index conflict.
// Note that object passed to tryUpdate may change across invocations of tryUpdate() if
// other writers are simultaneously updating it, so tryUpdate() needs to take into account
// the current contents of the object when deciding how the update object should look.
// If the key doesn't exist, it will return NotFound storage error if ignoreNotFound=false
// or zero value in 'ptrToType' parameter otherwise.
// If the object to update has the same value as previous, it won't do any update
// but will return the object in 'ptrToType' parameter.
// If 'suggestion' can contain zero or one element - in such case this can be used as
// a suggestion about the current version of the object to avoid read operation from
// storage to get it.
//
// Example:
//
// s := /* implementation of Interface */
// err := s.GuaranteedUpdate(
//     "myKey", &MyType{}, true,
//     func(input runtime.Object, res ResponseMeta) (runtime.Object, *uint64, error) {
//       // Before each incovation of the user defined function, "input" is reset to
//       // current contents for "myKey" in database.
//       curr := input.(*MyType)  // Guaranteed to succeed.
//
//       // Make the modification
//       curr.Counter++
//
//       // Return the modified object - return an error to stop iterating. Return
//       // a uint64 to alter the TTL on the object, or nil to keep it the same value.
//       return cur, nil, nil
//    }
// })
func (ps *policyStore) GuaranteedUpdate(
	ctx context.Context, key string, out runtime.Object, ignoreNotFound bool,
	precondtions *storage.Preconditions, userUpdate storage.UpdateFunc, suggestion ...runtime.Object) error {
	// If a suggestion was passed, use that as the initial object, otherwise
	// use Get() to retrieve it
	var initObj runtime.Object
	if len(suggestion) == 1 && suggestion[0] != nil {
		initObj = suggestion[0]
	} else {
		initObj = &aapi.NetworkPolicy{}
		if err := ps.Get(ctx, key, "", initObj, ignoreNotFound); err != nil {
			glog.Errorf("getting initial object (%s)", err)
			return aapiError(err, key)
		}
	}
	// In either case, extract current state from the initial object
	curState, err := ps.getStateFromObject(initObj)
	if err != nil {
		glog.Errorf("getting state from initial object (%s)", err)
		return err
	}
	// Loop until update succeeds or we get an error
	for {
		if err := checkPreconditions(key, precondtions, curState.obj); err != nil {
			glog.Errorf("checking preconditions (%s)", err)
			return err
		}

		// update the object by applying the userUpdate func & encode it
		updated, ttl, err := userUpdate(curState.obj, *curState.meta)
		if err != nil {
			glog.Errorf("applying user update: (%s)", err)
			return err
		}

		updatedData, err := runtime.Encode(ps.codec, updated)
		if err != nil {
			glog.Errorf("encoding candidate obj (%s)", err)
			return err
		}

		// figure out what the new "current state" of the object is for this loop iteration
		if bytes.Equal(updatedData, curState.data) {
			// If the candidate matches what we already have, then all we need to do is
			// decode into the out object
			return decode(ps.codec, updatedData, out)
		}

		// Apply Update
		networkPolicy := updated.(*aapi.NetworkPolicy)
		libcalicoPolicy := &libcalicoapi.NetworkPolicy{}
		libcalicoPolicy.TypeMeta = networkPolicy.TypeMeta
		libcalicoPolicy.ObjectMeta = networkPolicy.ObjectMeta
		libcalicoPolicy.Spec = networkPolicy.Spec
		if libcalicoPolicy.ResourceVersion == "" {
			// Resource Version needs to be set for libcaclio clientv2 Update call.
			libcalicoPolicy.ResourceVersion = "0"
		}

		pHandler := ps.client.NetworkPolicies()
		var opts options.SetOptions
		if ttl != nil {
			opts = options.SetOptions{TTL: time.Duration(*ttl) * time.Second}
		}
		createdLibcalicoPolicy, err := pHandler.Update(ctx, libcalicoPolicy, opts)
		if err != nil {
			e := aapiError(err, key)
			switch {
			case storage.IsNotFound(e):
				createdLibcalicoPolicy, err = pHandler.Create(ctx, libcalicoPolicy, opts)
				if err != nil {
					return aapiError(err, key)
				}
			case storage.IsConflict(e):
				glog.V(4).Infof(
					"GuaranteedUpdate of %s failed because of a conflict, going to retry",
					key,
				)
				newCurObj := &aapi.NetworkPolicy{}
				if err := ps.Get(ctx, key, "", newCurObj, ignoreNotFound); err != nil {
					glog.Errorf("getting new current object (%s)", err)
					return aapiError(err, key)
				}
				updatedObj, _, err := userUpdate(newCurObj, *curState.meta)
				ncs, err := ps.getStateFromObject(updatedObj)
				if err != nil {
					glog.Errorf("getting state from new current object (%s)", err)
					return err
				}
				curState = ncs
				continue
			}
			return e
		}
		networkPolicy = out.(*aapi.NetworkPolicy)
		networkPolicy.Spec = createdLibcalicoPolicy.Spec
		networkPolicy.TypeMeta = createdLibcalicoPolicy.TypeMeta
		networkPolicy.ObjectMeta = createdLibcalicoPolicy.ObjectMeta
		return nil
	}
}
