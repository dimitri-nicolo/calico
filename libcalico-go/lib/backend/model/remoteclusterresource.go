// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package model

import "regexp"

var (
	matchRemoteClusterResource = regexp.MustCompile("^/?cluster/([^/]+)(/.+)$")
)

type RemoteClusterResourceKey struct {
	ResourceKey

	// The name of the cluster that the resource is homed.
	Cluster string
}

func (key RemoteClusterResourceKey) defaultPath() (string, error) {
	return key.defaultDeletePath()
}

func (key RemoteClusterResourceKey) defaultDeletePath() (string, error) {
	p, err := key.ResourceKey.defaultPath()
	if err != nil {
		return "", err
	}
	return "/cluster/" + key.Cluster + p, nil
}
