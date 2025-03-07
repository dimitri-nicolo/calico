// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package checkpoint

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

// Coordinator will coordinate migrator checkpoints with
// its Storage component
type Coordinator struct {
	checkpoints <-chan operator.TimeInterval
	storage     Storage
	lastWritten time.Time
}

func NewCoordinator(checkpoints <-chan operator.TimeInterval, storage Storage) *Coordinator {
	return &Coordinator{checkpoints: checkpoints, storage: storage}
}

func (w *Coordinator) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("Stopping coordinator")
			return
		case checkpoint, ok := <-w.checkpoints:
			if ok {
				last := checkpoint.LastGeneratedTime()
				if !last.Equal(w.lastWritten) {
					logrus.Infof("Storing checkpoint for %s", last.String())
					err := w.storage.Write(ctx, last)
					if err != nil {
						logrus.WithError(err).Errorf("Error storing checkpoint for %s", last.String())
						continue
					}
					w.lastWritten = last
				}
			}
		}
	}
}
