// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package compliance

import (
	"context"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

type SnapshotsOperator struct {
	backend     bapi.SnapshotsBackend
	clusterInfo bapi.ClusterInfo
}

func (f SnapshotsOperator) Read(ctx context.Context, current operator.TimeInterval, pageSize int) (*v1.List[v1.Snapshot], *operator.TimeInterval, error) {
	logParams := v1.SnapshotParams{
		QueryParams: operator.QueryParams(pageSize, current),
		Sort:        operator.SortParameters(),
	}

	list, err := f.backend.List(ctx, f.clusterInfo, &logParams)
	if err != nil {
		return nil, nil, err
	}

	var lastGeneratedTime *time.Time
	items := list.Items
	if len(items) != 0 {
		lastGeneratedTime = items[len(items)-1].ResourceList.GeneratedTime
	} else {
		lastGeneratedTimeFromCursor := current.LastGeneratedTime()
		lastGeneratedTime = &lastGeneratedTimeFromCursor
	}

	return list, operator.Next(list.GetAfterKey(), lastGeneratedTime, current.Start), nil

}

func (f SnapshotsOperator) Write(ctx context.Context, items []v1.Snapshot) (*v1.BulkResponse, error) {
	return f.backend.Create(ctx, f.clusterInfo, items)
}

func NewSnapshotsOperator(backend bapi.SnapshotsBackend, clusterInfo bapi.ClusterInfo) SnapshotsOperator {
	return SnapshotsOperator{
		backend:     backend,
		clusterInfo: clusterInfo,
	}
}
