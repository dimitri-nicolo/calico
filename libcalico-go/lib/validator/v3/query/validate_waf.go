// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

var (
	wafKeys = map[string]Validator{
		"timestamp":     DateValidator,
		"owaspFile":     NullValidator,
		"owaspHost":     NullValidator,
		"owaspId":       PositiveIntValidator,
		"owaspLine":     PositiveIntValidator,
		"owaspSeverity": NullValidator,
		"uniqueId":      NullValidator,
		"uri":           NullValidator,
		"envoyHost":     NullValidator,
	}
)

func IsValidWAFAtom(a *Atom) error {
	if validator, ok := wafKeys[a.Key]; ok {
		return validator(a)
	}

	return nil
}
