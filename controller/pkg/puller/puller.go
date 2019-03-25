// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

type SyncFailFunction func()

type Puller interface {
	// Run activates the feed and returns a channel that sends snapshots of the
	// IPs that are considered suspicious.
	Run(context.Context, statser.Statser) SyncFailFunction
	SetFeed(*v3.GlobalThreatFeed)
	Close()
}
