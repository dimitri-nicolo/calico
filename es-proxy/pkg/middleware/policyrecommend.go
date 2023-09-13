// Copyright (c) 2019, 2022 Tigera, Inc. All rights reserved.
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/lma/pkg/policyrec"
	"github.com/projectcalico/calico/lma/pkg/rbac"

	k8srequest "k8s.io/apiserver/pkg/endpoints/request"
)

const (
	defaultTierName    = "default"
	defaultPolicyOrder = float64(1000)
)

// The response that we return.
type PolicyRecommendationResponse struct {
	*policyrec.Recommendation
}

// PolicyRecommendationHandler returns a handler that writes a json response containing recommended
// policies.
func PolicyRecommendationHandler(
	clientSetk8sClientFactory lmak8s.ClientSetFactory,
	k8sClientFactory datastore.ClusterCtxK8sClientFactory,
	c client.Client,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check that the request has the appropriate method. Should it be GET or POST?

		// Extract the recommendation parameters
		params, err := policyrec.ExtractPolicyRecommendationParamsFromRequest(req)
		if err != nil {
			createAndReturnError(err, "Error extracting policy recommendation parameters",
				http.StatusBadRequest, api.PolicyRec, w)
			return
		}

		clusterID := MaybeParseClusterNameFromRequest(req)
		log.WithField("cluster", clusterID).Debug("Cluster ID from request")

		// Get the k8s client set for this cluster.
		clientSet, err := clientSetk8sClientFactory.NewClientSetForApplication(clusterID)
		if err != nil {
			log.WithError(err).Error("failed to create a new client set for the cluster: ", clusterID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		authorizer, err := k8sClientFactory.RBACAuthorizerForCluster(clusterID)
		if err != nil {
			log.WithError(err).Error("failed to get authorizer from client factory")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		user, ok := k8srequest.UserFrom(req.Context())
		if !ok {
			log.WithError(err).Error("user not found in context")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check that user has sufficient permissions to list flows for the requested endpoint. This
		// happens in the selected cluster from the UI drop-down menu.
		flowHelper := rbac.NewCachedFlowHelper(user, authorizer)
		if stat, err := ValidateRecommendationPermissions(flowHelper, params); err != nil {
			createAndReturnError(err, "Not permitting user actions", stat, api.PolicyRec, w)
			return
		}

		// Query with the parameters provided
		query := policyrec.BuildQuery(params)
		pager := client.NewListPager[lapi.L3Flow](query)
		flows, err := policyrec.SearchFlows(req.Context(), c.L3Flows(clusterID).List, pager)
		if err != nil {
			createAndReturnError(err, "Error querying flows", http.StatusInternalServerError, api.PolicyRec, w)
			return
		}

		if len(flows) == 0 {
			log.WithField("params", params).Info("No matching flows found")
			return
		}

		// Setup the recommendation engine. We specify the default tier as the flows that we are
		// fetching is at the end of the default tier. Similarly we set the recommended policy order to
		// nil as well.
		// TODO(doublek): Tier and policy order should be obtained from the observation point policy.
		// Set order to 1000 so that the policy is in the middle of the tier and can be moved up or down
		// as necessary.

		recommendedPolicyOrder := defaultPolicyOrder
		recEngine := policyrec.NewEndpointRecommendationEngine(
			clientSet,
			params.EndpointName,
			params.Namespace,
			defaultTierName,
			&recommendedPolicyOrder,
		)
		for _, flow := range flows {
			log.WithField("flow", flow).Debug("Calling recommendation engine with flow")
			err := recEngine.ProcessFlow(*flow)
			if err != nil {
				log.WithError(err).WithField("flow", flow).Debug("Error processing flow")
			}
		}
		recommendation, err := recEngine.Recommend()
		if err != nil {
			createAndReturnError(err, err.Error(), http.StatusInternalServerError, api.PolicyRec, w)
			return
		}
		response := &PolicyRecommendationResponse{
			Recommendation: recommendation,
		}
		log.WithField("recommendation", recommendation).Debug("Policy recommendation response")
		recJSON, err := json.Marshal(response)
		if err != nil {
			createAndReturnError(err, "Error marshalling recommendation to JSON",
				http.StatusInternalServerError, api.PolicyRec, w)
			return
		}
		_, err = w.Write(recJSON)
		if err != nil {
			errorStr := fmt.Sprintf("Error writing JSON recommendation: %v", recommendation)
			createAndReturnError(err, errorStr, http.StatusInternalServerError, api.PolicyRec, w)
			return
		}
	})
}

// ValidateRecommendationPermissions checks that the user is able to list flows for the specified
// endpoint.
func ValidateRecommendationPermissions(
	flowHelper rbac.FlowHelper, params *policyrec.PolicyRecommendationParams,
) (int, error) {
	if params.Namespace != "" {
		if ok, err := flowHelper.CanListEndpoint(api.EndpointTypeWep, params.Namespace); err != nil {
			return http.StatusInternalServerError, err
		} else if !ok {
			return http.StatusForbidden, fmt.Errorf("user cannot list flows for pods in namespace %s",
				params.Namespace)
		}
	} else {
		if ok, err := flowHelper.CanListEndpoint(api.EndpointTypeHep, ""); err != nil {
			return http.StatusInternalServerError, err
		} else if !ok {
			return http.StatusForbidden, fmt.Errorf("user cannot list flows for host-endpoints")
		}
	}
	return http.StatusOK, nil
}
