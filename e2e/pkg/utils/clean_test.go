// Copyright (c) 2026 Tigera, Inc. All rights reserved.
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
	"testing"

	v3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// stubDeleteClient returns errs in order, one per Delete call, and nil once the
// list is spent. The embedded interface is nil so any other client method
// panics rather than silently doing nothing.
type stubDeleteClient struct {
	ctrlclient.Client

	errs  []error
	calls int
}

func (s *stubDeleteClient) Delete(context.Context, ctrlclient.Object, ...ctrlclient.DeleteOption) error {
	s.calls++
	if s.calls <= len(s.errs) {
		return s.errs[s.calls-1]
	}
	return nil
}

// TestDeleteResource covers why the helper exists: a delete served by the
// aggregated apiserver returns 503 for as long as that apiserver is down, and a
// spec should not fail for cleaning up during that window. A refusal, by
// contrast, must not be retried.
func TestDeleteResource(t *testing.T) {
	groupResource := schema.GroupResource{Group: "projectcalico.org", Resource: "networkpolicies"}

	for _, tc := range []struct {
		name      string
		errs      []error
		wantErr   bool
		wantCalls int
	}{
		{
			name:      "succeeds on the first call when the API is healthy",
			wantCalls: 1,
		},
		{
			name:      "retries through a transient 503 and succeeds",
			errs:      []error{apierrors.NewServiceUnavailable("service unavailable")},
			wantCalls: 2,
		},
		{
			name:      "treats an already-deleted resource as success",
			errs:      []error{apierrors.NewNotFound(groupResource, "deny-egress")},
			wantCalls: 1,
		},
		{
			name:      "does not retry a refusal",
			errs:      []error{apierrors.NewForbidden(groupResource, "deny-egress", nil)},
			wantErr:   true,
			wantCalls: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cli := &stubDeleteClient{errs: tc.errs}
			obj := &v3.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "deny-egress", Namespace: "ns"}}

			err := DeleteResource(context.Background(), cli, obj)
			if tc.wantErr && err == nil {
				t.Fatalf("DeleteResource() error = nil, want an error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("DeleteResource() error = %v, want nil", err)
			}
			if cli.calls != tc.wantCalls {
				t.Errorf("Delete called %d times, want %d", cli.calls, tc.wantCalls)
			}
		})
	}
}

// TestDeleteResourceSurfacesTheAPIError checks that exhausting the retry budget
// reports what the API was returning rather than a bare deadline, which is the
// difference between a diagnosable failure and a confusing one.
func TestDeleteResourceSurfacesTheAPIError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Nothing should be retried; the first attempt is all we get.

	cli := &stubDeleteClient{errs: []error{apierrors.NewServiceUnavailable("apiserver is rolling")}}
	obj := &v3.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "deny-egress", Namespace: "ns"}}

	err := DeleteResource(ctx, cli, obj)
	if err == nil {
		t.Fatalf("DeleteResource() error = nil, want the underlying API error")
	}
	if !apierrors.IsServiceUnavailable(err) {
		t.Errorf("DeleteResource() error = %v, want a ServiceUnavailable error", err)
	}
}
