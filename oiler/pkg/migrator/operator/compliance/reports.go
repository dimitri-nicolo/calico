// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package compliance

import (
	"context"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

type ReportsOperator struct {
	backend     bapi.ReportsBackend
	clusterInfo bapi.ClusterInfo
}

func (f ReportsOperator) Read(ctx context.Context, current operator.TimeInterval, pageSize int) (*v1.List[v1.ReportData], *operator.TimeInterval, error) {
	logParams := v1.ReportDataParams{
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

func (f ReportsOperator) Write(ctx context.Context, items []v1.ReportData) (*v1.BulkResponse, error) {
	return f.backend.Create(ctx, f.clusterInfo, items)
}

func NewReportsOperator(backend bapi.ReportsBackend, clusterInfo bapi.ClusterInfo) ReportsOperator {
	return ReportsOperator{
		backend:     backend,
		clusterInfo: clusterInfo,
	}
}
