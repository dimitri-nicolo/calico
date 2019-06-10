// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package targets

import (
	"fmt"
	"net/url"
)

type Targets struct {
	targets map[string]*url.URL
}

func New(targets map[string]*url.URL) *Targets {
	return &Targets{targets: targets}
}

func NewEmpty() *Targets {
	return &Targets{
		targets: make(map[string]*url.URL),
	}
}

func (t *Targets) Add(target string, dest string) error {
	url, err := url.Parse(dest)
	if err != nil {
		return err
	}
	t.targets[target] = url

	return nil
}

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
