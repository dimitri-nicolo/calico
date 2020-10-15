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
	api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"

	hp "github.com/tigera/honeypod-controller/pkg/processor"
	"github.com/tigera/honeypod-controller/pkg/snort"
)

func GetNodeName() (string, error) {
	//Get node name by using Get pod on our own pod
	nodename := os.Getenv("NODENAME")
	log.Info("Honeypod controller is running on node: ", nodename)
	if nodename == "" {
		return "", fmt.Errorf("Empty NodeName")
	}
	return nodename, nil
}

func GetPcaps(e *api.Alert, path string) ([]string, error) {
	var matches []string
	s := fmt.Sprintf("%s/%s/%s/%s", path, *e.Record.DestNamespace, hp.PacketCapture, *e.Record.DestNameAggr)
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

func Loop(p *hp.HoneypodLogProcessor, node string) error {
	//We only look at the past 10min of alerts
	endTime := time.Now()
	startTime := p.LastProcessingTime
	log.Info("Querying Elasticsearch for new Alerts between:", startTime, endTime)

	//We retrieve alerts from elastic and filter
	filteredAlerts := make(map[string](*api.Alert))
	for e := range p.LogHandler.SearchAlertLogs(p.Ctx, nil, &startTime, &endTime) {
		if e.Err != nil {
			log.WithError(e.Err).Error("Failed querying alert logs")
			return e.Err
		}

		//Skip alerts thats not Honeypod related and not on our node
		if !strings.Contains(e.Alert.Alert, "honeypod.") || *e.Record.HostKeyword != node {
			continue
		}
		log.Info("Valid alert Found: ", e.Alert)
		//Store Honeypod in buckets, using destination pod name aggregate
		if filteredAlerts[*e.Alert.Record.DestNameAggr] == nil {
			filteredAlerts[*e.Alert.Record.DestNameAggr] = e.Alert
		}
	}

	//Parallel processing of Honeypod alerts`
	for _, alert := range filteredAlerts {
		alertCopy := alert
		go func() {
			//Retrieve Pcap locations
			pcapArray, err := GetPcaps(alertCopy, hp.PcapPath)
			if err != nil {
				log.WithError(err).Error("Failed to retrieve Pcaps")
			}
			log.Info("Honeypod Controller scanning: ", pcapArray)
			//Run snort on each pcap and send new alerts to Elasticsearch
			for _, pcap := range pcapArray {
				err := snort.ScanSnort(alertCopy, pcap, hp.SnortPath)
				if err != nil {
					log.WithError(err).Error("Failed to run snort on pcap")
				}
			}
			err = snort.ProcessSnort(alertCopy, p, hp.SnortPath)
			if err != nil {
				log.WithError(err).Error("Failed to process snort on pcap")
			}
		}()
	}

	log.Info("Honeypod controller loop completed")
	p.UpdateLastProcessTime(endTime)

	return nil
}

func main() {

	//Get Default Elastic client config, then modify URL
	log.Info("Honeypod Controller started")
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

	//Create HoneypodLogProcessor
	p, err := hp.NewHoneypodLogProcessor(c, ctx)
	if err != nil {
		log.WithError(err).Panic("Unable to create HoneypodLog Processor")
	}
	//Retrieve controller's running Nodename
	node, err := GetNodeName()
	if err != nil {
		log.WithError(err).Panic("Error getting Nodename")
	}
	//Start controller loop
	for c := time.Tick(10 * time.Minute); ; <-c {
		log.Info("Honeypod controller loop started")
		snort.GlobalSnort = []snort.Snort{}
		err = Loop(&p, node)
		if err != nil {
			log.WithError(err).Error("Error running controller loop")
		}
	}
}
