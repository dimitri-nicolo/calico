package test

import (
	"net/http"
	"sync"

	"github.com/pkg/errors"
	authn "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	calicofake "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/fake"
	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

const (
	// Jane is a generic username to be used in testing
	Jane = "jane"
	// Developers is a generic group name to be used in testing
	Developers = "developers"
	// JaneBearerToken is the Bearer token associated with Jane
	JaneBearerToken = "Bearer jane'sToken"
	// BobBearerToken is the Bearer token associated with Jane
	BobBearerToken = "Bearer bob'sToken"
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
	}

	fake.clusters.cs = make(map[string]*cluster)
	fake.clusters.watcher = watch.NewFake()

	calico.Fake.PrependWatchReactor("managedclusters",
		k8stesting.DefaultWatchReactor(fake.clusters.watcher, nil))

	return fake
}

// K8sFake returns the Fake struct to acces k8s (re)actions
func (c *K8sFakeClient) K8sFake() *k8stesting.Fake {
	return &c.k8sFake.Fake
}

// CalicoFake retusn the Fake struct to access the calico (re)actions
func (c *K8sFakeClient) CalicoFake() *k8stesting.Fake {
	return c.calicoFakeCtrl
}

// AddJaneIdentity mocks k8s authentication response for Jane
// Expect username to match Jane and groups to match Developers
func (c *K8sFakeClient) AddJaneIdentity() {
	c.k8sFake.PrependReactor(
		"create", "tokenreviews",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			review := &authn.TokenReview{
				Spec: authn.TokenReviewSpec{
					Token: JaneBearerToken,
				},
				Status: authn.TokenReviewStatus{
					Authenticated: true,
					User: authn.UserInfo{
						Username: Jane,
						Groups:   []string{Developers},
					},
				},
			}
			return true, review, nil
		})
}

// AddBobIdentity mocks k8s authentication response for Bob
// Expect user not be authenticated
func (c *K8sFakeClient) AddBobIdentity() {
	c.k8sFake.PrependReactor(
		"create", "tokenreviews",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			review := &authn.TokenReview{
				Spec: authn.TokenReviewSpec{
					Token: BobBearerToken,
				},
				Status: authn.TokenReviewStatus{
					Authenticated: false,
				},
			}
			return true, review, nil
		})
}

type cluster struct {
	name string
}

type managedClusters struct {
	sync.Mutex
	cs      map[string]*cluster
	watcher *watch.FakeWatcher
}

func (mc *managedClusters) Get(id string) *cluster {
	return mc.cs[id]
}

func (mc *managedClusters) Add(id, name string) {
	mc.cs[id] = &cluster{
		name: name,
	}

	cl := apiv3.NewManagedCluster()
	cl.ObjectMeta.Name = name
	cl.ObjectMeta.UID = k8stypes.UID(id)

	mc.watcher.Add(cl)
}

func (mc *managedClusters) Delete(id string) {
	cl := apiv3.NewManagedCluster()
	cl.ObjectMeta.Name = mc.cs[id].name
	cl.ObjectMeta.UID = k8stypes.UID(id)

	delete(mc.cs, id)

	mc.watcher.Delete(cl)
}

// AddCluster adds a cluster resource
func (c *K8sFakeClient) AddCluster(id, name string) error {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	if c.clusters.Get(id) != nil {
		return errors.Errorf("cluster id %s already present", id)
	}

	c.clusters.Add(id, name)
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

// AddJaneToken adds JaneBearerToken on the request
func AddJaneToken(req *http.Request) {
	req.Header.Add("Authorization", JaneBearerToken)
}

// AddBobToken adds BobBearerToken on the request
func AddBobToken(req *http.Request) {
	req.Header.Add("Authorization", BobBearerToken)
}
