// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tigera/es-proxy/pkg/pip"

	log "github.com/sirupsen/logrus"
)

// PolicyImpactParamsHandler is a middleware http handler that moves
// policy actions request params, "policyActions:...", from the request body
// into a custom context value.
// This custom context value is picked up by the the policy impact mutator after
// the es proxy request has completed and passed to the primary pip function
func PolicyImpactParamsHandler(h http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		//only interested in post requests so skip otherwise
		if req.Method != http.MethodPost {
			h.ServeHTTP(w, req)
		}

		//extract the request body and unmarshal the policy impact parameters
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.WithError(err).Debug("Error reading request body ")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		params := PolicyImpactParams{}
		err = json.Unmarshal(b, &params)
		if err != nil {
			log.WithError(err).Debug("Error reading request json ")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//add the policy actions to the context
		h.ServeHTTP(w, req.WithContext(NewContextWithPolicyImpactActions(req.Context(), params)))

	})

}

func NewContextWithPolicyImpactActions(ctx context.Context, params PolicyImpactParams) context.Context {

	return context.WithValue(ctx, pip.PolicyImpactContextKey, params.PolicyActions)
}

type PolicyImpactParams struct {
	PolicyActions []pip.NetworkPolicyChange `json:"policyActions"`
}
