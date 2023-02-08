// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	clientv3 "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	lclient "github.com/projectcalico/calico/licensing/client"
	"github.com/projectcalico/calico/licensing/client/features"
	"github.com/projectcalico/calico/licensing/monitor"

	"github.com/projectcalico/calico/honeypod-controller/pkg/events"
	hp "github.com/projectcalico/calico/honeypod-controller/pkg/processor"
	"github.com/projectcalico/calico/honeypod-controller/pkg/snort"

	"github.com/projectcalico/calico/lma/pkg/api"
	"github.com/projectcalico/calico/lma/pkg/elastic"
)

func getNodeName() (string, error) {
	//Get node name by reading NODENAME env variable.
	nodename := os.Getenv("NODENAME")
	log.Info("Honeypod controller is running on node: ", nodename)
	if nodename == "" {
		return "", fmt.Errorf("empty NODENAME variable")
	}
	return nodename, nil
}

func GetPcaps(e *api.EventsData, path string) ([]string, error) {
	var matches []string
	s := fmt.Sprintf("%s/%s/%s/%s", path, e.DestNamespace, hp.PacketCapture, e.DestNameAggr)
	//Check if packet capture directory is missing and look for pcaps that matches Alert's destination pod
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.WithError(err).Error("/pcap directory missing")
		return matches, err
	}
	matches, err := filepath.Glob(s)
	if err != nil {
		log.WithError(err).Error("Failed to match pcap files")
		return matches, err
	}
	return matches, nil
}

func validateAlerts(res *api.EventResult, node string) error {
	eventsData := res.EventsData
	if !strings.Contains(eventsData.Origin, "honeypod.") {
		return fmt.Errorf("skipping non honeypod alert")
	}

	b, err := json.Marshal(eventsData.Record)
	if err != nil {
		return fmt.Errorf("failed to marshal honeypod alert.record")
	}
	record := &events.HoneypodAlertRecord{}
	err = json.Unmarshal(b, record)
	if err != nil {
		return fmt.Errorf("failed to unmarshal honeypod alert.record")
	}

	if eventsData.DestNameAggr == "" || eventsData.DestNamespace == "" || eventsData.SourceNameAggr == "" || eventsData.SourceNamespace == "" || record.HostKeyword == nil {
		return fmt.Errorf("skipping invalid honeypod alert")
	}

	if *record.HostKeyword != node {
		return fmt.Errorf("skipping non honeypod alert")
	}

	return nil
}

func loop(p *hp.HoneyPodLogProcessor, node string) error {
	// We only look at the past 10min of alerts
	endTime := time.Now()
	startTime := p.LastProcessingTime
	log.Info("Querying Elasticsearch for new Alerts between:", startTime, endTime)

	// We retrieve alerts from elastic and filter
	filteredAlerts := make(map[string]*api.EventsData)

	for eventResult := range p.Client.SearchSecurityEvents(p.Ctx, &startTime, &endTime, nil, false) {
		if eventResult.Err != nil {
			log.WithError(eventResult.Err).Error("Failed querying event logs")
			return eventResult.Err
		}

		err := validateAlerts(eventResult, node)
		if err != nil {
			continue
		}

		// Store HoneyPod in buckets, using destination pod name aggregate
		if filteredAlerts[eventResult.EventsData.DestNameAggr] == nil {
			filteredAlerts[eventResult.EventsData.DestNameAggr] = eventResult.EventsData
		}
	}

	var store = snort.NewStore(p.LastProcessingTime)
	// Parallel processing of HoneyPod alerts
	for _, alert := range filteredAlerts {
		go func(alert *api.EventsData) {
			log.Infof("Processing Alert: %v", alert.Origin)
			// Retrieve Pcap locations
			pcapArray, err := GetPcaps(alert, hp.PcapPath)
			if err != nil {
				log.WithError(err).Error("Failed to retrieve pcaps")
			}
			log.Infof("Alert: %v, scanning: %v", alert.Origin, pcapArray)
			// Run snort on each pcap and send new alerts to Elasticsearch
			for _, pcap := range pcapArray {
				err := snort.RunScanSnort(alert, pcap, hp.SnortPath)
				if err != nil {
					log.WithError(err).Error("Failed to run Snort on pcap")
				}
			}
			err = snort.ProcessSnort(alert, p, hp.SnortPath, store)
			if err != nil {
				log.WithError(err).Error("Failed to process Snort on pcap")
			}
			log.Infof("Alert: %v scanning completed", alert.Origin)
		}(alert)
	}

	p.LastProcessingTime = endTime

	return nil
}

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() bapi.Client
}

func main() {
	// Get Default Elastic client config, then modify URL
	log.Info("Honeypod controller started")
	cfg := elastic.MustLoadConfig()

	// Try to connect to Elasticsearch
	c, err := elastic.NewFromConfig(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initiate ES client.")
	}
	// Set up context
	ctx := context.Background()

	// Check if required index exists
	exists, err := c.EventsIndexExists(ctx)
	if err != nil || !exists {
		log.WithError(err).Fatal("Failed to check event index existence.")
	}

	clientCalico, err := clientv3.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("Failed to build calico client")
	}

	licenseMonitor := monitor.New(clientCalico.(backendClientAccessor).Backend())
	err = licenseMonitor.RefreshLicense(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get license from datastore; continuing without a license")
	}

	licenseChangedChan := make(chan struct{})

	// Define some of the callbacks for the license monitor. Any changes just send a signal back on the license changed channel.
	licenseMonitor.SetFeaturesChangedCallback(func() {
		licenseChangedChan <- struct{}{}
	})

	licenseMonitor.SetStatusChangedCallback(func(newLicenseStatus lclient.LicenseStatus) {
		licenseChangedChan <- struct{}{}
	})

	// Start the license monitor, which will trigger the callback above at start of day and then whenever the license
	// status changes.
	go func() {
		err := licenseMonitor.MonitorForever(context.Background())
		if err != nil {
			log.WithError(err).Warn("Error while continuously monitoring the license.")
		}
	}()

	p := hp.NewHoneyPodLogProcessor(c, ctx)
	// Retrieve controller's running NodeName
	node, err := getNodeName()
	if err != nil {
		log.WithError(err).Fatal("Error getting NodeName")
	}

	ticker := time.NewTicker(10 * time.Minute)
	done := make(chan bool)

	for {
		hasLicense := licenseMonitor.GetFeatureStatus(features.ThreatDefense)

		select {
		case <-ticker.C:
			if hasLicense {
				// Create HoneyPodLogProcessor and Es Writer
				// Start controller loop
				log.Info("Honeypod controller loop started")
				err = loop(p, node)
				if err != nil {
					log.WithError(err).Error("Error running controller loop")
				}
			} else {
				log.Info("Skip beat due to missing license")
			}
		case <-done:
			log.Info("Received done")
			return
		case <-licenseChangedChan:
			log.Info("License status has changed")
			continue
		}
	}
}
