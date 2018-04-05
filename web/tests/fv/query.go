// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package fv

import (
	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
)

type testQueryData struct {
	description string
	resources   []resourcemgr.ResourceObject
	query       interface{}
	response    interface{}
}

type errorResponse struct {
	text string
	code int
}
