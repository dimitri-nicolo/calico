package pip

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"

	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var (
	defaultTier = &v3.Tier{
		TypeMeta: resources.TypeCalicoTiers,
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}

	knpDefaultDeny = &networkingv1.NetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny",
			Namespace: "ns1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}

	knpAllowAll = &networkingv1.NetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allows",
			Namespace: "ns1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{},
				}},
			}},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}

	ns1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns1",
			Labels: map[string]string{
				"name": "ns1",
			},
		},
	}

	cfgCalcActionBefore = &pipcfg.Config{
		CalculateOriginalAction: true,
	}
)

var _ = Describe("Kubernetes Network Policy PIP tests", func() {
	var ep *policycalc.EndpointCache

	BeforeEach(func() {
		ep = policycalc.NewEndpointCache()
	})

	It("handles kubernetes network policy", func() {
		xc := xrefcache.NewXrefCache(&config.Config{}, func() {})
		xc.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("Adding the default tier and namespace ns1")
		xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(defaultTier),
			Resource:   defaultTier,
		}, {
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(ns1),
			Resource:   ns1,
		}})

		By("Having a single drop all ingress in namespace ns1 policy")
		xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(knpDefaultDeny),
			Resource:   knpDefaultDeny,
		}})
		rdBefore := resourceDataFromXrefCache(xc)

		By("Adding an allow all ingress rule before default-drop")
		xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(knpAllowAll),
			Resource:   knpAllowAll,
		}})
		modified := make(policycalc.ModifiedResources)
		modified.Add(knpAllowAll)
		rdAfter := resourceDataFromXrefCache(xc)

		By("Creating the policy calculators which calculates before and after")
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, ep, rdBefore, rdAfter, modified)

		By("Checking a flow with dest in ns1 is unaffected")
		f := &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Name:      "wep1-*",
				Namespace: "ns1",
				Labels: map[string]string{
					"any": "value",
				},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Name:      "wep2-*",
				Namespace: "ns1",
				Labels: map[string]string{
					"any": "value",
				},
			},
			Action: policycalc.ActionAllow,
		}

		_, before, after := pc.Calculate(f)
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		_, before, after = pc.Calculate(f)
		Expect(before.Destination.Action).To(Equal(policycalc.ActionDeny))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionAllow))
	})
})
