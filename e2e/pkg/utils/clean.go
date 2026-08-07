// Copyright (c) 2025-2026 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// TestResourceLabel is a label that is applied to any Calico resource created by the e2e tests.
	// It's used to identify resources that should be cleaned up.
	TestResourceLabel = "projectcalico.org/e2e"

	// deleteRetryTimeout bounds how long DeleteResource re-tries a delete that
	// is failing for a reason that tends to pass on its own.
	deleteRetryTimeout = 2 * time.Minute
)

// DeleteResource deletes obj, tolerating the API being briefly unable to serve
// the request. A resource that is already gone counts as deleted.
//
// Deletes of projectcalico.org/v3 resources are served by the Calico aggregated
// apiserver, which runs as a single replica in test clusters, so a restart or
// rollout makes every delete return 503 until it completes. Without retries
// that window fails whichever spec happens to be cleaning up inside it, which
// reads as a fault in that spec's feature rather than in the cluster. A bounded
// retry absorbs the outage while still failing on a resource that genuinely
// will not delete - a finalizer that never clears, or RBAC that forbids the
// call.
func DeleteResource(ctx context.Context, cli ctrlclient.Client, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
	ctx, cancel := context.WithTimeout(ctx, deleteRetryTimeout)
	defer cancel()

	var lastErr error
	err := wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		err := cli.Delete(ctx, obj, opts...)
		if err == nil || apierrors.IsNotFound(err) {
			return true, nil
		}
		if !retryableAPIError(err) {
			return false, err
		}
		lastErr = err
		return false, nil
	})
	if wait.Interrupted(err) && lastErr != nil {
		// Surface what the API was actually saying, not "context deadline exceeded".
		return lastErr
	}
	return err
}

// retryableAPIError reports whether err describes the API being temporarily
// unable to serve a request, as opposed to refusing it.
func retryableAPIError(err error) bool {
	return apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsInternalError(err)
}
