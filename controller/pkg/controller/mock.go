// Copyright (c) 2019 Tigera Inc. All rights reserved.

package controller

import (
	"context"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type mockSetsData struct {
	ipSet db.IPSet
}

func (d *mockSetsData) Put(ctx context.Context, name string, value interface{}) error {
	return d.ipSet.PutIPSet(ctx, name, value.(db.IPSetSpec))
}

func (d *mockSetsData) List(ctx context.Context) ([]db.Meta, error) {
	return d.ipSet.ListIPSets(ctx)
}

func (d *mockSetsData) Delete(ctx context.Context, m db.Meta) error {
	return d.ipSet.DeleteIPSet(ctx, m)
}

type mockStatser struct {
	errorConditions []v3.ErrorCondition
}

func (s *mockStatser) Run(context.Context) {
}

func (s *mockStatser) Close() {
}

func (s *mockStatser) Status() v3.GlobalThreatFeedStatus {
	return v3.GlobalThreatFeedStatus{
		ErrorConditions: append(s.errorConditions),
	}
}

func (s *mockStatser) Error(t string, err error) {
	s.errorConditions = append(s.errorConditions, v3.ErrorCondition{Type: t, Message: err.Error()})
}

func (s *mockStatser) ClearError(t string) {
	ec := []v3.ErrorCondition{}

	for _, errorCondition := range s.errorConditions {
		if errorCondition.Type != t {
			ec = append(ec, errorCondition)
		}
	}

	s.errorConditions = ec
}
