package main_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/cni-plugin/internal/pkg/testutils"
	"github.com/projectcalico/cni-plugin/internal/pkg/utils"
	k8sconversion "github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/logutils"
	"github.com/projectcalico/libcalico-go/lib/names"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// This file is to hold private only tests to try to reduce the possibility of
// merge conflicts from the OS repo.
var _ = Describe("CalicoCni Private", func() {
	// Create a random seed
	rand.Seed(time.Now().UTC().UnixNano())
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
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
	AfterEach(func() {
		testutils.WipeK8sPods()
		testutils.WipeEtcd()
	})

	Describe("Run Calico CNI plugin in K8s mode", func() {
		utils.ConfigureLogging("info")
		cniVersion := os.Getenv("CNI_SPEC_VERSION")

		Context("using host-local IPAM", func() {

			netconf := fmt.Sprintf(`
			{
			  "cniVersion": "%s",
			  "name": "net1",
			  "type": "calico",
			  "etcd_endpoints": "http://%s:2379",
			  "datastore_type": "%s",
			  "ipam": {
			    "type": "host-local",
			    "subnet": "10.0.0.0/8"
			  },
			  "kubernetes": {
			    "k8s_api_root": "http://127.0.0.1:8080"
			  },
			  "policy": {"type": "k8s"},
			  "nodename_file_optional": true,
			  "log_level":"info"
			}`, cniVersion, os.Getenv("ETCD_IP"), os.Getenv("DATASTORE_TYPE"))

			It("converts AWS SecurityGroup annotation to label", func() {
				config, err := clientcmd.DefaultClientConfig.ClientConfig()
				if err != nil {
					panic(err)
				}
				clientset, err := kubernetes.NewForConfig(config)
				ensureNamespace(clientset, "test2")

				if err != nil {
					panic(err)
				}

				name := fmt.Sprintf("run%d", rand.Uint32())

				// Create a K8s pod with AWS SG annotation
				_, err = clientset.CoreV1().Pods("test2").Create(&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        name,
						Annotations: map[string]string{k8sconversion.AnnotationSecurityGroups: "[\"sg-test\"]"},
					},
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
				_, _, _, _, _, contNs, err := testutils.CreateContainer(netconf, name, "test2", "")
				defer func() {
					_, err = testutils.DeleteContainer(netconf, contNs.Path(), name, "test2")
					Expect(err).ShouldNot(HaveOccurred())
				}()

				Expect(err).ShouldNot(HaveOccurred())

				// The endpoint is created
				endpoints, err := calicoClient.WorkloadEndpoints().List(ctx, options.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(endpoints.Items).Should(HaveLen(1))

				Expect(endpoints.Items[0].Labels).Should(
					HaveKeyWithValue(k8sconversion.SecurityGroupLabelPrefix+"/sg-test", ""))
			})
		})
	})
})
