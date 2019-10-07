// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package query

import (
	"fmt"
	"strconv"
)

const (
	MaxTCPUDPPortNum = 1<<16 - 1
)

func ProtoValidator(a *Atom) error {
	switch a.Value {
	case "icmp", "tcp", "udp", "ipip", "esp", "icmp6":
		return nil
	}

	_, err := strconv.ParseInt(a.Value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %s: %s", a.Key, a.Value, err)
	}

	return nil
}

var (
	flowsKeys = map[string]Validator{
		"start_time":               DateValidator,
		"end_time":                 DateValidator,
		"action":                   SetValidator("allow", "deny"),
		"bytes_in":                 PositiveIntValidator,
		"bytes_out":                PositiveIntValidator,
		"dest_ip":                  IPValidator,
		"dest_name":                DomainValidator,
		"dest_name_aggr":           DomainValidator,
		"dest_namespace":           DomainValidator,
		"dest_port":                IntRangeValidator(0, MaxTCPUDPPortNum),
		"dest_type":                SetValidator("wep", "hep", "ns", "net"),
		"dest_labels.labels":       RegexpValidator("^[^=]+=[^=]+$"),
		"reporter":                 SetValidator("src", "dst"),
		"num_flows":                PositiveIntValidator,
		"num_flows_completed":      PositiveIntValidator,
		"num_flows_started":        PositiveIntValidator,
		"http_requests_allowed_in": PositiveIntValidator,
		"http_requests_denied_in":  PositiveIntValidator,
		"packets_in":               PositiveIntValidator,
		"packets_out":              PositiveIntValidator,
		"proto":                    ProtoValidator,
		"policies.all_policies":    NullValidator,
		"source_ip":                IPValidator,
		"source_name":              DomainValidator,
		"source_name_aggr":         DomainValidator,
		"source_namespace":         DomainValidator,
		"source_port":              IntRangeValidator(0, MaxTCPUDPPortNum),
		"source_type":              SetValidator("wep", "hep", "ns", "net"),
		"source_labels.labels":     RegexpValidator("^[^=]+=[^=]+$"),
		"original_source_ips":      IPValidator,
		"num_original_source_ips":  PositiveIntValidator,
	}
)

func IsValidFlowsAtom(a *Atom) error {
	if validator, ok := flowsKeys[a.Key]; ok {
		return validator(a)
	}

	return fmt.Errorf("invalid key: %s", a.Key)
}
