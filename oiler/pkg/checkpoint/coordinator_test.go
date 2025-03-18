// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package checkpoint_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/oiler/pkg/checkpoint"
	"github.com/projectcalico/calico/oiler/pkg/checkpoint/fake"
	"github.com/projectcalico/calico/oiler/pkg/config"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

var ctx context.Context

func setupAndTeardown(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		logCancel()
		cancel()
	}
}

func TestCoordinator_Run(t *testing.T) {

	t.Run("Store only new values", func(t *testing.T) {
		defer setupAndTeardown(t)()

		checkpointsChan := make(chan operator.TimeInterval)
		defer close(checkpointsChan)
		storage := fake.NewStorage()
		coordinator := checkpoint.NewCoordinator(checkpointsChan, storage)

		go func() {
			coordinator.Run(ctx)
		}()
		checkpointsChan <- operator.TimeInterval{Start: ptrTime(time.Unix(1, 0))}
		checkpointsChan <- operator.TimeInterval{Start: ptrTime(time.Unix(1, 0))}

		ctx.Done()
		require.Equal(t, 1, storage.GetNumberOfWrites())
	})

	t.Run("Store only new value even we encounter an error", func(t *testing.T) {
		defer setupAndTeardown(t)()

		checkpointsChan := make(chan operator.TimeInterval)
		defer close(checkpointsChan)
		storage := fake.NewStorage()
		coordinator := checkpoint.NewCoordinator(checkpointsChan, storage)

		go func() {
			coordinator.Run(ctx)
		}()
		checkpointsChan <- operator.TimeInterval{Start: ptrTime(time.Unix(1, 0))}
		checkpointsChan <- operator.TimeInterval{Start: ptrTime(fake.ErrorCheckPoint)}
		checkpointsChan <- operator.TimeInterval{Start: ptrTime(time.Unix(1, 0))}

		ctx.Done()
		require.Equal(t, 1, storage.GetNumberOfWrites())
	})
}
