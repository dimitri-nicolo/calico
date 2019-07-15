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
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/elastic"

	"github.com/tigera/es-proxy/pkg/pip"
)

// PolicyImpactHandler is a middleware http handler that extracts PIP arguments from the request
// if they exist and uses them to execute a PIP request. It also checks that the user
// has the necessary permissions to execute this PIP request.
func PolicyImpactHandler(authz K8sAuthInterface, p pip.PIP, esClient elastic.Client, h http.Handler) http.Handler {
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
		}
		log.Debug("Policy Impact Permissions OK")

		_, err = p.GetPolicyCalculator(context.TODO(), params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusOK)
		}

		_, _ = w.Write([]byte("TODO: fill this with flow logs"))
	})

}

// PolicyImpactRequestProcessor will extract a PolicyImpactParams object if it exists
// in the request (resourceActions) It will also modify the request removing the
// resourceActions from the request body
func PolicyImpactRequestProcessor(req *http.Request) (p *pip.PolicyImpactParams, e error) {

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
	var reqRaw map[string]json.RawMessage
	err = json.Unmarshal(b, &reqRaw)
	if err != nil {
		log.WithError(err).Debug("Error unmarshaling query - just proxy it")
		return nil, nil
	}

	// Extract and remove the resourceActions value if present, if not present just exit immediately.
	resourceActionsRaw, ok := reqRaw["resourceActions"]
	if !ok {
		return nil, nil
	}
	log.WithField("resourceActionsRaw", resourceActionsRaw).Debug("Policy Impact request found")

	// Extract the start and end time of the query.
	queryRaw, ok := reqRaw["query"]
	q := query{}
	err = json.Unmarshal(queryRaw, &q)
	if err != nil {
		log.WithError(err).Debug("Error unmarshaling query - just proxy it")
		return nil, nil
	}

	now := time.Now()
	var fromTime, toTime *time.Time
	for _, e := range q.Bool.Must {
		if e.Range.EndTime.GTE != nil && e.Range.EndTime.LTE != nil {
			fromTime = ParseElasticsearchTime(now, e.Range.EndTime.GTE)
			toTime = ParseElasticsearchTime(now, e.Range.EndTime.LTE)
			log.Debugf("Extracted request time range: %v -> %v", fromTime, toTime)
			break
		}
	}

	// Delete resourceActions param and rebuild the request body without it.
	delete(reqRaw, "resourceActions")
	nb, err := json.Marshal(reqRaw)
	if err != nil {
		log.WithError(err).Debug("Error marshaling query with pip params removed")
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(nb))
	req.ContentLength = int64(len(nb))

	// It's a pip request, parse the resourceActions which should be a slice of pip.ResourceChange struct.
	pipParms := &pip.PolicyImpactParams{
		FromTime: fromTime,
		ToTime:   toTime,
	}
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

// Define structs to unpack the time query.
/*

	"query": {
		"bool": {
			"must": [
			{
				"range": {
					"end_time": {
						"gte": "now-15m",
						"lte": "now-0m"
					}
				}
			},
	...
*/
type query struct {
	Bool boolquery `json:"bool"`
}
type boolquery struct {
	Must []must `json:"must"`
}
type must struct {
	Range mustRange `json:"range"`
}
type mustRange struct {
	EndTime endTime `json:"end_time"`
}
type endTime struct {
	GTE *string `json:"gte"`
	LTE *string `json:"lte"`
}

// ParseElasticsearchTime parses the time string supplied in the ES query.
func ParseElasticsearchTime(now time.Time, tstr *string) *time.Time {
	if tstr == nil {
		return nil
	}
	clog := log.WithField("time", *tstr)
	// Expecting times in RFC3999 format, or now-<duration> format. Try the latter first.
	parts := strings.SplitN(*tstr, "-", 2)
	if strings.TrimSpace(parts[0]) == "now" {
		clog.Debug("Time is relative to now")

		// Make sure time is in UTC format.
		now = now.UTC()

		// Handle time string just being "now"
		if len(parts) == 1 {
			clog.Debug("Time is now")
			return &now
		}

		// Time string has section after the subtraction sign. Parse it as a duration.
		clog.Debugf("Time string in now-x format; x=%s", parts[1])
		sub, err := time.ParseDuration(strings.TrimSpace(parts[1]))
		if err != nil {
			clog.WithError(err).Debug("Error parsing duration string")
			return nil
		}
		t := now.Add(-sub)
		return &t
	}
	if t, err := time.Parse(time.RFC3339, *tstr); err == nil {
		clog.Debug("Time is in valid RFC3339 format")
		return &t
	}

	clog.Debug("Time format is not recognized")
	return nil
}
