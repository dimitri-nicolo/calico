// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package calicoresources

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	fakecalico "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"

	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
)

var _ = Describe("Tests policy recommendation controller", func() {
	var (
		ctx context.Context

		fakeClient *fakecalico.Clientset
		fakeCoreV1 corev1.CoreV1Interface

		mockClientSet          *lmak8s.MockClientSet
		mockConstructorTesting mockConstructorTestingTNewMockClientSet

		owner metav1.OwnerReference
	)

	Context("MaybeCreatePrivateNetworkSet", func() {
		BeforeEach(func() {
			ctx = context.Background()

			fakeClient = fakecalico.NewSimpleClientset()
			fakeCoreV1 = fakeK8s.NewSimpleClientset().CoreV1()

			mockConstructorTesting = mockConstructorTestingTNewMockClientSet{}
			mockClientSet = lmak8s.NewMockClientSet(mockConstructorTesting)

			mockClientSet.On("ProjectcalicoV3").Return(fakeClient.ProjectcalicoV3())
			mockClientSet.On("CoreV1").Return(fakeCoreV1)

			owner = metav1.OwnerReference{
				Kind: "GNS-Owner",
				Name: "GNS-Owner-Name",
			}
		})

		It("Creates a new 'private-network' if that doesn't exist", func() {
			expectedGnsName := "private-network"
			expectedGnsOwnerKind := "GNS-Owner"
			expectedGnsOwnerName := "GNS-Owner-Name"

			err := MaybeCreatePrivateNetworkSet(ctx, mockClientSet.ProjectcalicoV3(), owner)
			Expect(err).To(BeNil())

			gns, err := mockClientSet.ProjectcalicoV3().GlobalNetworkSets().Get(ctx, expectedGnsName, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(gns.Labels["projectcalico.org/kind"]).To(Equal("NetworkSet"))
			Expect(gns.Labels["projectcalico.org/name"]).To(Equal(expectedGnsName))
			Expect(gns.Annotations["policyrecommendation.tigera.io/scope"]).To(Equal("Private"))
			Expect(len(gns.OwnerReferences)).To(Equal(1))
			Expect(gns.OwnerReferences[0].Kind).To(Equal(expectedGnsOwnerKind))
			Expect(gns.OwnerReferences[0].Name).To(Equal(expectedGnsOwnerName))
			Expect(len(gns.Spec.Nets)).To(Equal(0))

		})

		It("Does not create a new 'private-network' if that already exists, and doesn't modify the data", func() {
			expectedGnsName := "private-network"
			expectedGnsOwnerKind := "GNS-Owner"
			expectedGnsOwnerName := "GNS-Owner-Name"

			newGns := &v3.GlobalNetworkSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "private-network",
					Annotations: map[string]string{
						"policyrecommendation.tigera.io/scope": "Private",
					},
					Labels: map[string]string{
						"projectcalico.org/kind": "NetworkSet",
						"projectcalico.org/name": "private-network",
					},
					OwnerReferences: []metav1.OwnerReference{owner},
				},
				Spec: v3.GlobalNetworkSetSpec{
					Nets: []string{
						"10.0.0.0/28",
						"192.1.0.3/24",
					},
				},
			}
			_, err := mockClientSet.ProjectcalicoV3().GlobalNetworkSets().Create(ctx, newGns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			err = MaybeCreatePrivateNetworkSet(ctx, mockClientSet.ProjectcalicoV3(), owner)
			Expect(err).To(BeNil())

			gns, err := mockClientSet.ProjectcalicoV3().GlobalNetworkSets().Get(ctx, expectedGnsName, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(gns.Labels["projectcalico.org/kind"]).To(Equal("NetworkSet"))
			Expect(gns.Labels["projectcalico.org/name"]).To(Equal(expectedGnsName))
			Expect(gns.Annotations["policyrecommendation.tigera.io/scope"]).To(Equal("Private"))
			Expect(len(gns.OwnerReferences)).To(Equal(1))
			Expect(gns.OwnerReferences[0].Kind).To(Equal(expectedGnsOwnerKind))
			Expect(gns.OwnerReferences[0].Name).To(Equal(expectedGnsOwnerName))
			Expect(gns.Spec.Nets).To(Equal([]string{"10.0.0.0/28", "192.1.0.3/24"}))
		})
	})
})

// Mock for testing.
type mockConstructorTestingTNewMockClientSet struct {
}

func (m mockConstructorTestingTNewMockClientSet) Cleanup(func()) {
}

func (m mockConstructorTestingTNewMockClientSet) Logf(format string, args ...interface{}) {
}

func (m mockConstructorTestingTNewMockClientSet) Errorf(format string, args ...interface{}) {
}

func (m mockConstructorTestingTNewMockClientSet) FailNow() {
}
