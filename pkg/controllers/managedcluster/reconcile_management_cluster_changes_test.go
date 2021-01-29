// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package managedcluster_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/kube-controllers/pkg/controllers/managedcluster"
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	relasticsearchfake "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch/fake"

	esv1 "github.com/elastic/cloud-on-k8s/pkg/apis/elasticsearch/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	tigeraapifake "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
)

var _ = Describe("Reconcile", func() {
	Context("When the elasticsearch components exist in management cluster", func() {
		var managementESCertSecret *corev1.Secret
		var managementKBCertSecret *corev1.Secret
		var managementESConfigMap *corev1.ConfigMap
		var es *esv1.Elasticsearch
		var managementK8sCli kubernetes.Interface
		var calicoK8sCLI tigeraapi.Interface
		var esK8sCLI *relasticsearchfake.RESTClient
		var notifier chan bool
		var r worker.Reconciler

		BeforeEach(func() {
			managementESCertSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.ElasticsearchCertSecret,
					Namespace: resource.OperatorNamespace,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("someescertbytes"),
					"tls.key": []byte("someeskeybytes"),
				},
			}

			managementKBCertSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.KibanaCertSecret,
					Namespace: resource.OperatorNamespace,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("somekbcertbytes"),
					"tls.key": []byte("somekbkeybytes"),
				},
			}

			managementESConfigMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.ElasticsearchConfigMapName,
					Namespace: resource.OperatorNamespace,
				},
				Data: map[string]string{
					"clusterName": "cluster",
					"replicas":    "1",
					"shards":      "5",
				},
			}
			es = &esv1.Elasticsearch{ObjectMeta: metav1.ObjectMeta{
				Name:              resource.DefaultTSEEInstanceName,
				Namespace:         resource.TigeraElasticsearchNamespace,
				CreationTimestamp: metav1.Now(),
			}}
			managementK8sCli = k8sfake.NewSimpleClientset(managementESCertSecret, managementKBCertSecret,
				managementESConfigMap)
			calicoK8sCLI = tigeraapifake.NewSimpleClientset()

			var err error
			esK8sCLI, err = relasticsearchfake.NewFakeRESTClient(es)
			Expect(err).ShouldNot(HaveOccurred())
			notifier = make(chan bool, 1)
			r = managedcluster.NewManagementClusterChangeReconciler(managementK8sCli, calicoK8sCLI, esK8sCLI, notifier)
			// Initial reconcile to set the change hash
			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			close(notifier)
		})

		It("notifies when the elasticsearch ConfigMap changes", func() {
			err := resource.WriteConfigMapToK8s(managementK8sCli, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.ElasticsearchConfigMapName,
					Namespace: resource.OperatorNamespace,
				},
				Data: map[string]string{
					"clusterName": "cluster",
					"replicas":    "2",
					"shards":      "5",
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())

			select {
			case <-notifier:
			default:
				Fail("wasn't notified about change to management cluster")
			}
		})

		It("notifies when elasticsearch cert Secret changes", func() {
			err := resource.WriteSecretToK8s(managementK8sCli, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.ElasticsearchCertSecret,
					Namespace: resource.OperatorNamespace,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("somedifferentcertbytes"),
					"tls.key": []byte("somekeybytes"),
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())

			select {
			case <-notifier:
			default:
				Fail("wasn't notified about change to management cluster")
			}
		})

		It("notifies when elasticsearch is recreated", func() {
			// Changing the creation time stamp signals that the cluster was recreated
			esK8sCLI.SetElasticsearch(&esv1.Elasticsearch{ObjectMeta: metav1.ObjectMeta{
				Name:              resource.DefaultTSEEInstanceName,
				Namespace:         resource.TigeraElasticsearchNamespace,
				CreationTimestamp: metav1.Now(),
			}})

			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())

			select {
			case <-notifier:
			default:
				Fail("wasn't notified about change to management cluster")
			}
		})

		It("doesn't notify when the elasticsearch ConfigMap data isn't changed", func() {
			cp := managementESConfigMap.DeepCopy()
			cp.Labels = map[string]string{
				"foo": "bar",
			}
			err := resource.WriteConfigMapToK8s(managementK8sCli, cp)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())

			select {
			case <-notifier:
				Fail("wasn't notified about change to management cluster")
			default:
			}
		})

		It("doesn't notify when the elasticsearch cert Secret data isn't changed", func() {
			cp := managementESCertSecret.DeepCopy()
			cp.Labels = map[string]string{
				"foo": "bar",
			}
			err := resource.WriteSecretToK8s(managementK8sCli, cp)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())

			select {
			case <-notifier:
				Fail("wasn't notified about change to management cluster")
			default:
			}
		})

		It("doesn't notify when Elasticsearches status changes", func() {
			cp := es.DeepCopy()
			cp.Status.Health = esv1.ElasticsearchYellowHealth
			esK8sCLI.SetElasticsearch(cp)
			Expect(r.Reconcile(types.NamespacedName{})).ShouldNot(HaveOccurred())

			select {
			case <-notifier:
				Fail("was notified about change to management cluster")
			default:
			}
		})
	})
})
