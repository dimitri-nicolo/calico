// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

import (
	"fmt"
)

var (
	L7LogsKeys = map[string]Validator{
		"host":                   NullValidator,
		"start_time":             DateValidator,
		"end_time":               DateValidator,
		"duration_mean":          PositiveIntValidator,
		"duration_max":           PositiveIntValidator,
		"bytes_in":               PositiveIntValidator,
		"bytes_out":              PositiveIntValidator,
		"count":                  PositiveIntValidator,
		"src_name_aggr":          DomainValidator,
		"src_namespace":          DomainValidator,
		"src_type":               SetValidator("wep", "ns", "net"),
		"dest_name_aggr":         DomainValidator,
		"dest_namespace":         DomainValidator,
		"dest_service_name":      DomainValidator,
		"dest_service_namespace": DomainValidator,
		"dest_service_port":      IntRangeValidator(0, MaxTCPUDPPortNum),
		"dest_type":              SetValidator("wep", "ns", "net"),
		"method":                 NullValidator,
		"user_agent":             NullValidator,
		"url":                    URLValidator,
		"response_code":          NullValidator,
		"type":                   NullValidator,
	}
)

func IsValidL7LogsAtom(a *Atom) error {
	if validator, ok := L7LogsKeys[a.Key]; ok {
		return validator(a)
	}

	return fmt.Errorf("invalid key: %s", a.Key)
}
