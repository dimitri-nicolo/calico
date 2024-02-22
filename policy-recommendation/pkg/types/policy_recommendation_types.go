// Copyright (c) 2023 Tigera Inc. All rights reserved.

package types

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
)

const (
	PolicyRecommendationScopeName = "default"
	PolicyRecommendationTierName  = "namespace-isolation"
)

type FlowLogData struct {
	Action    v3.Action
	Domains   []string
	Global    bool
	Name      string
	Namespace string
	Protocol  numorstring.Protocol
	Ports     []numorstring.Port
	Timestamp string
}
