// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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

	log "github.com/sirupsen/logrus"
	healthzv1 "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	_ healthzv1.HealthServer = (*healthCheckService)(nil)
)

// An implementation of the HealthzServer health check service.
type healthCheckService struct {
	healthzv1.UnimplementedHealthServer

	reporter ReadinessReporter
}

// ReadinessReporter is a type that knows how to report its readiness.
type ReadinessReporter interface {
	Readiness() bool
}

func NewHealthCheckService(h ReadinessReporter) *healthCheckService {
	return &healthCheckService{reporter: h}
}

//	Check - Implements the HealthServer interface
//
// We don't configure any liveness or readiness probes for Dikastes in Enterprise.
// But, we will add a startup probe for the new L7 implementation(s),
//
//	which will call this endpoint, and we want that to depend on Dikastes readiness.
func (h *healthCheckService) Check(ctx context.Context, req *healthzv1.HealthCheckRequest) (*healthzv1.HealthCheckResponse, error) {
	log.WithField("request", req).Debug("Health check request received")
	if h.reporter.Readiness() {
		return &healthzv1.HealthCheckResponse{Status: healthzv1.HealthCheckResponse_SERVING}, nil
	}
	return &healthzv1.HealthCheckResponse{Status: healthzv1.HealthCheckResponse_NOT_SERVING}, nil
}
