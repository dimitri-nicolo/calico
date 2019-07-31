package policycalc

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var _ = Describe("Namespace handler tests", func() {
	It("handles namespace selector caching", func() {
		nh := NewNamespaceHandler(nil, nil)

		By("creating a namespace selector")
		m1 := nh.GetNamespaceSelectorEndpointMatcher("all()")
		Expect(m1).NotTo(BeNil())

		By("creating the same namespace selector and checking the cache size")
		m2 := nh.GetNamespaceSelectorEndpointMatcher("all()")
		Expect(m2).NotTo(BeNil())
		Expect(reflect.ValueOf(m2)).To(Equal(reflect.ValueOf(m1)))
		Expect(nh.selectorMatchers).To(HaveLen(1))

		By("checking a different selector returns a different matcher function")
		m3 := nh.GetNamespaceSelectorEndpointMatcher("vegetable == 'turnip'")
		Expect(m3).NotTo(BeNil())
		Expect(reflect.ValueOf(m3)).NotTo(Equal(reflect.ValueOf(m1)))
		Expect(nh.selectorMatchers).To(HaveLen(2))
	})

	It("handles namespace selection", func() {
		nh := NewNamespaceHandler([]*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
					Labels: map[string]string{
						"vegetable": "turnip",
						"protein":   "chicken",
						"carb":      "potato",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns2",
					Labels: map[string]string{
						"vegetable": "turnip",
						"protein":   "beef",
						"carb":      "rice",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns3",
					Labels: map[string]string{
						"vegetable": "carrot",
						"protein":   "beef",
						"carb":      "rice",
					},
				},
			},
		}, nil)

		By("Creating a matcher on ns1 and ns2")
		m1 := nh.GetNamespaceSelectorEndpointMatcher("vegetable == 'turnip'")
		Expect(m1(nil, &FlowEndpointData{Namespace: "ns1"})).To(Equal(MatchTypeTrue))
		Expect(m1(nil, &FlowEndpointData{Namespace: "ns2"})).To(Equal(MatchTypeTrue))
		Expect(m1(nil, &FlowEndpointData{Namespace: "ns3"})).To(Equal(MatchTypeFalse))

		By("Creating a matcher on ns2 and ns3")
		m2 := nh.GetNamespaceSelectorEndpointMatcher("protein == 'beef' && carb == 'rice'")
		Expect(m2(nil, &FlowEndpointData{Namespace: "ns1"})).To(Equal(MatchTypeFalse))
		Expect(m2(nil, &FlowEndpointData{Namespace: "ns2"})).To(Equal(MatchTypeTrue))
		Expect(m2(nil, &FlowEndpointData{Namespace: "ns3"})).To(Equal(MatchTypeTrue))

		By("Getting the same selector matchers and rechecking")
		m1 = nh.GetNamespaceSelectorEndpointMatcher("vegetable == 'turnip'")
		m2 = nh.GetNamespaceSelectorEndpointMatcher("protein == 'beef' && carb == 'rice'")

		Expect(m1(nil, &FlowEndpointData{Namespace: "ns1"})).To(Equal(MatchTypeTrue))
		Expect(m1(nil, &FlowEndpointData{Namespace: "ns2"})).To(Equal(MatchTypeTrue))
		Expect(m1(nil, &FlowEndpointData{Namespace: "ns3"})).To(Equal(MatchTypeFalse))

		Expect(m2(nil, &FlowEndpointData{Namespace: "ns1"})).To(Equal(MatchTypeFalse))
		Expect(m2(nil, &FlowEndpointData{Namespace: "ns2"})).To(Equal(MatchTypeTrue))
		Expect(m2(nil, &FlowEndpointData{Namespace: "ns3"})).To(Equal(MatchTypeTrue))

		By("Checking the size of the selector cache")
		Expect(nh.selectorMatchers).To(HaveLen(2))
	})

	It("handles service account population", func() {
		nh := NewNamespaceHandler(nil, []*corev1.ServiceAccount{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "ns1",
					Labels: map[string]string{
						"vegetable": "turnip",
						"protein":   "chicken",
						"carb":      "potato",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa1",
					Namespace: "ns2",
					Labels: map[string]string{
						"vegetable": "turnip",
						"protein":   "beef",
						"carb":      "rice",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa2",
					Namespace: "ns2",
					Labels: map[string]string{
						"vegetable": "carrot",
						"protein":   "beef",
						"carb":      "rice",
					},
				},
			},
		})

		By("Checking the number of namespaces created implicitly")
		Expect(nh.namespaces).To(HaveLen(2))

		By("Checking the number of service accounts cached")
		Expect(nh.namespaces["ns1"].serviceAccountLabels).To(HaveLen(1))
		Expect(nh.namespaces["ns2"].serviceAccountLabels).To(HaveLen(2))

		By("Creating a service account matcher by name and checking for matches")
		m1 := nh.GetServiceAccountEndpointMatchers(&v3.ServiceAccountMatch{
			Names: []string{"sa1"},
		})
		Expect(m1(nil, &FlowEndpointData{Type: EndpointTypeHep})).To(Equal(MatchTypeFalse))
		Expect(m1(nil, &FlowEndpointData{Type: EndpointTypeWep})).To(Equal(MatchTypeUncertain))
		s := "sa1"
		Expect(m1(nil, &FlowEndpointData{
			Type:           EndpointTypeWep,
			Namespace:      "ns1",
			ServiceAccount: &s,
		})).To(Equal(MatchTypeTrue))

		By("Asking for the same matcher and verifying we get the same response")
		m2 := nh.GetServiceAccountEndpointMatchers(&v3.ServiceAccountMatch{
			Names: []string{"sa1"},
		})
		Expect(reflect.ValueOf(m2)).To(Equal(reflect.ValueOf(m1)))
		Expect(nh.serviceAccountMatchers).To(HaveLen(1))

		By("Creating a service account matcher by label and checking for matches")
		m3 := nh.GetServiceAccountEndpointMatchers(&v3.ServiceAccountMatch{
			Selector: "vegetable == 'carrot'",
		})
		Expect(m3(nil, &FlowEndpointData{Type: EndpointTypeHep})).To(Equal(MatchTypeFalse))
		Expect(m3(nil, &FlowEndpointData{Type: EndpointTypeWep})).To(Equal(MatchTypeUncertain))
		s = "sa1"
		Expect(m3(nil, &FlowEndpointData{
			Type:           EndpointTypeWep,
			Namespace:      "ns2",
			ServiceAccount: &s,
		})).To(Equal(MatchTypeFalse))
		s = "sa2"
		Expect(m3(nil, &FlowEndpointData{
			Type:           EndpointTypeWep,
			Namespace:      "ns2",
			ServiceAccount: &s,
		})).To(Equal(MatchTypeTrue))

		By("Asking for the same matcher and verifying we get the same response")
		m4 := nh.GetServiceAccountEndpointMatchers(&v3.ServiceAccountMatch{
			Selector: "vegetable == 'carrot'",
		})
		Expect(reflect.ValueOf(m4)).To(Equal(reflect.ValueOf(m3)))
		Expect(reflect.ValueOf(m4)).NotTo(Equal(reflect.ValueOf(m2)))
		Expect(nh.serviceAccountMatchers).To(HaveLen(2))

		By("Creating a service account matcher by name and label and checking for matches")
		m5 := nh.GetServiceAccountEndpointMatchers(&v3.ServiceAccountMatch{
			Names:    []string{"sa1"},
			Selector: "carb == 'rice'",
		})
		Expect(m5(nil, &FlowEndpointData{Type: EndpointTypeHep})).To(Equal(MatchTypeFalse))
		Expect(m5(nil, &FlowEndpointData{Type: EndpointTypeWep})).To(Equal(MatchTypeUncertain))
		s = "sa1"
		Expect(m5(nil, &FlowEndpointData{
			Type:           EndpointTypeWep,
			Namespace:      "ns1",
			ServiceAccount: &s,
		})).To(Equal(MatchTypeFalse))
		Expect(m5(nil, &FlowEndpointData{
			Type:           EndpointTypeWep,
			Namespace:      "ns2",
			ServiceAccount: &s,
		})).To(Equal(MatchTypeTrue))

		By("Creating a service account matcher that doesn't match anything")
		m6 := nh.GetServiceAccountEndpointMatchers(&v3.ServiceAccountMatch{
			Names:    []string{"sa1"},
			Selector: "protein == 'tofu'",
		})
		Expect(m6(nil, &FlowEndpointData{Type: EndpointTypeHep})).To(Equal(MatchTypeFalse))
		Expect(m6(nil, &FlowEndpointData{Type: EndpointTypeWep})).To(Equal(MatchTypeFalse))
	})
})
