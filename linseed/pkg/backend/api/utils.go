// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package api

import "github.com/sirupsen/logrus"

// ContextLogger returns a suitable context logger for use in a request to the backend.
func ContextLogger(i ClusterInfo) *logrus.Entry {
	f := logrus.Fields{
		"cluster": i.Cluster,
	}
	if i.Tenant != "" {
		f["tenant"] = i.Tenant
	}
	return logrus.WithFields(f)
}
