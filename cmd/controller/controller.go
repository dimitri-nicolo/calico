package main

import (
	"context"
	"fmt"
	"time"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
        api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/tigera/honeypod-controller/pkg/snort"
        hp "github.com/tigera/honeypod-controller/pkg/processor"
)

func GetNodeName() (string, error) {
    //Retrieve KubeConfig in Pod
    config, err := rest.InClusterConfig()
    if err != nil {
        return "", fmt.Errorf("Error retrieving cluster config for GetNodeName(): %v", err)
    }
    //Generate KubeApi client
    client, err := kubernetes.NewForConfig(config)
    if err != nil {
        return "", fmt.Errorf("Error getting KubeAPI client for GetNodeName(): %v", err)
    }
    //Get node name by using Get pod on our own pod
    hostname := os.Getenv("HOSTNAME")
    log.Info("Honeypod Controller hostname: ", hostname)
    pod, err := client.CoreV1().Pods("tigera-intrusion-detection").Get(hostname, metav1.GetOptions{})
    if err != nil {
	    return "", fmt.Errorf("Error getting NodeName for GetNodeName(): %v", err)
    }
    return pod.Spec.NodeName, nil
}

func GetPcaps(e *api.AlertResult, path string) ([]string, error) {
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

func Loop(p hp.HoneypodLogProcessor, node string) error {
    //We only look at the past 10min of alerts
    endTime := time.Now()
    startTime := endTime.Add(-10 * time.Minute)
    log.Info("Querying Elasticsearch for new Alerts")

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

	//Retrieve Pcap locations
	pcapArray, err := GetPcaps(e, hp.PcapPath)
	if err != nil {
            log.WithError(e.Err).Error("Failed to retrieve Pcaps")
	    continue
	}

	log.Info("Honeypod Controller scanning: ", pcapArray)
	//Run snort on each pcap and send new alerts to Elasticsearch
	for _ , pcap := range pcapArray {
            err := snort.ScanSnort(e, pcap, hp.SnortPath)
	    if err != nil {
                log.WithError(e.Err).Error("Failed to run snort on pcap")
	    }
	}
	err = snort.ProcessSnort(e, p, hp.SnortPath)
	if err != nil {
            log.WithError(e.Err).Error("Failed to process snort on pcap")
	}
    }
    log.Info("Honeypod controller loop completed")

    return nil
}

func main() {

    //Get Default Elastic client config, then modify URL
    log.Info("Honeypod Controller started")
    cfg := elastic.MustLoadConfig()
    cfg.ElasticURI = "https://tigera-secure-es-http.tigera-elasticsearch.svc:9200"
    cfg.ParsedElasticURL, _ = url.Parse(cfg.ElasticURI)

    //Try to connect to Elasticsearch
    c, err := elastic.NewFromConfig(cfg)
    if err != nil {
        log.WithError(err).Error("Failed to initiate ES client.")
	return
    }
    //Set up context
    ctx := context.Background()

    //Check if required index exists
    exists, err := c.Backend().IndexExists(hp.Index).Do(context.Background())
    if err != nil || !exists {
        log.WithError(err).Error("Error unable to access Index: ", hp.Index)
	return
    }

    //Create HoneypodLogProcessor
    p, err := hp.NewHoneypodLogProcessor(c, ctx)
    if err != nil {
       log.WithError(err).Error("Unable to create HoneypodLog Processor")
       return
    }
    //Retrieve controller's running Nodename
    node,err := GetNodeName()
    if err != nil {
       log.WithError(err).Error("Error getting Nodename")
       return
    }
    //Start controller loop
    for {
        err = Loop(p, node)
	if err != nil {
            log.WithError(err).Error("Error running controller loop")
	    return
	}
	//Sleep for 10 minutues and run controller loop again
	timer, err := time.ParseDuration("10m")
	if err != nil {
            log.WithError(err).Error("Error setting timer")
	    return
	}
	time.Sleep(timer)
    }
}
