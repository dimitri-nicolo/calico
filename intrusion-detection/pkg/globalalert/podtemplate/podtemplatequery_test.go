package podtemplate

import (
	context "context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	fakebatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1/fake"
	"k8s.io/client-go/testing"
)

var (
	podtemplatequery ADPodTemplateQuery
	mockK8sClient    kubernetes.Interface

	ctx    context.Context
	cancel context.CancelFunc
)
var _ = Describe("PodTemplateQuery", func() {
	BeforeEach(func() {
		mockK8sClient = fakeK8s.NewSimpleClientset()
		podtemplatequery = NewPodTemplateQuery(mockK8sClient)

		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
	})

	Context("GetPodTemplate", func() {
		It("returns error received from k8s client", func() {
			mockK8sClient.BatchV1().(*fakebatchv1.FakeBatchV1).PrependReactor("get", "podtemplates",
				func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("Error fetching podtemplate")
				})

			_, err := podtemplatequery.GetPodTemplate(ctx, "mocknamespace", "mockpodtemplate")
			Expect(err).ToNot(BeNil())
		})

		It("returns intendedPodTemplate", func() {
			mockK8sClient.BatchV1().(*fakebatchv1.FakeBatchV1).PrependReactor("get", "podtemplates",
				func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.PodTemplate{}, nil
				})

			podtemplate, err := podtemplatequery.GetPodTemplate(ctx, "mocknamespace", "mockpodtemplate")
			Expect(podtemplate).ToNot(BeNil())
			Expect(err).To(BeNil())
		})
	})
})
