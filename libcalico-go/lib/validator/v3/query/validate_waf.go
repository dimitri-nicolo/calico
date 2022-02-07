// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

var (
	wafKeys = map[string]Validator{
		"timestamp":            DateValidator,
		"path":                 NullValidator,
		"method":               NullValidator,
		"protocol":             NullValidator,
		"source.ip":            IPValidator,
		"source.port_num":      IntRangeValidator(0, MaxTCPUDPPortNum),
		"source.hostname":      DomainValidator,
		"destination.ip":       IPValidator,
		"destination.port_num": IntRangeValidator(0, MaxTCPUDPPortNum),
		"destination.hostname": DomainValidator,
		"rule_info":            NullValidator,
		"node":                 NullValidator,
	}
)

func IsValidWAFAtom(a *Atom) error {
	if validator, ok := wafKeys[a.Key]; ok {
		return validator(a)
	}

	return nil
}
