// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package targets

import (
	"net/url"
)

func CreateStaticTargets() *Targets {
	targets := make(map[string]*url.URL)

	targets["^/api"] = parse("https://kubernetes.default")
	targets["^/tigera-elasticsearch*"] = parse("http://localhost:8002")

	return &Targets{targets: targets}
}

func CreateStaticTargetsForServer() *Targets {
	targets := make(map[string]*url.URL)

	targets["api"] = parse("http://localhost:3000")

	return &Targets{targets: targets}
}

func parse(rawUrl string) *url.URL {
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}

	return url
}
