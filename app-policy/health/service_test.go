// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

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

package health

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	healthzv1 "google.golang.org/grpc/health/grpc_health_v1"
)

type reporter struct {
	Ready bool
}

func (r *reporter) Readiness() bool {
	return r.Ready
}

func TestHealthService(t *testing.T) {
	g := NewWithT(t)
	reporter := &reporter{
		Ready: false,
	}
	s := NewHealthCheckService(reporter)

	req := &healthzv1.HealthCheckRequest{}
	resp, err := s.Check(context.Background(), req)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resp.Status).To(Equal(healthzv1.HealthCheckResponse_NOT_SERVING))

	reporter.Ready = true
	resp, err = s.Check(context.Background(), req)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resp.Status).To(Equal(healthzv1.HealthCheckResponse_SERVING))
}
