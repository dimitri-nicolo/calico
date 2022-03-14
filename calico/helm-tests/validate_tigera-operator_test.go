// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package helm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// This file holds all of the functions for validating
// expected values on the Tigera operator deployment.
var _ = Describe("Tigera Operator Helm Chart", func() {
	Context("With tigera operator on a kubernetes datastore", func() {
		values := HelmValues{}

		resources, err := render(values)

		It("Renders the helm resources without issue", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		// Tigera Prometheus
		It("Creates the calico-prometheus-operator service account", func() {
			Expect(resources["ServiceAccount,tigera-prometheus,calico-prometheus-operator"]).NotTo(BeNil())
		})

		It("Creates the calico-prometheus-operator cluster role binding", func() {
			Expect(resources["ClusterRoleBinding,,calico-prometheus-operator"]).NotTo(BeNil())
		})

		It("Creates the calico-prometheus-operator deployment", func() {
			Expect(resources["Deployment,tigera-prometheus,calico-prometheus-operator"]).NotTo(BeNil())
		})

		It("Creates the tigera-prometheus namespace", func() {
			Expect(resources["Namespace,,tigera-prometheus"]).NotTo(BeNil())
		})

		It("Creates the calico-prometheus-operator cluster role", func() {
			Expect(resources["ClusterRole,tigera-prometheus,calico-prometheus-operator"]).NotTo(BeNil())
		})

		// Tigera Operator
		It("Creates the tigera-operator cluster role", func() {
			Expect(resources["ClusterRole,,tigera-operator"]).NotTo(BeNil())
		})

		It("Creates the tigera-operator cluster role binding", func() {
			Expect(resources["ClusterRoleBinding,,tigera-operator"]).NotTo(BeNil())
		})

		It("Creates the tigera-operator deployment", func() {
			Expect(resources["Deployment,tigera-operator,tigera-operator"]).NotTo(BeNil())
		})

		It("Creates the tigera-operator namespace", func() {
			Expect(resources["Namespace,,tigera-operator"]).NotTo(BeNil())
		})

		It("Creates the tigera-operator pod security policy", func() {
			Expect(resources["PodSecurityPolicy,,tigera-operator"]).NotTo(BeNil())
		})

		It("Creates the tigera-operator service account", func() {
			Expect(resources["ServiceAccount,tigera-operator,tigera-operator"]).NotTo(BeNil())
		})

		// Custom Resources have not been created.
		It("Does not create the tigera-secure api server", func() {
			Expect(resources["ApiServer,,tigera-secure"]).To(BeNil())
		})

		It("Does not create the tigera-secure intrusion detection resource", func() {
			Expect(resources["IntrusionDetection,,tigera-secure"]).To(BeNil())
		})

		It("Does not create the tigera-operator log collector", func() {
			Expect(resources["LogCollector,,tigera-secure"]).To(BeNil())
		})

		It("Does not create the tigera-operator log storage", func() {
			Expect(resources["LogStorage,,tigera-secure"]).To(BeNil())
		})

		It("Does not create the tigera-operator manager", func() {
			Expect(resources["Manager,,tigera-secure"]).To(BeNil())
		})

		It("Does not create the tigera-operator monitor", func() {
			Expect(resources["Monitor,,tigera-secure"]).To(BeNil())
		})

		It("Does not create the tigera-operator compliance resource", func() {
			Expect(resources["Compliance,,tigera-secure"]).To(BeNil())
		})
	})
})
