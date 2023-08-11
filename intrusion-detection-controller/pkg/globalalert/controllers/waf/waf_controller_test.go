package waf

import (
	"context"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WAF Controller", func() {
	var (
		numOfAlerts = 2
		mockClient = client.NewMockClient("", rest.MockResult{})
		wafCache = WafEventsCache{
			lastWafTimestamp: time.Now(),
		}
		wac        = &wafAlertController{
			clusterName:      "clusterName",
			wafLogs:          newMockWAFLogs(mockClient, "clustername"),
			events:           newMockEvents(mockClient, "clustername"),
			eventsCache: wafCache,
		}
	)

	Context("Test Waf Controller", func() {
		It("Test Waf ProcessWAFLogs", func() {
			ctx := context.Background()

			err := wac.ProcessWafLogs(ctx)
			Expect(err).ToNot(HaveOccurred())

			now := time.Now()
			params := &v1.WAFLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: wac.eventsCache.lastWafTimestamp,
						To:   now,
					},
				},
			}

			logs, err := wac.events.List(ctx, params)
			Expect(err).ToNot(HaveOccurred())

			Expect(logs).To(Equal(numOfAlerts))

		})
	})

	Context("Test WAF Caching", func() {
		It("Test WAF caching", func() {
			ctx := context.Background()

			err := wac.ProcessWafLogs(ctx)
			Expect(err).ToNot(HaveOccurred())

			now := time.Now()
			params := &v1.WAFLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: wac.eventsCache.lastWafTimestamp,
						To:   now,
					},
				},
			}

			logs, err := wac.events.List(ctx, params)
			Expect(err).ToNot(HaveOccurred())

			Expect(logs).To(Equal(numOfAlerts))

			// run the process again to make sure no new events are generated
			err = wac.ProcessWafLogs(ctx)
			Expect(err).ToNot(HaveOccurred())

			params.QueryParams.TimeRange.To = time.Now()

			logs2, err := wac.events.List(ctx, params)
			Expect(err).ToNot(HaveOccurred())
			// no new Events should have been created 
			Expect(logs2).To(Equal(numOfAlerts))

		})
	})

})
