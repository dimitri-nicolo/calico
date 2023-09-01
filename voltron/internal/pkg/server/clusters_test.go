// Copyright (c) 2019-2023 Tigera, Inc. All rights reserved.

package server

// test is in pkg server to be able to access internal clusters without
// exporting them outside, not part of the pkg API

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	calicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	jclust "github.com/projectcalico/calico/voltron/internal/pkg/clusters"
	vcfg "github.com/projectcalico/calico/voltron/internal/pkg/config"
	"github.com/projectcalico/calico/voltron/internal/pkg/test"
)

var _ = Describe("Clusters", func() {
	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
	clusters := &clusters{
		clusters:   make(map[string]*cluster),
		k8sCLI:     k8sAPI,
		voltronCfg: &vcfg.Config{},
	}

	var wg sync.WaitGroup
	clusterID := "resource-name"
	clusterName := "resource-name"

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
			annotations := map[string]string{
				AnnotationActiveCertificateFingerprint: "active-fingerprint-hash-1",
			}
			Expect(k8sAPI.AddCluster(clusterID, clusterName, annotations)).ShouldNot(HaveOccurred())
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
		})

		It("should be able to update cluster active fingerprint", func() {
			Expect(clusters.clusters[clusterID].ActiveFingerprint).To(Equal("active-fingerprint-hash-1"))
			mc, err := k8sAPI.ManagedClusters().Get(ctx, clusterID, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(mc.GetAnnotations()).To(HaveKeyWithValue(AnnotationActiveCertificateFingerprint, "active-fingerprint-hash-1"))

			err = clusters.clusters[clusterID].updateActiveFingerprint("active-fingerprint-hash-2")
			Expect(err).NotTo(HaveOccurred())

			Expect(clusters.clusters[clusterID].ActiveFingerprint).To(Equal("active-fingerprint-hash-2"))
			mc, err = k8sAPI.ManagedClusters().Get(ctx, clusterID, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(mc.GetAnnotations()).To(HaveKeyWithValue(AnnotationActiveCertificateFingerprint, "active-fingerprint-hash-2"))
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

var _ = Describe("Update certificates", func() {
	k8sAPI := test.NewK8sSimpleFakeClient(nil, nil)
	clusters := &clusters{
		clusters:              make(map[string]*cluster),
		k8sCLI:                k8sAPI,
		voltronCfg:            &vcfg.Config{},
		clientCertificatePool: x509.NewCertPool(),
	}

	var (
		err                  error
		voltronTunnelCert    *x509.Certificate
		voltronTunnelPrivKey *rsa.PrivateKey

		cluster1Cert *x509.Certificate
		cluster2Cert *x509.Certificate

		cluster1CertTemplate *x509.Certificate
		cluster2CertTemplate *x509.Certificate
	)

	const (
		cluster1ID = "cluster-1"
		cluster2ID = "cluster-2"
		cluster3ID = "cluster-3"
	)

	BeforeEach(func() {
		voltronTunnelCertTemplate := test.CreateCACertificateTemplate("voltron")
		voltronTunnelPrivKey, voltronTunnelCert, err = test.CreateCertPair(voltronTunnelCertTemplate, nil, nil)
		Expect(err).ShouldNot(HaveOccurred())

	})

	It("should update the certificate pool when a managed cluster containing a certificate is added", func() {
		cluster1CertTemplate = test.CreateClientCertificateTemplate(cluster1ID, "localhost")
		_, cluster1Cert, err = test.CreateCertPair(cluster1CertTemplate, voltronTunnelCert, voltronTunnelPrivKey)
		Expect(err).NotTo(HaveOccurred())

		cluster2CertTemplate = test.CreateClientCertificateTemplate(cluster2ID, "localhost")
		_, cluster2Cert, err = test.CreateCertPair(cluster2CertTemplate, voltronTunnelCert, voltronTunnelPrivKey)
		Expect(err).NotTo(HaveOccurred())

		// Add a cluster
		mc := &jclust.ManagedCluster{
			ID:          cluster1ID,
			Certificate: test.CertToPemBytes(cluster1Cert),
		}

		_, err = clusters.add(mc)
		Expect(err).NotTo(HaveOccurred())
		// Add a second cluster
		mc = &jclust.ManagedCluster{
			ID:          cluster2ID,
			Certificate: test.CertToPemBytes(cluster2Cert),
		}
		_, err = clusters.add(mc)
		Expect(err).NotTo(HaveOccurred())

		// Validate the certificates are in the map
		expectedCertCluster1, err := parseCertificatePEMBlock(test.CertToPemBytes(cluster1Cert))
		Expect(err).NotTo(HaveOccurred())
		expectedCertCluster2, err := parseCertificatePEMBlock(test.CertToPemBytes(cluster2Cert))
		Expect(err).NotTo(HaveOccurred())

		// Validate the certificates are in the pool
		//nolint:staticcheck // Ignore SA1019 deprecated
		Expect(clusters.clientCertificatePool.Subjects()).To(HaveLen(2))
		//nolint:staticcheck // Ignore SA1019 deprecated
		Expect(clusters.clientCertificatePool.Subjects()).To(ContainElement(expectedCertCluster1.RawSubject))
		//nolint:staticcheck // Ignore SA1019 deprecated
		Expect(clusters.clientCertificatePool.Subjects()).To(ContainElement(expectedCertCluster2.RawSubject))
	})

	It("should add a new certificate to the pool when a cluster certificate has been updated", func() {
		cluster1CertTemplate = test.CreateClientCertificateTemplate("cluster-1-update", "localhost")
		_, cluster1Cert, err = test.CreateCertPair(cluster1CertTemplate, voltronTunnelCert, voltronTunnelPrivKey)
		Expect(err).NotTo(HaveOccurred())

		// Update the certificate for cluster-1
		mc := &jclust.ManagedCluster{
			ID:          cluster1ID,
			Certificate: test.CertToPemBytes(cluster1Cert),
		}

		err = clusters.update(mc)
		Expect(err).NotTo(HaveOccurred())

		expectedCertCluster1, err := parseCertificatePEMBlock(test.CertToPemBytes(cluster1Cert))
		Expect(err).NotTo(HaveOccurred())

		// Validate the certificates are in the pool
		//nolint:staticcheck // Ignore SA1019 deprecated
		Expect(clusters.clientCertificatePool.Subjects()).To(HaveLen(3))
		//nolint:staticcheck // Ignore SA1019 deprecated
		Expect(clusters.clientCertificatePool.Subjects()).To(ContainElement(expectedCertCluster1.RawSubject))
	})
})
