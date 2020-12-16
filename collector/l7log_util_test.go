// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("L7 log utility functions", func() {
	Describe("getAddressAndPort tests", func() {
		Context("With an IP and port", func() {
			It("Should properly split the IP and port", func() {
				addr, port := getAddressAndPort("10.10.10.10:80")
				Expect(addr).To(Equal("10.10.10.10"))
				Expect(port).To(Equal(80))
			})
		})

		Context("With an IP without a port", func() {
			It("Should properly return the IP", func() {
				addr, port := getAddressAndPort("10.10.10.10")
				Expect(addr).To(Equal("10.10.10.10"))
				Expect(port).To(Equal(0))
			})
		})

		Context("With a service name and port", func() {
			It("Should properly split the service name and port", func() {
				addr, port := getAddressAndPort("my-svc:80")
				Expect(addr).To(Equal("my-svc"))
				Expect(port).To(Equal(80))
			})
		})

		Context("With a service name and no port", func() {
			It("Should properly return the service name", func() {
				addr, port := getAddressAndPort("my-svc")
				Expect(addr).To(Equal("my-svc"))
				Expect(port).To(Equal(0))
			})
		})

		Context("With a malformed address", func() {
			It("Should not return anything", func() {
				addr, port := getAddressAndPort("asdf:qewr:asdf:jkl")
				Expect(addr).To(Equal(""))
				Expect(port).To(Equal(0))
			})
		})
	})

	Describe("extractK8sServiceNameAndNamespace tests", func() {
		Context("With a Kubernetes service DNS name", func() {
			It("Should properly extract the service name and namespace", func() {
				name, ns := extractK8sServiceNameAndNamespace("my-svc.svc-namespace.svc.cluster.local")
				Expect(name).To(Equal("my-svc"))
				Expect(ns).To(Equal("svc-namespace"))
			})
		})

		Context("With a Kubernetes service DNS name without a namespace", func() {
			It("Should properly extract the service name and namespace", func() {
				name, ns := extractK8sServiceNameAndNamespace("my-svc.svc.cluster.local")
				Expect(name).To(Equal("my-svc"))
				Expect(ns).To(Equal(""))
			})
		})

		Context("With a Kubernetes service DNS name with a subdomain", func() {
			It("Should properly extrac the service name, subdomain, and namespace", func() {
				name, ns := extractK8sServiceNameAndNamespace("my-svc.place.svc-namespace.svc.cluster.local")
				Expect(name).To(Equal("my-svc.place"))
				Expect(ns).To(Equal("svc-namespace"))
			})
		})

		Context("With an invalid Kubernetes service DNS name", func() {
			It("Should return nothing", func() {
				// Pod DNS
				name, ns := extractK8sServiceNameAndNamespace("my-pod.pod-namespace.pod.cluster.local")
				Expect(name).To(Equal(""))
				Expect(ns).To(Equal(""))

				// Non Kubernetes DNS
				name, ns = extractK8sServiceNameAndNamespace("my-external-svc.com")
				Expect(name).To(Equal(""))
				Expect(ns).To(Equal(""))
			})
		})
	})
})
