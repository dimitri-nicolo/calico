// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"fmt"

	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"

	hp "github.com/tigera/honeypod-controller/pkg/processor"
	"github.com/tigera/honeypod-controller/pkg/snort"
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

func GetPcaps(a *api.Alert, path string) ([]string, error) {
	var matches []string
	s := fmt.Sprintf("%s/%s/%s/%s", path, *a.Record.DestNamespace, hp.PacketCapture, *a.Record.DestNameAggr)
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

func validateAlerts(res *api.AlertResult, node string) error {

	if !strings.Contains(res.Alert.Alert, "honeypod.") {
		return fmt.Errorf("skipping non honeypod alert")
	}

	record := res.Alert.Record
	if record.DestNameAggr == nil || record.DestNamespace == nil || record.SourceNameAggr == nil || record.SourceNamespace == nil || record.HostKeyword == nil {
		return fmt.Errorf("skipping invalid honeypod alert")
	}

	if *record.HostKeyword != node {
		return fmt.Errorf("skipping non honeypod alert")
	}

	return nil
}

func loop(p *hp.HoneyPodLogProcessor, node string) error {
	//We only look at the past 10min of alerts
	endTime := time.Now()
	startTime := p.LastProcessingTime
	log.Info("Querying Elasticsearch for new Alerts between:", startTime, endTime)

	//We retrieve alerts from elastic and filter
	filteredAlerts := make(map[string]*api.Alert)
	for e := range p.LogHandler.SearchAlertLogs(p.Ctx, nil, &startTime, &endTime) {
		if e.Err != nil {
			log.WithError(e.Err).Error("Failed querying alert logs")
			return e.Err
		}

		err := validateAlerts(e, node)
		if err != nil {
			continue
		}

		//Store HoneyPod in buckets, using destination pod name aggregate
		if filteredAlerts[*e.Alert.Record.DestNameAggr] == nil {
			filteredAlerts[*e.Alert.Record.DestNameAggr] = e.Alert
		}
	}

	var store = snort.NewStore(p.LastProcessingTime)
	//Parallel processing of HoneyPod alerts
	for _, alert := range filteredAlerts {
		// Alina: go leak routines
		go func(alert *api.Alert) {
			log.Infof("Processing Alert: %v", alert.Alert)
			//Retrieve Pcap locations
			pcapArray, err := GetPcaps(alert, hp.PcapPath)
			if err != nil {
				log.WithError(err).Error("Failed to retrieve pcaps")
			}
			log.Infof("Alert: %v, scanning: %v", alert.Alert, pcapArray)
			//Run snort on each pcap and send new alerts to Elasticsearch
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
			log.Infof("Alert: %v scanning completed", alert.Alert)
		}(alert)
	}

	p.LastProcessingTime = endTime

	return nil
}

func main() {

	//Get Default Elastic client config, then modify URL
	log.Info("Honeypod controller started")
	cfg := elastic.MustLoadConfig()

	//Try to connect to Elasticsearch
	c, err := elastic.NewFromConfig(cfg)
	if err != nil {
		log.WithError(err).Panic("Failed to initiate ES client.")
	}
	//Set up context
	ctx := context.Background()

	//Check if required index exists
	exists, err := c.Backend().IndexExists(hp.Index).Do(context.Background())
	if err != nil || !exists {
		log.WithError(err).Panic("Error unable to access Index: ", hp.Index)
	}

	//Create HoneyPodLogProcessor and Es Writer
	p := hp.NewHoneyPodLogProcessor(c, ctx)
	//Retrieve controller's running NodeName
	node, err := getNodeName()
	if err != nil {
		log.WithError(err).Panic("Error getting NodeName")
	}
	//Start controller loop
	// Alina : Make a ticker
	for c := time.Tick(10 * time.Minute); ; <-c {
		log.Info("Honeypod controller loop started")
		// Alina: What is loop lasts longer ?
		err = loop(p, node)
		if err != nil {
			log.WithError(err).Error("Error running controller loop")
		}
	}
}
