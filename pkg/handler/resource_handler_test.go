// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handler_test

import (
	"context"
	"reflect"

	"github.com/tigera/deep-packet-inspection/pkg/config"

	"github.com/tigera/deep-packet-inspection/pkg/exec"

	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"

	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/deep-packet-inspection/pkg/handler"
	"github.com/tigera/deep-packet-inspection/pkg/processor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Resource Handler", func() {
	dpiName1 := "dpiKey-test-1"
	dpiName2 := "dpiKey-test-2"
	dpiNs := "test-ns"
	ifaceName1 := "wepKey1-iface"
	ifaceName2 := "random-califace"
	dpiKey1 := model.ResourceKey{
		Name:      dpiName1,
		Namespace: dpiNs,
		Kind:      "DeepPacketInspection",
	}
	dpiKey2 := model.ResourceKey{
		Name:      dpiName2,
		Namespace: dpiNs,
		Kind:      "DeepPacketInspection",
	}
	wepKey1 := model.WorkloadEndpointKey{
		Hostname:       "127.0.0.1",
		OrchestratorID: "k8s",
		WorkloadID:     "test-dpiKey/pod1",
		EndpointID:     "eth0",
	}
	wepKey2 := model.WorkloadEndpointKey{
		Hostname:       "127.0.0.1",
		OrchestratorID: "k8s",
		WorkloadID:     "test-dpiKey/pod2",
		EndpointID:     "eth0",
	}

	It("Adds, updates and deletes DPI and WEP resource", func() {
		mockProcessor1 := &processor.MockProcessor{}
		mockProcessor2 := &processor.MockProcessor{}
		mockSnortProcessor := func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string, snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor {
			if reflect.DeepEqual(dpiKey, dpiKey1) {
				return mockProcessor1
			}
			return mockProcessor2
		}
		ctx := context.Background()
		hndler := handler.NewResourceController(nil, "", &config.Config{}, mockSnortProcessor)

		By("adding a new WorkLoadEndpoint doesn't call processor")
		// during OnUpdate, no calls are made to porocessor as there is no matching selector
		updateWEPResource(ctx, hndler, wepKey1, ifaceName1, map[string]string{"projectcalico.org/namespace": dpiNs})

		By("adding a new DeepPacketInspection resource with WEP that has matching label")
		mockProcessor1.On("Add", ctx, wepKey1, ifaceName1).Return(nil).Times(1)
		updateDPIResource(ctx, hndler, dpiKey1, dpiName1, dpiNs, "k8s-app=='dpiKey'")

		By("adding a second DeepPacketInspection resource that selects all WEPs")
		mockProcessor2.On("Add", ctx, wepKey1, ifaceName1).Return(nil).Times(1)
		updateDPIResource(ctx, hndler, dpiKey2, dpiName1, dpiNs, "all()")

		By("update existing DeepPacketInspection resource no not select any WEPs")
		// Stops snort and removes interfaces that are no longer valid
		mockProcessor1.On("Remove", wepKey1).Return(nil).Times(1)
		mockProcessor1.On("WEPInterfaceCount").Return(0)
		mockProcessor1.On("Close").Return(nil)
		updateDPIResource(ctx, hndler, dpiKey1, dpiName1, dpiNs, "k8s-app=='none'")

		By("update existing WorkLoadEndpoint resource's interface name")
		// if WEP interface changes, the old interface is removed and newer one are be added.
		mockProcessor2.On("Add", ctx, wepKey1, ifaceName2, mock.Anything).Return(nil).Times(1)
		mockProcessor2.On("Remove", wepKey1).Return(nil).Times(2)
		mockProcessor2.On("WEPInterfaceCount").Return(0)
		mockProcessor2.On("Close").Return(nil)
		updateWEPResource(ctx, hndler, wepKey1, ifaceName2, map[string]string{"projectcalico.org/namespace": dpiNs})

		By("delete WorkLoadEndpoint resource")
		deleteResource(ctx, hndler, wepKey1)
	})

	It("Adds DPI resource before WEP resource", func() {
		mockProcessor1 := &processor.MockProcessor{}
		mockSnortProcessor := func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string, snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor {
			return mockProcessor1
		}
		ctx := context.Background()
		hndler := handler.NewResourceController(nil, "", &config.Config{}, mockSnortProcessor)

		By("adding a new DeepPacketInspection doesn't call processor")
		updateDPIResource(ctx, hndler, dpiKey1, dpiName1, dpiNs, "k8s-app=='dpiKey'")

		By("adding a new WorkLoadEndpoint that matches the DPI selector")
		mockProcessor1.On("Add", ctx, wepKey1, ifaceName1).Return(nil).Times(1)
		updateWEPResource(ctx, hndler, wepKey1, ifaceName1, map[string]string{"projectcalico.org/namespace": dpiNs})
	})

	It("Deletes DPI resource before WEP resource", func() {
		mockProcessor1 := &processor.MockProcessor{}
		mockSnortProcessor := func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string, snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor {
			return mockProcessor1
		}
		ctx := context.Background()
		hndler := handler.NewResourceController(nil, "", &config.Config{}, mockSnortProcessor)

		By("adding a new DeepPacketInspection doesn't call processor")
		updateDPIResource(ctx, hndler, dpiKey1, dpiName1, dpiNs, "k8s-app=='dpiKey'")

		By("adding a new WorkLoadEndpoint that matches the DPI selector")
		mockProcessor1.On("Add", ctx, wepKey1, ifaceName1).Return(nil).Times(1)
		updateWEPResource(ctx, hndler, wepKey1, ifaceName1, map[string]string{"projectcalico.org/namespace": dpiNs})

		By("deleting a DPI resource that has snort running")
		mockProcessor1.On("Remove", wepKey1).Return(nil).Times(1)
		mockProcessor1.On("WEPInterfaceCount").Return(0).Times(1)
		mockProcessor1.On("Close").Return(nil).Times(1)
		deleteResource(ctx, hndler, dpiKey1)
	})

	It("Deletes non-existing/non-cached DPI and WEP resource", func() {
		// This scenario might happen if the dpi pods starts after delete DPI/WEP resource is initiated.
		mockProcessor1 := &processor.MockProcessor{}
		mockSnortProcessor := func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string, snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor {
			return mockProcessor1
		}
		ctx := context.Background()
		hndler := handler.NewResourceController(nil, "", &config.Config{}, mockSnortProcessor)

		By("deleting a DPI resource that doesn't have a processor")
		deleteResource(ctx, hndler, dpiKey1)
		//	No calls are made to the mockProcessor
	})

	It("Deletes and adds the same DPI and WEP resource", func() {
		mockProcessor1 := &processor.MockProcessor{}
		mockSnortProcessor := func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string, snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor {
			return mockProcessor1
		}
		ctx := context.Background()
		hndler := handler.NewResourceController(nil, "", &config.Config{}, mockSnortProcessor)

		By("adding a new DeepPacketInspection doesn't call processor")
		updateDPIResource(ctx, hndler, dpiKey1, dpiName1, dpiNs, "k8s-app=='dpiKey'")

		By("adding a new WorkLoadEndpoint that matches the DPI selector")
		mockProcessor1.On("Add", ctx, wepKey1, ifaceName1).Return(nil).Times(1)
		updateWEPResource(ctx, hndler, wepKey1, ifaceName1, map[string]string{"projectcalico.org/namespace": dpiNs})

		By("adding another WorkLoadEndpoint that matches the DPI selector")
		mockProcessor1.On("Add", ctx, wepKey2, ifaceName2).Return(nil).Times(1)
		updateWEPResource(ctx, hndler, wepKey1, ifaceName2, map[string]string{"projectcalico.org/namespace": dpiNs})

		By("deleting a DPI resource that has snort running")
		mockProcessor1.On("Remove", wepKey1).Return(nil).Times(2)
		totalCall := 2
		mockProcessor1.On("WEPInterfaceCount").Run(func(args mock.Arguments) {
			for _, c := range mockProcessor1.ExpectedCalls {
				// After the first call to "Remove" there must be one interface left and zero after second call
				totalCall--
				c.ReturnArguments = mock.Arguments{totalCall}
			}
		}).Times(2)
		mockProcessor1.On("Close").Return(nil).Times(1)
		deleteResource(ctx, hndler, wepKey1)
	})

	It("Doesn't start snort on WEP in a different namespace", func() {
		mockProcessor1 := &processor.MockProcessor{}
		mockSnortProcessor := func(ctx context.Context, calicoClient clientv3.Interface, dpiKey interface{}, nodeName string, snortExecFn exec.Snort, snortAlertFileBasePath string, snortAlertFileSize int, snortCommunityRulesFile string) processor.Processor {
			return mockProcessor1
		}
		ctx := context.Background()
		hndler := handler.NewResourceController(nil, "", &config.Config{}, mockSnortProcessor)

		By("adding a new DeepPacketInspection doesn't call processor")
		updateDPIResource(ctx, hndler, dpiKey1, dpiName1, dpiNs, "k8s-app=='dpiKey'")

		By("adding a new WorkLoadEndpoint that belongs to different namespace")
		mockProcessor1.On("Add", ctx, wepKey1, ifaceName1).Return(nil).Times(1)
		updateWEPResource(ctx, hndler, wepKey1, ifaceName1, map[string]string{"projectcalico.org/namespace": "randomNs"})
		// No calls are made to the processor
	})

})

func deleteResource(ctx context.Context, hndler handler.Handler, key model.Key) {
	hndler.OnUpdate(ctx, []handler.CacheRequest{
		{
			UpdateType: bapi.UpdateTypeKVDeleted,
			KVPair: model.KVPair{
				Key: key,
			},
		},
	})
}

func updateWEPResource(ctx context.Context, hndler handler.Handler, wepKey model.WorkloadEndpointKey, ifaceName string, labels map[string]string) {
	hndler.OnUpdate(ctx, []handler.CacheRequest{
		{
			UpdateType: bapi.UpdateTypeKVNew,
			KVPair: model.KVPair{
				Key: wepKey,
				Value: &model.WorkloadEndpoint{
					Name:   ifaceName,
					Labels: labels,
				},
			},
		},
	})
}

func updateDPIResource(ctx context.Context, hndler handler.Handler, dpiKey1 model.ResourceKey, dpiName string, ns string, selector string) {
	hndler.OnUpdate(ctx, []handler.CacheRequest{
		{
			UpdateType: bapi.UpdateTypeKVNew,
			KVPair: model.KVPair{
				Key: dpiKey1,
				Value: &v3.DeepPacketInspection{
					ObjectMeta: metav1.ObjectMeta{Name: dpiName, Namespace: ns},
					Spec:       v3.DeepPacketInspectionSpec{Selector: selector},
				},
			},
		},
	})
}
