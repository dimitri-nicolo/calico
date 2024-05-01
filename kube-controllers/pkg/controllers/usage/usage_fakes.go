package usage

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8swatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"

	usagev1 "github.com/projectcalico/calico/libcalico-go/lib/apis/usage.tigera.io/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/ipam"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// fakerInformer extends FakeInformer such that it returns ResourceEventHandlerRegistration objects that return true for HasSynced, rather than panicking.
type fakerInformer struct {
	controllertest.FakeInformer
}

type fakeEventHandlerRegistration struct{}

func (f fakeEventHandlerRegistration) HasSynced() bool {
	return true
}

func (f *fakerInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	return fakeEventHandlerRegistration{}, nil
}

// fakeCalicoClient only responds to License GET requests (with a 404).
type fakeCalicoClient struct{}

func (f fakeCalicoClient) LicenseKey() clientv3.LicenseKeyInterface {
	return fakeLicenseKeyClient{}
}

type fakeLicenseKeyClient struct{}

func (f fakeLicenseKeyClient) Get(ctx context.Context, name string, opts options.GetOptions) (*v3.LicenseKey, error) {
	return nil, errors.NewNotFound(schema.GroupResource{Group: v3.Group, Resource: "LicenseKey"}, name)
}

func (f fakeLicenseKeyClient) Create(ctx context.Context, res *v3.LicenseKey, opts options.SetOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeLicenseKeyClient) Update(ctx context.Context, res *v3.LicenseKey, opts options.SetOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeLicenseKeyClient) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeLicenseKeyClient) List(ctx context.Context, opts options.ListOptions) (*v3.LicenseKeyList, error) {
	panic("implement me")
}

func (f fakeLicenseKeyClient) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (f fakeCalicoClient) Create(ctx context.Context, res *v3.LicenseKey, opts options.SetOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeCalicoClient) Update(ctx context.Context, res *v3.LicenseKey, opts options.SetOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeCalicoClient) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeCalicoClient) Get(ctx context.Context, name string, opts options.GetOptions) (*v3.LicenseKey, error) {
	panic("implement me")
}

func (f fakeCalicoClient) List(ctx context.Context, opts options.ListOptions) (*v3.LicenseKeyList, error) {
	panic("implement me")
}

func (f fakeCalicoClient) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (f fakeCalicoClient) Nodes() clientv3.NodeInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalNetworkPolicies() clientv3.GlobalNetworkPolicyInterface {
	panic("implement me")
}

func (f fakeCalicoClient) NetworkPolicies() clientv3.NetworkPolicyInterface {
	panic("implement me")
}

func (f fakeCalicoClient) IPPools() clientv3.IPPoolInterface {
	panic("implement me")
}

func (f fakeCalicoClient) IPReservations() clientv3.IPReservationInterface {
	panic("implement me")
}

func (f fakeCalicoClient) Profiles() clientv3.ProfileInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalNetworkSets() clientv3.GlobalNetworkSetInterface {
	panic("implement me")
}

func (f fakeCalicoClient) NetworkSets() clientv3.NetworkSetInterface {
	panic("implement me")
}

func (f fakeCalicoClient) HostEndpoints() clientv3.HostEndpointInterface {
	panic("implement me")
}

func (f fakeCalicoClient) WorkloadEndpoints() clientv3.WorkloadEndpointInterface {
	panic("implement me")
}

func (f fakeCalicoClient) BGPPeers() clientv3.BGPPeerInterface {
	panic("implement me")
}

func (f fakeCalicoClient) BGPFilter() clientv3.BGPFilterInterface {
	panic("implement me")
}

func (f fakeCalicoClient) IPAM() ipam.Interface {
	panic("implement me")
}

func (f fakeCalicoClient) BGPConfigurations() clientv3.BGPConfigurationInterface {
	panic("implement me")
}

func (f fakeCalicoClient) FelixConfigurations() clientv3.FelixConfigurationInterface {
	panic("implement me")
}

func (f fakeCalicoClient) ClusterInformation() clientv3.ClusterInformationInterface {
	panic("implement me")
}

func (f fakeCalicoClient) KubeControllersConfiguration() clientv3.KubeControllersConfigurationInterface {
	panic("implement me")
}

func (f fakeCalicoClient) CalicoNodeStatus() clientv3.CalicoNodeStatusInterface {
	panic("implement me")
}

func (f fakeCalicoClient) IPAMConfig() clientv3.IPAMConfigInterface {
	panic("implement me")
}

func (f fakeCalicoClient) BlockAffinities() clientv3.BlockAffinityInterface {
	panic("implement me")
}

func (f fakeCalicoClient) StagedGlobalNetworkPolicies() clientv3.StagedGlobalNetworkPolicyInterface {
	panic("implement me")
}

func (f fakeCalicoClient) StagedNetworkPolicies() clientv3.StagedNetworkPolicyInterface {
	panic("implement me")
}

func (f fakeCalicoClient) StagedKubernetesNetworkPolicies() clientv3.StagedKubernetesNetworkPolicyInterface {
	panic("implement me")
}

func (f fakeCalicoClient) PolicyRecommendationScopes() clientv3.PolicyRecommendationScopeInterface {
	panic("implement me")
}

func (f fakeCalicoClient) Tiers() clientv3.TierInterface {
	panic("implement me")
}

func (f fakeCalicoClient) UISettingsGroups() clientv3.UISettingsGroupInterface {
	panic("implement me")
}

func (f fakeCalicoClient) UISettings() clientv3.UISettingsInterface {
	panic("implement me")
}

func (f fakeCalicoClient) RemoteClusterConfigurations() clientv3.RemoteClusterConfigurationInterface {
	panic("implement me")
}

func (f fakeCalicoClient) AlertExceptions() clientv3.AlertExceptionInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalAlerts() clientv3.GlobalAlertInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalAlertTemplates() clientv3.GlobalAlertTemplateInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalThreatFeeds() clientv3.GlobalThreatFeedInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalReportTypes() clientv3.GlobalReportTypeInterface {
	panic("implement me")
}

func (f fakeCalicoClient) GlobalReports() clientv3.GlobalReportInterface {
	panic("implement me")
}

func (f fakeCalicoClient) ManagedClusters() clientv3.ManagedClusterInterface {
	panic("implement me")
}

func (f fakeCalicoClient) PacketCaptures() clientv3.PacketCaptureInterface {
	panic("implement me")
}

func (f fakeCalicoClient) DeepPacketInspections() clientv3.DeepPacketInspectionInterface {
	panic("implement me")
}

func (f fakeCalicoClient) ExternalNetworks() clientv3.ExternalNetworkInterface {
	panic("implement me")
}

func (f fakeCalicoClient) EgressGatewayPolicy() clientv3.EgressGatewayPolicyInterface {
	panic("implement me")
}

func (f fakeCalicoClient) SecurityEventWebhook() clientv3.SecurityEventWebhookInterface {
	panic("implement me")
}

func (f fakeCalicoClient) EnsureInitialized(ctx context.Context, calicoVersion, cnxVersion, clusterType string) error {
	panic("implement me")
}

// errorReturningFakeRuntimeClient responds only to Create requests with the specified err, unless the resolveAtRequestCount has been reached.
type errorReturningFakeRuntimeClient struct {
	requestCount          int
	resolveAtRequestCount *int
	err                   error
}

func (e *errorReturningFakeRuntimeClient) Create(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.CreateOption) error {
	e.requestCount++
	if e.resolveAtRequestCount == nil || e.requestCount < *e.resolveAtRequestCount {
		return e.err
	} else {
		return nil
	}
}

func (e *errorReturningFakeRuntimeClient) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) List(ctx context.Context, list ctrlclient.ObjectList, opts ...ctrlclient.ListOption) error {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) Delete(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) Update(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.UpdateOption) error {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) Patch(ctx context.Context, obj ctrlclient.Object, patch ctrlclient.Patch, opts ...ctrlclient.PatchOption) error {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) DeleteAllOf(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.DeleteAllOfOption) error {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) Status() ctrlclient.SubResourceWriter {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) SubResource(subResource string) ctrlclient.SubResourceClient {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) Scheme() *runtime.Scheme {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) RESTMapper() meta.RESTMapper {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	panic("implement me")
}

func (e *errorReturningFakeRuntimeClient) Watch(ctx context.Context, obj ctrlclient.ObjectList, opts ...ctrlclient.ListOption) (watch.Interface, error) {
	panic("implement me")
}

// callCountingObjectTracker tracker simply tracks calls made to the datastore.
type callCountingObjectTracker struct {
	creates int
	updates int
	lists   int
	deletes int
	watches int
}

func (c *callCountingObjectTracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	c.creates++
	return nil
}

func (c *callCountingObjectTracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	c.updates++
	return nil
}

func (c *callCountingObjectTracker) List(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind, ns string) (runtime.Object, error) {
	c.lists++
	return nil, nil
}

func (c *callCountingObjectTracker) Delete(gvr schema.GroupVersionResource, ns, name string) error {
	c.deletes++
	return nil
}

func (c *callCountingObjectTracker) Watch(gvr schema.GroupVersionResource, ns string) (k8swatch.Interface, error) {
	c.watches++
	return nil, nil
}

func (c *callCountingObjectTracker) Add(obj runtime.Object) error {
	panic("implement me")
}

func (c *callCountingObjectTracker) Get(gvr schema.GroupVersionResource, ns, name string) (runtime.Object, error) {
	panic("implement me")
}

func (c *callCountingObjectTracker) noCallsMade() bool {
	return *c == callCountingObjectTracker{}
}

func (c *callCountingObjectTracker) clear() {
	*c = callCountingObjectTracker{}
}

// createScheme creates the scheme used for LicenseUsageReports.
func createScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(usagev1.UsageGroupVersion, &usagev1.LicenseUsageReport{}, &usagev1.LicenseUsageReportList{})
	metav1.AddToGroupVersion(scheme, usagev1.UsageGroupVersion)
	return scheme
}
