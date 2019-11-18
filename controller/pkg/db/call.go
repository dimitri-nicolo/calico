// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
)

type Call struct {
	Method      string
	GNS         *v3.GlobalNetworkSet
	Name        string
	Value       interface{}
	Version     *int64
	SeqNo       *int64
	PrimaryTerm *int64
}
