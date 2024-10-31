// Copyright 2019 Tigera Inc. All rights reserved.

package util

import (
	"reflect"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FeedsNamespace = "calico-monitoring"
)

func NewGlobalThreatFeedFromName(name string) *v3.GlobalThreatFeed {
	return &v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: FeedsNamespace,
		},
		Spec: v3.GlobalThreatFeedSpec{},
	}
}

func FeedNeedsRestart(a, b *v3.GlobalThreatFeed) bool {
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

func ParseFeedDuration(f *v3.GlobalThreatFeed) time.Duration {
	if f.Spec.Pull.Period != "" {
		period, err := time.ParseDuration(f.Spec.Pull.Period)
		if err == nil {
			return period
		}
	}

	return v3.DefaultPullPeriod
}

func NewGlobalNetworkSet(tfName string) *v3.GlobalNetworkSet {
	s := &v3.GlobalNetworkSet{
		ObjectMeta: v1.ObjectMeta{Name: GlobalNetworkSetNameFromThreatFeed(tfName)},
	}
	return s
}

func GlobalNetworkSetNameFromThreatFeed(tfName string) string {
	return "threatfeed." + tfName
}
