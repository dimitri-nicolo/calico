// Copyright 2019 Tigera Inc. All rights reserved.

package util

import (
	"time"

	v33 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v32 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FeedsNamespace = "calico-monitoring"
)

func NewGlobalThreatFeedFromName(name string) *v32.GlobalThreatFeed {
	return &v32.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: FeedsNamespace,
		},
		Spec: v33.GlobalThreatFeedSpec{},
	}
}

func FeedNeedsRestart(a, b *v32.GlobalThreatFeed) bool {
	if a.Spec.Pull == nil && b.Spec.Pull == nil {
		return false
	}
	if a.Spec.Pull == nil || b.Spec.Pull == nil {
		return true
	}

	aPeriod := ParseFeedDuration(a)
	bPeriod := ParseFeedDuration(b)
	return aPeriod != bPeriod
}

func ParseFeedDuration(f *v32.GlobalThreatFeed) time.Duration {
	period := v33.DefaultPullPeriod
	if f.Spec.Pull.Period != "" {
		var err error
		period, err = time.ParseDuration(f.Spec.Pull.Period)
		if err == nil {
			return period
		}
	}

	return v33.DefaultPullPeriod
}
