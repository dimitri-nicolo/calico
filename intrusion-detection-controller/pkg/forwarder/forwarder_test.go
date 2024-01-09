// Copyright 2021 Tigera Inc. All rights reserved.

package forwarder

import (
	"context"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/storage"
	v3 "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	lsv1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var _ = Describe("Event forwarder", func() {
	var (
		ctx            context.Context
		cancel         context.CancelFunc
		storageService *storage.Service
		clusterName    string
		startTime      time.Time
		endTime        time.Time
		totalDocs      int
	)

	BeforeEach(func() {
		now := time.Now()
		startTime = now.Add(time.Duration(-2) * time.Minute)

		clusterName = "cluster"
		err := os.Setenv("CLUSTER_NAME", clusterName)
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel = context.WithCancel(context.Background())

		// mock controller runtime client.
		scheme := scheme.Scheme
		err = v3.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		storageService = storage.NewService(nil, fakeClient, "")
		storageService.Run(ctx)

		// Populate events index with enough test data that needs scrolling
		totalDocs = 1550
		for i := 0; i < totalDocs; i++ {
			err := storageService.PutSecurityEventWithID(ctx, []lsv1.Event{
				{
					Time:            lsv1.NewEventDate(time.Now()),
					Type:            "global_alert",
					Description:     "test event fwd",
					Severity:        100,
					Origin:          "event-fwd-resource",
					SourceNamespace: "sample-fwd-ns",
					DestNameAggr:    "sample-dest-*",
					Host:            "node0",
					Record:          map[string]string{"key1": "value1", "key2": "value2"},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
		}

		now = time.Now()
		endTime = now.Add(time.Duration(2) * time.Minute)
	})

	It("should read from events index and dispatches them", func() {
		dispatcher := &MockLogDispatcher{}
		dispatcher.On("Initialize").Return(nil)
		dispatchCount := 0
		dispatcher.On("Dispatch", mock.Anything).Run(func(args mock.Arguments) {
			for _, c := range dispatcher.ExpectedCalls {
				if c.Method == "Dispatch" {
					dispatchCount++
				}
			}
		}).Return(nil)

		eventFwdr := &eventForwarder{
			logger: log.WithFields(log.Fields{
				"context": "eventforwarder",
				"uid":     "fwd-test",
			}),
			once:       sync.Once{},
			cancel:     cancel,
			ctx:        ctx,
			events:     storageService,
			dispatcher: dispatcher,
			config:     &storage.ForwarderConfig{},
		}

		err := eventFwdr.retrieveAndForward(startTime, endTime, 1, 30*time.Second)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() int { return dispatchCount }).Should(Equal(totalDocs))
	})
})
