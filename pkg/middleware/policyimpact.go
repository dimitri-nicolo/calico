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
func PolicyImpactParamsHandler(authz K8sAuthInterface, h http.Handler) http.Handler {

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

		//check permissions for the policy actions being previewed
		ok, err := checkPolicyActionsPermissions(params.PolicyActions, req, authz)
		if err != nil {
			log.WithError(err).Debug("Error reading policy permissions ")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !ok {
			log.Debug("Policy Impact permission denied")
			http.Error(w, "Policy action not allowed for user", http.StatusUnauthorized)
			return
		} else {
			log.Debug("Policy Impact Permissions OK")
		}

		//add the policy actions to the context
		h.ServeHTTP(w, req.WithContext(NewContextWithPolicyImpactActions(req.Context(), *params)))

	})

}

// PolicyImpactRequestProcessor will extract a PolicyImpactParams object if it exists
// in the request (policyActions) It will also modify the request removing the
// policyActions from the request body
func PolicyImpactRequestProcessor(req *http.Request) (p *PolicyImpactParams, e error) {

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

	//if we later return without returning params, reset the request body
	defer func() {
		if p == nil {
			req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			req.ContentLength = int64(len(b))
		}
	}()

	//unmarshal the body data to a map
	var data interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}

	//look for the pip parameter in the map if not a pip request
	//restore the body and return
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

	return &pip, nil
}

func checkPolicyActionsPermissions(actions []pip.NetworkPolicyChange, req *http.Request, authz K8sAuthInterface) (bool, error) {
	factory := NewStandardPolicyImpactRbacHelperFactory(authz)
	rbac := factory.NewPolicyImpactRbacHelper(req)
	for i, _ := range actions {
		ok, err := rbac.CanPreviewPolicyAction(actions[i].ChangeAction, actions[i].NetworkPolicy)
		if err != nil {
			return false, err
		}
		if ok == false {
			return false, nil
		}
	}
	return true, nil
}

func NewContextWithPolicyImpactActions(ctx context.Context, params PolicyImpactParams) context.Context {

	return context.WithValue(ctx, pip.PolicyImpactContextKey, params.PolicyActions)
}

type PolicyImpactParams struct {
	PolicyActions []pip.NetworkPolicyChange `json:"policyActions"`
}
