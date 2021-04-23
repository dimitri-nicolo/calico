// Copyright 2019 Tigera Inc. All rights reserved.

package util

import (
	"reflect"
	"time"

	v32 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	v33 "github.com/projectcalico/libcalico-go/lib/apis/v3"
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
	if aPeriod != bPeriod {
		return true
	}

	if a.Spec.Pull.HTTP == nil && b.Spec.Pull.HTTP == nil {
		return false
	}
	if a.Spec.Pull.HTTP == nil || b.Spec.Pull.HTTP == nil {
		return true
	}

	if !reflect.DeepEqual(a.Spec.Pull.HTTP.Format, b.Spec.Pull.HTTP.Format) {
		aFmt := a.Spec.Pull.HTTP.Format
		bFmt := b.Spec.Pull.HTTP.Format

		// Handle the case where default (NewlineDelimited) is specified in one
		// but not the other
		if aFmt.NewlineDelimited != nil && bFmt.JSON == nil && bFmt.CSV == nil {
			return false
		}
		if bFmt.NewlineDelimited != nil && aFmt.JSON == nil && aFmt.CSV == nil {
			return false
		}

		return true
	}

	return !reflect.DeepEqual(a.Spec.GlobalNetworkSet, b.Spec.GlobalNetworkSet)
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

func NewGlobalNetworkSet(tfName string) *v32.GlobalNetworkSet {
	s := &v32.GlobalNetworkSet{
		ObjectMeta: v1.ObjectMeta{Name: GlobalNetworkSetNameFromThreatFeed(tfName)},
	}
	return s
}

func GlobalNetworkSetNameFromThreatFeed(tfName string) string {
	return "threatfeed." + tfName
}
