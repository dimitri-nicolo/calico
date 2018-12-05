package main_windows_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/containernetworking/cni/pkg/types/current"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/cni-plugin/internal/pkg/testutils"
	"github.com/projectcalico/cni-plugin/internal/pkg/utils"
	"github.com/projectcalico/cni-plugin/pkg/types"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	k8sconversion "github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/logutils"
	"github.com/projectcalico/libcalico-go/lib/names"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func ensureNamespace(clientset *kubernetes.Clientset, name string) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_, err := clientset.CoreV1().Namespaces().Create(ns)
	if errors.IsAlreadyExists(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("Kubernetes CNI tests", func() {
	// Create a random seed
	rand.Seed(time.Now().UTC().UnixNano())
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.InfoLevel)
	hostname, _ := names.Hostname()
	ctx := context.Background()
	calicoClient, err := client.NewFromEnv()
	if err != nil {
		panic(err)
	}

	BeforeEach(func() {
		testutils.WipeK8sPods()
		testutils.WipeEtcd()
	})

	utils.ConfigureLogging("info")
	cniVersion := os.Getenv("CNI_SPEC_VERSION")

	Context("using host-local IPAM", func() {
		netconf := fmt.Sprintf(`
			{
			  "cniVersion": "%s",
			  "name": "net1",
			  "type": "calico",
			  "etcd_endpoints": "%s",
			  "datastore_type": "%s",
			  "ipam": {
			    "type": "host-local",
			    "subnet": "10.0.0.0/8"
			  },
			  "kubernetes": {
			    "k8s_api_root": "%s"
			  },
			  "policy": {"type": "k8s"},
			  "nodename_file_optional": true,
			  "log_level":"debug"
			}`, cniVersion, os.Getenv("ETCD_ENDPOINTS"), os.Getenv("DATASTORE_TYPE"), os.Getenv("KUBERNETES_MASTER"))

		It("successfully networks the namespace", func() {
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			if err != nil {
				panic(err)
			}
			config = testutils.SetCertFilePath(config)
			clientset, err := kubernetes.NewForConfig(config)

			if err != nil {
				panic(err)
			}

			// Create the Namespace before the tests
			var period int64
			period = 10
			log.Infof("\nrosh:: creating new namespace\n")
			_ = clientset.CoreV1().Namespaces().Delete("none", &metav1.DeleteOptions{
				GracePeriodSeconds: &period,
			})
			_, err = clientset.CoreV1().Namespaces().Create(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					//Name:        "test",
					Name:        "none",
					Annotations: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			name := fmt.Sprintf("run%d", rand.Uint32())

			// Create a K8s pod w/o any special params
			//ensureNamespace(clientset, testutils.K8S_TEST_NS)
			ensureNamespace(clientset, "none")
			_, err = clientset.CoreV1().Pods("none").Create(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  name,
						Image: "ignore",
					}},
					NodeName: hostname,
				},
			})
			if err != nil {
				panic(err)
			}

			log.Infof("\nrosh:: creating container\n")
			containerID, result, contVeth, contAddresses, contRoutes, err := testutils.CreateContainer(netconf, name, "none", "")
			Expect(err).ShouldNot(HaveOccurred())
			if err != nil {
				log.Debugf("\nrosh:: containerID :%v , result: %v ,icontVeth : %v , contAddresses : %v ,contRoutes : %v \n", containerID, result, contVeth, contAddresses, contRoutes)
			}

			Expect(len(result.IPs)).Should(Equal(1))
			ip := result.IPs[0].Address.IP.String()
			log.Infof("ip is %v ", ip)
			result.IPs[0].Address.IP = result.IPs[0].Address.IP.To4() // Make sure the IP is respresented as 4 bytes
			Expect(result.IPs[0].Address.Mask.String()).Should(Equal("ff000000"))

			// datastore things:
			// TODO Make sure the profile doesn't exist

			ids := names.WorkloadEndpointIdentifiers{
				Node:         hostname,
				Orchestrator: api.OrchestratorKubernetes,
				Endpoint:     "eth0",
				Pod:          name,
				ContainerID:  containerID,
			}

			wrkload, err := ids.CalculateWorkloadEndpointName(false)
			log.WithField("wrkload: ", wrkload).Info("Akhilesh")
			Expect(err).NotTo(HaveOccurred())

		})

		Context("when a named port is set", func() {
			It("it is added to the workload endpoint", func() {
				config, err := clientcmd.DefaultClientConfig.ClientConfig()
				if err != nil {
					panic(err)
				}
				config = testutils.SetCertFilePath(config)
				clientset, err := kubernetes.NewForConfig(config)

				if err != nil {
					panic(err)
				}

				name := fmt.Sprintf("run%d", rand.Uint32())

				// Create a K8s pod w/o any special params
				_, err = clientset.CoreV1().Pods("none").Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: name},
					Spec: v1.PodSpec{
						Containers: []v1.Container{{
							Name:  fmt.Sprintf("container-%s", name),
							Image: "ignore",
							Ports: []v1.ContainerPort{{
								Name:          "anamedport",
								ContainerPort: 555,
							}},
						}},
						NodeName: hostname,
					},
				})
				if err != nil {
					panic(err)
				}
				containerID, result, contVeth, _, _, err := testutils.CreateContainer(netconf, name, "none", "")
				Expect(err).ShouldNot(HaveOccurred())
				log.Infof("contVeth %v ", contVeth)
				log.Infof("containerID %v ", containerID)

				log.WithField("result %v ", result).Info("AKHILESH")
				Expect(len(result.IPs)).Should(Equal(1))
				result.IPs[0].Address.IP = result.IPs[0].Address.IP.To4() // Make sure the IP is respresented as 4 bytes
				Expect(result.IPs[0].Address.Mask.String()).Should(Equal("ff000000"))

				// datastore things:
				// TODO Make sure the profile doesn't exist

				ids := names.WorkloadEndpointIdentifiers{
					Node:         hostname,
					Orchestrator: api.OrchestratorKubernetes,
					Endpoint:     "eth0",
					Pod:          name,
					ContainerID:  containerID,
				}

				wrkload, err := ids.CalculateWorkloadEndpointName(false)
				interfaceName := k8sconversion.VethNameForWorkload("none", name)
				Expect(err).NotTo(HaveOccurred())
				log.Infof("interfaceName : %v", interfaceName)

				// The endpoint is created
				endpoints, err := calicoClient.WorkloadEndpoints().List(ctx, options.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(endpoints.Items).Should(HaveLen(1))
				log.WithField("endpoints :", endpoints).Info("Akhilesh")

				Expect(endpoints.Items[0].Name).Should(Equal(wrkload))
				Expect(endpoints.Items[0].Namespace).Should(Equal("none"))
				Expect(endpoints.Items[0].Labels).Should(Equal(map[string]string{
					"projectcalico.org/namespace":      "none",
					"projectcalico.org/orchestrator":   api.OrchestratorKubernetes,
					"projectcalico.org/serviceaccount": "default",
				}))
				Expect(endpoints.Items[0].Spec.Pod).Should(Equal(name))
				Expect(endpoints.Items[0].Spec.InterfaceName).Should(Equal(interfaceName))
				Expect(endpoints.Items[0].Spec.Node).Should(Equal(hostname))
				Expect(endpoints.Items[0].Spec.Endpoint).Should(Equal("eth0"))
				Expect(endpoints.Items[0].Spec.ContainerID).Should(Equal(containerID))
				Expect(endpoints.Items[0].Spec.Orchestrator).Should(Equal(api.OrchestratorKubernetes))
				Expect(endpoints.Items[0].Spec.Ports).Should(Equal([]api.EndpointPort{{
					Name:     "anamedport",
					Protocol: numorstring.ProtocolFromString("TCP"),
					Port:     555,
				}}))

				_, err = testutils.DeleteContainerWithId(netconf, name, "none", containerID)
				Expect(err).ShouldNot(HaveOccurred())

				// Make sure there are no endpoints anymore
				endpoints, err = calicoClient.WorkloadEndpoints().List(ctx, options.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(endpoints.Items).Should(HaveLen(0))

			})
		})

	})

	Context("after a pod has already been networked once", func() {
		var nc types.NetConf
		var netconf string
		var workloadName, containerID, name string
		var endpointSpec api.WorkloadEndpointSpec
		var result *current.Result

		checkIPAMReservation := func() {
			// IPAM reservation should still be in place.
			handleID, _ := utils.GetHandleID("calico-uts", containerID, workloadName)
			ipamIPs, err := calicoClient.IPAM().IPsByHandle(context.Background(), handleID)
			ExpectWithOffset(1, err).NotTo(HaveOccurred(), "error getting IPs")
			ExpectWithOffset(1, ipamIPs).To(HaveLen(1),
				"There should be an IPAM handle for endpoint")
			ExpectWithOffset(1, ipamIPs[0].String()+"/32").To(Equal(endpointSpec.IPNetworks[0]))
		}

		BeforeEach(func() {
			time.Sleep(30000 * time.Millisecond)
			// Create a new ipPool.
			testutils.MustCreateNewIPPool(calicoClient, "10.0.0.0/24", false, false, true, 26)

			// Create a network config.
			nc = types.NetConf{
				CNIVersion:           cniVersion,
				Name:                 "calico-uts",
				Type:                 "calico",
				EtcdEndpoints:        fmt.Sprintf("http://%s:2389", os.Getenv("ETCD_IP")),
				DatastoreType:        os.Getenv("DATASTORE_TYPE"),
				Kubernetes:           types.Kubernetes{K8sAPIRoot: os.Getenv("KUBERNETES_MASTER")},
				Policy:               types.Policy{PolicyType: "k8s"},
				NodenameFileOptional: true,
				LogLevel:             "info",
			}
			nc.IPAM.Type = "calico-ipam"
			ncb, err := json.Marshal(nc)
			Expect(err).NotTo(HaveOccurred())
			netconf = string(ncb)

			// Now create a K8s pod.
			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			Expect(err).NotTo(HaveOccurred())
			config = testutils.SetCertFilePath(config)
			clientset, err := kubernetes.NewForConfig(config)
			Expect(err).NotTo(HaveOccurred())
			name = fmt.Sprintf("run%d", rand.Uint32())
			pod, err := clientset.CoreV1().Pods("none").Create(
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{{
							Name:  name,
							Image: "ignore",
						}},
						NodeName: hostname,
					},
				})
			Expect(err).NotTo(HaveOccurred())
			log.Infof("Created POD object: %v", pod)

			// Run the CNI plugin.
			containerID, result, _, _, _, err = testutils.CreateContainer(netconf, name, "none", "")
			Expect(err).ShouldNot(HaveOccurred())
			log.Printf("Unmarshalled result from first ADD: %v\n", result)

			// The endpoint is created in etcd
			endpoints, err := calicoClient.WorkloadEndpoints().List(ctx, options.ListOptions{})
			log.WithField("endpoints:", endpoints).Info("AKHILESH :")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(endpoints.Items).Should(HaveLen(1))
			ids := names.WorkloadEndpointIdentifiers{
				Node:         hostname,
				Orchestrator: api.OrchestratorKubernetes,
				Endpoint:     "eth0",
				Pod:          name,
				ContainerID:  containerID,
			}
			workloadName, err = ids.CalculateWorkloadEndpointName(false)
			log.WithField("workloadName:", workloadName).Info("AKHILESH :")
			Expect(err).NotTo(HaveOccurred())
			Expect(endpoints.Items[0].Name).Should(Equal(workloadName))
			//Expect(endpoints.Items[0].Namespace).Should(Equal(testutils.K8S_TEST_NS))
			Expect(endpoints.Items[0].Namespace).Should(Equal("none"))
			Expect(endpoints.Items[0].Labels).Should(Equal(map[string]string{
				//"projectcalico.org/namespace":      "test",
				"projectcalico.org/namespace":      "none",
				"projectcalico.org/orchestrator":   api.OrchestratorKubernetes,
				"projectcalico.org/serviceaccount": "default",
			}))
			endpointSpec = endpoints.Items[0].Spec
			log.WithField("endpointSpec:", endpointSpec).Info("AKHILESH :")
			Expect(endpointSpec.ContainerID).Should(Equal(containerID))
			checkIPAMReservation()
			time.Sleep(30000 * time.Millisecond)
		})

		AfterEach(func() {
			_, err = testutils.DeleteContainerWithId(netconf, name, "none", containerID)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("a second ADD for the same container should work, assigning a new IP", func() {
			// Try to create the same pod with a different container (so CNI receives the ADD for the same endpoint again)
			resultSecondAdd, _, _, _, err := testutils.RunCNIPluginWithId(netconf, name, "none", "", "new-container-id", "eth0")
			log.WithField("resultSecondAdd:", resultSecondAdd).Info("AKHILESH :")
			Expect(err).NotTo(HaveOccurred())
			log.Printf("Unmarshalled result from second ADD: %v\n", resultSecondAdd)

			// The IP addresses shouldn't be the same, since we'll reassign one.
			log.Infof("resultSecondAdd.IPs: %v and result.IPs: %v ", resultSecondAdd.IPs, result.IPs)
			Expect(resultSecondAdd.IPs).ShouldNot(Equal(result.IPs))

			// Otherwise, they should be the same.
			resultSecondAdd.IPs = nil
			result.IPs = nil
			Expect(resultSecondAdd).Should(Equal(result))

			// IPAM reservation should still be in place.
			checkIPAMReservation()
			time.Sleep(30000 * time.Millisecond)
		})
	})

	Context("With a /29 IPAM blockSize", func() {
		netconf := fmt.Sprintf(`
					{
					  "cniVersion": "%s",
					  "name": "net10",
					  "type": "calico",
					  "etcd_endpoints": "http://%s:2389",
					  "datastore_type": "%s",
	           			  "nodename_file_optional": true,
					  "windows_use_single_network":true,
					  "log_level": "debug",
				 	  "ipam": {
					    "type": "calico-ipam"
					  },
					  "kubernetes": {
					    "k8s_api_root": "%s"
					  },
					  "policy": {"type": "k8s"}
					}`, cniVersion, os.Getenv("ETCD_IP"), os.Getenv("DATASTORE_TYPE"), os.Getenv("KUBERNETES_MASTER"))

		It("with windows single network flag set,should successfully network 4 pods but reject networking 5th", func() {
			// Create a new ipPool.
			time.Sleep(30000 * time.Millisecond)
			testutils.MustCreateNewIPPool(calicoClient, "10.0.0.0/26", false, false, true, 29)

			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			Expect(err).NotTo(HaveOccurred())

			config = testutils.SetCertFilePath(config)
			clientset, err := kubernetes.NewForConfig(config)
			Expect(err).NotTo(HaveOccurred())

			// Now create a K8s pod.
			name := "mypod-1"
			for i := 0; i < 4; i++ {
				name = fmt.Sprintf("mypod-%d", i)
				pod, err := clientset.CoreV1().Pods("none").Create(
					&v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:  name,
								Image: "ignore",
							}},
							NodeName: hostname,
						},
					})

				Expect(err).NotTo(HaveOccurred())
				log.Infof("Created POD object: %v", pod)

				// Create the container, which will call CNI and by default it will create the container with interface name 'eth0'.
				containerID, result, _, _, _, err := testutils.CreateContainer(netconf, name, "none", "")
				Expect(err).ShouldNot(HaveOccurred())
				log.WithField("result: ", result).Info("AKHILESH")
				time.Sleep(10000 * time.Millisecond)
				// Make sure the pod gets cleaned up, whether we fail or not.
				defer func() {
					_, err := testutils.DeleteContainerWithId(netconf, name, "none", containerID)
					Expect(err).ShouldNot(HaveOccurred())
				}()
			}
			pod, err := clientset.CoreV1().Pods("none").Create(
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mypod-5",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{{
							Name:  "mypod-5",
							Image: "ignore",
						}},
						NodeName: hostname,
					},
				})

			Expect(err).NotTo(HaveOccurred())
			log.Infof("Created POD object: %v", pod)

			// Create the container, which will call CNI and by default it will create the container with interface name 'eth0'.
			containerID, result, _, _, _, err := testutils.CreateContainer(netconf, name, "none", "")
			Expect(err).Should(HaveOccurred())
			log.WithField("result: ", result).Info("AKHILESH")
			time.Sleep(10000 * time.Millisecond)
			// Make sure the pod gets cleaned up, whether we fail or not.
			defer func() {
				_, err := testutils.DeleteContainerWithId(netconf, name, "none", containerID)
				Expect(err).ShouldNot(HaveOccurred())
			}()

		})

	})
	Context("AKHILESH :With a /29 IPAM blockSize", func() {
		netconf := fmt.Sprintf(`
					{
					  "cniVersion": "%s",
					  "name": "net10",
					  "type": "calico",
					  "etcd_endpoints": "http://%s:2389",
					  "datastore_type": "%s",
	           			  "nodename_file_optional": true,
					  "log_level": "debug",
				 	  "ipam": {
					    "type": "calico-ipam"
					  },
					  "kubernetes": {
					    "k8s_api_root": "%s"
					  },
					  "policy": {"type": "k8s"}
					}`, cniVersion, os.Getenv("ETCD_IP"), os.Getenv("DATASTORE_TYPE"), os.Getenv("KUBERNETES_MASTER"))
		It("with windows single network flag not set,should successfully network 4 pods and sucessfully create new network for 5th", func() {
			// Create a new ipPool.
			time.Sleep(30000 * time.Millisecond)
			testutils.MustCreateNewIPPool(calicoClient, "10.0.0.0/26", false, false, true, 29)

			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			Expect(err).NotTo(HaveOccurred())

			config = testutils.SetCertFilePath(config)
			clientset, err := kubernetes.NewForConfig(config)
			Expect(err).NotTo(HaveOccurred())

			// Now create a K8s pod.
			name := "mypod"
			for i := 0; i < 5; i++ {
				name = fmt.Sprintf("mypod-%d", i)
				pod, err := clientset.CoreV1().Pods("none").Create(
					&v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:  name,
								Image: "ignore",
							}},
							NodeName: hostname,
						},
					})

				Expect(err).NotTo(HaveOccurred())
				log.Infof("Created POD object: %v", pod)

				// Create the container, which will call CNI and by default it will create the container with interface name 'eth0'.
				containerID, result, _, _, _, err := testutils.CreateContainer(netconf, name, "none", "")
				Expect(err).ShouldNot(HaveOccurred())
				log.WithField("result: ", result).Info("AKHILESH")
				time.Sleep(10000 * time.Millisecond)
				// Make sure the pod gets cleaned up, whether we fail or not.
				defer func() {
					_, err := testutils.DeleteContainerWithId(netconf, name, "none", containerID)
					Expect(err).ShouldNot(HaveOccurred())
				}()
			}
		})
		It("create 4 pods; delete 3 pods; create 3 pods, should still have only one network", func() {
			// Create a new ipPool.
			time.Sleep(30000 * time.Millisecond)
			testutils.MustCreateNewIPPool(calicoClient, "10.0.0.0/26", false, false, true, 29)

			config, err := clientcmd.DefaultClientConfig.ClientConfig()
			Expect(err).NotTo(HaveOccurred())

			config = testutils.SetCertFilePath(config)
			clientset, err := kubernetes.NewForConfig(config)
			Expect(err).NotTo(HaveOccurred())

			// Now create a K8s pod.
			podName := []string{}
			containerid := []string{}
			name := "mynewpod"
			for i := 0; i < 4; i++ {
				name = fmt.Sprintf("mynewpod-%d", i)
				podName = append(podName, name)
				pod, err := clientset.CoreV1().Pods("none").Create(
					&v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:  name,
								Image: "ignore",
							}},
							NodeName: hostname,
						},
					})

				Expect(err).NotTo(HaveOccurred())
				log.Infof("Created POD object: %v", pod)

				// Create the container, which will call CNI and by default it will create the container with interface name 'eth0'.
				containerID, result, _, _, _, err := testutils.CreateContainer(netconf, name, "none", "")
				Expect(err).ShouldNot(HaveOccurred())
				containerid = append(containerid, containerID)
				log.WithField("result: ", result).Info("AKHILESH")
				time.Sleep(10000 * time.Millisecond)
			}
			for i := 0; i < 3; i++ {
				_, err := testutils.DeleteContainerWithId(netconf, podName[i], "none", containerid[i])
				Expect(err).ShouldNot(HaveOccurred())
			}
			for i := 0; i < 3; i++ {
				name = fmt.Sprintf("recreatenewpod-%d", i)
				pod, err := clientset.CoreV1().Pods("none").Create(
					&v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:  name,
								Image: "ignore",
							}},
							NodeName: hostname,
						},
					})

				Expect(err).NotTo(HaveOccurred())
				log.Infof("Created POD object: %v", pod)

				// Create the container, which will call CNI and by default it will create the container with interface name 'eth0'.
				containerID, result, _, _, _, err := testutils.CreateContainer(netconf, name, "none", "")
				Expect(err).ShouldNot(HaveOccurred())
				log.WithField("result: ", result).Info("AKHILESH")
				time.Sleep(10000 * time.Millisecond)
				defer func() {
					_, err := testutils.DeleteContainerWithId(netconf, name, "none", containerID)
					Expect(err).ShouldNot(HaveOccurred())
				}()
			}
		})
	})
})
