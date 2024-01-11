// Copyright 2021 Tigera Inc. All rights reserved.

package forwarder

import (
	"context"
	"net/http"
	"net/url"
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
	lmaAPI "github.com/projectcalico/calico/lma/pkg/api"
	lma "github.com/projectcalico/calico/lma/pkg/elastic"
)

var _ = Describe("Event forwarder", func() {
	var (
		ctx         context.Context
		cancel      context.CancelFunc
		esSvc       *storage.Service
		clusterName string
		startTime   time.Time
		endTime     time.Time
		totalDocs   int
		lmaESCli    lma.Client
	)

	BeforeEach(func() {
		now := time.Now()
		startTime = now.Add(time.Duration(-2) * time.Minute)

		clusterName = "cluster"
		err := os.Setenv("CLUSTER_NAME", clusterName)
		Expect(err).ShouldNot(HaveOccurred())

		u := &url.URL{
			Scheme: "http",
			Host:   "localhost:9200",
		}
		ctx, cancel = context.WithCancel(context.Background())
		lmaESCli, err = lma.New(&http.Client{}, u, "", "", clusterName, 1, 0, true, 0, 5)
		if err != nil {
			panic("could not create unit under test: " + err.Error())
		}

		_, err = lmaESCli.Backend().DeleteIndex("tigera_secure_ee_events*").Do(ctx)
		Expect(err).NotTo(HaveOccurred())

		err = lmaESCli.CreateEventsIndex(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		// mock controller runtime client.
		scheme := scheme.Scheme
		err = v3.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		esSvc = storage.NewService(lmaESCli, nil, fakeClient, "")
		esSvc.Run(ctx)

		// Populate events index with enough test data that needs scrolling
		totalDocs = 1550
		for i := 0; i < totalDocs; i++ {
			_, err := lmaESCli.PutSecurityEvent(ctx, lmaAPI.EventsData{
				Time:            time.Now().Unix(),
				Type:            "global_alert",
				Description:     "test event fwd",
				Severity:        100,
				Origin:          "event-fwd-resource",
				SourceNamespace: "sample-fwd-ns",
				DestNameAggr:    "sample-dest-*",
				Host:            "node0",
				Record:          map[string]string{"key1": "value1", "key2": "value2"},
			})
			Expect(err).ShouldNot(HaveOccurred())
		}

		now = time.Now()
		endTime = now.Add(time.Duration(2) * time.Minute)

		// Sleep for the data to persist in ES
		time.Sleep(10 * time.Second)
	})

	AfterEach(func() {
		_, err := lmaESCli.Backend().DeleteIndex("tigera_secure_ee_events*").Do(ctx)
		Expect(err).NotTo(HaveOccurred())
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
			events:     esSvc,
			dispatcher: dispatcher,
			config:     &storage.ForwarderConfig{},
		}

		err := eventFwdr.retrieveAndForward(startTime, endTime, 1, 30*time.Second)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() int { return dispatchCount }).Should(Equal(totalDocs))
	})
})
