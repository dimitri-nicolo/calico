package managedcluster

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/waf"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
)

var _ = Describe("Managed Cluster Reconcile", func() {
	var (
		mcr             managedClusterReconciler
		clusterName     = "test-cluster"
		clusterName2    = "test-cluster-2"
		wafClusterName  = "WAF-test-cluster"
		wafClusterName2 = "WAF-test-cluster-2"
		namespace       = "default"
	)

	BeforeEach(func() {
		k8sConfig, err := rest.InClusterConfig()
		if err != nil {
			k8sConfig = &rest.Config{}
		}
		mockClient := waf.NewMockClient()
		calicoctl := util.ManagedClusterClient(k8sConfig, "", "")
		mcr = managedClusterReconciler{
			client:                          MockClientWithWatch{},
			alertNameToAlertControllerState: map[string]alertControllerState{},
			createManagedCalicoCLI:          calicoctl,
			lsClient:                        mockClient,
		}

	})

	It("Managed Cluster Reconcile: reconcile cluster add connected cluster", func() {

		Expect(mcr.alertNameToAlertControllerState[clusterName].alertController).To(BeNil())
		Expect(mcr.alertNameToAlertControllerState[wafClusterName].alertController).To(BeNil())

		Expect(mcr.alertNameToAlertControllerState[clusterName2].alertController).To(BeNil())
		Expect(mcr.alertNameToAlertControllerState[wafClusterName2].alertController).To(BeNil())

		err := mcr.Reconcile(types.NamespacedName{Name: clusterName, Namespace: namespace})

		Expect(err).To(BeNil())

		Expect(mcr.alertNameToAlertControllerState[clusterName]).To(Not(BeNil()))
		Expect(mcr.alertNameToAlertControllerState[wafClusterName]).To(Not(BeNil()))

		err = mcr.Reconcile(types.NamespacedName{Name: clusterName2, Namespace: namespace})

		Expect(err).To(BeNil())

		Expect(mcr.alertNameToAlertControllerState[clusterName2]).To(Not(BeNil()))
		Expect(mcr.alertNameToAlertControllerState[wafClusterName2]).To(Not(BeNil()))

		// change the conditions of the managedcluster struct to say it's disconnected
		for _, cluster := range Clusters {
			if cluster.ObjectMeta.Name == clusterName || cluster.ObjectMeta.Name == wafClusterName {
				*cluster = v3.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:      cluster.ObjectMeta.Name,
						Namespace: cluster.ObjectMeta.Namespace,
					},
					Status: v3.ManagedClusterStatus{
						Conditions: []v3.ManagedClusterStatusCondition{
							{
								Status: v3.ManagedClusterStatusValueFalse,
							},
						},
					},
				}
			}
		}

		err = mcr.Reconcile(types.NamespacedName{Name: clusterName, Namespace: namespace})

		Expect(err).To(BeNil())
		Expect(mcr.alertNameToAlertControllerState[clusterName].alertController).To(BeNil())
		Expect(mcr.alertNameToAlertControllerState[wafClusterName].alertController).To(BeNil())

		Expect(mcr.alertNameToAlertControllerState[clusterName2]).To(Not(BeNil()))
		Expect(mcr.alertNameToAlertControllerState[wafClusterName2]).To(Not(BeNil()))

	})

})
