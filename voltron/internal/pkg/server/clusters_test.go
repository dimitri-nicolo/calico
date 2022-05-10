// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package server

// test is in pkg server to be able to access internal clusters without
// exporting them outside, not part of the pkg API

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/voltron/internal/pkg/test"

	calicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var _ = Describe("Clusters", func() {
	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
	clusters := &clusters{
		clusters: make(map[string]*cluster),
		k8sCLI:   k8sAPI,
	}

	var wg sync.WaitGroup
	var clusterID = "resource-name"
	var clusterName = "resource-name"

	Describe("basic functionality", func() {
		ctx, cancel := context.WithCancel(context.Background())

		By("starting watching", func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = clusters.watchK8s(ctx, nil)
			}()
		})

		It("should be possible to add a cluster", func() {
			Expect(k8sAPI.AddCluster(clusterID, clusterName, nil)).ShouldNot(HaveOccurred())
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
		})

		It("should be possible to delete a cluster", func() {
			Expect(k8sAPI.DeleteCluster(clusterID)).ShouldNot(HaveOccurred())
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(0))
		})

		It("should stop watch", func() {
			cancel()
			wg.Wait()
		})
	})

	When("watch is down", func() {
		ctx, cancel := context.WithCancel(context.Background())
		It("should cluster added should be seen after watch restarts", func() {
			Expect(len(clusters.List())).To(Equal(0))
			Expect(k8sAPI.AddCluster(clusterID, clusterName, nil)).ShouldNot(HaveOccurred())
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = clusters.watchK8s(ctx, nil)
			}()
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
		})

		It("should stop watch", func() {
			cancel()
			wg.Wait()
		})
	})

	When("watch restarts", func() {
		ctx, cancel := context.WithCancel(context.Background())
		It("should delete a cluster deleted while watch was down", func() {
			Expect(len(clusters.List())).To(Equal(1))
			Expect(k8sAPI.DeleteCluster(clusterID)).ShouldNot(HaveOccurred())
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = clusters.watchK8s(ctx, nil)
			}()
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(0))
		})

		It("should add a cluster after watch restarted due to an error", func() {
			Expect(len(clusters.List())).To(Equal(0))
			k8sAPI.BreakWatcher()
			k8sAPI.WaitForManagedClustersWatched() // indicates a watch restart
			Expect(k8sAPI.AddCluster("X", "X", nil)).ShouldNot(HaveOccurred())
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
		})

		It("should add a cluster before watch restarted due to an error", func() {
			Expect(len(clusters.List())).To(Equal(1))
			k8sAPI.BlockWatches()
			k8sAPI.BreakWatcher()
			Expect(k8sAPI.AddCluster("Y", "Y", nil)).ShouldNot(HaveOccurred())
			k8sAPI.UnblockWatches()
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(2))
		})

		It("should stop watch", func() {
			cancel()
			wg.Wait()
		})
	})

	When("new watch", func() {
		ctx, cancel := context.WithCancel(context.Background())
		clusterName := "sample-restart-cluster"

		It("should set ManagedClusterConnected status to false if it is true during startup.", func() {
			Expect(len(clusters.List())).To(Equal(2))
			Expect(k8sAPI.AddCluster(clusterName, clusterName, nil, calicov3.ManagedClusterStatusValueTrue)).ShouldNot(HaveOccurred())

			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = clusters.watchK8s(ctx, nil)
			}()

			Eventually(func() calicov3.ManagedClusterStatusValue {
				c, _ := k8sAPI.ManagedClusters().Get(context.Background(), clusterName, metav1.GetOptions{})
				return c.Status.Conditions[0].Status
			}, 10*time.Second, 1*time.Second).Should(Equal(calicov3.ManagedClusterStatusValueFalse))
			Expect(len(clusters.List())).To(Equal(3))
		})

		It("should stop watch", func() {
			cancel()
			wg.Wait()
		})
	})

})
