// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

const (
	actual = iota
	typical
)

type Job struct {
	DatafeedID string
	Detectors  map[int]*template.Template
}

var templateFuncs = template.FuncMap{
	"actual":     func(r elastic.RecordSpec) string { return getActualOrTypical(r, actual) },
	"typical":    func(r elastic.RecordSpec) string { return getActualOrTypical(r, typical) },
	"influencer": getInfluencer,
}

func getActualOrTypical(r elastic.RecordSpec, keyId int) string {
	var res []string

	var key string
	switch keyId {
	case actual:
		key = "actual"
		for _, v := range r.Actual {
			res = append(res, fmt.Sprintf("%v", v))
		}
	case typical:
		key = "typical"
		for _, v := range r.Typical {
			res = append(res, fmt.Sprintf("%v", v))
		}
	default:
		panic(fmt.Sprintf("Unknown keyID: %d", keyId))
	}

	for _, i := range r.Causes {
		cause, ok := i.(map[string]interface{})
		if !ok {
			return "ERROR"
		}

		i2, ok := cause[key]
		if !ok {
			continue
		}

		actual, ok := i2.([]interface{})
		if !ok {
			return "ERROR"
		}

		for _, v := range actual {
			res = append(res, fmt.Sprintf("%v", v))
		}
	}

	switch len(res) {
	case 0:
		return "-"
	case 1:
		return res[0]
	default:
		return "[" + strings.Join(res, ", ") + "]"
	}
}

func getInfluencer(r elastic.RecordSpec, key string) string {
	var res []string
	for _, i := range r.Influencers {
		if i.FieldName == key {
			for _, v := range i.FieldValues {
				res = append(res, fmt.Sprintf("%v", v))
			}
		}
	}

	switch len(res) {
	case 0:
		return "nil"
	case 1:
		return res[0]
	default:
		return "[" + strings.Join(res, ", ") + "]"
	}
}

// This needs to be kept in sync with the jobs in /install/data
var Jobs = map[string]Job{
	"inbound_connection_spike": {
		"datafeed-inbound_connection_spike",
		map[int]*template.Template{
			0: template.Must(
				template.New("inbound_connection_spike[0]").
					Funcs(templateFuncs).
					Parse("Inbound connection spike for pod {{.OverFieldValue}} within replica set" +
						" {{.PartitionFieldValue}}: {{actual .}} >> {{typical .}}")),
		},
	},
	"ip_sweep_pods": {
		"datafeed-ip_sweep_pods",
		map[int]*template.Template{
			0: template.Must(
				template.New("ip_sweep_pods[0]").
					Funcs(templateFuncs).
					Parse(`Possible IP sweep by pod {{influencer . "source_namespace"}}/{{.OverFieldValue}}:` +
						" {{actual .}} >> {{typical .}} unique destination IPs as compared to all pods")),
		},
	},
	"port_scan_pods": {
		"datafeed-port_scan_pods",
		map[int]*template.Template{
			0: template.Must(
				template.New("port_scan_pods[0]").
					Funcs(templateFuncs).
					Parse(`Possible port scan by pod {{influencer . "source_namespace"}}/{{.OverFieldValue}}:` +
						" {{actual .}} >> {{typical .}} unique destination ports as compared to all pods")),
		},
	},
	"inbound_service_bytes_anomaly": {
		"datafeed-inbound_service_bytes_anomaly",
		map[int]*template.Template{
			0: template.Must(
				template.New("inbound_service_bytes_anomaly[0]").
					Funcs(templateFuncs).
					Parse("Input bytes spike for pod {{.OverFieldValue}} within replica set {{.PartitionFieldValue}}:" +
						"{{actual .}} >> {{typical .}}")),
		},
	},
	"outbound_service_bytes_anomaly": {
		"datafeed-outbound_service_bytes_anomaly",
		map[int]*template.Template{
			0: template.Must(
				template.New("outbound_service_bytes_anomaly[0]").
					Funcs(templateFuncs).
					Parse("Output bytes spike for pod {{.OverFieldValue}} within replica set {{.PartitionFieldValue}}:" +
						"{{actual .}} >> {{typical .}}")),
		},
	},
}
