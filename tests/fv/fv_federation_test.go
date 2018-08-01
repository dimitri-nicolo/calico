// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/kube-controllers/pkg/controllers/federatedservices"
	"github.com/projectcalico/kube-controllers/tests/testutils"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

var (
	eventuallyTimeout   = "15s"
	eventuallyPoll      = "500ms"
	consistentlyTimeout = "2s"
	consistentlyPoll    = "500ms"

	node1Name = "node-1"
	node2Name = "node-2"
	ns1Name   = "ns-1"
	ns2Name   = "ns-2"
	ctx       = context.Background()

	// svc1 has the federation label
	svc1 = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"federate": "yes",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "port1",
					Port:     1234,
					Protocol: v1.ProtocolUDP,
				},
				{
					Name:     "port2",
					Port:     1234,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "port3",
					Port:     1200,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}

	eps1 = &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.1",
						NodeName: &node1Name,
					},
				},
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.1",
						NodeName: &node1Name,
					},
				},
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port3",
						Port:     1200,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "2.0.0.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port2",
						Port:     1234,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "3.0.0.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port3",
						Port:     1200,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	// svc2 has the federation label and a different set of endpoints from svc1
	svc2 = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"federate": "yes",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "port1",
					Port:     1234,
					Protocol: v1.ProtocolUDP,
				},
				{
					Name:     "port2",
					Port:     1234,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "port3",
					Port:     1200,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}

	eps2 = &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"federate": "yes",
			},
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.1",
						NodeName: &node1Name,
					},
				},
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
		},
	}
)

var _ = Describe("[federation] kube-controllers Federated Services FV tests", func() {
	var (
		localEtcd            *containers.Container
		localApiserver       *containers.Container
		localCalicoClient    client.Interface
		localK8sClient       *kubernetes.Clientset
		federationController *containers.Container
		remoteEtcd           *containers.Container
		remoteApiserver      *containers.Container
		remoteK8sClient      *kubernetes.Clientset
		remoteKubeconfig     string
	)

	const (
		localK8sNodeName     = "k8snodename-local"
		localCalicoNodeName  = "caliconodename-local"
		remoteK8sNodeName    = "k8snodename-remote"
		remoteCalicoNodeName = "caliconodename-remote"
	)

	getSubsets := func(namespace, name string) []v1.EndpointSubset {
		eps, err := localK8sClient.CoreV1().Endpoints(namespace).Get(name, metav1.GetOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			return nil
		}
		Expect(err).NotTo(HaveOccurred())
		return federatedservices.GetOrderedEndpointSubsets(eps.Subsets)
	}

	getSubsetsFn := func(namespace, name string) func() []v1.EndpointSubset {
		return func() []v1.EndpointSubset {
			return getSubsets(namespace, name)
		}
	}

	setup := func(isCalicoEtcdDatastore bool) {
		// Create local etcd and run the local apiserver. Wait for the API server to come online.
		localEtcd = testutils.RunEtcd()
		localApiserver = testutils.RunK8sApiserver(localEtcd.IP)

		// Write out a kubeconfig file for the local API server, and create a k8s client.
		lkubeconfig, err := ioutil.TempFile("", "ginkgo-localcluster")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(lkubeconfig.Name())
		data := fmt.Sprintf(testutils.KubeconfigTemplate, localApiserver.IP)
		lkubeconfig.Write([]byte(data))
		localK8sClient, err = testutils.GetK8sClient(lkubeconfig.Name())

		// Create the appropriate local Calico client depending on whether this is an etcd or kdd test.
		if isCalicoEtcdDatastore {
			localCalicoClient = testutils.GetCalicoClient(localEtcd.IP)
		} else {
			localCalicoClient = testutils.GetCalicoKubernetesClient(lkubeconfig.Name())
		}

		// Create remote etcd and run the remote apiserver.
		remoteEtcd = testutils.RunEtcd()
		remoteApiserver = testutils.RunK8sApiserver(remoteEtcd.IP)

		// Write out a kubeconfig file for the remote API server.
		rkubeconfig, err := ioutil.TempFile("", "ginkgo-remotecluster")
		Expect(err).NotTo(HaveOccurred())
		remoteKubeconfig = rkubeconfig.Name()
		data = fmt.Sprintf(testutils.KubeconfigTemplate, remoteApiserver.IP)
		rkubeconfig.Write([]byte(data))
		remoteK8sClient, err = testutils.GetK8sClient(remoteKubeconfig)

		// Wait for the api servers to be available.
		Eventually(func() error {
			_, err := localK8sClient.CoreV1().Namespaces().List(metav1.ListOptions{})
			return err
		}, eventuallyTimeout, eventuallyPoll).Should(BeNil())
		Eventually(func() error {
			_, err := remoteK8sClient.CoreV1().Namespaces().List(metav1.ListOptions{})
			return err
		}, eventuallyTimeout, eventuallyPoll).Should(BeNil())

		// For Kubernetes backend on the local cluster, configured the required CRDs. These are defined in libcalico-go
		// which we can get from our vendor directory.
		if !isCalicoEtcdDatastore {
			// Copy CRD registration manifest into the API server container, and apply it.
			err = localApiserver.CopyFileIntoContainer("../../vendor/github.com/projectcalico/libcalico-go/test/crds.yaml", "/crds.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = localApiserver.ExecMayFail("kubectl", "apply", "-f", "/crds.yaml")
			Expect(err).NotTo(HaveOccurred())
		}

		// Run the federation controller on the local cluster.
		federationController = testutils.RunFederationController(
			localEtcd.IP,
			lkubeconfig.Name(),
			[]string{remoteKubeconfig},
			isCalicoEtcdDatastore,
		)

		// Create two test namespaces in both kubernetes clusters.
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns1Name,
			},
			Spec: v1.NamespaceSpec{},
		}
		_, err = localK8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).NotTo(HaveOccurred())
		_, err = remoteK8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).NotTo(HaveOccurred())

		ns = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns2Name,
			},
			Spec: v1.NamespaceSpec{},
		}
		_, err = localK8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).NotTo(HaveOccurred())
		_, err = remoteK8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).NotTo(HaveOccurred())
	}

	makeService := func(base *v1.Service, name string, prev *v1.Service) *v1.Service {
		copy := base.DeepCopy()
		if prev == nil {
			copy.Name = name
			return copy
		}
		copy.ObjectMeta = *prev.ObjectMeta.DeepCopy()
		return copy
	}
	makeEndpoints := func(base *v1.Endpoints, name string, prev *v1.Endpoints) *v1.Endpoints {
		copy := base.DeepCopy()
		if prev == nil {
			copy.Name = name
			return copy
		}
		copy.ObjectMeta = *prev.ObjectMeta.DeepCopy()
		return copy
	}

	AfterEach(func() {
		federationController.Stop()
		localApiserver.Stop()
		localEtcd.Stop()
		remoteApiserver.Stop()
		remoteEtcd.Stop()
		os.Remove(remoteKubeconfig)
	})

	DescribeTable("Test with specific local Calico datastore type", func(isCalicoEtcdDatastore bool) {

		By("Setting up the local and remote clusters")
		setup(isCalicoEtcdDatastore)

		By("Creating two identical backing services and endpoints")
		svcBacking1, err := localK8sClient.CoreV1().Services(ns1Name).Create(makeService(svc1, "backing1", nil))
		Expect(err).NotTo(HaveOccurred())

		_, err = localK8sClient.CoreV1().Endpoints(ns1Name).Create(makeEndpoints(eps1, "backing1", nil))
		Expect(err).NotTo(HaveOccurred())

		_, err = localK8sClient.CoreV1().Services(ns1Name).Create(makeService(svc1, "backing2", nil))
		Expect(err).NotTo(HaveOccurred())

		epsBacking2, err := localK8sClient.CoreV1().Endpoints(ns1Name).Create(makeEndpoints(eps1, "backing2", nil))
		Expect(err).NotTo(HaveOccurred())

		By("Creating a service with endpoints in a different namespace but the correct labels")
		_, err = localK8sClient.CoreV1().Services(ns2Name).Create(makeService(svc2, "wrongns", nil))
		Expect(err).NotTo(HaveOccurred())

		_, err = localK8sClient.CoreV1().Endpoints(ns2Name).Create(makeEndpoints(eps2, "wrongns", nil))
		Expect(err).NotTo(HaveOccurred())

		By("Creating a federated service which matches the two indentical backing services")
		fedCfg, err := localK8sClient.CoreV1().Services(ns1Name).Create(&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "federated",
				Namespace: ns1Name,
				Annotations: map[string]string{
					"federation.tigera.io/serviceSelector": "federate == 'yes'",
				},
			},
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Name:     "port1",
						Port:     8080,
						Protocol: v1.ProtocolUDP,
					},
					{
						Name:     "port2",
						Port:     80,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints contain the expected set ips/ports")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal([]v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "2.0.0.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port2",
						Port:     1234,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		}))

		By("Updating backing2 to have a different set of endpoints")
		epsBacking2, err = localK8sClient.CoreV1().Endpoints(ns1Name).Update(makeEndpoints(eps2, "backing2", epsBacking2))
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints contain the expected set ips/ports [2]")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal([]v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "2.0.0.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port2",
						Port:     1234,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		}))

		By("Updating backing1 to have no labels")
		svcBacking1.Labels = nil
		svcBacking1, err = localK8sClient.CoreV1().Services(ns1Name).Update(svcBacking1)
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints contain the expected set ips/ports [3]")
		// Store this set of expected endpoints as we'll use it a few times below.
		es := []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
		}
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal(es))

		By("Removing the federation annotation")
		fedCfg.Annotations = nil
		fedCfg, err = localK8sClient.CoreV1().Services(ns1Name).Update(fedCfg)
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints has been deleted")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(BeNil())

		By("Adding the federation annotation back")
		fedCfg.Annotations = map[string]string{
			"federation.tigera.io/serviceSelector": "federate == 'yes'",
		}
		fedCfg, err = localK8sClient.CoreV1().Services(ns1Name).Update(fedCfg)
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints contain the expected set ips/ports [4]")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal(es))

		By("Removing the federation selector from the federated endpoints to disable controller updates")
		eps, err := localK8sClient.CoreV1().Endpoints(ns1Name).Get("federated", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		eps.Annotations = nil
		eps, err = localK8sClient.CoreV1().Endpoints(ns1Name).Update(eps)
		Expect(err).NotTo(HaveOccurred())

		By("Modifying the federation annotation to be a no match")
		fedCfg.Annotations = map[string]string{
			"federation.tigera.io/serviceSelector": "federate == 'idontthinkso'",
		}
		fedCfg, err = localK8sClient.CoreV1().Services(ns1Name).Update(fedCfg)
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints remains unchanged")
		Consistently(getSubsetsFn(ns1Name, "federated"), consistentlyTimeout, consistentlyPoll).Should(Equal(es))
		Expect(getSubsets(ns1Name, "federated")).ToNot(BeNil())

		By("Removing the federated endpoints completely")
		err = localK8sClient.CoreV1().Endpoints(ns1Name).Delete("federated", &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Modifying the federation annotation again - but still is a no match")
		fedCfg.Annotations = map[string]string{
			"federation.tigera.io/serviceSelector": "foo == 'bar'",
		}
		fedCfg, err = localK8sClient.CoreV1().Services(ns1Name).Update(fedCfg)
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints is recreated but contains no subsets")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).ShouldNot(BeNil())
		Expect(getSubsets(ns1Name, "federated")).To(HaveLen(0))

		By("Adding the federation annotation back")
		fedCfg.Annotations = map[string]string{
			"federation.tigera.io/serviceSelector": "federate == 'yes'",
		}
		fedCfg, err = localK8sClient.CoreV1().Services(ns1Name).Update(fedCfg)
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints contain the expected set ips/ports [5]")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal(es))

		By("Adding backing service to the remote cluster")
		_, err = remoteK8sClient.CoreV1().Services(ns1Name).Create(makeService(svc1, "backing1", nil))
		Expect(err).NotTo(HaveOccurred())

		_, err = remoteK8sClient.CoreV1().Endpoints(ns1Name).Create(makeEndpoints(eps1, "backing1", nil))
		Expect(err).NotTo(HaveOccurred())

		By("Adding remote cluster config")
		rcc, err := localCalicoClient.RemoteClusterConfigurations().Create(ctx, &apiv3.RemoteClusterConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-remote",
			},
			Spec: apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "kubernetes",
				KubeConfig: apiv3.KubeConfig{
					Kubeconfig: remoteKubeconfig,
				},
			},
		}, options.SetOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints contain the expected set ips/ports [6]")
		// Any port that has a name in the target object, should be updated to include the remote cluster name.
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal([]v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "my-remote/pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "10.10.10.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "2.0.0.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port2",
						Port:     1234,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		}))

		By("Deleting the local services")
		localK8sClient.CoreV1().Services(ns1Name).Delete("backing1", &metav1.DeleteOptions{})
		localK8sClient.CoreV1().Services(ns1Name).Delete("backing2", &metav1.DeleteOptions{})
		localK8sClient.CoreV1().Services(ns2Name).Delete("wrongns", &metav1.DeleteOptions{})

		By("Checking the federated endpoints contain the expected set ips/ports [7]")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(Equal([]v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.1",
						NodeName: &node1Name,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "1.0.0.2",
						NodeName: &node2Name,
						TargetRef: &v1.ObjectReference{
							Kind:      "Pod",
							Namespace: ns1Name,
							Name:      "my-remote/pod1",
						},
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port1",
						Port:     1234,
						Protocol: v1.ProtocolUDP,
					},
				},
			},
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "2.0.0.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "port2",
						Port:     1234,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		}))

		By("Deleting the RemoteClusterConfiguration")
		_, err = localCalicoClient.RemoteClusterConfigurations().Delete(ctx, "my-remote", options.DeleteOptions{ResourceVersion: rcc.ResourceVersion})
		Expect(err).NotTo(HaveOccurred())

		By("Checking the federated endpoints is present but contains no subsets")
		Eventually(getSubsetsFn(ns1Name, "federated"), eventuallyTimeout, eventuallyPoll).Should(HaveLen(0))
		Expect(getSubsets(ns1Name, "federated")).ToNot(BeNil())
	},
		Entry("etcd datastore", true),
		Entry("kubernetes datastore", false))
})
