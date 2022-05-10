// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package snort

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/honeypod-controller/pkg/events"
	hp "github.com/projectcalico/calico/honeypod-controller/pkg/processor"

	api "github.com/projectcalico/calico/lma/pkg/api"
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

func RunScanSnort(e *api.EventsData, pcap string, outPath string) error {
	//We setup the directory for the snort alert result to be stored in
	output := fmt.Sprintf("%s/%s", outPath, e.DestNameAggr)

	log.Info("Running Snort Scan on: ", pcap)
	err := os.MkdirAll(output, 0755)
	if err != nil {
		log.WithError(err).Error("Failed to create Snort folder")
		return err
	}
	//Exec snort with pre-set flags, and redirect Stdout to a buffer
	// -q                        : quiet
	// -k none                   : checksum level
	// -y                        : show year in timestamp
	// -c /etc/snort/snort.conf  : configuration file
	// -r $pcap                  : pcap input file/directory
	// -l $output                : alert output directory
	cmd := exec.Command("snort", "-q", "-k", "none", "-y", "-c", "/etc/snort/snort.conf", "-r", pcap, "-l", output)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		log.WithError(err).Errorf("failed to run alert on pcap: %s, error: %s", pcap, stderr.String())
		return err
	}
	return nil
}

func ProcessSnort(e *api.EventsData, p *hp.HoneyPodLogProcessor, outPath string, store *Store) error {
	//We look at the directory in which the snort alerts is stored
	snortAlertDirs, err := filepath.Glob(fmt.Sprintf("%s/*", outPath))
	if err != nil {
		log.WithError(err).Error("Error matching Snort directory")
		return err
	}

	log.Info("Parsing Snort result: ", snortAlertDirs)
	for _, match := range snortAlertDirs {
		//If found, we iterate each entry, parse it and filter the ones that we already sent to elasticsearch
		path := fmt.Sprintf("%s/alert", match)
		if _, err := os.Stat(path); os.IsNotExist(err) {
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
		err = SendEvents(newest, p, e)
		if err != nil {
			log.WithError(err).Error("Error sending Alert alert")
			continue
		}
	}
	return nil
}
func SendEvents(snortEvents []Alert, p *hp.HoneyPodLogProcessor, e *api.EventsData) error {
	b, err := json.Marshal(e.Record)
	if err != nil {
		return fmt.Errorf("failed to marshal event.record")
	}
	record := &events.HoneypodAlertRecord{}
	err = json.Unmarshal(b, record)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event.record to honeypod alert record")
	}

	host := ""
	if record.HostKeyword != nil {
		host = *record.HostKeyword
	}

	// Iterate list of snort alerts and send them to Elasticsearch
	for _, event := range snortEvents {
		description := fmt.Sprintf("[Alert] Signature Triggered on %s/%s", e.DestNamespace, e.DestNameAggr)
		he := events.HoneypodEvent{
			EventsData: api.EventsData{
				Time:            time.Now().Unix(),
				Type:            "honeypod",
				Description:     description,
				Host:            host,
				Severity:        100,
				Origin:          "honeypod-controller.snort",
				DestNameAggr:    e.DestNameAggr,
				DestNamespace:   e.DestNamespace,
				SourceNameAggr:  e.SourceNameAggr,
				SourceNamespace: e.SourceNamespace,
				Record: events.HoneypodSnortEventRecord{
					Snort: &events.Snort{
						Description: event.SigName,
						Category:    event.Category,
						Occurrence:  string(event.DateSrcDst),
						Flags:       event.Flags,
						Other:       event.Other,
					},
				},
			},
		}

		if _, err := p.Client.PutSecurityEvent(p.Ctx, he.EventData()); err != nil {
			// If theres any issue sending the snort alert to ES, we log and return error.
			log.WithError(err).Errorf("Failed to put snort security event: %s", description)
			return err
		}

		log.Infof("Alert: %v Snort signature triggered: %v", e.Origin, event.SigName)
	}
	log.Infof("%v Snort events created and sent to Elastic", len(snortEvents))
	return nil
}
