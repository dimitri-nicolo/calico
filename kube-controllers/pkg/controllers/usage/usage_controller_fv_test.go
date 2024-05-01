package usage

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcalico/calico/felix/fv/containers"
	"github.com/projectcalico/calico/kube-controllers/tests/testutils"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	usagev1 "github.com/projectcalico/calico/libcalico-go/lib/apis/usage.tigera.io/v1"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	licenseClient "github.com/projectcalico/calico/licensing/client"
	"github.com/projectcalico/calico/licensing/utils"
)

const (
	secondsBetweenUsageReports = 4
	secondsPerDay              = 60 * 60 * 24
	reportsPerDay              = secondsPerDay / secondsBetweenUsageReports
	reportsPerTest             = 3
)

// These FVs are responsible for ensuring that the following is under test:
// - usageController: whether it constructs its pipeline correctly and that the pipeline functions against a real datastore
// - reportWriter: whether it enriches basic reports properly for different values of license presence, last report UID, and uptime
// These FVs are _NOT_ responsible for the following being under test - they are handled by UTs:
// - reportGenerator: whether basic reports are generated correctly in all conceivable permutations of input events
// - reportWriter: whether retries of report writing are performed correctly, and that incomplete reports are not written to the datastore.
// - usageController: whether it handles stop channel sends correctly
var _ = Describe("Calico usage controller FV tests (KDD mode)", func() {
	var (
		etcd              *containers.Container
		controller        *containers.Container
		apiserver         *containers.Container
		usageClient       runtimeClient.Client
		calicoClient      clientv3.Interface
		k8sClient         *kubernetes.Clientset
		controllerManager *containers.Container
		kconfigfile       *os.File
	)

	BeforeEach(func() {
		// Run etcd.
		etcd = testutils.RunEtcd()

		// Run apiserver.
		apiserver = testutils.RunK8sApiserver(etcd.IP)

		// Write out a kubeconfig file.
		var err error
		kconfigfile, err = os.CreateTemp("", "ginkgo-usage-controller")
		Expect(err).NotTo(HaveOccurred())
		//defer os.Remove(kconfigfile.Name())
		data := testutils.BuildKubeconfig(apiserver.IP)
		_, err = kconfigfile.Write([]byte(data))
		Expect(err).NotTo(HaveOccurred())
		Expect(kconfigfile.Chmod(os.ModePerm)).NotTo(HaveOccurred())

		// Create the k8s client from the kubeconfig file.
		k8sClient, err = testutils.GetK8sClient(kconfigfile.Name())
		Expect(err).NotTo(HaveOccurred())

		// Wait for the API server to be available.
		listNamespaces := func() error {
			_, err := k8sClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			return err
		}
		Eventually(listNamespaces, 30*time.Second, 1*time.Second).Should(BeNil())
		Consistently(listNamespaces, 10*time.Second, 1*time.Second).Should(BeNil())

		// Apply the necessary CRDs. There can sometimes be a delay between starting
		// the API server and when CRDs are apply-able, so retry here.
		apply := func() error {
			out, err := apiserver.ExecOutput("kubectl", "apply", "-f", "/crds/")
			if err != nil {
				return fmt.Errorf("%s: %s", err, out)
			}
			return nil
		}
		Eventually(apply, 10*time.Second).ShouldNot(HaveOccurred())

		// Make a Calico client.
		calicoClient = testutils.GetCalicoClient(apiconfig.Kubernetes, "", kconfigfile.Name())

		// Make a usage client.
		config, err := clientcmd.BuildConfigFromFlags("", kconfigfile.Name())
		Expect(err).NotTo(HaveOccurred())
		usageClient, err = createUsageClient(config)
		Expect(err).NotTo(HaveOccurred())

		// Create nodes.
		_, err = k8sClient.CoreV1().Nodes().Create(context.Background(),
			&v1.Node{
				TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
				Spec:       v1.NodeSpec{},
				Status: v1.NodeStatus{
					Capacity: v1.ResourceList{
						v1.ResourceCPU: resource.MustParse("10"),
					},
				},
			},
			metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sClient.CoreV1().Nodes().Create(context.Background(),
			&v1.Node{
				TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "node-b"},
				Spec:       v1.NodeSpec{},
				Status: v1.NodeStatus{
					Capacity: v1.ResourceList{
						v1.ResourceCPU: resource.MustParse("20"),
					},
				},
			},
			metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Run the usage controller.
		controller = runUsageControllerForFV(apiconfig.Kubernetes, kconfigfile.Name(), reportsPerDay)

		// Run controller manager.
		controllerManager = testutils.RunK8sControllerManager(apiserver.IP)
	})

	AfterEach(func() {
		controllerManager.Stop()
		controller.Stop()
		apiserver.Stop()
		etcd.Stop()
	})

	Context("Mainline FV tests", func() {
		for _, loopLicensePresent := range []bool{true, false} {
			licensePresent := loopLicensePresent
			It(fmt.Sprintf("should write usage reports according to the configured reports per day (license present: %v)", licensePresent), func() {
				// Create a license if required.
				var licenseClaims licenseClient.LicenseClaims
				if licensePresent {
					var err error
					licenseKey := utils.ValidEnterpriseTestLicense()
					licenseClaims, err = licenseClient.Decode(*licenseKey)
					Expect(err).NotTo(HaveOccurred())
					_, err = calicoClient.LicenseKey().Create(context.Background(), licenseKey, options.SetOptions{})
					Expect(err).NotTo(HaveOccurred())
				}

				// Get the list of reports, waiting until the expected amount of reports have been flushed.
				var usageReportList usagev1.LicenseUsageReportList
				getUsageReports := func() []usagev1.LicenseUsageReport {
					err := usageClient.List(context.Background(), &usageReportList)
					Expect(err).NotTo(HaveOccurred())
					return usageReportList.Items
				}
				timeout := fmt.Sprintf("%ds", reportsPerTest*secondsBetweenUsageReports*2)
				Eventually(getUsageReports, timeout, "1s").Should(HaveLen(reportsPerTest))

				// Sort the list by time ascending.
				slices.SortFunc(usageReportList.Items, func(a, b usagev1.LicenseUsageReport) int {
					return a.CreationTimestamp.Compare(b.CreationTimestamp.Time)
				})

				// Get the kube-system namespace for report validation.
				ksNamespace, err := k8sClient.CoreV1().Namespaces().Get(context.Background(), "kube-system", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				reportDatas := convertToTypedReportData(usageReportList.Items)
				for _, report := range reportDatas {
					// Validate interval start/end values.
					intervalLength := report.IntervalEnd.Sub(report.IntervalStart).Seconds()
					Expect(intervalLength).To(BeNumerically("~", secondsBetweenUsageReports, 0.01))

					// Validate subject UID.
					Expect(report.SubjectUID).To(Equal(string(ksNamespace.UID)))

					// Validate license UID.
					Expect(report.LicenseUID).To(Equal(licenseClaims.LicenseID))

					// Validate counts. Min and max should be the same as nodes were static.
					Expect(report.VCPUs).To(Equal(Stats{Min: 30, Max: 30}))
					Expect(report.Nodes).To(Equal(Stats{Min: 2, Max: 2}))
				}

				// Validate reporter uptime values: the first reports value should be something greater than zero, and the seconds should be roughly the time between reports.
				Expect(reportDatas[0].ReporterUptime).To(BeNumerically(">", 0))
				Expect(reportDatas[1].ReporterUptime - reportDatas[0].ReporterUptime).To(BeNumerically("~", secondsBetweenUsageReports, 1))

				// Validate last published report UID values.
				Expect(reportDatas[0].LastPublishedReportUID).To(BeEmpty())
				Expect(reportDatas[1].LastPublishedReportUID).To(Equal(string(usageReportList.Items[0].UID)))

				// Ensure the HMAC can be validated on read-back.
				for _, datastoreReport := range usageReportList.Items {
					computedHMAC := ComputeHMAC(datastoreReport.Spec.ReportData)
					Expect(computedHMAC).To(Equal(datastoreReport.Spec.HMAC))
				}
			})
		}
	})
})

func runUsageControllerForFV(datastoreType apiconfig.DatastoreType, kconfigfile string, reportsPerDay int) *containers.Container {
	return containers.Run("calico-kube-controllers",
		containers.RunOpts{AutoRemove: true},
		"-e", fmt.Sprintf("DATASTORE_TYPE=%s", datastoreType),
		"-e", fmt.Sprintf("KUBECONFIG=%s", kconfigfile),
		"-v", fmt.Sprintf("%s:%s", kconfigfile, kconfigfile),
		"-e", fmt.Sprintf("USAGE_REPORTS_PER_DAY=%d", reportsPerDay),
		"-e", "ENABLED_CONTROLLERS=node,service,federatedservices,usage",
		"-e", "LOG_LEVEL=debug",
		"-e", "KUBE_CONTROLLERS_CONFIG_NAME=default",
		os.Getenv("CONTAINER_NAME"))
}

func convertToTypedReportData(datastoreReports []usagev1.LicenseUsageReport) (reportDatas []LicenseUsageReportData) {
	for _, datastoreReport := range datastoreReports {
		reportData, err := NewLicenseUsageReportDataFromMessage(datastoreReport.Spec.ReportData)
		Expect(err).NotTo(HaveOccurred())
		reportDatas = append(reportDatas, reportData)
	}
	return
}
