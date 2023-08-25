package waf

import (
	"context"
	"math/rand"
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
		ctx context.Context
		numOfAlerts = 2
		mockClient  = client.NewMockClient("", rest.MockResult{})
		wafCache    = WafLogsCache{
			lastWafTimestamp: time.Now(),
		}
		wac = &wafAlertController{
			clusterName: "clusterName",
			wafLogs:     newMockWAFLogs(mockClient, "clustername"),
			events:      newMockEvents(mockClient, "clustername"),
			logsCache:   wafCache,
		}
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("Test Waf Controller", func() {
		It("Test Waf ProcessWAFLogs", func() {
			ctx := context.Background()

			err := wac.ProcessWafLogs(ctx)
			Expect(err).ToNot(HaveOccurred())

			now := time.Now()
			params := &v1.WAFLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: wac.logsCache.lastWafTimestamp,
						To:   now,
					},
				},
			}

			logs, err := wac.events.List(ctx, params)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(logs.Items)).To(Equal(numOfAlerts))

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
						From: wac.logsCache.lastWafTimestamp,
						To:   now,
					},
				},
			}

			logs, err := wac.events.List(ctx, params)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(logs.Items)).To(Equal(numOfAlerts))

			// run the process again to make sure no new events are generated
			err = wac.ProcessWafLogs(ctx)
			Expect(err).ToNot(HaveOccurred())

			params.QueryParams.TimeRange.To = time.Now()

			logs2, err := wac.events.List(ctx, params)
			Expect(err).ToNot(HaveOccurred())
			// no new Events should have been created
			Expect(len(logs2.Items)).To(Equal(numOfAlerts))

		})
		It("Test Waf Cache Management", func() {
			wac.logsCache.wafLogs = genCacheInfo()
			wac.ManageCache(ctx)
			Expect(len(wac.logsCache.wafLogs)).To(Equal(10))
		})
	})

})

func genCacheInfo() []cacheInfo{
	cache := []cacheInfo{}
	for i := 1; i <= 11; i++ {
		t := rand.Intn(25)
		newEntry := cacheInfo{
			timestamp: time.Now().Add(-(time.Duration(t) * time.Minute)),
			requestID: "",
		}
		cache = append(cache, newEntry)
	}

	oldEntry := cacheInfo{
		timestamp: time.Now().Add(-(31* time.Minute)),
		requestID: "",
	}

	cache = append(cache, oldEntry)

	return cache
}
