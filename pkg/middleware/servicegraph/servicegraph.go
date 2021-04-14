// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/authorization"
	"github.com/tigera/es-proxy/pkg/middleware/flows"
)

// This file implements the main HTTP handler factory for service graph. This is the main entry point for service
// graph in es-proxy. The handler pulls together various components to parse the request, query the flow data,
// filter and aggregate the flows. All HTTP request processing is handled here.

type ServiceGraph interface {
	Handler() http.HandlerFunc
}

func NewServiceGraph(client lmaelastic.Client) ServiceGraph {
	return &serviceGraph{
		flowCache: NewFlowCache(client),
	}
}

type serviceGraph struct {
	// Flows cache.
	flowCache FlowCache
	// TODO(rlb): Probably worth caching some data for faster response times.
}

func (s *serviceGraph) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		// Extract the request from the body.
		sgr, err := s.getServiceGraphRequest(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse the view IDs.
		indexL3 := flows.GetFlowsIndex(req)
		indexL7 := flows.GetL7FlowsIndex(req)
		rbacFilter := s.getRBACFilter(req)

		// Get the filtered flow from the cache.
		if f, err := s.flowCache.GetFilteredFlowData(indexL3, indexL7, sgr.TimeRange, rbacFilter); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if pv, err := ParseViewIDs(sgr, f.ServiceGroups); err != nil {
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
	return sgr, nil
}

// getRBACFilter creates an RBACFilter from the authorized resource verbs attached to the HTTP request context.
// It is assumed an earlier HTTP handler is included in the handler chain that will perform this query and add it to
// the context.
func (s *serviceGraph) getRBACFilter(req *http.Request) RBACFilter {
	// Extract the auth review from the context.
	verbs := authorization.GetAuthorizedResourceVerbs(req)
	return NewRBACFilterFromAuth(verbs)
}
