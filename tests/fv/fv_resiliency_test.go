// Copyright (c) 2017 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fv_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/kube-controllers/tests/testutils"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

var _ = Describe("[Resilience] PolicyController", func() {
	var (
		calicoEtcd       *containers.Container
		policyController *containers.Container
		k8sEtcd          *containers.Container
		apiserver        *containers.Container
		calicoClient     client.Interface
		k8sClient        *kubernetes.Clientset
		kfConfigFile     *os.File

		policyName      string
		genPolicyName   string
		policyNamespace string

		consistentlyTimeout = "2s"
		consistentlyPoll    = "500ms"
	)

	BeforeEach(func() {
		// Run etcd.
		calicoEtcd = testutils.RunEtcd()
		calicoClient = testutils.GetCalicoClient(calicoEtcd.IP)

		// Run apiserver.
		k8sEtcd = testutils.RunEtcd()
		apiserver = testutils.RunK8sApiserver(k8sEtcd.IP)

		// Write out a kubeconfig file
		var err error
		kfConfigFile, err = ioutil.TempFile("", "ginkgo-policycontroller")
		Expect(err).NotTo(HaveOccurred())
		data := fmt.Sprintf(testutils.KubeconfigTemplate, apiserver.IP)
		kfConfigFile.Write([]byte(data))

		k8sClient, err = testutils.GetK8sClient(kfConfigFile.Name())
		Expect(err).NotTo(HaveOccurred())

		// Wait for the apiserver to be available and for the default namespace to exist.
		Eventually(func() error {
			_, err := k8sClient.CoreV1().Namespaces().Get("default", metav1.GetOptions{})
			return err
		}, 15*time.Second, 500*time.Millisecond).Should(BeNil())

		// Create a Kubernetes NetworkPolicy.
		policyName = "jelly"
		genPolicyName = "knp.default." + policyName
		policyNamespace = "default"
		var np *networkingv1.NetworkPolicy
		np = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      policyName,
				Namespace: policyNamespace,
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"fools": "gold",
					},
				},
			},
		}
		err = k8sClient.NetworkingV1().RESTClient().
			Post().
			Resource("networkpolicies").
			Namespace(policyNamespace).
			Body(np).
			Do().Error()
		Expect(err).NotTo(HaveOccurred())

		policyController = testutils.RunPolicyController(calicoEtcd.IP, kfConfigFile.Name())

		// Wait for it to appear in Calico's etcd.
		Eventually(func() *api.NetworkPolicy {
			policy, _ := calicoClient.NetworkPolicies().Get(context.Background(), policyNamespace, genPolicyName, options.GetOptions{})
			return policy
		}, time.Second*5, 500*time.Millisecond).ShouldNot(BeNil())
	})

	AfterEach(func() {
		calicoEtcd.Stop()
		policyController.Stop()
		k8sEtcd.Stop()
		apiserver.Stop()
		os.Remove(kfConfigFile.Name())
	})

	Context("when apiserver goes down momentarily and data is removed from calico's etcd", func() {
		It("should eventually add the data to calico's etcd", func() {
			Skip("TODO: improve FV framework to handle pod restart")
			testutils.Stop(apiserver)
			_, err := calicoClient.NetworkPolicies().Delete(context.Background(), policyNamespace, genPolicyName, options.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			testutils.Start(apiserver)
			Eventually(func() error {
				_, err := calicoClient.NetworkPolicies().Get(context.Background(), policyNamespace, genPolicyName, options.GetOptions{})
				return err
			}, time.Second*15, 500*time.Millisecond).ShouldNot(HaveOccurred())
		})
	})
	Context("when calico's etcd goes down momentarily and data is removed from k8s-apiserver", func() {
		It("should eventually remove the data from calico's etcd", func() {
			Skip("TODO: improve FV framework to handle pod restart")
			// Delete the Policy.
			testutils.Stop(calicoEtcd)
			err := k8sClient.NetworkingV1().RESTClient().
				Delete().
				Resource("networkpolicies").
				Namespace(policyNamespace).
				Name(policyName).
				Do().Error()
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(10 * time.Second)
			testutils.Start(calicoEtcd)
			Eventually(func() error {
				_, err := calicoClient.NetworkPolicies().Get(context.Background(), policyNamespace, genPolicyName, options.GetOptions{})
				return err
			}, time.Second*15, 500*time.Millisecond).Should(HaveOccurred())
		})
	})
	Context("when licenses are applied for CNX controllers", func() {
		It("should not exit for a license change that adds functionality", func() {
			By("Applying a valid license")
			infrastructure.ApplyValidLicense(calicoClient)
			By("Checking that the controller continues to run")
			Consistently(policyController.ListedInDockerPS(), consistentlyTimeout, consistentlyPoll).Should(Equal(true))
		})

		It("should not exit for a license change that removes functionality since the policy controller does not run licensed controllers by default", func() {
			By("Applying a valid license")
			infrastructure.ApplyValidLicense(calicoClient)
			By("Checking the controller continues to run")
			Consistently(policyController.ListedInDockerPS(), consistentlyTimeout, consistentlyPoll).Should(Equal(true))

			By("Applying an invalid license")
			infrastructure.ApplyExpiredLicense(calicoClient)
			By("Checking the controller continues to run")
			Consistently(policyController.ListedInDockerPS(), consistentlyTimeout, consistentlyPoll).Should(Equal(true))
		})
	})
})
