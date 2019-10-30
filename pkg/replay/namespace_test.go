// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package replay_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/libcalico-go/lib/resources"
	. "github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/lma/pkg/list"
	"github.com/tigera/lma/pkg/api"
)

var (
	now           = time.Now()
	nowMinus24Hrs = now.Add(-24 * time.Hour)
	nowMinus48Hrs = nowMinus24Hrs.Add(-24 * time.Hour)
	namespace1    = "namespace1"
	namespace2    = "namespace2"

	gnp1 = apiv3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "gnp1",
			ResourceVersion: "1",
		},
	}
	sgnp1 = apiv3.StagedGlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoStagedGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "gnp1",
			ResourceVersion: "1",
		},
	}
	gns1 = apiv3.GlobalNetworkSet{
		TypeMeta: resources.TypeCalicoGlobalNetworkSets,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "gns1",
			ResourceVersion: "1",
		},
	}
	hep1 = apiv3.HostEndpoint{
		TypeMeta: resources.TypeCalicoHostEndpoints,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "hep1",
			ResourceVersion: "1",
		},
	}
	np1 = apiv3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "np1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	snp1 = apiv3.StagedNetworkPolicy{
		TypeMeta: resources.TypeCalicoStagedNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "np1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	np2 = apiv3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "np2",
			Namespace:       namespace2,
			ResourceVersion: "1",
		},
	}
	tier1 = apiv3.Tier{
		TypeMeta: resources.TypeCalicoTiers,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tier1",
			ResourceVersion: "1",
		},
	}
	ep1 = corev1.Endpoints{
		TypeMeta: resources.TypeK8sEndpoints,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "svc1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	ep2 = corev1.Endpoints{
		TypeMeta: resources.TypeK8sEndpoints,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "svc2",
			Namespace:       namespace2,
			ResourceVersion: "1",
		},
	}
	ns1 = corev1.Namespace{
		TypeMeta: resources.TypeK8sNamespaces,
		ObjectMeta: metav1.ObjectMeta{
			Name:            namespace1,
			ResourceVersion: "1",
		},
	}
	ns2 = corev1.Namespace{
		TypeMeta: resources.TypeK8sNamespaces,
		ObjectMeta: metav1.ObjectMeta{
			Name:            namespace2,
			ResourceVersion: "1",
		},
	}
	knp1 = networkingv1.NetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "knp1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	sknp1 = apiv3.StagedKubernetesNetworkPolicy{
		TypeMeta: resources.TypeCalicoStagedKubernetesNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "knp1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	knp2 = networkingv1.NetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "knp2",
			Namespace:       namespace2,
			ResourceVersion: "1",
		},
	}
	pod1 = corev1.Pod{
		TypeMeta: resources.TypeK8sPods,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "pod1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	pod2 = corev1.Pod{
		TypeMeta: resources.TypeK8sPods,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "pod2",
			Namespace:       namespace2,
			ResourceVersion: "1",
		},
	}
	sa1 = corev1.ServiceAccount{
		TypeMeta: resources.TypeK8sServiceAccounts,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "sa1",
			Namespace:       namespace1,
			ResourceVersion: "1",
		},
	}
	sa2 = corev1.ServiceAccount{
		TypeMeta: resources.TypeK8sServiceAccounts,
		ObjectMeta: metav1.ObjectMeta{
			Name:            "sa2",
			Namespace:       namespace2,
			ResourceVersion: "1",
		},
	}
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type mockNamespaceTestClient struct {
	deleteBeforeSync bool
	getAuditCalls    int
}

func (c *mockNamespaceTestClient) RetrieveList(tm metav1.TypeMeta, from, to *time.Time, sortAscendingTime bool) (*list.TimestampedResourceList, error) {
	var l resources.ResourceList
	switch tm {
	case resources.TypeCalicoGlobalNetworkPolicies:
		l = &apiv3.GlobalNetworkPolicyList{
			TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
			Items:    []apiv3.GlobalNetworkPolicy{gnp1},
		}
	case resources.TypeCalicoStagedGlobalNetworkPolicies:
		l = &apiv3.StagedGlobalNetworkPolicyList{
			TypeMeta: resources.TypeCalicoStagedGlobalNetworkPolicies,
			Items:    []apiv3.StagedGlobalNetworkPolicy{sgnp1},
		}
	case resources.TypeCalicoGlobalNetworkSets:
		l = &apiv3.GlobalNetworkSetList{
			TypeMeta: resources.TypeCalicoGlobalNetworkSets,
			Items:    []apiv3.GlobalNetworkSet{gns1},
		}
	case resources.TypeCalicoHostEndpoints:
		l = &apiv3.HostEndpointList{
			TypeMeta: resources.TypeCalicoHostEndpoints,
			Items:    []apiv3.HostEndpoint{hep1},
		}
	case resources.TypeCalicoNetworkPolicies:
		l = &apiv3.NetworkPolicyList{
			TypeMeta: resources.TypeCalicoNetworkPolicies,
			Items:    []apiv3.NetworkPolicy{np1, np2},
		}
	case resources.TypeCalicoStagedNetworkPolicies:
		l = &apiv3.StagedNetworkPolicyList{
			TypeMeta: resources.TypeCalicoStagedNetworkPolicies,
			Items:    []apiv3.StagedNetworkPolicy{snp1},
		}
	case resources.TypeCalicoTiers:
		l = &apiv3.TierList{
			TypeMeta: resources.TypeCalicoTiers,
			Items:    []apiv3.Tier{tier1},
		}
	case resources.TypeK8sEndpoints:
		l = &corev1.EndpointsList{
			TypeMeta: resources.TypeK8sEndpoints,
			Items:    []corev1.Endpoints{ep1, ep2},
		}
	case resources.TypeK8sNamespaces:
		l = &corev1.NamespaceList{
			TypeMeta: resources.TypeK8sNamespaces,
			Items:    []corev1.Namespace{ns1, ns2},
		}
	case resources.TypeK8sNetworkPolicies:
		l = &networkingv1.NetworkPolicyList{
			TypeMeta: resources.TypeK8sNetworkPolicies,
			Items:    []networkingv1.NetworkPolicy{knp1, knp2},
		}
	case resources.TypeCalicoStagedKubernetesNetworkPolicies:
		l = &apiv3.StagedKubernetesNetworkPolicyList{
			TypeMeta: resources.TypeCalicoStagedKubernetesNetworkPolicies,
			Items:    []apiv3.StagedKubernetesNetworkPolicy{sknp1},
		}
	case resources.TypeK8sPods:
		l = &corev1.PodList{
			TypeMeta: resources.TypeK8sPods,
			Items:    []corev1.Pod{pod1, pod2},
		}
	case resources.TypeK8sServiceAccounts:
		l = &corev1.ServiceAccountList{
			TypeMeta: resources.TypeK8sServiceAccounts,
			Items:    []corev1.ServiceAccount{sa1, sa2},
		}
	default:
		panic(fmt.Errorf("Unexpected resource type: %v", tm))
	}

	return &list.TimestampedResourceList{
		ResourceList:              l,
		RequestStartedTimestamp:   metav1.Time{nowMinus48Hrs},
		RequestCompletedTimestamp: metav1.Time{nowMinus48Hrs},
	}, nil
}

func (c *mockNamespaceTestClient) StoreList(tm metav1.TypeMeta, resourceList *list.TimestampedResourceList) error {
	panic(fmt.Errorf("StoreList should not be called from replayer: %v", tm))
}

func (c *mockNamespaceTestClient) GetAuditEvents(cxt context.Context, start *time.Time, end *time.Time) <-chan *api.AuditEventResult {
	// We expect this to be called twice. Once to get determine the initial start of day snapshot and once to determine
	// the changing events within the requested interval.
	c.getAuditCalls++

	Expect(c.getAuditCalls).To(BeNumerically("<", 3))

	ch := make(chan *api.AuditEventResult, 2)

	// Return a namespace deletion for namespace1 depending on when the test wants the delete sent.
	if (c.getAuditCalls == 1 && c.deleteBeforeSync) || (c.getAuditCalls == 2 && !c.deleteBeforeSync) {
		ch <- &api.AuditEventResult{
			Event: &auditv1.Event{
				Stage: auditv1.StageResponseComplete,
				Verb:  api.EventVerbDelete,
				ObjectRef: &auditv1.ObjectReference{
					Resource:        "namespaces",
					Name:            namespace1,
					APIGroup:        "",
					APIVersion:      "v1",
					ResourceVersion: "",
				},
			},
		}
	}

	// Close the channel, the current contents of the channel can still be read.
	close(ch)

	return ch
}

var _ = Describe("Replay namespace deletion", func() {
	var cb *mockCallbacks
	var replayer syncer.Starter
	var client *mockNamespaceTestClient
	BeforeEach(func() {
		cb = &mockCallbacks{}
		client = &mockNamespaceTestClient{}
		replayer = New(nowMinus24Hrs, now, client, client, cb)
	})

	It("should handle namespace deletion before the in-sync", func() {
		By("running the replayed with a namespace deletion in the first event query")
		client.deleteBeforeSync = true
		replayer.Start(context.Background())

		By("Checking for status update complete")
		Expect(cb.statusUpdates).To(ContainElement(syncer.NewStatusUpdateComplete()))

		By("Checking for the expected updates")
		// There should be one update of each kind and none from namespace 2. We should get no deletion events.
		Expect(cb.updates).To(HaveLen(11))
		checkUpdates(cb.updates, syncer.UpdateTypeSet, []resources.Resource{
			&gnp1, &sgnp1, &gns1, &tier1, &hep1, &np2, &knp2, &ep2, &pod2, &sa2, &ns2,
		})
	})

	It("should handle namespace deletion after the in-sync", func() {
		By("running the replayed with a namespace deletion in the second event query")
		client.deleteBeforeSync = false
		replayer.Start(context.Background())

		By("Checking for status update complete")
		Expect(cb.statusUpdates).To(ContainElement(syncer.NewStatusUpdateComplete()))

		By("Checking for the expected updates")
		// There should be one update of each resource, followed by delete events for everything in namespace2.
		Expect(cb.updates).To(HaveLen(27))
		checkUpdates(cb.updates[:19], syncer.UpdateTypeSet, []resources.Resource{
			&gnp1, &sgnp1, &gns1, &tier1, &hep1, &np1, &snp1, &np2, &knp1, &sknp1, &knp2, &ep1, &ep2, &pod1, &pod2, &sa1, &sa2, &ns1, &ns2,
		})
		checkUpdates(cb.updates[19:], syncer.UpdateTypeDeleted, []resources.Resource{
			&np1, &snp1, &knp1, &sknp1, &ep1, &pod1, &sa1, &ns1,
		})
	})
})

func checkUpdates(updates []syncer.Update, expectedType syncer.UpdateType, expectedResources []resources.Resource) {
	r := []resources.Resource{}
	for i, update := range updates {
		Expect(update.Type).To(Equal(expectedType), fmt.Sprintf("Unexpected type at index %d", i))
		r = append(r, update.Resource)
	}
	Expect(r).Should(ConsistOf(expectedResources))
}
