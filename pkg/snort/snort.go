// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package snort

import (
	"os"
        "os/exec"
	"strings"
	"fmt"
	"time"
        "bytes"
	"path/filepath"
	"io/ioutil"

        api "github.com/tigera/lma/pkg/api"
	log "github.com/sirupsen/logrus"

	hp "github.com/tigera/honeypod-controller/pkg/processor"
)

var GlobalSnort []Snort

type Snort struct {
    SigName string
    Category string
    Date_Src_Dst string
    Flags string
    Other string
}

func ParseSnort(reader_str string) ([]Snort, error) {
    //We will do generic splits of the Snort result for now
    alerts := strings.Split(reader_str,"\n\n")
    var result []Snort
    for _, lines := range alerts {
	item := strings.Split(lines,"\n")
	if len(item) >= 5 {
	    var tmp Snort
	    tmp.SigName = item[0]
	    tmp.Category = item[1]
	    tmp.Date_Src_Dst = item[2]
	    tmp.Flags = item[3]
	    tmp.Other = item[4]
	    result = append(result, tmp)
	}
    }
    return result, nil
}

func FilterSnort(SnortList []Snort) ([]Snort, error) {
     //We use a global Snortlist to keep track of Snort entries that we already send to elastic.
     var tmpList []Snort
     if len(GlobalSnort) == 0 {
         GlobalSnort = append(GlobalSnort, SnortList...)
         return SnortList, nil
     }
     for _, items := range SnortList {
	 found := 0
         for _, items2 := range GlobalSnort {
             //We are matching the Timestamp, Source and Destinaton of the Snort entry for uniqueness.
	     if items.Date_Src_Dst == items2.Date_Src_Dst {
	         found = 1
	     }
	 }
	 if found == 0 {
             tmpList = append(tmpList, items)
         }
     }
     GlobalSnort = append(GlobalSnort, tmpList...)
     return tmpList, nil
}

func ScanSnort(e *api.AlertResult, pcap string, outpath string) error {
    //We setup the directory for the snort alert result to be stored in
    output := fmt.Sprintf("%s/%s", outpath, *e.Record.DestNameAggr)
    if _, err := os.Stat(output); os.IsNotExist(err) {
        err = os.Mkdir(output, 0755)
        if err != nil {
           log.WithError(err).Error("can't create snort folder")
	   return err
        }
    }
    //Exec snort with pre-set flags, and redirect Stdout to a buffer
    cmd := exec.Command("snort", "-q", "-k", "none", "-c", "/etc/snort/snort.conf", "-r", pcap, "-l", output)
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        log.WithError(err).Error("Error running Snort on pcap: ", output)
	return err
    }
    log.Info("Signature Triggered")
    return nil
}

func ProcessSnort(e *api.AlertResult, p hp.HoneypodLogProcessor, outpath string) error {
    //We look at the directory in which the snort alerts is stored
    path := fmt.Sprintf("%s/*", outpath)
    snortArray, err := filepath.Glob(path)
    if err != nil {
        log.WithError(err).Error("Error matching snort directory")
    }
    log.Info("Snort Directory found: ", snortArray)
    for _, match := range snortArray {
        //If found, we iterate each entry, parse it and filter the ones that we already sent to elasticsearch
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
        SnortList, err := ParseSnort(reader_str)
        if err != nil {
            log.WithError(err).Error("Error parsing Snort alert: ", path)
	    continue
        }
        FilterList, err := FilterSnort(SnortList)
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
    return nil
}
func SendEvents(SnortList []Snort, p hp.HoneypodLogProcessor, e api.AlertResult) error {

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
        res, err := p.Client.Backend().Index().Index(hp.Index).Id("").BodyJson(json_res).Do(p.Ctx)

	//If theres any issue sending the snort alert to ES, we log and exit
        if err != nil {
            log.WithError(err).Error("Error sending Snort alert")
	    return err
        }
	log.Info("Snort alert sent: ", res)
    }
    return nil
}
