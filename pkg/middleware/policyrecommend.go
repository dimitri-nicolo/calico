// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"

	lmaauth "github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/policyrec"
	"github.com/tigera/lma/pkg/rbac"
)

const (
	defaultTierName = "default"
)

// The response that we return
type PolicyRecommendationResponse struct {
	*policyrec.Recommendation
}

// PolicyRecommendationHandler returns a handler that writes a json response containing recommended policies.
func PolicyRecommendationHandler(authz lmaauth.K8sAuthInterface, k8sClient k8s.Interface, c policyrec.CompositeAggregator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check that the request has the appropriate method. Should it be GET or POST?

		// Extract the recommendation parameters
		params, err := policyrec.ExtractPolicyRecommendationParamsFromRequest(req)
		if err != nil {
			log.WithError(err).Info("Error extracting policy recommendation parameters")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check that the user is allowed to access flow logs.
		if stat, err := policyrec.ValidatePermissions(req, authz); err != nil {
			log.Infof("Not permitting user actions (code=%d): %v", stat, err)
			http.Error(w, err.Error(), stat)
			return
		}

		// Check that user has sufficient permissions to list flows for the requested endpoint.
		if stat, err := ValidateRecommendationPermissions(req, authz, params); err != nil {
			log.Infof("Not permitting user actions (code=%d): %v", stat, err)
			http.Error(w, err.Error(), stat)
			return
		}

		// Query elasticsearch with the parameters provided
		flows, err := policyrec.QueryElasticsearchFlows(req.Context(), c, params)
		if err != nil {
			log.WithError(err).Errorf("Error querying elasticsearch")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(flows) == 0 {
			log.WithField("params", params).Info("No matching flows found")
			err = fmt.Errorf("No matching flows found for endpoint name '%v' in namespace '%v' within the time range '%v:%v'", params.EndpointName, params.Namespace, params.StartTime, params.EndTime)
			log.WithError(err).Info("No matching flows found")
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		policyName := policyrec.GeneratePolicyName(k8sClient, params)
		// Setup the recommendation engine. We specify the default tier as the flows that we are fetching
		// is at the end of the default tier. Similarly we set the recommended policy order to nil as well.
		// TODO(doublek): Tier and policy order should be obtained from the observation point policy.
		// Set order to 1000 so that the policy is in the middle of the tier and can be moved up or down as necessary.
		recommendedPolicyOrder := float64(1000)
		recEngine := policyrec.NewEndpointRecommendationEngine(params.EndpointName, params.Namespace, policyName, defaultTierName, &recommendedPolicyOrder)
		for _, flow := range flows {
			log.WithField("flow", flow).Debug("Calling recommendation engine with flow")
			err := recEngine.ProcessFlow(*flow)
			if err != nil {
				log.WithError(err).WithField("flow", flow).Debug("Error processing flow")
			}
		}
		recommendation, err := recEngine.Recommend()
		if err != nil {
			log.WithError(err).Error("Error when generating recommended policy")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := &PolicyRecommendationResponse{
			Recommendation: recommendation,
		}
		log.WithField("recommendation", recommendation).Debug("Policy recommendation response")
		recJson, err := json.Marshal(response)
		if err != nil {
			log.WithError(err).Error("Error marshalling recommendation to JSON")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(recJson)
		if err != nil {
			log.WithError(err).Infof("Error writing JSON recommendation: %v", recommendation)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// ValidateRecommendationPermissions checks that the user is able to list flows for the specified endpoint.
func ValidateRecommendationPermissions(req *http.Request, k8sAuth lmaauth.K8sAuthInterface, params *policyrec.PolicyRecommendationParams) (int, error) {
	fh := rbac.NewCachedFlowHelper(&userAuthorizer{k8sAuth: k8sAuth, userReq: req})
	if params.Namespace != "" {
		if ok, err := fh.CanListPods(params.Namespace); err != nil {
			return http.StatusInternalServerError, err
		} else if !ok {
			return http.StatusForbidden, fmt.Errorf("user cannot list flows for pods in namespace %s", params.Namespace)
		}
	} else {
		if ok, err := fh.CanListHostEndpoints(); err != nil {
			return http.StatusInternalServerError, err
		} else if !ok {
			return http.StatusForbidden, fmt.Errorf("user cannot list flows for hostendpoints")
		}
	}
	return http.StatusOK, nil
}
