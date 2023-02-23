// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package processes

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var (
	noProcess []v1.ProcessInfo
	processes = []v1.ProcessInfo{
		{
			Name:     "/proc1",
			Endpoint: "my-endpoint",
			Count:    1,
		},
		{
			Name:     "/usr/bin/curl",
			Endpoint: "my-endpoint",
			Count:    3,
		},
	}
)
