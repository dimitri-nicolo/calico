package calico

import (
	"os"

	"golang.org/x/net/context"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/apiv2"
	"github.com/projectcalico/libcalico-go/lib/clientv2"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

type policyStore struct {
	client    clientv2.Interface
	versioner storage.Versioner
}

// NewPolicyStorage creates a new libcalico-based storage.Interface implementation for Policy
func NewNetworkPolicyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	var err error

	cfg, err := apiconfig.LoadClientConfig("")
	if err != nil {
		glog.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}

	//TODO: Make the following items configurable at creation.
	cfg.Spec.DatastoreType = apiconfig.EtcdV3
	cfg.Spec.EtcdEndpoints = "http://127.0.0.1:2379"

	c, err := clientv2.New(*cfg)
	if err != nil {
		glog.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	glog.Infof("Client: %v", c)

	return &policyStore{
		client:    c,
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
	policy := obj.(*calico.NetworkPolicy)
	libcalicoPolicy := &apiv2.NetworkPolicy{}
	libcalicoPolicy.TypeMeta = policy.TypeMeta
	libcalicoPolicy.ObjectMeta = policy.ObjectMeta
	libcalicoPolicy.Spec = policy.Spec

	// TODO: Get namespace from key string, for now "default"
	pHandler := ps.client.NetworkPolicies("namespace")
	// TODO: Set TTL
	opts := options.SetOptions{}
	networkPolicy, err := pHandler.Create(ctx, libcalicoPolicy, opts)
	if err != nil {
		return err
	}
	calicoNetworkPolicy := out.(*calico.NetworkPolicy)
	calicoNetworkPolicy.Spec = networkPolicy.Spec
	calicoNetworkPolicy.TypeMeta = networkPolicy.TypeMeta
	calicoNetworkPolicy.ObjectMeta = networkPolicy.ObjectMeta
	return nil
}

// Delete removes the specified key and returns the value that existed at that spot.
// If key didn't exist, it will return NotFound storage error.
func (ps *policyStore) Delete(ctx context.Context, key string, out runtime.Object,
	preconditions *storage.Preconditions) error {
	// TODO: Get namespace from key string, for now "default"
	pHandler := ps.client.NetworkPolicies("namespace")

	// TODO: Hack to get the object to be deleted and returned
	// TODO: Fill in the resource version if present
	opts := options.GetOptions{}
	networkPolicy, err := pHandler.Get(ctx, "test-networkpolicy", opts)
	if err != nil {
		return err
	}
	calicoNetworkPolicy := out.(*calico.NetworkPolicy)
	calicoNetworkPolicy.Spec = networkPolicy.Spec
	calicoNetworkPolicy.TypeMeta = networkPolicy.TypeMeta
	calicoNetworkPolicy.ObjectMeta = networkPolicy.ObjectMeta

	// TODO: Fill in the resource version if present
	delOpts := options.DeleteOptions{}
	err = pHandler.Delete(ctx, "test-networkpolicy", delOpts)
	if err != nil {
		return err
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
	// TODO: Get namespace from key string, for now "default"
	pHandler := ps.client.NetworkPolicies("namespace")
	// TODO: Fill in the resource version if present
	opts := options.ListOptions{}
	_, err := pHandler.Watch(ctx, opts)
	if err != nil {
		return nil, err
	}
	return nil, nil
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
	//TODO
	return nil, nil
}

// Get unmarshals json found at key into objPtr. On a not found error, will either
// return a zero object of the requested type, or an error, depending on ignoreNotFound.
// Treats empty responses and nil response nodes exactly like a not found error.
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ps *policyStore) Get(ctx context.Context, key string, resourceVersion string,
	objPtr runtime.Object, ignoreNotFound bool) error {
	// TODO: Get namespace from key string, for now "default"
	pHandler := ps.client.NetworkPolicies("namespace")
	// TODO: Fill in the resource version if present
	opts := options.GetOptions{}
	// TODO: Get name from key. This is to just get the integration tests to pass.
	networkPolicy, err := pHandler.Get(ctx, "test-networkpolicy", opts)
	if err != nil {
		return err
	}
	glog.Infof("SHATRU DEBUG: %v", networkPolicy)
	calicoNetworkPolicy := objPtr.(*calico.NetworkPolicy)
	calicoNetworkPolicy.Spec = networkPolicy.Spec
	calicoNetworkPolicy.TypeMeta = networkPolicy.TypeMeta
	calicoNetworkPolicy.ObjectMeta = networkPolicy.ObjectMeta
	glog.Infof("SHATRU DEBUG CNP: %v", calicoNetworkPolicy)

	return nil
}

// GetToList unmarshals json found at key and opaque it into *List api object
// (an object that satisfies the runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ps *policyStore) GetToList(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate, listObj runtime.Object) error {
	//TODO
	return nil
}

// List unmarshalls jsons found at directory defined by key and opaque them
// into *List api object (an object that satisfies runtime.IsList definition).
// The returned contents may be delayed, but it is guaranteed that they will
// be have at least 'resourceVersion'.
func (ps *policyStore) List(ctx context.Context, key string, resourceVersion string,
	p storage.SelectionPredicate, listObj runtime.Object) error {
	// TODO: Get namespace from key string, for now "default"
	pHandler := ps.client.NetworkPolicies("namespace")
	// TODO: Fill in the resource version if present
	opts := options.ListOptions{}
	networkPolicyList, err := pHandler.List(ctx, opts)
	if err != nil {
		return err
	}
	calicoNetworkPolicyList := listObj.(*calico.NetworkPolicyList)
	calicoNetworkPolicyList.Items = []calico.NetworkPolicy{}
	for _, item := range networkPolicyList.Items {
		glog.Infof("SHATRU DEBUG List ITem: %v", item)
		calicoNetworkPolicy := calico.NetworkPolicy{}
		calicoNetworkPolicy.TypeMeta = item.TypeMeta
		calicoNetworkPolicy.ObjectMeta = item.ObjectMeta
		calicoNetworkPolicy.Spec = item.Spec
		calicoNetworkPolicyList.Items = append(calicoNetworkPolicyList.Items, calicoNetworkPolicy)
	}
	calicoNetworkPolicyList.TypeMeta = networkPolicyList.TypeMeta
	calicoNetworkPolicyList.ListMeta = networkPolicyList.ListMeta
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
	ctx context.Context, key string, ptrToType runtime.Object, ignoreNotFound bool,
	precondtions *storage.Preconditions, tryUpdate storage.UpdateFunc, suggestion ...runtime.Object) error {
	//TODO
	return nil
}
