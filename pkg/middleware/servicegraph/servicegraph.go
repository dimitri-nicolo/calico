// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/tigera/es-proxy/pkg/elastic"

	"github.com/projectcalico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/k8s"
)

// This file implements the main HTTP handler factory for service graph. This is the main entry point for service
// graph in es-proxy. The handler pulls together various components to parse the Request, query the flow data,
// filter and aggregate the flows. All HTTP Request processing is handled here.

const (
	defaultRequestTimeout = 60 * time.Second
)

type ServiceGraph interface {
	Handler() http.HandlerFunc
}

func NewServiceGraph(ctx context.Context, elasticClient lmaelastic.Client, clientSetFactory k8s.ClientSetFactory) ServiceGraph {
	return &serviceGraph{
		sgCache:          NewServiceGraphCache(ctx, elasticClient, clientSetFactory),
		clientSetFactory: clientSetFactory,
	}
}

// serviceGraph implements the ServiceGraph interface.
type serviceGraph struct {
	// Flows cache.
	sgCache          ServiceGraphCache
	clientSetFactory k8s.ClientSetFactory
}

// RequestData encapsulates data parsed from the request that is shared between the various components that construct
// the service graph.
type RequestData struct {
	Request        *v1.ServiceGraphRequest
	HostnameHelper HostnameHelper
	RBACFilter     RBACFilter
}

func (s *serviceGraph) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		// Extract the request specific data used to collate and filter the data.
		// - The parsed service graph request
		// - A bunch of client sets:
		//   - App/Cluster:     Query host names
		//   - User/Management: Determine which ES tables the user has access to
		//   - User/Cluster:    Flow endpoint RBAC, Events
		// - RBAC filter.
		sgr, err := s.getServiceGraphRequest(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Construct a context with timeout based on the service graph Request.
		ctx, cancel := context.WithTimeout(req.Context(), sgr.Timeout)
		defer cancel()

		csAppCluster, err := s.clientSetFactory.NewClientSetForApplication(sgr.Cluster)
		if err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadRequest)
			return
		}
		csUserManagement, err := s.clientSetFactory.NewClientSetForUser(req, "")
		if err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadRequest)
			return
		}
		csUserCluster, err := s.clientSetFactory.NewClientSetForUser(req, sgr.Cluster)
		if err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadRequest)
			return
		}

		// Run the following queries in parallel - this is used for filtering and modifying the logs, so we need to
		// do this first.
		// - Get the RBAC filter
		// - Get the host name mapping helper
		var rbacFilter RBACFilter
		var hostnameHelper HostnameHelper
		var errRBACFilter, errHostnameHelper error
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			rbacFilter, errRBACFilter = GetRBACFilter(ctx, csUserManagement, csUserCluster)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			hostnameHelper, errHostnameHelper = GetHostnameHelper(ctx, csAppCluster, sgr.SelectedView.HostAggregationSelectors)
		}()
		wg.Wait()
		if errRBACFilter != nil {
			log.WithError(err).Error("Failed to discover users permissions")
			http.Error(w, "Failed to discover users permissions", http.StatusBadRequest)
			return
		} else if errHostnameHelper != nil {
			log.WithError(err).Error("Failed to query cluster hosts")
			http.Error(w, "Failed to query cluster hosts", http.StatusBadRequest)
			return
		}

		// Create the Request data.
		rd := &RequestData{
			Request:        sgr,
			HostnameHelper: hostnameHelper,
			RBACFilter:     rbacFilter,
		}

		// Get the filtered flow from the cache.
		if f, err := s.sgCache.GetFilteredServiceGraphData(ctx, rd); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if pv, err := ParseViewIDs(rd, f.ServiceGroups); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if sg, err := GetServiceGraphResponse(rd, f, pv); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		} else {
			// Write the response.
			w.Header().Set("Content-Type", "application/json")
			if err = json.NewEncoder(w).Encode(sg); err != nil {
				log.WithError(err).Info("Encoding search results failed")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Infof("ServicePort graph request took %s; returning %d nodes and %d edges",
				time.Since(start), len(sg.Nodes), len(sg.Edges))
		}
	}
}

// getServiceGraphRequest parses the ServiceGraphRequest from the HTTP request body.
func (s *serviceGraph) getServiceGraphRequest(req *http.Request) (*v1.ServiceGraphRequest, error) {
	// Extract the request from the body.
	sgr := &v1.ServiceGraphRequest{}
	if err := json.NewDecoder(req.Body).Decode(&sgr); err != nil {
		return nil, err
	}
	if sgr.Timeout == 0 {
		sgr.Timeout = defaultRequestTimeout
	}
	if sgr.Cluster == "" {
		sgr.Cluster = elastic.DefaultClusterName
	}

	// Sanity check any user configuration that may potentially break the API. In particular all user defined names
	// that may be embedded in an ID should adhere to the IDValueRegex.
	allLayers := set.New()
	for _, layer := range sgr.SelectedView.Layers {
		if !IDValueRegex.MatchString(layer.Name) {
			return nil, fmt.Errorf("invalid layer name: %s", layer.Name)
		}
		if allLayers.Contains(layer.Name) {
			return nil, fmt.Errorf("duplicate layer name specified: %s", layer.Name)
		}
		allLayers.Add(layer.Name)
	}

	allAggrHostnames := set.New()
	for _, selector := range sgr.SelectedView.HostAggregationSelectors {
		if !IDValueRegex.MatchString(selector.Name) {
			return nil, fmt.Errorf("invalid aggregated host name: %s", selector.Name)
		}
		if allAggrHostnames.Contains(selector.Name) {
			return nil, fmt.Errorf("duplicate aggregated host name specified: %s", selector.Name)
		}
		allAggrHostnames.Add(selector.Name)
	}

	return sgr, nil
}
