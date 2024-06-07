// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package query

import (
	"fmt"
)

const WafEventType = "waf"

var (
	EventsKeys = map[string]Validator{
		"attack_vector":    NullValidator,
		"description":      NullValidator,
		"dest_ip":          IPValidator,
		"dest_name":        DomainValidator,
		"dest_name_aggr":   DomainValidator,
		"dest_namespace":   DomainValidator,
		"dest_port":        IntRangeValidator(0, MaxTCPUDPPortNum),
		"dismissed":        SetValidator("true", "false"),
		"host":             NullValidator,
		"mitre_tactic":     NullValidator,
		"name":             NullValidator,
		"origin":           NullValidator,
		"severity":         IntRangeValidator(0, 100),
		"source_ip":        IPValidator,
		"source_name":      DomainValidator,
		"source_name_aggr": DomainValidator,
		"source_namespace": DomainValidator,
		"source_port":      IntRangeValidator(0, MaxTCPUDPPortNum),
		"time":             PositiveIntValidator,
		// sync with manager ApiSecurityEventType if anything changes.
		"type": SetValidator(
			"alert",
			"anomaly_detection_job",
			"deep_packet_inspection",
			"global_alert",
			"gtf_suspicious_dns_query",
			"gtf_suspicious_flow",
			"runtime_security",
			"suspicious_dns_query",
			"suspicious_flow",
			"waf",
		),
	}
)

func IsValidEventsKeysAtom(a *Atom) error {
	if validator, ok := EventsKeys[a.Key]; ok {
		return validator(a)
	}

	return fmt.Errorf("invalid key: %s", a.Key)
}
