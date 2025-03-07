// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package fake

import (
	"context"
	"fmt"
	"time"

	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

var (
	ErrorCheckPoint time.Time = time.Unix(0, 0)
)

type Storage struct {
	data           time.Time
	numberOfWrites int
}

func NewStorage() *Storage {
	return &Storage{}
}

func (s *Storage) Read(ctx context.Context) (operator.TimeInterval, error) {
	return operator.TimeInterval{Start: &s.data}, nil
}

func (s *Storage) Write(ctx context.Context, checkpoint time.Time) error {
	if checkpoint.Equal(ErrorCheckPoint) {
		return fmt.Errorf("ErrorCheckPoint")
	}
	s.data = checkpoint
	s.numberOfWrites++
	return nil
}

func (s *Storage) GetNumberOfWrites() int {
	return s.numberOfWrites
}
