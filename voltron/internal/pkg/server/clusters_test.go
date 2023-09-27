// Copyright (c) 2019-2023 Tigera, Inc. All rights reserved.

package server

// test is in pkg server to be able to access internal clusters without
// exporting them outside, not part of the pkg API

import (
	"context"

	"crypto/rsa"
	"crypto/x509"

	"k8s.io/apimachinery/pkg/types"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	calicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	jclust "github.com/projectcalico/calico/voltron/internal/pkg/clusters"
	vcfg "github.com/projectcalico/calico/voltron/internal/pkg/config"
	"github.com/projectcalico/calico/voltron/internal/pkg/test"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func describe(name string, testFn func(string)) bool {
	Describe(name+" cluster-scoped", func() { testFn("") })
	Describe(name+" namespace-scoped", func() { testFn("resource-ns") })
	return true
}

var _ = describe("Clusters", func(clusterNamespace string) {
	scheme := kscheme.Scheme
	err := v3.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	var wg sync.WaitGroup
	clusterID := "resource-name"
	//clusterNamespace := "resource-ns"

	clusters := &clusters{
		clusters:   make(map[string]*cluster),
		client:     fakeClient,
		voltronCfg: &vcfg.Config{TenantNamespace: clusterNamespace},
	}
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
		err := fakeClient.Create(context.Background(), &v3.ManagedCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       v3.KindManagedCluster,
				APIVersion: v3.GroupVersionCurrent,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        clusterID,
				Namespace:   clusterNamespace,
				Annotations: annotations,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
	})

	It("should be able to update cluster active fingerprint", func() {
		Expect(clusters.clusters[clusterID].ActiveFingerprint).To(Equal("active-fingerprint-hash-1"))
		mc := &v3.ManagedCluster{}
		err := fakeClient.Get(context.Background(), types.NamespacedName{Name: clusterID, Namespace: clusterNamespace}, mc)
		Expect(err).NotTo(HaveOccurred())
		Expect(mc.GetAnnotations()).To(HaveKeyWithValue(AnnotationActiveCertificateFingerprint, "active-fingerprint-hash-1"))

		err = clusters.clusters[clusterID].updateActiveFingerprint("active-fingerprint-hash-2")
		Expect(err).NotTo(HaveOccurred())

		Expect(clusters.clusters[clusterID].ActiveFingerprint).To(Equal("active-fingerprint-hash-2"))

		mc = &v3.ManagedCluster{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: clusterID, Namespace: clusterNamespace}, mc)

		Expect(err).NotTo(HaveOccurred())
		Expect(mc.GetAnnotations()).To(HaveKeyWithValue(AnnotationActiveCertificateFingerprint, "active-fingerprint-hash-2"))
	})

	It("should be possible to delete a cluster", func() {
		Expect(fakeClient.Delete(context.Background(), &v3.ManagedCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       v3.KindManagedCluster,
				APIVersion: v3.GroupVersionCurrent,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterID,
				Namespace: clusterNamespace,
			},
		})).ShouldNot(HaveOccurred())
		Eventually(func() int { return len(clusters.List()) }).Should(Equal(0))
	})

	It("should stop watch", func() {
		cancel()
		wg.Wait()
	})

	When("watch is down", func() {
		ctx, cancel := context.WithCancel(context.Background())
		It("should cluster added should be seen after watch restarts", func() {
			Expect(len(clusters.List())).To(Equal(0))
			Expect(fakeClient.Create(context.Background(), &v3.ManagedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterID,
					Namespace: clusterNamespace,
				},
			})).NotTo(HaveOccurred())
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
			Expect(fakeClient.Delete(context.Background(), &v3.ManagedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterID,
					Namespace: clusterNamespace,
				},
			})).ShouldNot(HaveOccurred())
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = clusters.watchK8s(ctx, nil)
			}()
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(0))
		})

		It("should add a cluster after watch restarted due to an error", func() {

			mcList := &v3.ManagedClusterList{}
			watch, err := fakeClient.Watch(context.Background(), mcList, &ctrlclient.ListOptions{})
			watch.Stop()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(clusters.List())).To(Equal(0))
			Expect(fakeClient.Create(context.Background(), &v3.ManagedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "X",
					Namespace: clusterNamespace,
				},
			})).NotTo(HaveOccurred())
			Eventually(func() int { return len(clusters.List()) }).Should(Equal(1))
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
			Expect(len(clusters.List())).To(Equal(1))
			Expect(fakeClient.Create(context.Background(), &v3.ManagedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       v3.KindManagedCluster,
					APIVersion: v3.GroupVersionCurrent,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: clusterNamespace,
				},
				Status: calicov3.ManagedClusterStatus{
					Conditions: []calicov3.ManagedClusterStatusCondition{
						{
							Status: calicov3.ManagedClusterStatusValueTrue,
							Type:   "ManagedClusterConnected",
						},
					},
				},
			})).NotTo(HaveOccurred())

			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = clusters.watchK8s(ctx, nil)
			}()

			Eventually(func() calicov3.ManagedClusterStatusValue {
				mc := &v3.ManagedCluster{}
				err = fakeClient.Get(context.Background(), types.NamespacedName{Name: clusterName, Namespace: clusterNamespace}, mc)
				return mc.Status.Conditions[0].Status
			}, 10*time.Second, 1*time.Second).Should(Equal(calicov3.ManagedClusterStatusValueFalse))

			Expect(len(clusters.List())).To(Equal(2))
		})

		It("should stop watch", func() {
			cancel()
			wg.Wait()
		})
	})
})

var _ = describe("Update certificates", func(clusterNamespace string) {

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
