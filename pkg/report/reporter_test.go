// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/flow"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

// Fake replayer
type fakeReplayer struct {
	started bool
	stopped bool
}

func (r *fakeReplayer) Start(ctx context.Context) {
	defer GinkgoRecover()

	r.started = true
	Eventually(func() bool { return r.stopped }, "5s", "0.1s").Should(BeTrue())
}

// Fake auditer.
type fakeAuditer struct {
	created []resources.Resource
	deleted []resources.Resource
	patched []resources.Resource
	updated []resources.Resource
}

func (a *fakeAuditer) SearchAuditEvents(ctx context.Context, filter *apiv3.AuditEventsSelection, start, end *time.Time) <-chan *event.AuditEventResult {
	ch := make(chan *event.AuditEventResult)

	send := func(verb string, rs []resources.Resource) {
		for _, r := range rs {
			var ro *runtime.Unknown
			tm := resources.GetTypeMeta(r)
			rh := resources.GetResourceHelperByTypeMeta(tm)
			if verb != event.VerbDelete {
				raw, _ := json.Marshal(r)
				ro = &runtime.Unknown{
					TypeMeta: runtime.TypeMeta{
						Kind:       r.GetObjectKind().GroupVersionKind().Kind,
						APIVersion: r.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					},
					Raw: raw,
				}
			}
			ch <- &event.AuditEventResult{
				Event: &auditv1.Event{
					Stage: auditv1.StageResponseComplete,
					Verb:  verb,
					ObjectRef: &auditv1.ObjectReference{
						Name:       r.GetObjectMeta().GetName(),
						Namespace:  r.GetObjectMeta().GetNamespace(),
						APIGroup:   r.GetObjectKind().GroupVersionKind().Group,
						APIVersion: r.GetObjectKind().GroupVersionKind().Version,
						Resource:   rh.Plural(),
					},
					ResponseObject: ro,
				},
			}
		}
	}

	go func() {
		defer close(ch)
		send("create", a.created)
		send("patch", a.patched)
		send("update", a.updated)
		send("delete", a.deleted)
	}()

	return ch
}

type fakeReportStorer struct {
	data *ArchivedReportData
	time time.Time
}

func (r *fakeReportStorer) StoreArchivedReport(d *ArchivedReportData, t time.Time) error {
	r.data = d
	r.time = t
	return nil
}

// Fake flow reporter
type fakeFlowReporter struct {
}

func (f *fakeFlowReporter) SearchFlowLogs(ctx context.Context, namespaces []string, start, end *time.Time) <-chan *flow.FlowLogResult {
	return nil
}

var _ = Describe("Report tests", func() {
	var r *reporter
	var xc *XrefCacheTester
	var healthCnt int
	var replayer *fakeReplayer
	var auditer *fakeAuditer
	var reportStorer *fakeReportStorer
	var stop func()

	BeforeEach(func() {
		// Reset the health count.
		healthCnt = 0

		// Create a config.
		baseCfg := config.MustLoadConfig()
		baseCfg.ReportName = "report"
		cfg := &Config{
			Config: *baseCfg,
			Report: &v3.GlobalReport{
				ObjectMeta: metav1.ObjectMeta{
					Name: "report",
				},
				Spec: apiv3.ReportSpec{
					ReportType: "report-type",
					Schedule:   "@daily",
					Endpoints: &apiv3.EndpointsSelection{
						Selector: "has(label1)",
					},
				},
			},
			ReportType: &v3.GlobalReportType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "report-type",
				},
				Spec: apiv3.ReportTypeSpec{
					IncludeEndpointData:  true,
					AuditEventsSelection: &apiv3.AuditEventsSelection{},
				},
			},
		}

		// We'll use an xrefcache tester to feed in config.
		xc = NewXrefCacheTester()

		// Create a reporter "by hand" passing in test interfaces.
		replayer = &fakeReplayer{}
		auditer = &fakeAuditer{}
		reportStorer = &fakeReportStorer{}
		auditer = &fakeAuditer{}
		flowReporter := &fakeFlowReporter{}
		r = &reporter{
			ctx: context.Background(),
			cfg: cfg,
			clog: logrus.WithFields(logrus.Fields{
				"name":  cfg.Report.Name,
				"type":  cfg.ReportType.Name,
				"start": cfg.ParsedReportStart.Format(time.RFC3339),
				"end":   cfg.ParsedReportEnd.Format(time.RFC3339),
			}),
			auditer:          auditer,
			flowlogger:       flowReporter,
			archiver:         reportStorer,
			xc:               xc.XrefCache,
			replayer:         replayer,
			healthy:          func() { healthCnt++ },
			inScopeEndpoints: make(map[apiv3.ResourceID]*reportEndpoint),
			services:         make(map[apiv3.ResourceID]xrefcache.CacheEntryFlags),
			namespaces:       make(map[apiv3.ResourceID]xrefcache.CacheEntryFlags),
			serviceAccounts:  resources.NewSet(),
			policies:         resources.NewSet(),
			data: &apiv3.ReportData{
				ReportName:     "report",
				ReportTypeName: "report-type",
				ReportSpec:     cfg.Report.Spec,
				ReportTypeSpec: cfg.ReportType.Spec,
				StartTime:      metav1.Time{cfg.ParsedReportStart},
				EndTime:        metav1.Time{cfg.ParsedReportEnd},
			},
			flowLogFilter: flow.NewFlowLogFilter(),
		}

		// Start the reporter and wait until start has been called.
		var completed bool
		go func() {
			r.run()
			completed = true
		}()
		Eventually(func() bool { return replayer.started }, "5s", "0.1s").Should(BeTrue())

		// Send an in-sync so that we start gathering data for the report.
		xc.OnStatusUpdate(syncer.StatusUpdate{
			Type: syncer.StatusTypeInSync,
		})

		stop = func() {
			xc.OnStatusUpdate(syncer.StatusUpdate{
				Type: syncer.StatusTypeComplete,
			})

			// This will cause the Start() method to return.
			replayer.stopped = true

			// Which will in turn cause run() to finish.
			Eventually(func() bool { return completed }, "5s", "0.1s").Should(BeTrue())
		}
	})

	It("should handle no data at all", func() {
		stop()
	})

	It("should handle filtering policy based on endpoints", func() {
		By("applying pod1 IP1 (this matches the EP selector)")
		pod1 := xc.SetPod(Name1, Namespace1, Label1, IP1, Name1, NoPodOptions)
		pod1ID := resources.GetResourceID(pod1)

		By("applying pod2 IP2 (this does not match the EP selector)")
		xc.SetPod(Name2, Namespace1, Label2, IP2, Name2, NoPodOptions)

		By("Setting GNP1, NP1 and k8sNP1 to match pod1 only")
		gnp1 := xc.SetGlobalNetworkPolicy(Name1, Select1,
			nil,
			[]apiv3.Rule{},
		)
		np1 := xc.SetNetworkPolicy(Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{},
		)
		knp1 := xc.SetK8sNetworkPolicy(Name1, Namespace1, Select1,
			nil,
			[]networkingv1.NetworkPolicyEgressRule{},
		)

		By("Setting GNP2, NP2 and k8sNP2 to match pod2 only")
		gnp2 := xc.SetGlobalNetworkPolicy(Name2, Select2,
			nil,
			[]apiv3.Rule{},
		)
		np2 := xc.SetNetworkPolicy(Name2, Namespace1, Select2,
			nil,
			[]apiv3.Rule{},
		)
		knp2 := xc.SetK8sNetworkPolicy(Name2, Namespace1, Select2,
			nil,
			[]networkingv1.NetworkPolicyEgressRule{},
		)

		By("Updating GNP1, NP1 and k8sNP1 to match pod2 only -  they should all remain in-scope though")
		gnp1_2 := xc.SetGlobalNetworkPolicy(Name1, Select2,
			nil,
			[]apiv3.Rule{},
		)
		np1_2 := xc.SetNetworkPolicy(Name1, Namespace1, Select2,
			nil,
			[]apiv3.Rule{},
		)
		knp1_2 := xc.SetK8sNetworkPolicy(Name1, Namespace1, Select2,
			nil,
			[]networkingv1.NetworkPolicyEgressRule{},
		)

		By("Setting the auditer to include events for all of these resource types")
		auditer.created = []resources.Resource{
			gnp1, gnp2, np1, np2, knp1, knp2,
		}
		auditer.patched = []resources.Resource{
			gnp1, gnp2, np1, np2, knp1, knp2,
		}
		auditer.updated = []resources.Resource{
			gnp1, gnp2, np1, np2, knp1, knp2, gnp1_2, np1_2, knp1_2,
		}
		auditer.deleted = []resources.Resource{
			gnp1, gnp2,
		}

		By("Completing the event replay")
		stop()

		By("Checking that pod1 is the only enumerated endpoint")
		Expect(reportStorer.data).ToNot(BeNil())
		Expect(reportStorer.data.EndpointsSummary.NumTotal).To(Equal(1))
		Expect(reportStorer.data.EndpointsSummary.NumServiceAccounts).To(Equal(1))
		Expect(reportStorer.data.NamespacesSummary.NumTotal).To(Equal(1))
		Expect(reportStorer.data.Endpoints).To(HaveLen(1))
		Expect(reportStorer.data.Endpoints[0].Endpoint).To(Equal(pod1ID))

		By("Checking that pod1 has the correct three applied policies")
		Expect(reportStorer.data.Endpoints[0].AppliedPolicies).To(HaveLen(3))
		Expect(reportStorer.data.Endpoints[0].AppliedPolicies).To(ConsistOf(
			resources.GetResourceID(gnp1),
			resources.GetResourceID(np1),
			resources.GetResourceID(knp1),
		))

		By("Checking that the audit logs were filtered based on the in-scope endpoint policies")
		Expect(reportStorer.data.AuditSummary.NumCreate).To(Equal(3))
		Expect(reportStorer.data.AuditSummary.NumModify).To(Equal(9))
		Expect(reportStorer.data.AuditSummary.NumDelete).To(Equal(1))
		Expect(reportStorer.data.AuditSummary.NumTotal).To(Equal(13))
		Expect(reportStorer.data.AuditEvents).To(HaveLen(13))
	})
})
