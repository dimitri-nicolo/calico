// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

var (
	wafKeys = map[string]Validator{
		"timestamp":      DateValidator,
		"unique_id":      NullValidator,
		"uri":            NullValidator,
		"owasp_host":     IPValidator,
		"owasp_file":     NullValidator,
		"owasp_line":     PositiveIntValidator,
		"owasp_id":       PositiveIntValidator,
		"owasp_severity": PositiveIntValidator,
		"node":           NullValidator,
	}
)

func IsValidWAFAtom(a *Atom) error {
	if validator, ok := wafKeys[a.Key]; ok {
		return validator(a)
	}

	return nil
}
