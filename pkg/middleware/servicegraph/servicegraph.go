// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/k8s"
)

// This file implements the main HTTP handler factory for service graph. This is the main entry point for service
// graph in es-proxy. The handler pulls together various components to parse the request, query the flow data,
// filter and aggregate the flows. All HTTP request processing is handled here.

var (
	authReviewAttrListEndpoints = []apiv3.AuthorizationReviewResourceAttributes{{
		APIGroup: "projectcalico.org",
		Resources: []string{
			"hostendpoints", "networksets", "globalnetworksets",
		},
		Verbs: []string{"list"},
	}, {
		APIGroup:  "",
		Resources: []string{"pods"},
		Verbs:     []string{"list"},
	}}
)

type ServiceGraph interface {
	Handler() http.HandlerFunc
}

func NewServiceGraph(elasticClient lmaelastic.Client, clientSetFactory k8s.ClientSetFactory) ServiceGraph {
	return &serviceGraph{
		flowCache:        NewServiceGraphCache(elasticClient),
		clientSetFactory: clientSetFactory,
	}
}

// serviceGraph implements the ServiceGraph interface.
type serviceGraph struct {
	// Flows cache.
	flowCache        ServiceGraphCache
	clientSetFactory k8s.ClientSetFactory
}

// RequestData encapsulates data parsed from the request that is shared between the various components that construct
// the service graph.
type RequestData struct {
	request        *v1.ServiceGraphRequest
	appCluster     k8s.ClientSet
	userManagement k8s.ClientSet
	userCluster    k8s.ClientSet
}

func (s *serviceGraph) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		ctx := req.Context()
		var err error
		rd := &RequestData{}

		// Extract the request specific data used to collate and filter the data.
		// - The parsed service graph request
		// - A bunch of client sets:
		//   - App/Cluster:     Query host names
		//   - User/Management: Determine which ES tables the user has access to
		//   - User/Cluster:    Flow endpoint RBAC, Events
		// - RBAC filter.
		if rd.request, err = s.getServiceGraphRequest(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if rd.appCluster, err = s.clientSetFactory.NewClientSetForApplication(rd.request.Cluster); err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadRequest)
			return
		} else if rd.userManagement, err = s.clientSetFactory.NewClientSetForUser(req, ""); err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadRequest)
			return
		} else if rd.userCluster, err = s.clientSetFactory.NewClientSetForUser(req, rd.request.Cluster); err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadRequest)
			return
		}

		// Get the filtered flow from the cache.
		if f, err := s.flowCache.GetFilteredServiceGraphData(ctx, rd); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if pv, err := ParseViewIDs(rd, f.ServiceGroups); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if sg, err := GetServiceGraphResponse(f, pv); err != nil {
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
