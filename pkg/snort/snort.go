// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package snort

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	hp "github.com/tigera/honeypod-controller/pkg/processor"
	api "github.com/tigera/lma/pkg/api"
)

type dateSrcDst string

// Alina: Does it need to be exported ?
type Alert struct {
	SigName    string
	Category   string
	DateSrcDst dateSrcDst
	Flags      string
	Other      string
}

func parse(input string) ([]Alert, error) {
	//We will do generic splits of the Alert result for now
	alerts := strings.Split(input, "\n\n")
	var result []Alert
	for _, lines := range alerts {
		item := strings.Split(lines, "\n")
		if len(item) >= 5 {
			var tmp Alert
			tmp.SigName = item[0]
			tmp.Category = item[1]
			tmp.DateSrcDst = dateSrcDst(item[2])
			tmp.Flags = item[3]
			tmp.Other = item[4]
			result = append(result, tmp)
		}
	}
	return result, nil
}

func parseTime(timeStr dateSrcDst) (time.Time, error) {
	//Convert Alert's timestamp to time.Time
	res := strings.Split(string(timeStr), " ")
	layout := "01/02/06-15:04:05.000000"
	newTime, err := time.Parse(layout, res[0])
	if err != nil {
		log.WithError(err).Error("Failed to parse time:", res[0])
		return newTime, err
	}
	return newTime, nil
}

func RunScanSnort(a *api.Alert, pcap string, outPath string) error {
	//We setup the directory for the snort alert result to be stored in
	output := fmt.Sprintf("%s/%s", outPath, *a.Record.DestNameAggr)
	err := os.MkdirAll(output, 0755)
	if err != nil {
		log.WithError(err).Error("Failed to create snort folder")
		return err
	}
	//Exec snort with pre-set flags, and redirect Stdout to a buffer
	// -q                        : quiet
	// -k none                   :
	// -y                        : show year in timestamp
	// -c /etc/snort/snort.conf  : configuration file
	// -r $pcap                   : pcap input file/directory
	// -l $output
	cmd := exec.Command("snort", "-q", "-k", "none", "-y", "-c", "/etc/snort/snort.conf", "-r", pcap, "-l", output)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.WithError(err).Error("Error running Alert on pcap: ", output)
		return err
	}
	log.Info("Signature Triggered")
	return nil
}

func ProcessSnort(a *api.Alert, p *hp.HoneyPodLogProcessor, outPath string, store *Store) error {
	//We look at the directory in which the snort alerts is stored
	snortAlertDirs, err := filepath.Glob(fmt.Sprintf("%s/*", outPath))
	if err != nil {
		log.WithError(err).Error("Error matching snort directory")
		return err
	}

	log.Info("Alert Directory found: ", snortAlertDirs)
	for _, match := range snortAlertDirs {
		//If found, we iterate each entry, parse it and filter the ones that we already sent to elasticsearch
		path := fmt.Sprintf("%s/alert", match)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.WithError(err).Error("Error file missing: ", path)
			continue
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.WithError(err).Error("Error reading file: ", path)
			continue
		}
		alerts, err := parse(string(content))
		if err != nil {
			log.WithError(err).Error("Error parsing Alert alert: ", path)
			continue
		}
		newest := store.Apply(alerts, Uniques, Newest)
		store.Update(newest)
		err = SendEvents(newest, p, a)
		if err != nil {
			log.WithError(err).Error("Error sending Alert alert")
			continue
		}
	}
	return nil
}
func SendEvents(snortEvents []Alert, p *hp.HoneyPodLogProcessor, a *api.Alert) error {

	//Iterate list of Alert alerts and send them to Elasticsearch
	for _, alert := range snortEvents {
		description := fmt.Sprintf("[Alert] Signature Triggered on %s/%s", *a.Record.DestNamespace, *a.Record.DestNameAggr)
		body := map[string]interface{}{
			"severity":    100,
			"description": description,
			"alert":       "honeypod-controller.snort",
			"type":        "alert",
			"record": map[string]interface{}{
				"snort": map[string]interface{}{
					"Description": alert.SigName,
					"Category":    alert.Category,
					"Occurence":   alert.DateSrcDst,
					"Flags":       alert.Flags,
					"Other":       alert.Other,
				},
			},
			"time": time.Now(),
		}
		res, err := p.Client.Backend().Index().Index(hp.Index).Id("").BodyJson(body).Do(p.Ctx)

		//If theres any issue sending the snort alert to ES, we log and exit
		if err != nil {
			log.WithError(err).Error("Error sending Alert alert")
			return err
		}
		log.Infof("Alert alert sent: %v", res)
	}
	return nil
}
