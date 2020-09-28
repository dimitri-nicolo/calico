package main_test

import (
	"net/url"
	"context"
	"fmt"
	"time"
	"encoding/json"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/lma/pkg/elastic"
        api "github.com/tigera/lma/pkg/api"

        controller "github.com/tigera/honeypod-controller/cmd/controller"
        "github.com/tigera/honeypod-controller/pkg/snort"
        hp "github.com/tigera/honeypod-controller/pkg/processor"

)

var _ = Describe("Test Honeypod Controller Processor Test", func() {
	// Setup Elastic Client Config
        cfg := elastic.MustLoadConfig()
        cfg.ElasticURI = "http://localhost:9200"
        cfg.ParsedElasticURL, _ = url.Parse(cfg.ElasticURI)
        c, err := elastic.NewFromConfig(cfg)
        if err != nil {
            fmt.Println("Failed to initiate ES client.")
	    return
        }
        //Set up context
        ctx := context.Background()
        //Create HoneypodLogProcessor
        p, err := hp.NewHoneypodLogProcessor(c, ctx)
	if err != nil {
            fmt.Println("Failed to initiate HoneypodLogProcessor.")
	    return
	}
	It("Should be able to create index mapping to Elastisearch", func() {
	    //Insert mapping for Event Index to simulate actual Elasticsearch instance
            _, err = p.Client.Backend().CreateIndex("tigera_secure_ee_events.cluster").BodyJson(map[string]interface{}{"mappings": json.RawMessage(hp.EventMapping),}).Do(ctx)
	    Expect(err).To(BeNil())
	})
        It("Should be able to create 1 event to Elastisearch", func() {
            json_res := map[string]interface{}{
                "severity": 100,
                "description": "Test Event",
                "alert": "test.event",
                "type" : "alert",
	        "dest_namespace": "Don't Care",
	        "source_namespace": "Don't Care",
		"record": map[string]interface{}{
                    "source_name_aggr": "Don't Care",
	            "source_namespace": "Don't Care",
	            "dest_namespace": "Don't Care",
	            "host.keyword": "Don't Care",
                    "dest_name_aggr": "Don't Care",
                    "count": 1,
		},
                "time" : time.Now(),
            }
	    //Create 1 generic entry to ensure it works
            _, err = p.Client.Backend().Index().Index("tigera_secure_ee_events.cluster").BodyJson(json_res).Do(ctx)
	    Expect(err).To(BeNil())
	})
        It("Should be able retrieve 1 alert from Elasticsearch(empty)", func() {
            end := time.Now()
            start := end.Add(-48 * time.Hour)
	    var result string
	    //Elastic require a bit of time to store the entry so we loop until we see the entry
	    for result == "" {
	        for alert := range p.LogHandler.SearchAlertLogs(ctx, nil, &start, &end) {
		    result = alert.Alert.Alert
	        }
            }
	    //We want to check that the previous entry was created and retrievable.
	    Expect(result).To(Equal("test.event"))
	})
        It("Should be able to create 1 Honeypod event to Elastisearch", func() {
	    //We use a legitimate alert and store it in Elasticsearch
	    reader, err := ioutil.ReadFile("../../test/honeypod_alert_good")
	    if err != nil {
                fmt.Println("Failed to read alert.")
		return
	    }
	    var alert api.Alert
	    err = alert.UnmarshalJSON(reader)
	    if err != nil {
                fmt.Println("Failed to create alert.")
		return
	    }
            _, err = p.Client.Backend().Index().Index("tigera_secure_ee_events.cluster").BodyJson(alert).Do(ctx)
	    Expect(err).To(BeNil())
	})
	It("should be able to retrieve Honeypod Event in Elasticsearch and find relevant pcap", func() {
	    //We retrieve the previous Honeypod Event by limitng the time
            end, err := time.Parse(time.RFC3339 ,"2020-09-25T20:17:37Z")
	    if err != nil {
		fmt.Println("Failed to parse time.")
		return
	    }
            start := end.Add(-48 * time.Hour)
	    var res *api.AlertResult
	    var result string
	    //Elastic require a bit of time to store the entry so we loop until we see the entry
	    for result == "" {
	        for alert := range p.LogHandler.SearchAlertLogs(ctx, nil, &start, &end) {
		    result = alert.Alert.Alert
		    res = alert
	        }
            }
	    Expect(result).To(Equal("honeypod.fake.svc"))
	    //We modify the path to pcap due to being a test
	    path := "../../test/pcap"
	    //Once we can see that the entry was created, we try to retrieve the pcap location
	    matches, err := controller.GetPcaps(res, path)
	    if err != nil {
                fmt.Println("Failed to get pcaps")
	    }
	    Expect(matches[0]).To(Equal("../../test/pcap/tigera-internal/capture-honey/tigera-internal-3-6b97f5d974-vd6c2_calid322b8d6606.pcap"))
        })
	It("should be able to simulate a snort scan and find scan result", func() {
	    //We retrieve the previous Honeypod Event 
	    end, err := time.Parse(time.RFC3339 ,"2020-09-25T20:17:37Z")
	    if err != nil {
		fmt.Println("Failed to parse time.")
		return
	    }
            start := end.Add(-48 * time.Hour)
	    var res *api.AlertResult
	    var result string
	    for result == "" {
	        for alert := range p.LogHandler.SearchAlertLogs(ctx, nil, &start, &end) {
		    result = alert.Alert.Alert
		    res = alert
	        }
            }
	    //We modify the path to snort alert due to being a test
	    snortPath := "../../test/snort"
	    //We pass our pre-create alerts to be processed
	    err = snort.ProcessSnort(res, p , snortPath)
	    if err != nil {
                fmt.Println("Failed to Process snort")
	    }
            end = time.Now()
            start = end.Add(-10 * time.Minute)
	    //We should be seeing 3 entries, loopp until all 3 are filled
	    var result2 []string
	    for len(result2) < 3 {
		result2 = []string{}
	        for alert := range p.LogHandler.SearchAlertLogs(ctx, nil, &start, &end) {
	            result2 = append(result2, alert.Alert.Alert)
	        }
	    }
	    count := 0
	    //We count to the number of honeypod snort entries
	    for _, entry := range result2 {
	        if entry == "honeypod-controller.snort" {
                    count += 1
		}
	    }
	    Expect(count).To(Equal(3))
	})
})
