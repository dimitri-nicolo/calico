package globalnetworksets

import (
	"context"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v3client "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const DefaultClientRetries = 5
const DefaultResyncPeriod = time.Hour
const LabelKey = "tigera.io/creator"
const LabelValue = "intrusion-detection-controller"

type Controller interface {
	// Add, Delete, and GC alter the desired state the controller will attempt to
	// maintain, by syncing with the Kubernetes API server.

	// Add or update a new GlobalNetworkSet including the spec
	Add(*v3.GlobalNetworkSet)

	// Delete removes a GlobalNetworkSet from the desired state.
	Delete(*v3.GlobalNetworkSet)

	// NoGC marks a GlobalNetworkSet as not eligible for garbage collection
	// until deleted. This is useful when we don't know the contents of a
	// GlobalNetworkSet, but know it should not be deleted.
	NoGC(*v3.GlobalNetworkSet)

	// RegisterFailFunc registers a function the controller should call when
	// the given key fails to sync, so that a rescheduled pull can be triggered.
	RegisterFailFunc(string, func())

	// Run starts synching GlobalNetworkSets.  All required sets should be added
	// or marked NoGC() before calling run, as any extra will be deleted by the
	// controller.
	Run(context.Context)
}

type controller struct {
	client   v3client.GlobalNetworkSetInterface
	local    cache.Store
	remote   cache.Indexer
	informer cache.Controller
	queue    workqueue.RateLimitingInterface

	noGC    map[string]struct{}
	gcMutex sync.RWMutex

	failFuncs map[string]func()
	ffMutex   sync.RWMutex
}

// Wrapper for clientset errors, used in retry processing.
type clientsetError struct {
	e error
}

func NewController(client v3client.GlobalNetworkSetInterface) Controller {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			// We only care about GlobalNetworkSets created by this controller
			options.LabelSelector = LabelKey + " = " + LabelValue
			return client.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = LabelKey + " = " + LabelValue
			return client.Watch(options)
		},
	}
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	remote, informer := cache.NewIndexerInformer(lw, &v3.GlobalNetworkSet{}, DefaultResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	local := cache.NewStore(cache.MetaNamespaceKeyFunc)
	return &controller{local: local, queue: queue, remote: remote, informer: informer, noGC: make(map[string]struct{})}
}

func (c *controller) Add(s *v3.GlobalNetworkSet) {
	ss := s.DeepCopy()

	// The "creator" key ensures this object will be watched/listed by
	ss.Labels[LabelKey] = LabelValue
	err := c.local.Add(ss)
	if err != nil {
		// Add to local cache only returns error if we fail to extract a key,
		// which is a bug if it ever happens.
		panic(err)
	}
	key, err := cache.MetaNamespaceKeyFunc(ss)
	if err != nil {
		panic(err)
	}
	c.queue.Add(key)
}

func (c *controller) Delete(s *v3.GlobalNetworkSet) {
	// don't bother copying, since we won't keep a reference to the Set
	err := c.local.Delete(s)
	if err != nil {
		// Delete from local cache only returns error if we fail to extract a key,
		// which is a bug if it ever happens.
		panic(err)
	}
	key, err := cache.MetaNamespaceKeyFunc(s)
	if err != nil {
		panic(err)
	}

	// Mark as safe to garbage collect
	c.gcMutex.Lock()
	delete(c.noGC, key)
	c.gcMutex.Unlock()

	// Don't notify puller of failures any more, since the GNS is no longer
	// needed.
	c.ffMutex.Lock()
	delete(c.failFuncs, key)
	c.ffMutex.Unlock()

	c.queue.Add(key)
}

func (c *controller) NoGC(s *v3.GlobalNetworkSet) {
	// don't bother copying, since we're only going to extract a key.
	key, err := cache.MetaNamespaceKeyFunc(s)
	if err != nil {
		panic(err)
	}
	c.gcMutex.Lock()
	defer c.gcMutex.Unlock()
	c.noGC[key] = struct{}{}
	// don't add the Set to the queue.  NoGC just prevents garbage collection,
	// but doesn't trigger any direct action.
}

func (c *controller) RegisterFailFunc(key string, f func()) {
	c.ffMutex.Lock()
	defer c.ffMutex.Unlock()
	c.failFuncs[key] = f
}

func (c *controller) Run(ctx context.Context) {

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Info("Starting GlobalNetworkSet controller")

	go c.informer.Run(ctx.Done())

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		// WaitForCacheSync returns false if the context expires before sync is successful.
		// If that happens, the controller is no longer needed, so just log the error.
		log.Error("Failed to sync GlobalNetworkSet controller")
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping GlobalNetworkSet controller")
			return
		default:
			c.processNextItem()
		}
	}
}

func (c *controller) processNextItem() {
	item, shutdown := c.queue.Get()
	if shutdown {
		log.Info("GlobalNetworkSet workqueue shut down")
		return
	}
	defer c.queue.Done(item)
	key := item.(string)

	il, okl, err := c.local.GetByKey(key)
	if err != nil {
		// Local cache should never error.
		panic(err)
	}
	ir, okr, err := c.remote.GetByKey(key)
	if err != nil {
		// Remote is a cache and should never error.
		panic(err)
	}
	defer c.handleErr(key)
	switch {
	case okl && okr:
		// Local and remote copies exist.  Are they identical?
		sl := il.(*v3.GlobalNetworkSet)
		sr := ir.(*v3.GlobalNetworkSet)
		if setIdentical(sl, sr) {
			return
		} else {
			c.update(sl)
		}
	case okl && !okr:
		// Local exists, but remote does not.
		sl := il.(*v3.GlobalNetworkSet)
		c.create(sl)
	case !okl && okr:
		// Local does not exist, but remote does.
		if c.okToGC(key) {
			sr := ir.(*v3.GlobalNetworkSet)
			c.delete(sr)
		}
	case !okl && !okr:
		// Neither local nor remote exist
		return
	}
}

// handleErr recovers from panics adding, deleting, or updating resources on
// the remote API Server.
func (c *controller) handleErr(key string) {
	e := recover()
	if e == nil {
		// Forget any rate limiting history for this key.
		c.queue.Forget(key)
		return
	}
	// Re-raise if not our "exception" type
	f, ok := e.(clientsetError)
	if !ok {
		panic(e)
	}

	// Try to requeue and reprocess.  But, if we try and fail too many times,
	// give up.  The hourly full resync will try again later.
	if c.queue.NumRequeues(key) < DefaultClientRetries {
		log.WithError(f.e).Errorf("Error handling %v, will retry", key)
		c.queue.AddRateLimited(key)
		return
	}
	// Give up
	c.queue.Forget(key)
	log.WithError(f.e).Errorf("Dropping key %q out of the work queue", key)

	// Inform Puller of failure, if it has registered to be notified.
	c.ffMutex.RLock()
	fn, ok := c.failFuncs[key]
	c.ffMutex.RUnlock()
	if ok {
		fn()
	}
}

func (c *controller) okToGC(key string) bool {
	c.gcMutex.RLock()
	defer c.gcMutex.RUnlock()
	_, ok := c.noGC[key]
	return !ok
}

func (c *controller) update(s *v3.GlobalNetworkSet) {
	_, err := c.client.Update(s)
	if err != nil {
		panic(clientsetError{err})
	}
}

func (c *controller) create(s *v3.GlobalNetworkSet) {
	_, err := c.client.Create(s)
	if err != nil {
		panic(clientsetError{err})
	}
}

func (c *controller) delete(s *v3.GlobalNetworkSet) {
	err := c.client.Delete(s.Name, &metav1.DeleteOptions{})
	if err != nil {
		panic(clientsetError{err})
	}
}

func setIdentical(s1, s2 *v3.GlobalNetworkSet) bool {
	// We only care about labels and spec in this comparison.  This makes sure
	// resource versions, create times, etc don't enter into the comparison.
	if !reflect.DeepEqual(s1.Labels, s2.Labels) {
		return false
	}
	if !reflect.DeepEqual(s1.Spec, s2.Spec) {
		return false
	}
	return true
}
