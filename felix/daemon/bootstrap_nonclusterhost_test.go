// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package daemon

import (
	"context"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"github.com/projectcalico/calico/libcalico-go/lib/nonclusterhost"
)

var _ = Describe("Felix daemon NonClusterHost bootstrap tests", func() {
	Context("Retrieve Typha endpoint", func() {
		var (
			fakeDynamicClient *fake.FakeDynamicClient
		)

		BeforeEach(func() {
			gvrListKind := map[schema.GroupVersionResource]string{
				nonclusterhost.NonClusterHostGVR: "NonClusterHostList",
			}

			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(nonclusterhost.NonClusterHostGVR.GroupVersion(), &unstructured.Unstructured{})
			fakeDynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrListKind)
			Expect(fakeDynamicClient).NotTo(BeNil())
		})

		It("should extract and validate typhaEndpoint from NonClusterHost custom resource", func() {
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "operator.tigera.io/v1",
					"kind":       "NonClusterHost",
					"metadata": map[string]interface{}{
						"name": "tigera-secure",
					},
					"spec": map[string]interface{}{
						"some-field":    "some-value",
						"typhaEndpoint": "1.2.3.4:5678",
					},
				},
			}
			nch, err := fakeDynamicClient.Resource(nonclusterhost.NonClusterHostGVR).Create(context.TODO(), obj, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(nch).NotTo(BeNil())
			Expect(nch.GetName()).To(Equal("tigera-secure"))

			fakeTyphaAddressExtractor := &typhaAddressExtractor{
				ctx:              context.TODO(),
				k8sDynamicClient: fakeDynamicClient,
			}
			addr, err := fakeTyphaAddressExtractor.typhaAddress()
			Expect(err).NotTo(HaveOccurred())
			Expect(addr).To(Equal("1.2.3.4:5678"))
		})

		It("should resolve host to IP address", func() {
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "operator.tigera.io/v1",
					"kind":       "NonClusterHost",
					"metadata": map[string]interface{}{
						"name": "tigera-secure",
					},
					"spec": map[string]interface{}{
						"some-field":    "some-value",
						"typhaEndpoint": "localhost:5678",
					},
				},
			}
			nch, err := fakeDynamicClient.Resource(nonclusterhost.NonClusterHostGVR).Create(context.TODO(), obj, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(nch).NotTo(BeNil())
			Expect(nch.GetName()).To(Equal("tigera-secure"))

			fakeTyphaAddressExtractor := &typhaAddressExtractor{
				ctx:              context.TODO(),
				k8sDynamicClient: fakeDynamicClient,
			}
			addr, err := fakeTyphaAddressExtractor.typhaAddress()
			Expect(err).NotTo(HaveOccurred())
			Expect(addr).To(BeElementOf([]string{"127.0.0.1:5678", "[::1]:5678"}))
		})

		table.DescribeTable("should return error when typhaEndpoint is invalid",
			func(endpoint string) {
				obj := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "operator.tigera.io/v1",
						"kind":       "NonClusterHost",
						"metadata": map[string]interface{}{
							"name": "tigera-secure",
						},
						"spec": map[string]interface{}{
							"some-field":    "some-value",
							"typhaEndpoint": endpoint,
						},
					},
				}
				nch, err := fakeDynamicClient.Resource(nonclusterhost.NonClusterHostGVR).Create(context.TODO(), obj, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(nch).NotTo(BeNil())
				Expect(nch.GetName()).To(Equal("tigera-secure"))

				fakeTyphaAddressExtractor := &typhaAddressExtractor{
					ctx:              context.TODO(),
					k8sDynamicClient: fakeDynamicClient,
				}
				addr, err := fakeTyphaAddressExtractor.typhaAddress()
				Expect(err).To(HaveOccurred())
				Expect(addr).To(BeEmpty())
			},

			table.Entry("endpoint is not a valid ip:port format", "some-random-format"),
			table.Entry("endpoint is missing IP address", ":5678"),
			table.Entry("endpoint is missing port number", "1.2.3.4:"),
			table.Entry("invalid IP address", "333.444.555.666:5678"),
			table.Entry("invalid port number", "1.2.3.4:abcd"),
		)
	})
})
