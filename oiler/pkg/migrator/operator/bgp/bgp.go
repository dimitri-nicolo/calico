// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package bgp

import (
	"context"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

type Operator struct {
	backend     bapi.BGPBackend
	clusterInfo bapi.ClusterInfo
}

func (f Operator) Read(ctx context.Context, current operator.TimeInterval, pageSize int) (*v1.List[v1.BGPLog], *operator.TimeInterval, error) {
	logParams := v1.BGPLogParams{
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
		lastGeneratedTime = items[len(items)-1].GeneratedTime
	} else {
		lastGeneratedTimeFromCursor := current.LastGeneratedTime()
		lastGeneratedTime = &lastGeneratedTimeFromCursor
	}

	return list, operator.Next(list.GetAfterKey(), lastGeneratedTime, current.Start), nil

}

func (f Operator) Write(ctx context.Context, items []v1.BGPLog) (*v1.BulkResponse, error) {
	return f.backend.Create(ctx, f.clusterInfo, items)
}

func NewOperator(backend bapi.BGPBackend, clusterInfo bapi.ClusterInfo) Operator {
	return Operator{
		backend:     backend,
		clusterInfo: clusterInfo,
	}
}
