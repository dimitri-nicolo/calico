// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

import (
	"fmt"
)

var (
	EventsKeys = map[string]Validator{
		"_id":   NullValidator,
		"alert": NullValidator,
		"type":  NullValidator,
	}
)

func IsValidEventsKeysAtom(a *Atom) error {
	if validator, ok := EventsKeys[a.Key]; ok {
		return validator(a)
	}

	return fmt.Errorf("invalid key: %s", a.Key)
}
