// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"bytes"
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

		params, err := PolicyImpactRequestProcessor(req)
		if err != nil {
			log.WithError(err).Debug("Policy impact request process failure")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//if no params were returned this is not a pip request
		if params == nil {
			h.ServeHTTP(w, req)
			return
		}

		//add the policy actions to the context
		h.ServeHTTP(w, req.WithContext(NewContextWithPolicyImpactActions(req.Context(), *params)))

	})

}

// PolicyImpactRequestProcessor will extract a PolicyImpactParams object if it exists
// in the request (policyActions) It will also modify the request removing the
// policyActions from the request body
func PolicyImpactRequestProcessor(req *http.Request) (*PolicyImpactParams, error) {

	//if it's not a post request no modifications to the request are needed
	//pip requests are always post requests
	if req.Method != http.MethodPost {
		return nil, nil
	}

	//read the body data
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	//and then put it back (the read is destructive)
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	//unmarshal the body data to a map
	var data interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}

	//look for the pip parameter in the map and return if not a pip request
	_, ok := data.(map[string]interface{})["policyActions"]
	if !ok {
		return nil, nil
	}

	//delete policyActions param and rebuild the request body without it
	delete(data.(map[string]interface{}), "policyActions")
	nb, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(nb))
	req.ContentLength = int64(len(nb))

	//now we know it's is a pip request
	log.Debug("Policy Impact request found")

	//parse out the pip params
	pip := PolicyImpactParams{}
	err = json.Unmarshal(b, &pip)
	if err != nil {
		return nil, err
	}

	log.Info("PIP:", pip)

	return &pip, nil
}

func NewContextWithPolicyImpactActions(ctx context.Context, params PolicyImpactParams) context.Context {

	return context.WithValue(ctx, pip.PolicyImpactContextKey, params.PolicyActions)
}

type PolicyImpactParams struct {
	PolicyActions []pip.NetworkPolicyChange `json:"policyActions"`
}
