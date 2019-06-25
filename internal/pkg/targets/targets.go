// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package targets

import (
	"fmt"
	"net/url"
)

// Targets maps a proxy pattern to a destination
type Targets struct {
	targets map[string]*url.URL
}

// New returns new Targets creates from a prepopulated map
func New(targets map[string]*url.URL) *Targets {
	return &Targets{targets: targets}
}

// NewEmpty returns new empty Targets
func NewEmpty() *Targets {
	return &Targets{
		targets: make(map[string]*url.URL),
	}
}

// Add will add a new key-value pair composed of proxy target and destination
// Add will return an error in case the destination is not a valid URL
func (t *Targets) Add(target string, dest string) error {
	url, err := url.Parse(dest)
	if err != nil {
		return err
	}
	t.targets[target] = url

	return nil
}

// List will list the key-value pairs composed of proxy targets and destinations
func (t *Targets) List() map[string]*url.URL {
	return t.targets
}

func (t *Targets) String() string {
	str := "["

	for k, v := range t.targets {
		str = fmt.Sprintf("%s %s-> %s", str, k, v)
	}

	str = str + " ]"

	return str
}
