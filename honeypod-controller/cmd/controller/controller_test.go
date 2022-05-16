// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	controller "github.com/projectcalico/calico/honeypod-controller/cmd/controller"
	"github.com/projectcalico/calico/honeypod-controller/pkg/events"
	hp "github.com/projectcalico/calico/honeypod-controller/pkg/processor"
	"github.com/projectcalico/calico/honeypod-controller/pkg/snort"

	"github.com/projectcalico/calico/lma/pkg/api"
	"github.com/projectcalico/calico/lma/pkg/elastic"
)

var _ = Describe("Test Honeypod Controller Processor Test", func() {
	var (
		c   elastic.Client
		cfg *elastic.Config
		ctx context.Context
		p   *hp.HoneyPodLogProcessor
	)

	BeforeEach(func() {
		// Setup Elastic Client Config
		var err error
		cfg = elastic.MustLoadConfig()
		cfg.ElasticURI = "http://localhost:9200"
		cfg.ParsedElasticURL, _ = url.Parse(cfg.ElasticURI)
		c, err = elastic.NewFromConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		// Set up context
		ctx = context.Background()

		// Create event index and alias
		exists, err := c.EventsIndexExists(ctx)
		Expect(err).NotTo(HaveOccurred())
		if !exists {
			err := c.CreateEventsIndex(ctx)
			Expect(err).NotTo(HaveOccurred())
		}

		// Create HoneyPodLogProcessor
		p = hp.NewHoneyPodLogProcessor(c, ctx)
		Expect(p).NotTo(BeNil())
	})

	It("Should be able to create 1 Honeypod event to Elastisearch", func() {
		snortEvent := snort.Alert{
			SigName:    "snort.alert.signame",
			Category:   "snort.alert.category",
			DateSrcDst: "snort.alert.datesrcdst",
			Flags:      "snort.alert.flags",
			Other:      "snort.alert.other",
		}
		// Values are translated from test/honeypod_alert_good
		count := int64(1)
		hostKeyword := "my-host-keyword"
		he := events.HoneypodEvent{
			EventsData: api.EventsData{
				DestNameAggr:    "tigera-internal-3-6b97f5d974-*",
				DestNamespace:   "tigera-internal",
				SourceNameAggr:  "attacker-app-774579d456-*",
				SourceNamespace: "default",
				Record: events.HoneypodAlertRecord{
					Count:       &count,
					HostKeyword: &hostKeyword,
				},
			},
		}

		data := he.EventData()
		err := snort.SendEvents([]snort.Alert{snortEvent}, p, &data)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should be able retrieve 1 alert from Elasticsearch", func() {
		// We retrieve the previous Honeypod Event by limiting the time
		end := time.Now().Add(1 * time.Minute)
		start := end.Add(-2 * time.Minute)
		var eventsData *api.EventsData
		// Elastic require a bit of time to store the entry so we loop until we see the entry
		for eventsData == nil {
			for eventResult := range p.Client.SearchSecurityEvents(p.Ctx, &start, &end, nil, false) {
				eventsData = eventResult.EventsData
			}
		}
		// We want to check that the previous entry was created and retrievable.
		Expect(eventsData.Type).To(Equal("honeypod"))
		Expect(eventsData.Description).To(Equal("[Alert] Signature Triggered on tigera-internal/tigera-internal-3-6b97f5d974-*"))
		Expect(eventsData.Host).To(Equal("my-host-keyword"))
		Expect(eventsData.Severity).To(Equal(100))
		Expect(eventsData.Origin).To(Equal("honeypod-controller.snort"))
		Expect(eventsData.DestNameAggr).To(Equal("tigera-internal-3-6b97f5d974-*"))
		Expect(eventsData.DestNamespace).To(Equal("tigera-internal"))
		Expect(eventsData.SourceNameAggr).To(Equal("attacker-app-774579d456-*"))
		Expect(eventsData.SourceNamespace).To(Equal("default"))

		b, err := json.Marshal(eventsData.Record)
		Expect(err).NotTo(HaveOccurred())
		record := &events.HoneypodSnortEventRecord{}
		err = json.Unmarshal(b, record)
		Expect(err).NotTo(HaveOccurred())
		Expect(record.Snort.Description).To(Equal("snort.alert.signame"))
		Expect(record.Snort.Category).To(Equal("snort.alert.category"))
		Expect(record.Snort.Occurrence).To(Equal("snort.alert.datesrcdst"))
		Expect(record.Snort.Flags).To(Equal("snort.alert.flags"))
		Expect(record.Snort.Other).To(Equal("snort.alert.other"))
	})

	It("should be able to retrieve Honeypod Event in Elasticsearch and find relevant pcap", func() {
		end := time.Now().Add(1 * time.Minute)
		start := end.Add(-2 * time.Minute)
		var eventsData *api.EventsData
		// Elastic require a bit of time to store the entry so we loop until we see the entry
		for eventsData == nil {
			for eventResult := range p.Client.SearchSecurityEvents(p.Ctx, &start, &end, nil, false) {
				eventsData = eventResult.EventsData
			}
		}
		Expect(eventsData.Origin).To(Equal("honeypod-controller.snort"))
		// We modify the path to pcap due to being a test
		path := "../../test/pcap"
		// Once we can see that the entry was created, we try to retrieve the pcap location
		matches, err := controller.GetPcaps(eventsData, path)
		if err != nil {
			fmt.Println("Failed to get pcaps")
		}
		Expect(matches[0]).To(Equal("../../test/pcap/tigera-internal/capture-honey/tigera-internal-3-6b97f5d974-vd6c2_calid322b8d6606.pcap"))
	})

	It("should be able to simulate a snort scan and find scan result", func() {
		// Values are translated from test/honeypod_alert_good
		t, err := time.Parse(time.RFC3339, "2020-09-25T20:16:37.312Z")
		Expect(err).NotTo(HaveOccurred())
		hostKeyword := "garwood-bz-n990-kadm-infra-0"
		alert := &api.EventsData{
			Description:     "[Honeypod] Fake debug service accessed by default/attacker-app-774579d456-* on port 8080",
			DestNameAggr:    "tigera-internal-3-6b97f5d974-*",
			DestNamespace:   "tigera-internal",
			Origin:          "honeypod.fake.svc",
			Severity:        100,
			SourceNameAggr:  "attacker-app-774579d456-*",
			SourceNamespace: "default",
			Time:            t.Unix(),
			Type:            "honeypod",
			Record: events.HoneypodAlertRecord{
				HostKeyword: &hostKeyword,
			},
		}
		// We modify the path to snort alert due to being a test
		snortPath := "../../test/snort"
		// We pass our pre-create alerts to be processed
		var store = snort.NewStore(t)
		err = snort.ProcessSnort(alert, p, snortPath, store)
		Expect(err).NotTo(HaveOccurred())
		end := time.Now()
		start := end.Add(-10 * time.Minute)
		// We should be seeing 3 entries, loop until all 3 are filled
		var result2 []string
		for len(result2) < 3 {
			result2 = []string{}
			for eventResult := range p.Client.SearchSecurityEvents(p.Ctx, &start, &end, nil, false) {
				// filter out snort events from previous test cases by alert.record["host.keyword"]
				if eventResult.Host == hostKeyword {
					result2 = append(result2, eventResult.Origin)
				}
			}
		}
		// We count to the number of honeypod snort entries
		count := 0
		for _, entry := range result2 {
			if entry == "honeypod-controller.snort" {
				count += 1
			}
		}
		Expect(count).To(Equal(3))
	})
})
