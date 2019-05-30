// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package targets

import "net/url"

type Targets struct {
	targets map[string]*url.URL
}

func New(targets map[string]*url.URL) *Targets {
	return &Targets{targets: targets}
}

func (targets *Targets) Add(target string, destination *url.URL) {
	targets.targets[target] = destination
}

func (targets *Targets) List() map[string]*url.URL {
	return targets.targets
}

