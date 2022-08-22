// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"

	"github.com/projectcalico/calico/lma/pkg/auth"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/projectcalico/calico/lma/pkg/k8s"
)

// This file implements the main HTTP handler factory for service graph. This is the main entry point for service
// graph in es-proxy. The handler pulls together various components to parse the request, query the flow data,
// filter and aggregate the flows. All HTTP request processing is handled here.

const (
	defaultRequestTimeout = 60 * time.Second
)

func NewServiceGraphHandler(
	ctx context.Context,
	authz auth.RBACAuthorizer,
	elasticClient lmaelastic.Client,
	clientSetFactory k8s.ClientSetFactory,
	cfg *Config,
) http.Handler {
	return NewServiceGraphHandlerWithBackend(ctx, &realServiceGraphBackend{
		authz:            authz,
		elastic:          elasticClient,
		clientSetFactory: clientSetFactory,
		config:           cfg,
	}, cfg)
}

func NewServiceGraphHandlerWithBackend(ctx context.Context, backend ServiceGraphBackend, cfg *Config) http.Handler {
	noServiceGroups := NewServiceGroups()
	noServiceGroups.FinishMappings()
	return &serviceGraph{
		sgCache:         NewServiceGraphCache(ctx, backend, cfg),
		noServiceGroups: noServiceGroups,
	}
}

// serviceGraph implements the ServiceGraph interface.
type serviceGraph struct {
	// Flows cache.
	sgCache ServiceGraphCache

	// An empty service groups helper.  Used to initially validate the format of the view data.
	noServiceGroups ServiceGroups
}

// RequestData encapsulates data parsed from the request that is shared between the various components that construct
// the service graph.
type RequestData struct {
	HTTPRequest         *http.Request
	ServiceGraphRequest *v1.ServiceGraphRequest
}

func (s *serviceGraph) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	// Extract the request specific data used to collate and filter the data.
	sgr, err := s.getServiceGraphRequest(w, req)
	if err != nil {
		httputils.EncodeError(w, err)
		return
	}

	// Construct a context with timeout based on the service graph request.
	ctx, cancel := context.WithTimeout(req.Context(), sgr.Timeout.Duration)
	defer cancel()

	// Create the request data.
	rd := &RequestData{
		HTTPRequest:         req,
		ServiceGraphRequest: sgr,
	}

	// Process the request:
	// - do a first parse of the view IDs (but with no service group info)
	// - get the filtered service graph raw data
	// - parse the view IDs, this time with service group info
	// - Compile the graph
	// - Write the response.
	if _, err := ParseViewIDs(rd, s.noServiceGroups); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if f, err := s.sgCache.GetFilteredServiceGraphData(ctx, rd); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if pv, err := ParseViewIDs(rd, f.ServiceGroups); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if sg, err := GetServiceGraphResponse(f, pv); err != nil {
		httputils.EncodeError(w, err)
		return
	} else {
		httputils.Encode(w, sg)

		log.Infof("Service graph request took %s; returning %d nodes and %d edges",
			time.Since(start), len(sg.Nodes), len(sg.Edges))
	}
}

// getServiceGraphRequest parses the request from the HTTP request body.
func (s *serviceGraph) getServiceGraphRequest(w http.ResponseWriter, req *http.Request) (*v1.ServiceGraphRequest, error) {
	// Extract the request from the body.
	var sgr v1.ServiceGraphRequest
	if err := httputils.Decode(w, req, &sgr); err != nil {
		return nil, err
	}

	// Validate parameters.
	if err := validator.Validate(sgr); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    fmt.Sprintf("Request body contains invalid data: %v", err),
			Err:    err,
		}
	}

	if sgr.Timeout.Duration == 0 {
		sgr.Timeout.Duration = defaultRequestTimeout
	}
	if sgr.Cluster == "" {
		sgr.Cluster = "cluster"
	}

	// Sanity check any user configuration that may potentially break the API. In particular all user defined names
	// that may be embedded in an ID should adhere to the IDValueRegex.
	allLayers := set.New[string]()
	for _, layer := range sgr.SelectedView.Layers {
		if !IDValueRegex.MatchString(layer.Name) {
			return nil, httputils.NewHttpStatusErrorBadRequest(fmt.Sprintf("Request body contains an invalid layer name: %s", layer.Name), nil)
		}
		if allLayers.Contains(layer.Name) {
			return nil, httputils.NewHttpStatusErrorBadRequest(fmt.Sprintf("Request body contains a duplicate layer name: %s", layer.Name), nil)
		}
		allLayers.Add(layer.Name)
	}

	allAggrHostnames := set.New[string]()
	for _, selector := range sgr.SelectedView.HostAggregationSelectors {
		if !IDValueRegex.MatchString(selector.Name) {
			return nil, httputils.NewHttpStatusErrorBadRequest(fmt.Sprintf("Request body contains an invalid aggregated host name: %s", selector.Name), nil)
		}
		if allAggrHostnames.Contains(selector.Name) {
			return nil, httputils.NewHttpStatusErrorBadRequest(fmt.Sprintf("Request body contains a duplicate aggregated host name: %s", selector.Name), nil)
		}
		allAggrHostnames.Add(selector.Name)
	}

	return &sgr, nil
}
