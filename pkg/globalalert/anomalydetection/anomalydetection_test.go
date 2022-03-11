package anomalydetection

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/podtemplate"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeK8s "k8s.io/client-go/kubernetes/fake"

	idcontroller "github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"

	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
)

const (
	alertName   = "sample-test"
	clusterName = "ut-cluster"
	namespace   = "ut-namespace"
)

var (
	mockCalicoCLI                  calicoclient.Interface
	mockK8sClient                  kubernetes.Interface
	mockPodTemplateQuery           *podtemplate.MockPodTemplateQuery
	mockAnomalyDetectionController *idcontroller.MockADJobController
	ctx                            context.Context
	cancel                         context.CancelFunc
)

var _ = Describe("AnomalyDetection Service", func() {

	var adjService Service

	now := time.Now()
	lastExecutedTime := now.Add(-2 * time.Second)

	defaultSampleGlobalAlert := &v3.GlobalAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: alertName,
		},
		Spec: v3.GlobalAlertSpec{
			Type:        v3.GlobalAlertTypeAnomalyDetection,
			Detector:    "port-scan",
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

	BeforeEach(func() {
		mockCalicoCLI = fake.NewSimpleClientset(defaultSampleGlobalAlert)
		mockK8sClient = fakeK8s.NewSimpleClientset()
		mockPodTemplateQuery = &podtemplate.MockPodTemplateQuery{}
		mockAnomalyDetectionController = &idcontroller.MockADJobController{}

		ctx, cancel = context.WithCancel(context.Background())
		mockPodTemplateQuery.On("GetPodTemplate", ctx, namespace, ADDetectionJobTemplateName).Return(defaultPodTemplate, nil)
		mockAnomalyDetectionController.On("RemoveManagedJob", mock.AnythingOfType("string")).Return(nil)

		var err error
		adjService, err = NewService(mockCalicoCLI, mockK8sClient, mockPodTemplateQuery, mockAnomalyDetectionController,
			clusterName, namespace, defaultSampleGlobalAlert)

		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
		adjService.Stop()
	})

	It("Start exits with error globalAlert status if podtemplatequery throws error", func() {
		errMockeTemplateQuery := &podtemplate.MockPodTemplateQuery{}
		errMockeTemplateQuery.On("GetPodTemplate", mock.Anything, mock.Anything, mock.Anything).Return(
			nil, errors.New("unsuccessful attempt at retrieving adj podtemplate"),
		)

		var err error
		adjService, err = NewService(mockCalicoCLI, mockK8sClient, errMockeTemplateQuery, mockAnomalyDetectionController,
			clusterName, namespace, defaultSampleGlobalAlert)

		Expect(err).ShouldNot(HaveOccurred())

		result := adjService.Start(ctx)

		Expect(len(result.ErrorConditions)).To(BeNumerically(">", 0))
		Expect(result.Healthy).To(BeFalse())
		Expect(result.Active).To(BeFalse())
	})
})
