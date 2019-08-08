package server

// test is in pkg server to be able to access internal clusters without
// exporting them outside, not part of the pkg API

import (
	"context"
	"crypto"
	"crypto/x509"
	"io"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	jclust "github.com/tigera/voltron/internal/pkg/clusters"
	"github.com/tigera/voltron/internal/pkg/test"
)

var _ = Describe("Clusters", func() {
	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)

	clusters := &clusters{
		clusters: make(map[string]*cluster),
		generateCreds: func(*jclust.Cluster) (*x509.Certificate, crypto.Signer, error) {
			return nil, nil, nil
		},
		renderManifest: func(wr io.Writer, cert *x509.Certificate, key crypto.Signer) error {
			return nil
		},
		watchAdded: true,
	}

	var wg sync.WaitGroup

	Describe("basic functionality", func() {
		ctx, cancel := context.WithCancel(context.Background())

		By("starting watching", func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				clusters.watchK8s(ctx, k8sAPI, nil)
			}()
		})

		It("should be possible to add a cluster", func() {
			k8sAPI.AddCluster("A", "A")
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
		})

		It("should be possible to delete a cluster", func() {
			k8sAPI.DeleteCluster("A")
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
			k8sAPI.AddCluster("A", "A")
			wg.Add(1)
			go func() {
				defer wg.Done()
				clusters.watchK8s(ctx, k8sAPI, nil)
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
			k8sAPI.DeleteCluster("A")
			wg.Add(1)
			go func() {
				defer wg.Done()
				clusters.watchK8s(ctx, k8sAPI, nil)
			}()
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(0))
		})

		It("should add a cluster after watch restarted due to an error", func() {
			Expect(len(clusters.List())).To(Equal(0))
			k8sAPI.BreakWatcher()
			k8sAPI.WaitForManagedClustersWatched() // indicates a watch restart
			k8sAPI.AddCluster("X", "X")
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
		})

		It("should add a cluster before watch restarted due to an error", func() {
			Expect(len(clusters.List())).To(Equal(1))
			k8sAPI.BlockWatches()
			k8sAPI.BreakWatcher()
			k8sAPI.AddCluster("Y", "Y")
			k8sAPI.UnblockWatches()
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(2))
		})

		It("should stop watch", func() {
			cancel()
			wg.Wait()
		})
	})
})
