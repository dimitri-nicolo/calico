package pip_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/es-proxy/pkg/pip"
)

var (
	r1 = &v3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "np",
		},
		Spec: v3.NetworkPolicySpec{
			Selector: "foobarbaz",
		},
	}
	r2 = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "gnp",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Selector: "foobazbar",
		},
	}
	r3 = &networkingv1.NetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8s-np",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
	r4 = &corev1.Namespace{
		TypeMeta: resources.TypeK8sNamespaces,
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace",
		},
	}
)

var _ = Describe("Test resourcechange unmarshaling and marshaling", func() {
	It("handles flag checks correctly", func() {
		test := []pip.ResourceChange{
			{
				Action:   "update",
				Resource: r1,
			},
			{
				Action:   "create",
				Resource: r2,
			},
			{
				Action:   "delete",
				Resource: r3,
			},
			{
				Action:   "exterminate",
				Resource: r4,
			},
		}

		By("Marshalling a slice of ResourceChange structs")
		j, err := json.Marshal(test)
		Expect(err).NotTo(HaveOccurred())

		By("Unmarshalling the json output")
		var output []pip.ResourceChange
		err = json.Unmarshal(j, &output)
		Expect(err).NotTo(HaveOccurred())

		By("Comparing the data")
		Expect(output).To(Equal(test))
	})
})
