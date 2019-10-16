// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"

	lmaauth "github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/policyrec"
)

const (
	defaultTierName = "default"
)

// The response that we return
type PolicyRecommendationResponse struct {
	*policyrec.Recommendation

	ErrorMessage string `json:"errorMessage"`
}

// PolicyRecommendationHandler returns a handler that writes a json response containing recommended policies.
func PolicyRecommendationHandler(authz lmaauth.K8sAuthInterface, k8sClient k8s.Interface, c policyrec.CompositeAggregator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check that the request has the appropriate method. Should it be GET or POST?

		// Extract the recommendation parameters
		params, err := policyrec.ExtractPolicyRecommendationParamsFromRequest(req)
		if err != nil {
			log.Infof("Error extracting policy recommendation parameters: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check permissions for the namespaces and endpoint names requested
		if stat, err := policyrec.ValidatePermissions(req, authz); err != nil {
			log.Infof("Not permitting user actions (code=%d): %v", stat, err)
			http.Error(w, err.Error(), stat)
			return
		}

		// Query elasticsearch with the parameters provided
		flows, err := policyrec.QueryElasticsearchFlows(context.TODO(), c, params)
		if err != nil {
			log.Infof("Error querying elasticsearch: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var (
			recommendation *policyrec.Recommendation
			recErr         error
		)
		if len(flows) == 0 {
			log.WithField("params", params).Info("No matching flows found")
			recErr = fmt.Errorf("No matching flows found for endpoint name '%v' in namespace '%v' within the time range '%v:%v'", params.EndpointName, params.Namespace, params.StartTime, params.EndTime)
		} else {

			policyName := policyrec.GeneratePolicyName(k8sClient, params)
			// Setup the recommendation engine. We specify the default tier as the flows that we are fetching
			// is at the end of the default tier. Similarly we set the recommended policy order to nil as well.
			// TODO(doublek): Tier and policy order should be obtained from the observation point policy.
			recEngine := policyrec.NewEndpointRecommendationEngine(params.EndpointName, params.Namespace, policyName, defaultTierName, nil)
			for _, flow := range flows {
				log.WithField("flow", flow).Debug("Calling recommendation engine with flow")
				err := recEngine.ProcessFlow(*flow)
				if err != nil {
					log.WithError(err).WithField("flow", flow).Debug("Error processing flow")
				}
			}
			recommendation, recErr = recEngine.Recommend()
		}
		response := &PolicyRecommendationResponse{
			Recommendation: recommendation,
		}
		if recErr != nil {
			response.ErrorMessage = recErr.Error()
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
