// Copyright 2021 Tigera Inc. All rights reserved.
package panorama

import (
	"context"
	"errors"
	"sync"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	clientv3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

const FakeGnsClientErrorKey = "error-key"
const FakeGnsClientSomeErrorMessage = "some error"

// FakeDagCalicoClient is a fake client for use in the dynamic address groups tests.
type FakeDagCalicoClient struct {
	gnsClient clientv3.GlobalNetworkSetInterface
}

func NewFakeDagCalicoClient(input []v3.GlobalNetworkSet) *FakeDagCalicoClient {
	gnsClient := FakeGnsClient{
		gnsm: make(map[string]v3.GlobalNetworkSet),
	}

	for _, gns := range input {
		gnsClient.gnsm[gns.Name] = gns
	}

	f := FakeDagCalicoClient{
		gnsClient: &gnsClient,
	}

	return &f
}

func (c FakeDagCalicoClient) RESTClient() rest.Interface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) AlertExceptions() clientv3.AlertExceptionInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) AuthenticationReviews() clientv3.AuthenticationReviewInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) AuthorizationReviews() clientv3.AuthorizationReviewInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) BGPConfigurations() clientv3.BGPConfigurationInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) BGPPeers() clientv3.BGPPeerInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) CalicoNodeStatuses() clientv3.CalicoNodeStatusInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) ClusterInformations() clientv3.ClusterInformationInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) DeepPacketInspections(namespace string) clientv3.DeepPacketInspectionInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) FelixConfigurations() clientv3.FelixConfigurationInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) GlobalAlerts() clientv3.GlobalAlertInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) GlobalAlertTemplates() clientv3.GlobalAlertTemplateInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) GlobalNetworkPolicies() clientv3.GlobalNetworkPolicyInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) GlobalNetworkSets() clientv3.GlobalNetworkSetInterface {
	return c.gnsClient
}

func (c FakeDagCalicoClient) GlobalReports() clientv3.GlobalReportInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) GlobalReportTypes() clientv3.GlobalReportTypeInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) GlobalThreatFeeds() clientv3.GlobalThreatFeedInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) HostEndpoints() clientv3.HostEndpointInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) IPPools() clientv3.IPPoolInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) IPReservations() clientv3.IPReservationInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) KubeControllersConfigurations() clientv3.KubeControllersConfigurationInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) LicenseKeys() clientv3.LicenseKeyInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) ManagedClusters() clientv3.ManagedClusterInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) NetworkPolicies(namespace string) clientv3.NetworkPolicyInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) NetworkSets(namespace string) clientv3.NetworkSetInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) PacketCaptures(namespace string) clientv3.PacketCaptureInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) Profiles() clientv3.ProfileInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) RemoteClusterConfigurations() clientv3.RemoteClusterConfigurationInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) StagedGlobalNetworkPolicies() clientv3.StagedGlobalNetworkPolicyInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) StagedKubernetesNetworkPolicies(namespace string) clientv3.StagedKubernetesNetworkPolicyInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) StagedNetworkPolicies(namespace string) clientv3.StagedNetworkPolicyInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) Tiers() clientv3.TierInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) UISettings() clientv3.UISettingsInterface {
	panic("not implemented") // TODO: Implement
}

func (c FakeDagCalicoClient) UISettingsGroups() clientv3.UISettingsGroupInterface {
	panic("not implemented") // TODO: Implement
}

// FakeGnsClient implements the client GlobalNetworkSetInterface for testing purposes.
type FakeGnsClient struct {
	sync.Mutex
	gnsm map[string]v3.GlobalNetworkSet
}

func (f *FakeGnsClient) Create(ctx context.Context, globalNetworkSet *v3.GlobalNetworkSet, opts metav1.CreateOptions) (*v3.GlobalNetworkSet, error) {
	f.Lock()
	defer f.Unlock()

	key := globalNetworkSet.Name
	f.gnsm[key] = *globalNetworkSet
	item := f.gnsm[key]

	return &item, nil
}

// Update defines a map value even the key is not present.
func (f *FakeGnsClient) Update(ctx context.Context, globalNetworkSet *v3.GlobalNetworkSet, opts metav1.UpdateOptions) (*v3.GlobalNetworkSet, error) {
	f.Lock()
	defer f.Unlock()

	key := globalNetworkSet.Name
	f.gnsm[key] = *globalNetworkSet
	item := f.gnsm[key]

	return &item, nil
}

func (f *FakeGnsClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	f.Lock()
	defer f.Unlock()

	delete(f.gnsm, name)

	return nil
}

func (f *FakeGnsClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("not implemented") // TODO: Implement
}

func (f *FakeGnsClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v3.GlobalNetworkSet, error) {
	f.Lock()
	defer f.Unlock()

	if name == FakeGnsClientErrorKey {
		return &v3.GlobalNetworkSet{}, errors.New(FakeGnsClientSomeErrorMessage)
	}
	if gns, found := f.gnsm[name]; !found {
		return &v3.GlobalNetworkSet{}, kerrors.NewNotFound(schema.GroupResource{}, "NotFound")
	} else {
		return &gns, nil
	}
}

func (f *FakeGnsClient) List(ctx context.Context, opts metav1.ListOptions) (*v3.GlobalNetworkSetList, error) {
	f.Lock()
	defer f.Unlock()

	gnsList := v3.GlobalNetworkSetList{
		Items: make([]v3.GlobalNetworkSet, 0, len(f.gnsm)),
	}
	for _, val := range f.gnsm {
		gnsList.Items = append(gnsList.Items, val)
	}

	return &gnsList, nil
}

func (f *FakeGnsClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("not implemented") // TODO: Implement
}

func (f *FakeGnsClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v3.GlobalNetworkSet, err error) {
	panic("not implemented") // TODO: Implement
}
