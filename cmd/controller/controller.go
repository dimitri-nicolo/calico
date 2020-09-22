package main

import (
	"context"
	"fmt"
	"time"
	"net/url"
	"os"
	"os/exec"
	"bytes"
	"path/filepath"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"
	//rule "github.com/tigera/honeypod-recommendation/pkg/rule"
	//model "github.com/tigera/honeypod-recommendation/pkg/model"
        api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/tigera/honeypod-controller/pkg/snort"
        hp "github.com/tigera/honeypod-controller/pkg/processor"
)

const (
    Index           = "tigera_secure_ee_events.cluster"
    PacketCapture   = "capture-honey"
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

func SendEvents(SnortList []snort.Snort, p hp.HoneypodLogProcessor, e api.AlertResult) error {

    //Iterate list of Snort alerts and send them to Elasticsearch
    for _, alert := range SnortList {
        snort_description := fmt.Sprintf("[Snort] Signature Triggered on %s/%s", *e.Record.DestNamespace, *e.Record.DestNameAggr)
        json_res := map[string]interface{}{
            "severity": 100,
            "description": snort_description,
            "alert": "honeypod-controller.snort",
            "type" : "alert",
            "record": map[string]interface{}{
	        "snort": map[string]interface{}{
			"Descripton": alert.SigName,
			"Category": alert.Category,
			"Occurance": alert.Date_Src_Dst,
			"Flags": alert.Flags,
			"Other": alert.Other,
		},
            },
            "time" : time.Now(),
        }
        //fmt.Println(json_res)
        res, err := p.Client.Backend().Index().Index(Index).Id("").BodyJson(json_res).Do(p.Ctx)

	//If theres any issue sending the snort alert to ES, we log and exit
        if err != nil {
            log.WithError(err).Error("Error sending Snort alert")
	    return err
        }
	log.Info("Snort alert sent: ", res)
    }
    return nil
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
	if strings.Contains(e.Alert.Alert, "honeypod.") == false || *e.Record.HostKeyword != node {
            continue
	}
        log.Info("Valid alert Found: ", e.Alert)

	s := fmt.Sprintf("/pcap/%s/%s/%s", *e.Record.DestNamespace, PacketCapture, *e.Record.DestNameAggr)

	//Check if packet capture directory is missing and look for pcaps that matches Alert's destination pod
	if _, err := os.Stat("/pcap/"); os.IsNotExist(err) {
	    log.WithError(err).Error("/pcap directory missing")
	    return err
	}
        matches, err := filepath.Glob(s)
	if err != nil {
	    log.WithError(err).Error("Failed to match pcap files")
	    return err
	}

	log.Info("Honeypod Controller scanning: ", matches)
	for _ , match := range matches {
	    output := fmt.Sprintf("/snort/%s", *e.Record.DestNameAggr)
            if _, err := os.Stat(output); os.IsNotExist(err) {
                err = os.Mkdir(output, 755)
	        if err != nil {
	           fmt.Errorf("can't create snort folder")
                }
	    }
	    cmd := exec.Command("snort", "-q", "-k", "none", "-c", "/etc/snort/snort.conf", "-r", match, "-l", output)
	    var out bytes.Buffer
	    cmd.Stdout = &out
	    err := cmd.Run()
	    if err != nil {
                log.WithError(err).Error("Error running Snort on pcap: ", output)
	    }
	    log.Info("Signature Triggered: ", out.String())
	}
        matches, err = filepath.Glob("/snort/*")
	if err != nil {
	    log.WithError(err).Error("Error matching snort directory")
	}
	log.Info("Snort Directory found: ", matches)
	for _, match := range matches {
	    path := fmt.Sprintf("%s/alert", match)
	    if _, err := os.Stat(path); os.IsNotExist(err) {
	        log.WithError(err).Error("Error file missing: ", path)
		continue
	    }
	    reader, err := ioutil.ReadFile(path)
	    if err != nil {
	        log.WithError(err).Error("Error reading file: ", path)
		continue
	    }
	    reader_str := string(reader)
	    SnortList, err := snort.ParseSnort(reader_str)
	    if err != nil {
	        log.WithError(err).Error("Error parsing Snort alert: ", path)
		continue
	    }
	    FilterList, err := snort.FilterSnort(SnortList)
	    if err != nil {
	        log.WithError(err).Error("Error filtering Snort alert")
		continue
	    }
	    w := *e
	    err = SendEvents(FilterList, p, w)
	    if err != nil {
	        log.WithError(err).Error("Error sending Snort alert")
		continue
	    }
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
    exists, err := c.Backend().IndexExists(Index).Do(context.Background())
    if err != nil || !exists {
        log.WithError(err).Error("Error unable to access Index: ", Index)
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

    //fmt.Println("get settings")
    //settings, err := c.Backend().IndexGetSettings(index).Do(context.Background())
    //if err != nil {
    //    fmt.Println("settings bad")
    //}
    //indexSettings := settings[index].Settings["index"].(map[string]interface{})
    //for key,value := range indexSettings {
	//    fmt.Println(key)
	//   fmt.Println(value)
    //}
    //fmt.Println("done3")
}
