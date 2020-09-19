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
	//"html"

	//log "github.com/sirupsen/logrus"
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
    PacketCapture  = "capture-honey"
)

func GetNodeName() string {
    config, err := rest.InClusterConfig()
    if err != nil {
	fmt.Println("bad rest")
        return ""
    }
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
	fmt.Println("bad config")
	return ""
    }
    hostname := os.Getenv("HOSTNAME")
    fmt.Println("Hostname: ", hostname)
    pod, err := clientset.CoreV1().Pods("tigera-intrusion-detection").Get(hostname, metav1.GetOptions{})
    if err != nil {
	fmt.Println("bad clientset",err)
        return ""
    }
    return pod.Spec.NodeName
}

func SendEvents(SnortList []snort.Snort, p hp.HoneypodLogProcessor, e api.AlertResult) error {

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
        fmt.Println(json_res)
        res, err := p.Client.Backend().Index().Index(Index).Id("").BodyJson(json_res).Do(p.Ctx)
        if err != nil {
            fmt.Println("Send Failed", err)
        }
        fmt.Println(res)
    }
    return nil
}
func loop(p hp.HoneypodLogProcessor, node string) error {
    //We only look at the past 10min of alerts
    endTime := time.Now()
    startTime := endTime.Add(-10 * time.Minute)
    fmt.Println("Querying Elasticsearch for new Alerts.")

    for e := range p.LogHandler.SearchAlertLogs(p.Ctx, nil, &startTime, &endTime) {
        if e.Err != nil {
	    fmt.Println("Search failed")
	    return e.Err
	}
	//fmt.Println(e.SourceNamespace)
	//fmt.Println(e.DestNamespace)

	//fmt.Println("Type: ", e.Type)
	//fmt.Println("Description: ", e.Description)
	//fmt.Println("Alert: ", e.Alert.Alert)
	//fmt.Println("Record SourceNameAggr: ", *e.Record.SourceNameAggr)
	//fmt.Println("Record SourceNamespace: ", *e.Record.SourceNamespace)
	//fmt.Println("Record DestNamespace:", *e.Record.DestNamespace)
	//fmt.Println("Record DestNameAggr: ", *e.Record.DestNameAggr)
	//fmt.Println("Record HostKeyword: ", *e.Record.HostKeyword)

	//Skip alerts thats not Honeypod related and not on our node
	if strings.Contains(e.Alert.Alert, "honeypod.") == false || *e.Record.HostKeyword != node {
            continue
	}

        fmt.Println("Valid alert Found: ", e.Alert)

	s := fmt.Sprintf("/pcap/%s/%s/%s", *e.Record.DestNamespace, PacketCapture, *e.Record.DestNameAggr)
	//fmt.Println(s)

	//Check if packet capture directory is missing and look for pcaps that matches Alert's destination pod
	if _, err := os.Stat("/pcap/"); os.IsNotExist(err) {
	    fmt.Println("/pcap directory missing.")
	    return err
	}
        matches, err := filepath.Glob(s)
	if err != nil {
	    fmt.Println("/pcap file.")
	    return err
	}

	fmt.Println("Scanning: ", matches)
	for _ , match := range matches {
	    output := fmt.Sprintf("/snort/%s", *e.Record.DestNameAggr)
            if _, err := os.Stat(output); os.IsNotExist(err) {
                err = os.Mkdir(output, 755)
	        if err != nil {
	           fmt.Println("can't create snort folder")
                }
	    }
	    cmd := exec.Command("snort", "-q", "-k", "none", "-c", "/etc/snort/snort.conf", "-r", match, "-l", output)
	    //fmt.Println("Exec: ", cmd.String())
	    var out bytes.Buffer
	    cmd.Stdout = &out
	    err := cmd.Run()
	    if err != nil {
	        fmt.Println("exec failed")
	    }
	    fmt.Println(out.String())
	}
        matches, err = filepath.Glob("/snort/*")
	if err != nil {
	    fmt.Println("/snort file.")
	}
	fmt.Println(matches)
	for _, match := range matches {
	    path := fmt.Sprintf("%s/alert", match)
	    if _, err := os.Stat(path); os.IsNotExist(err) {
	        fmt.Println(path, " missing.")
	    }
	    reader, err := ioutil.ReadFile(path)
	    if err != nil {
	        fmt.Println("Read Error")
	    }
	    //fmt.Println(string(reader))
	    reader_str := string(reader)
	    SnortList, err := snort.ParseSnort(reader_str)
	    if err != nil {
                fmt.Println("Parse Error")
	    }
	    FilterList, err := snort.FilterSnort(SnortList)
	    if err != nil {
                fmt.Println("Filter Error")
	    }
	    w := *e
	    err = SendEvents(FilterList, p, w)
	    if err != nil {
	        fmt.Println("SendEvent Failed.")
	    }
	}

    }
    fmt.Println("Loop Completed.")

    return nil
}

func main() {

    //Get Default Elastic client config, then modify URL
    fmt.Println("Retrieving Elastic Client.")
    cfg := elastic.MustLoadConfig()
    cfg.ElasticURI = "https://tigera-secure-es-http.tigera-elasticsearch.svc:9200"
    cfg.ParsedElasticURL, _ = url.Parse(cfg.ElasticURI)

    //Try to connect to Elasticsearch
    c, err := elastic.NewFromConfig(cfg)
    if err != nil {
        fmt.Println("Failed to initiate ES client.")
	return
    }
    //Set up context
    ctx := context.Background()

    //Check if required index exists
    //index := "tigera_secure_ee_events.cluster"
    exists, err := c.Backend().IndexExists(Index).Do(context.Background())
    if err != nil {
        fmt.Println("Index can't be accessed")
	return
    }
    if exists != true {
        fmt.Println("Index does not exist")
	return
    }

    p, err := hp.NewHoneypodLogProcessor(c, ctx)
    if err != nil {
       fmt.Println("Unable to create HoneypodLog Processor")
       return
    }
    node := GetNodeName()
    if node == "" {
       fmt.Println("Didn't get node name.")
    }
    fmt.Println(node)

    for {
        err = loop(p, node)
	if err != nil {
            fmt.Println("ded loop")
	    break
	}
	timer,err := time.ParseDuration("10m")
	if err != nil {
	    fmt.Println("bad time")
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
