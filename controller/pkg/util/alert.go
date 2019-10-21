// Copyright (c) 2019 Tigera Inc. All rights reserved.

package util

import (
	"fmt"

	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AlertsNamespace = "calico-monitoring"
)

// NewGlobalAlert generates an alert that you might want to use for testing purposes
func NewGlobalAlert(name string) *v3.GlobalAlert {
	return &v3.GlobalAlert{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: AlertsNamespace,
		},
		Spec: calicov3.GlobalAlertSpec{
			Description: fmt.Sprintf("test alert: %s", name),
			Severity:    100,
			DataSet:     "flows",
		},
	}
}
