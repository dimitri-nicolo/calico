// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dpi_test

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapifake "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/kube-controllers/pkg/controllers/dpi"
)

var _ = Describe("DPI Controller tests", func() {

	It("caches the DPI resource", func() {
		ns := "dpi-ns"
		resName := "dpi-res"
		cli := tigeraapifake.NewSimpleClientset()
		ctx := context.Background()
		reconciler := dpi.NewReconciler(cli)
		var err error

		By("verifying the cache is empty before reconcile", func() {
			cache := reconciler.GetCachedDPIs()
			Expect(len(cache)).Should(Equal(0))
		})

		By("verifying cache stores existing DPI resource", func() {
			_, err = cli.ProjectcalicoV3().DeepPacketInspections(ns).Create(ctx, &v3.DeepPacketInspection{
				ObjectMeta: metav1.ObjectMeta{Name: resName, Namespace: ns},
			}, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			err = reconciler.Reconcile(types.NamespacedName{Name: resName, Namespace: ns})
			Expect(err).NotTo(HaveOccurred())

			cache := reconciler.GetCachedDPIs()
			Expect(len(cache)).Should(Equal(1))
			Expect(cache[0].Name).Should(Equal(resName))
			Expect(cache[0].Namespace).Should(Equal(ns))
		})

		By("verifying cache removes DPI resource that doesn't exist", func() {
			err = cli.ProjectcalicoV3().DeepPacketInspections(ns).Delete(ctx, resName, metav1.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			err = reconciler.Reconcile(types.NamespacedName{Name: resName, Namespace: ns})
			Expect(err).NotTo(HaveOccurred())

			cache := reconciler.GetCachedDPIs()
			Expect(len(cache)).Should(Equal(0))
		})
	})
})
