package server

// test is in pkg server to be able to access internal clusters without
// exporting them outside, not part of the pkg API

import (
	"context"
	"crypto"
	"crypto/x509"
	"io"
	"sync"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/tigera/voltron/internal/pkg/test"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	jclust "github.com/tigera/voltron/internal/pkg/clusters"
)

var _ = Describe("Clusters", func() {
	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)

	clusters := &clusters{
		clusters: make(map[string]*cluster),
		generateCreds: func(*jclust.ManagedCluster) (*x509.Certificate, crypto.Signer, error) {
			return &x509.Certificate{Raw: []byte{}}, nil, nil
		},
		renderManifest: func(wr io.Writer, cert *x509.Certificate, key crypto.Signer) error {
			return nil
		},
		watchAdded: true,
		k8sCLI:     k8sAPI,
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
})

func TestStoringFingerprint(t *testing.T) {
	g := NewGomegaWithT(t)
	data := []struct {
		name        string
		fingerprint string
		annotations map[string]string
	}{
		{"cluster", "hex", nil},
		{"cluster", "hex", make(map[string]string)},
		{"cluster", "hex-new", map[string]string{AnnotationActiveCertificateFingerprint: "hex-old"}},
	}

	for _, entry := range data {
		k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)

		var err error
		// Mock the K8S Api so that we can perform a get and update on a managed cluster
		err = k8sAPI.AddCluster(entry.name, entry.name, entry.annotations)
		g.Expect(err).NotTo(HaveOccurred())

		err = storeFingerprint(k8sAPI, entry.name, entry.fingerprint)
		g.Expect(err).NotTo(HaveOccurred())

		cluster, err := k8sAPI.ManagedClusters().Get(entry.name, metav1.GetOptions{})
		g.Expect(cluster.ObjectMeta.Annotations).NotTo(BeNil())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cluster.ObjectMeta.Annotations[AnnotationActiveCertificateFingerprint]).To(Equal(entry.fingerprint))
	}
}

func TestFailToStoreFingerprint(t *testing.T) {
	g := NewGomegaWithT(t)
	data := []struct {
		verb     string
		reaction k8stesting.ReactionFunc
	}{
		{"get", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.Errorf("any error")
		}},
		{"update", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.Errorf("any error")
		}},
	}

	for _, entry := range data {
		// Mock the k8s api response so that it returns an error when performing get or update
		k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
		k8sAPI.CalicoFake().PrependReactor(entry.verb, "managedclusters", entry.reaction)

		// Expect storeFingerprint() to fail
		var err = storeFingerprint(k8sAPI, "cluster", "hex")
		g.Expect(err).To(HaveOccurred())
	}
}
