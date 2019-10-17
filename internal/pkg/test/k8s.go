package test

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	authn "k8s.io/api/authentication/v1"
	authz "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	apiv3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	calicofake "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/fake"
	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

const (
	// Jane is a generic username to be used in testing
	Jane = "jane"
	// Bob is a generic username to be used in testing
	Bob = "bob"
	// AnyUser is a generic username to be used in testing
	AnyUser = "anyUser"
	// Developers is a generic group name to be used in testing
	Developers = "developers"
	// JaneBearerToken is the Bearer token associated with Jane
	JaneBearerToken = "jane'sToken"
	// BobBearerToken is the Bearer token associated with Jane
	BobBearerToken = "bob'sToken"
	// JanePassword is the password associated with Jane
	JanePassword = "jane:password"
	// BobPassword is the password associated with Bob
	BobPassword = "bob:password"
	// AnyUserPassword is the password associated with AnyUser
	AnyUserPassword = "anyUser:password"
)

type k8sFake = fake.Clientset
type calicoFake = clientv3.ProjectcalicoV3Interface

// K8sFakeClient is the actual client
type K8sFakeClient struct {
	*k8sFake
	calicoFake

	calicoFakeCtrl *k8stesting.Fake

	clusters managedClusters
	reviews  tokenReviews
}

// FakeK8sClientGenerator maps K8sFakeClients to usernames
type FakeK8sClientGenerator struct {
	sync.Mutex
	apis map[string]*K8sFakeClient
}

// NewFakeK8sClientGenerator generates K8sFakeClients to access K8s API
func NewFakeK8sClientGenerator() *FakeK8sClientGenerator {
	return &FakeK8sClientGenerator{apis: make(map[string]*K8sFakeClient)}
}

// Generate returns a K8sFakeClient that maps to an user. If an user is not found an error will be returned
func (apiGen *FakeK8sClientGenerator) Generate(user string, password string) (k8s.Interface, error) {
	k8s, found := apiGen.get(user)

	if !found {
		return nil, errors.New("Could not generate api")
	}

	return k8s, nil
}

// AddJaneAccessReview mocks k8s authentication response for Jane access to any resource.
// The default response will be to allow Jane to access it
// Expect username to match Jane
func (apiGen *FakeK8sClientGenerator) AddJaneAccessReview() {
	k8sFake := NewK8sSimpleFakeClient(nil, nil)
	k8sFake.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		accessReview := &authz.SelfSubjectAccessReview{Status: authz.SubjectAccessReviewStatus{Allowed: true}}
		return true, accessReview, nil
	})

	apiGen.add(Jane, k8sFake)
}

// AddBobAccessReview mocks k8s authentication response for Bob access to any resource.
// The default response will be to not to allow Bob to access it
// Expect username to match Bob
func (apiGen *FakeK8sClientGenerator) AddBobAccessReview() {
	k8sFake := NewK8sSimpleFakeClient(nil, nil)
	k8sFake.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		accessReview := &authz.SelfSubjectAccessReview{Status: authz.SubjectAccessReviewStatus{Allowed: false}}
		return true, accessReview, nil
	})

	apiGen.add(Bob, k8sFake)
}

// AddErrorAccessReview mocks k8s authentication response for AnyUser's access to any resource.
// The default response will be to error
// Expect username to match AnyUser
func (apiGen *FakeK8sClientGenerator) AddErrorAccessReview() {
	k8sFake := NewK8sSimpleFakeClient(nil, nil)
	k8sFake.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authz.SelfSubjectAccessReview{}, errors.New("Any error")
	})

	apiGen.add(AnyUser, k8sFake)
}

func (apiGen *FakeK8sClientGenerator) add(user string, api *K8sFakeClient) {
	apiGen.Lock()
	defer apiGen.Unlock()

	apiGen.apis[user] = api
}

func (apiGen *FakeK8sClientGenerator) get(user string) (k8s.Interface, bool) {
	apiGen.Lock()
	defer apiGen.Unlock()

	k8s, found := apiGen.apis[user]
	return k8s, found
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
		reviews: tokenReviews{
			reviews: make(map[string]*authn.TokenReview),
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

	fake.k8sFake.PrependReactor("create", "tokenreviews", fake.reviews.Reactor)

	return fake
}

// K8sFake returns the Fake struct to access k8s (re)actions
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
	c.reviews.Add(JaneBearerToken,
		&authn.TokenReview{
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
		})
}

// AddBobIdentity mocks k8s authentication response for Bob
// Expect user not be authenticated
func (c *K8sFakeClient) AddBobIdentity() {
	c.reviews.Add(BobBearerToken,
		&authn.TokenReview{
			Spec: authn.TokenReviewSpec{
				Token: BobBearerToken,
			},
			Status: authn.TokenReviewStatus{
				Authenticated: false,
			},
		})
}

type cluster struct {
	name string
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

func (mc *managedClusters) Add(id, name string) {
	mc.cs[id] = &cluster{
		name: name,
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
		},
	}

	if mc.watcher != nil {
		mc.watcher.Add(cl)
	}
}

func (mc *managedClusters) Update(id string) {
	cl := &apiv3.ManagedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       calicov3.KindManagedCluster,
			APIVersion: calicov3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mc.cs[id].name,
			UID:  k8stypes.UID(id),
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
			Name: mc.cs[id].name,
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
				Name: c.name,
				UID:  k8stypes.UID(id),
			},
		})
	}

	return true, list, nil
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
func (c *K8sFakeClient) AddCluster(id, name string) error {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	if c.clusters.Get(id) != nil {
		return errors.Errorf("cluster id %s already present", id)
	}

	c.clusters.Add(id, name)
	return nil
}

// UpdateCluster modifies a cluster resource
//
// its action is currently void, but will be used when it comes to cert rotation
// etc.
func (c *K8sFakeClient) UpdateCluster(id string) error {
	c.clusters.Lock()
	defer c.clusters.Unlock()

	if c.clusters.Get(id) == nil {
		return errors.Errorf("cluster id %s not present", id)
	}

	c.clusters.Update(id)
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

// AddJaneToken adds JaneBearerToken on the request
func AddJaneToken(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+JaneBearerToken)
}

// AddBobToken adds BobBearerToken on the request
func AddBobToken(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+BobBearerToken)
}

type tokenReviews struct {
	sync.Mutex
	reviews map[string]*authn.TokenReview
}

func (rw *tokenReviews) Get(token string) *authn.TokenReview {
	rw.Lock()
	defer rw.Unlock()

	return rw.reviews[token]
}

func (rw *tokenReviews) Add(token string, review *authn.TokenReview) {
	rw.Lock()
	defer rw.Unlock()

	rw.reviews[token] = review
}

func (rw *tokenReviews) Reactor(action k8stesting.Action) (bool, runtime.Object, error) {
	token := action.(k8stesting.CreateActionImpl).Object.(*authn.TokenReview).Spec.Token
	r := rw.Get(token)
	if r != nil {
		return true, r, nil
	}

	return true, &authn.TokenReview{}, errors.Errorf("token %q does not exist", token)
}
