// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/lma/pkg/policyrec"
)

// handlePolicyRecommendation returns a json with recommended policies.
func (s *server) handlePolicyRecommendation(response http.ResponseWriter, request *http.Request) {
	log.Info(request.URL)

	// Extract the recommendation parameters
	params, err := policyrec.ExtractPolicyRecommendationParamsFromRequest(request)
	if err != nil {
		log.Infof("Error extracting policy recommendation parameters: %v", err)
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	rbac := s.rhf.NewPolicyRecommendationRbacHelper(request)

	if canGet, err := rbac.CanGetPolicyRecommendation(); err != nil {
		log.WithError(err).Error("Unable to determine access permissions for request")
		http.Error(response, err.Error(), http.StatusServiceUnavailable)
		return
	} else if !canGet {
		log.Debug("Requester has insufficient permissions to get policy recommendation")
		http.Error(response, "Access denied", http.StatusUnauthorized)
		return
	}

	policyRecResponse, err := s.pre.GetRecommendation(request.Context(), params)

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(response, policyRecResponse, false)
}
