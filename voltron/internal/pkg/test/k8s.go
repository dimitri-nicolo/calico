package test

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicofake "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	clientv3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

type k8sFake = fake.Clientset
type calicoFake = clientv3.ProjectcalicoV3Interface

// K8sFakeClient is the actual client
type K8sFakeClient struct {
	*k8sFake
	calicoFake

	calicoFakeCtrl *k8stesting.Fake

	clusters managedClusters
}

// NewK8sSimpleFakeClient returns a new aggregated fake client that satisfies
// server.K8sClient interface to access both k8s and calico resources
func NewK8sSimpleFakeClient(k8sObj []runtime.Object, calicoObj []runtime.Object) *K8sFakeClient {
	calico := calicofake.NewSimpleClientset(calicoObj...)

	fake := &K8sFakeClient{
		k8sFake:        fake.NewSimpleClientset(k8sObj...),
		calicoFake:     calico.ProjectcalicoV3(),
		calicoFakeCtrl: &calico.Fake,
		clusters: managedClusters{
			cs:      make(map[string]*cluster),
			watched: make(chan struct{}, 1000), // large enough to accomodate many watch restarts
		},
	}

	calico.Fake.PrependWatchReactor("managedclusters",
		func(action k8stesting.Action) (bool, watch.Interface, error) {
			defer func() { fake.clusters.watched <- struct{}{} }()
			fake.clusters.block.Lock()
			defer fake.clusters.block.Unlock()
			watcher := fake.clusters.newWatcher()
			return k8stesting.DefaultWatchReactor(watcher, nil)(action)
		})
	calico.Fake.PrependReactor("list", "managedclusters", fake.clusters.listReactor)
	calico.Fake.PrependReactor("get", "managedclusters", fake.clusters.getReactor)
	calico.Fake.PrependReactor("update", "managedclusters", fake.clusters.updateReactor)

	return fake
}

// K8sFake returns the Fake struct to access k8s (re)actions
func (c *K8sFakeClient) K8sFake() *k8stesting.Fake {
	return &c.k8sFake.Fake
}

// CalicoFake returns the Fake struct to access the calico (re)actions
func (c *K8sFakeClient) CalicoFake() *k8stesting.Fake {
	return c.calicoFakeCtrl
}

type cluster struct {
	id                      string
	annotations             map[string]string
	managedClusterConnected calicov3.ManagedClusterStatusValue
}

type managedClusters struct {
	sync.Mutex
	block   sync.Mutex
	version int
	cs      map[string]*cluster
	watcher *mcWatcher
	watched chan struct{}
}

type mcWatcher struct {
	*watch.FakeWatcher
	stop func()
}

func (w *mcWatcher) Stop() {
	w.stop()
}

func (mc *managedClusters) newWatcher() watch.Interface {
	mc.Lock()
	defer mc.Unlock()

	mc.watcher = &mcWatcher{
		FakeWatcher: watch.NewFakeWithChanSize(100, true),
		stop: func() {
			mc.Lock()
			defer mc.Unlock()
			mc.watcher = nil
		},
	}

	return mc.watcher
}

func (mc *managedClusters) versionStr() string {
	return fmt.Sprintf("%d", mc.version)
}

func (mc *managedClusters) Get(id string) *cluster {
	return mc.cs[id]
}

func (mc *managedClusters) Add(id, name string, annotations map[string]string, status calicov3.ManagedClusterStatusValue) {
	// We now use the resource name as the ID
	mc.cs[name] = &cluster{
		id:                      name,
		annotations:             annotations,
		managedClusterConnected: status,
	}

	mc.version++

	cl := &apiv3.ManagedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       calicov3.KindManagedCluster,
			APIVersion: calicov3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			UID:             k8stypes.UID(id),
			ResourceVersion: mc.versionStr(),
			Annotations:     annotations,
		},
		Status: calicov3.ManagedClusterStatus{
			Conditions: []calicov3.ManagedClusterStatusCondition{
				{
					Status: status,
					Type:   "ManagedClusterConnected",
				},
			},
		},
	}

	if mc.watcher != nil {
		mc.watcher.Add(cl)
	}
}

func (mc *managedClusters) Update(id string, annotations map[string]string) {
	cl := &apiv3.ManagedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       calicov3.KindManagedCluster,
			APIVersion: calicov3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        mc.cs[id].id,
			UID:         k8stypes.UID(id),
			Annotations: annotations,
		},
	}

	mc.watcher.Modify(cl)
}

func (mc *managedClusters) Delete(id string) {
	cl := &apiv3.ManagedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       calicov3.KindManagedCluster,
			APIVersion: calicov3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mc.cs[id].id,
			UID:  k8stypes.UID(id),
		},
	}

	delete(mc.cs, id)

	if mc.watcher != nil {
		mc.watcher.Delete(cl)
	}
}

func (mc *managedClusters) StopWatcher() {
	if mc.watcher != nil {
		w := mc.watcher
		mc.watcher = nil
		w.FakeWatcher.Stop()
	}
}

func (mc *managedClusters) listReactor(action k8stesting.Action) (
	handled bool, ret runtime.Object, err error) {

	list := &apiv3.ManagedClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       calicov3.KindManagedClusterList,
			APIVersion: calicov3.GroupVersionCurrent,
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: mc.versionStr(),
		},
	}

	mc.Lock()
	defer mc.Unlock()

	for id, c := range mc.cs {
		list.Items = append(list.Items, apiv3.ManagedCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       calicov3.KindManagedCluster,
				APIVersion: calicov3.GroupVersionCurrent,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.id,
				UID:  k8stypes.UID(id),
			},
			Status: calicov3.ManagedClusterStatus{
				Conditions: []calicov3.ManagedClusterStatusCondition{
					{
						Status: c.managedClusterConnected,
						Type:   "ManagedClusterConnected",
					},
				},
			},
		})
	}

	return true, list, nil
}

func (mc *managedClusters) getReactor(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
	mc.Lock()
	defer mc.Unlock()

	givenName := action.(k8stesting.GetActionImpl).Name
	cluster, ok := mc.cs[givenName]

	if !ok {
		return true, nil, errors.Errorf("Missing mocked cluster")
	}

	managedCluster := &apiv3.ManagedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       calicov3.KindManagedCluster,
			APIVersion: calicov3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        givenName,
			UID:         k8stypes.UID(cluster.id),
			Annotations: cluster.annotations,
		},
		Status: calicov3.ManagedClusterStatus{
			Conditions: []calicov3.ManagedClusterStatusCondition{
				{
					Status: cluster.managedClusterConnected,
					Type:   "ManagedClusterConnected",
				},
			},
		},
	}

	return true, managedCluster, nil
}

func (mc *managedClusters) updateReactor(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
	mc.Lock()
	defer mc.Unlock()

	managedCluster := action.(k8stesting.UpdateActionImpl).GetObject().(*apiv3.ManagedCluster)
	_, ok := mc.cs[managedCluster.Name]

	if !ok {
		return true, nil, errors.Errorf("Missing mocked cluster")
	}

	var managedClusterConnectedStatus calicov3.ManagedClusterStatusValue
	for _, c := range managedCluster.Status.Conditions {
		if c.Type == calicov3.ManagedClusterStatusTypeConnected {
			managedClusterConnectedStatus = c.Status
		}
	}

	mc.cs[managedCluster.Name] = &cluster{
		id:                      managedCluster.Name,
		annotations:             managedCluster.Annotations,
		managedClusterConnected: managedClusterConnectedStatus,
	}

	return true, managedCluster, nil
}

// WaitForManagedClustersWatched returns when the ManagedClusters start being
// watched to sync with tests
func (c *K8sFakeClient) WaitForManagedClustersWatched() {
	c.clusters.Lock()
	w := c.clusters.watched
	c.clusters.Unlock()
	<-w
}

// AddCluster adds a cluster resource
func (c *K8sFakeClient) AddCluster(id, name string, annotations map[string]string, status ...calicov3.ManagedClusterStatusValue) error {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	var connectionStatus calicov3.ManagedClusterStatusValue
	if len(status) == 1 {
		connectionStatus = status[0]
	} else {
		connectionStatus = calicov3.ManagedClusterStatusValueUnknown
	}
	if c.clusters.Get(id) != nil {
		return errors.Errorf("cluster id %s already present", id)
	}

	c.clusters.Add(id, name, annotations, connectionStatus)
	return nil
}

// UpdateCluster modifies a cluster resource
//
// its action is currently void, but will be used when it comes to cert rotation
// etc.
func (c *K8sFakeClient) UpdateCluster(id string, annotations map[string]string) error {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	if c.clusters.Get(id) == nil {
		return errors.Errorf("cluster id %s not present", id)
	}

	c.clusters.cs[id].annotations = annotations
	c.clusters.Update(id, annotations)
	return nil
}

// DeleteCluster remove the cluster resource
func (c *K8sFakeClient) DeleteCluster(id string) error {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	if c.clusters.Get(id) == nil {
		return errors.Errorf("cluster id %s not present", id)
	}

	c.clusters.Delete(id)
	return nil
}

// BlockWatches sync clients Watch call until UnblockWatches is called
func (c *K8sFakeClient) BlockWatches() {
	c.clusters.block.Lock()
}

// UnblockWatches allows clients Watch call to proceed
func (c *K8sFakeClient) UnblockWatches() {
	c.clusters.block.Unlock()
}

// BreakWatcher stops the watcher so that the client sees the event channel to
// close
func (c *K8sFakeClient) BreakWatcher() {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	c.clusters.watched = make(chan struct{}, 1000)
	c.clusters.StopWatcher()
}
