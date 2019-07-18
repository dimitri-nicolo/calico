// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tigera/es-proxy/pkg/pip"

	log "github.com/sirupsen/logrus"
)

// PolicyImpactHandler is a middleware http handler that extracts PIP arguments from the request
// if they exist and uses them to execute a PIP request. It also checks that the user
// has the necessary permissions to execute this PIP request.
func PolicyImpactHandler(authz K8sAuthInterface, p pip.PIP, h http.Handler) http.Handler {
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
		ok, err := checkPolicyActionsPermissions(params.ResourceActions, req, authz)
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

		_, err = p.GetPolicyCalculator(context.TODO(), params.ResourceActions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusOK)
		}

		w.Write([]byte("TODO: fill this with flow logs"))

		//add the policy actions to the context
		h.ServeHTTP(w, req)

	})

}

// PolicyImpactRequestProcessor will extract a PolicyImpactParams object if it exists
// in the request (resourceActions) It will also modify the request removing the
// resourceActions from the request body
func PolicyImpactRequestProcessor(req *http.Request) (p *PolicyImpactParams, e error) {

	// If it's not a post request no modifications to the request are needed.
	// PIP requests are always post requests
	if req.Method != http.MethodPost {
		return nil, nil
	}

	// Read the body data
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Debug("Error reading request body")
		return nil, err
	}

	// If we later return without returning params, reset the request body
	defer func() {
		if p == nil {
			req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			req.ContentLength = int64(len(b))
		}
	}()

	// Unmarshal the body data to a map of raw JSON messages.
	var query map[string]json.RawMessage
	err = json.Unmarshal(b, &query)
	if err != nil {
		log.WithError(err).Debug("Error unmarshaling query - just proxy it")
		return nil, nil
	}

	// Extract and remove the resourceActions value if present, if not present just exit immediately.
	resourceActionsRaw, ok := query["resourceActions"]
	if !ok {
		return nil, nil
	}
	log.WithField("resourceActionsRaw", resourceActionsRaw).Debug("Policy Impact request found")

	// Delete resourceActions param and rebuild the request body without it.
	delete(query, "resourceActions")
	nb, err := json.Marshal(query)
	if err != nil {
		log.WithError(err).Debug("Error marshaling query with pip params removed")
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(nb))
	req.ContentLength = int64(len(nb))

	// It's a pip request, parse the resourceActions which should be a slice of pip.ResourceChange struct.
	pipParms := &PolicyImpactParams{}
	err = json.Unmarshal([]byte(resourceActionsRaw), &pipParms.ResourceActions)
	if err != nil {
		log.WithError(err).Debug("Error unmarshaling pip params")
		return nil, err
	}

	return pipParms, nil
}

func checkPolicyActionsPermissions(actions []pip.ResourceChange, req *http.Request, authz K8sAuthInterface) (bool, error) {
	factory := NewStandardPolicyImpactRbacHelperFactory(authz)
	rbac := factory.NewPolicyImpactRbacHelper(req)
	for i, _ := range actions {
		err := validateAction(actions[i].Action)
		if err != nil {
			return false, err
		}
		ok, err := rbac.CanPreviewPolicyAction(actions[i].Action, actions[i].Resource)
		if err != nil {
			log.WithError(err).Debug("Unable to check permissions")
			return false, err
		}
		if ok == false {
			return false, nil
		}
	}
	return true, nil
}

func validateAction(action string) error {
	switch strings.ToLower(action) {
	case "create":
		fallthrough
	case "update":
		fallthrough
	case "delete":
		return nil
	}
	return fmt.Errorf("Invalid action: %v", action)
}

type PolicyImpactParams struct {
	ResourceActions []pip.ResourceChange `json:"resourceActions"`
}
