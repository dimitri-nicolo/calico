package anomalydetection

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeK8s "k8s.io/client-go/kubernetes/fake"

	adjcontroller "github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/anomalydetection"
	idscontroller "github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/controllers/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/globalalert/podtemplate"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
)

const (
	alertName   = "sample-test"
	clusterName = "ut-cluster"
	namespace   = "ut-namespace"
)

var (
	mockCalicoCLI             calicoclient.Interface
	mockK8sClient             kubernetes.Interface
	mockPodTemplateQuery      *podtemplate.MockPodTemplateQuery
	mockADDetectionController *idscontroller.MockAnomalyDetectionController
	mockADTrainingController  *idscontroller.MockAnomalyDetectionController
	ctx                       context.Context
	cancel                    context.CancelFunc
)

var _ = Describe("AnomalyDetection Service", func() {

	var adjService ADService

	now := time.Now()
	lastExecutedTime := now.Add(-2 * time.Second)

	defaultSampleGlobalAlert := &v3.GlobalAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: alertName,
		},
		Spec: v3.GlobalAlertSpec{
			Type: v3.GlobalAlertTypeAnomalyDetection,
			Detector: &v3.DetectorParams{
				Name: "port-scan",
			},
			Description: fmt.Sprintf("test anomalyDetection alert: %s", alertName),
			Severity:    100,
			Period:      &metav1.Duration{Duration: 5 * time.Second},
			Lookback:    &metav1.Duration{Duration: 1 * time.Second},
		},
		Status: v3.GlobalAlertStatus{
			LastUpdate:   &metav1.Time{Time: now},
			Active:       false,
			Healthy:      true,
			LastExecuted: &metav1.Time{Time: lastExecutedTime},
		},
	}

	defaultPodTemplate := &v1.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mock-adjob-podtemplate",
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mock-adjob-podtemplate-spec",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:    podtemplate.ADJobsContainerName,
						Command: []string{"python", "-m", "adj"},
					},
				},
			},
		},
	}

	noTenant := ""
	tenant := "tenant"

	BeforeEach(func() {
		mockCalicoCLI = fake.NewSimpleClientset(defaultSampleGlobalAlert)
		mockK8sClient = fakeK8s.NewSimpleClientset()
		mockPodTemplateQuery = &podtemplate.MockPodTemplateQuery{}
		mockADDetectionController = &idscontroller.MockAnomalyDetectionController{}
		mockADTrainingController = &idscontroller.MockAnomalyDetectionController{}

		ctx, cancel = context.WithCancel(context.Background())
		mockPodTemplateQuery.On("GetPodTemplate", ctx, namespace, adjcontroller.ADDetectionJobTemplateName).Return(defaultPodTemplate, nil)

		mockADDetectionController.On("AddDetector", mock.AnythingOfType("anomalydetection.DetectionCycleRequest")).Return(nil)
		mockADDetectionController.On("RemoveDetector", mock.AnythingOfType("anomalydetection.DetectionCycleRequest")).Return(nil)

		mockADTrainingController.On("AddDetector", mock.AnythingOfType("anomalydetection.TrainingDetectorsRequest")).Return(nil)
		mockADTrainingController.On("RemoveDetector", mock.AnythingOfType("anomalydetection.TrainingDetectorsRequest")).Return(nil)
	})

	AfterEach(func() {
		cancel()
	})

	DescribeTable("Start exits with error globalAlert status if ADDetectionController throws error", func(tenant string) {
		errMockADDetectionController := &idscontroller.MockAnomalyDetectionController{}
		errMockADDetectionController.On("AddDetector", mock.AnythingOfType("anomalydetection.DetectionCycleRequest")).Return(
			errors.New("unsuccessful attempt at creating detection cycle"),
		)
		errMockADDetectionController.On("RemoveDetector", mock.AnythingOfType("anomalydetection.DetectionCycleRequest")).Return(nil)

		var err error
		adjService, err = NewService(mockCalicoCLI, mockK8sClient, errMockADDetectionController,
			mockADTrainingController, clusterName, tenant, namespace, defaultSampleGlobalAlert)
		defer adjService.Stop()

		Expect(err).ShouldNot(HaveOccurred())

		result := adjService.Start()

		Expect(len(result.ErrorConditions)).To(BeNumerically(">", 0))
		Expect(result.Healthy).To(BeFalse())
		Expect(result.Active).To(BeFalse())
	},
		Entry("no tenant", noTenant),
		Entry("with tenant", tenant),
	)

	DescribeTable("Start exits with error globalAlert status if ADTrainingController throws error", func(tenant string) {
		errMockADTrainingController := &idscontroller.MockAnomalyDetectionController{}
		errMockADTrainingController.On("AddDetector", mock.AnythingOfType("anomalydetection.TrainingDetectorsRequest")).Return(
			errors.New("unsuccessful attempt at adding to training cycle"),
		)
		errMockADTrainingController.On("RemoveDetector", mock.AnythingOfType("anomalydetection.TrainingDetectorsRequest")).Return(nil)

		var err error
		adjService, err = NewService(mockCalicoCLI, mockK8sClient, mockADDetectionController,
			errMockADTrainingController, clusterName, tenant, namespace, defaultSampleGlobalAlert)

		Expect(err).ShouldNot(HaveOccurred())

		result := adjService.Start()

		Expect(len(result.ErrorConditions)).To(BeNumerically(">", 0))
		Expect(result.Healthy).To(BeFalse())
		Expect(result.Active).To(BeFalse())
	},
		Entry("no tenant", noTenant),
		Entry("with tenant", tenant),
	)

	DescribeTable("Stop exits with error globalAlert status if ADDetectionController throws error", func(tenant string) {
		errMockADDetectionController := &idscontroller.MockAnomalyDetectionController{}
		errMockADDetectionController.On("RemoveDetector", mock.AnythingOfType("anomalydetection.DetectionCycleRequest")).Return(
			errors.New("unsuccessful attempt at deleting detection cycle"),
		)
		errMockADDetectionController.On("AddDetector", mock.AnythingOfType("anomalydetection.DetectionCycleRequest")).Return(nil)

		var err error
		adjService, err = NewService(mockCalicoCLI, mockK8sClient, errMockADDetectionController,
			mockADTrainingController, clusterName, tenant, namespace, defaultSampleGlobalAlert)

		Expect(err).ShouldNot(HaveOccurred())

		result := adjService.Stop()

		Expect(len(result.ErrorConditions)).To(BeNumerically(">", 0))
		Expect(result.Healthy).To(BeFalse())
		Expect(result.Active).To(BeFalse())
	},
		Entry("no tenant", noTenant),
		Entry("with tenant", tenant),
	)

	DescribeTable("Stop exits with error globalAlert status if ADTrainingController throws error", func(tenant string) {
		errMockADTrainingController := &idscontroller.MockAnomalyDetectionController{}
		errMockADTrainingController.On("RemoveDetector", mock.AnythingOfType("anomalydetection.TrainingDetectorsRequest")).Return(
			errors.New("unsuccessful attempt at removing from training cycle"),
		)
		errMockADTrainingController.On("AddDetector", mock.AnythingOfType("anomalydetection.TrainingDetectorsRequest")).Return(nil)

		var err error
		adjService, err = NewService(mockCalicoCLI, mockK8sClient, mockADDetectionController,
			errMockADTrainingController, clusterName, tenant, namespace, defaultSampleGlobalAlert)

		Expect(err).ShouldNot(HaveOccurred())

		result := adjService.Stop()

		Expect(len(result.ErrorConditions)).To(BeNumerically(">", 0))
		Expect(result.Healthy).To(BeFalse())
		Expect(result.Active).To(BeFalse())
	},
		Entry("no tenant", noTenant),
		Entry("with tenant", tenant),
	)
})
