package event_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/compliance/mockdata/replayer"
	. "github.com/tigera/compliance/pkg/event"
)

var _ = Describe("Event", func() {
	Context("ExtractResourceFromAuditEvent", func() {
		It("should produce a resource GVK that the resources package can work with", func() {
			kubeEvents, err := replayer.GetKubeAuditEvents()
			Expect(err).ToNot(HaveOccurred())
			eeEvents, err := replayer.GetKubeAuditEvents()
			Expect(err).ToNot(HaveOccurred())

			for _, ev := range append(kubeEvents, eeEvents...) {
				res, err := ExtractResourceFromAuditEvent(ev)
				Expect(err).ToNot(HaveOccurred(), "failed on event: "+ev.String())

				gvk := res.GetObjectKind().GroupVersionKind()
				Expect(gvk.Group).To(Equal(ev.ObjectRef.APIGroup))
				Expect(gvk.Version).To(Equal(ev.ObjectRef.APIVersion))
			}
		})
	})
})
