package calico

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apiv2"
	"github.com/projectcalico/libcalico-go/lib/clientv2"
	"github.com/projectcalico/libcalico-go/lib/options"
	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

type tierStore struct {
	client    clientv2.Interface
	codec     runtime.Codec
	versioner storage.Versioner
}

// NewTierStorage creates a new libcalico-based storage.Interface implementation for Tiers
func NewTierStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
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

	return &tierStore{
		client:    c,
		codec:     opts.RESTOptions.StorageConfig.Codec,
		versioner: etcd.APIObjectVersioner{},
	}, func() {}
}

// Versioned returns the versioned associated with this interface
func (ts *tierStore) Versioner() storage.Versioner {
	return ts.versioner
}

// Create adds a new object at a key unless it already exists. 'ttl' is time-to-live
// in seconds (0 means forever). If no error is returned and out is not nil, out will be
// set to the read value from database.
func (ts *tierStore) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	tier := obj.(*aapi.Tier)
	libcalicoTier := &libcalicoapi.Tier{}
	convertToLibcalicoTier(tier, libcalicoTier)

	tHandler := ts.client.Tiers()
	opts := options.SetOptions{TTL: time.Duration(ttl) * time.Second}
	createdLibcalicoTier, err := tHandler.Create(ctx, libcalicoTier, opts)
	if err != nil {
		return aapiError(err, key)
	}
	tier = out.(*aapi.Tier)
	convertToAAPITier(tier, createdLibcalicoTier)
	return nil
}

// Delete removes the specified key and returns the value that existed at that spot.
// If key didn't exist, it will return NotFound storage error.
func (ts *tierStore) Delete(ctx context.Context, key string, out runtime.Object,
	preconditions *storage.Preconditions) error {
	tHandler := ts.client.Tiers()
	_, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return err
	}

	delOpts := options.DeleteOptions{}
	if preconditions != nil {
		// Get the object to check for validity of UID
		opts := options.GetOptions{}
		libcalicoTier, err := tHandler.Get(ctx, name, opts)
		if err != nil {
			return aapiError(err, key)
		}
		tier := &aapi.Tier{}
		convertToAAPITier(tier, libcalicoTier)
		if err := checkPreconditions(key, preconditions, tier); err != nil {
			return err
		}
		// Set the Resource Version for Deletion
		delOpts.ResourceVersion = tier.ResourceVersion
	}

	libcalicoTier, err := tHandler.Delete(ctx, name, delOpts)
	if err != nil {
		return aapiError(err, key)
	}
	tier := out.(*aapi.Tier)
	convertToAAPITier(tier, libcalicoTier)
	return nil
}

// Watch begins watching the specified key. Events are decoded into API objects,
// and any items selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will get current object at given key
// and send it in an "ADDED" event, before watch starts.
func (ts *tierStore) Watch(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate) (watch.Interface, error) {
	_, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return nil, err
	}
	return ts.watch(ctx, resourceVersion, p, name)
}

// WatchList begins watching the specified key's items. Items are decoded into API
// objects and any item selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will list current objects directory defined by key
// and send them in "ADDED" events, before watch starts.
func (ts *tierStore) WatchList(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate) (watch.Interface, error) {
	_, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return nil, err
	}
	return ts.watch(ctx, resourceVersion, p, name)
}

func (ts *tierStore) watch(ctx context.Context, resourceVersion string,
	p storage.SelectionPredicate, name string) (watch.Interface, error) {
	tHandler := ts.client.Tiers()
	opts := options.ListOptions{Name: name, ResourceVersion: resourceVersion}
	lWatch, err := tHandler.Watch(ctx, opts)
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
func (ts *tierStore) Get(ctx context.Context, key string, resourceVersion string,
	out runtime.Object, ignoreNotFound bool) error {
	_, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return err
	}
	tHandler := ts.client.Tiers()
	opts := options.GetOptions{ResourceVersion: resourceVersion}
	libcalicoTier, err := tHandler.Get(ctx, name, opts)
	if err != nil {
		e := aapiError(err, key)
		if storage.IsNotFound(e) && ignoreNotFound {
			return runtime.SetZeroValue(out)
		}
		return e
	}
	tier := out.(*aapi.Tier)
	convertToAAPITier(tier, libcalicoTier)
	return nil
}

// GetToList unmarshals json found at key and opaque it into *List api object
// (an object that satisfies the runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ts *tierStore) GetToList(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate, listObj runtime.Object) error {
	return ts.List(ctx, key, resourceVersion, p, listObj)
}

// List unmarshalls jsons found at directory defined by key and opaque them
// into *List api object (an object that satisfies runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ts *tierStore) List(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate, listObj runtime.Object) error {
	_, name, err := NamespaceAndNameFromKey(key)
	if err != nil {
		return err
	}
	tHandler := ts.client.Tiers()
	opts := options.ListOptions{Name: name, ResourceVersion: resourceVersion}
	libcalicoTierList, err := tHandler.List(ctx, opts)
	if err != nil {
		e := aapiError(err, key)
		if storage.IsNotFound(e) {
			tierList := listObj.(*aapi.TierList)
			tierList.Items = []aapi.Tier{}
			return nil
		}
		return e
	}
	filterFunc := storage.SimpleFilter(p)
	tierList := listObj.(*aapi.TierList)
	tierList.Items = []aapi.Tier{}
	for _, item := range libcalicoTierList.Items {
		tier := aapi.Tier{}
		convertToAAPITier(&tier, &item)
		if filterFunc(&tier) {
			tierList.Items = append(tierList.Items, tier)
		}
	}
	tierList.TypeMeta = libcalicoTierList.TypeMeta
	tierList.ListMeta = libcalicoTierList.ListMeta
	return nil
}

func (ts *tierStore) getStateFromObject(obj runtime.Object) (*objState, error) {
	state := &objState{
		obj:  obj,
		meta: &storage.ResponseMeta{},
	}

	rv, err := ts.versioner.ObjectResourceVersion(obj)
	if err != nil {
		return nil, fmt.Errorf("couldn't get resource version: %v", err)
	}
	state.rev = int64(rv)
	state.meta.ResourceVersion = uint64(state.rev)

	state.data, err = runtime.Encode(ts.codec, obj)
	if err != nil {
		return nil, err
	}

	return state, nil
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
func (ts *tierStore) GuaranteedUpdate(
	ctx context.Context, key string, out runtime.Object, ignoreNotFound bool,
	precondtions *storage.Preconditions, userUpdate storage.UpdateFunc, suggestion ...runtime.Object) error {
	// If a suggestion was passed, use that as the initial object, otherwise
	// use Get() to retrieve it
	var initObj runtime.Object
	if len(suggestion) == 1 && suggestion[0] != nil {
		initObj = suggestion[0]
	} else {
		initObj = &aapi.Tier{}
		if err := ts.Get(ctx, key, "", initObj, ignoreNotFound); err != nil {
			glog.Errorf("getting initial object (%s)", err)
			return aapiError(err, key)
		}
	}
	// In either case, extract current state from the initial object
	curState, err := ts.getStateFromObject(initObj)
	if err != nil {
		glog.Errorf("getting state from initial object (%s)", err)
		return err
	}

	// Loop until update succeeds or we get an error
	// Check count to avoid an infinite loop in case of any issues
	totalLoopCount := 0
	for totalLoopCount < 5 {
		totalLoopCount++

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

		updatedData, err := runtime.Encode(ts.codec, updated)
		if err != nil {
			glog.Errorf("encoding candidate obj (%s)", err)
			return err
		}

		// figure out what the new "current state" of the object is for this loop iteration
		if bytes.Equal(updatedData, curState.data) {
			// If the candidate matches what we already have, then all we need to do is
			// decode into the out object
			return decode(ts.codec, updatedData, out)
		}

		// Apply Update
		tier := updated.(*aapi.Tier)
		// Check for Revision no. If not set or less than the current version then set it
		// to latest
		accessor, err := meta.Accessor(tier)
		if err != nil {
			return err
		}
		revInt, _ := strconv.Atoi(accessor.GetResourceVersion())
		if tier.ResourceVersion == "" || revInt < int(curState.rev) {
			tier.ResourceVersion = strconv.FormatInt(curState.rev, 10)
		}
		libcalicoTier := &libcalicoapi.Tier{}
		convertToLibcalicoTier(tier, libcalicoTier)

		tHandler := ts.client.Tiers()
		var opts options.SetOptions
		if ttl != nil {
			opts = options.SetOptions{TTL: time.Duration(*ttl) * time.Second}
		}
		var createdLibcalicoTier *libcalicoapi.Tier
		createdLibcalicoTier, err = tHandler.Update(ctx, libcalicoTier, opts)
		if err != nil {
			e := aapiError(err, key)
			if storage.IsConflict(e) {
				glog.V(4).Infof(
					"GuaranteedUpdate of %s failed because of a conflict, going to retry",
					key,
				)
				newCurObj := &aapi.Tier{}
				if err := ts.Get(ctx, key, "", newCurObj, ignoreNotFound); err != nil {
					glog.Errorf("getting new current object (%s)", err)
					return aapiError(err, key)
				}
				ncs, err := ts.getStateFromObject(newCurObj)
				if err != nil {
					glog.Errorf("getting state from new current object (%s)", err)
					return err
				}
				curState = ncs
				continue
			}
			return e
		}
		tier = out.(*aapi.Tier)
		convertToAAPITier(tier, createdLibcalicoTier)
		return nil
	}
	glog.Errorf("GuaranteedUpdate failed.")
	return nil
}
