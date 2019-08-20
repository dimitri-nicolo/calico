// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/elastic"
)

const (
	//TODO(rlb): We should have a nice set of these and their mappings defined somewhere sensible that we can pull and
	//           use in the various places.
	esFlowPrefix = "tigera_secure_ee_flows"
	esSearch     = "_search"
)

// PolicyImpactHandler is a middleware http handler that extracts PIP arguments from the request
// if they exist and uses them to execute a PIP request. It also checks that the user
// has the necessary permissions to execute this PIP request.
func PolicyImpactHandler(authz K8sAuthInterface, p pip.PIP, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// If it's not a post request no modifications to the request are needed. PIP requests are always post requests.
		if req.Method != http.MethodPost {
			log.Debug("Method is not Post so cannot be a PIP request - proxying request")
			h.ServeHTTP(w, req)
			return
		}

		// Split the path by path separator. We expect a PIP URL to be an ES search query of the format:
		//     /tigera_secure_ee_flows*/_search
		// Any deviation from that and we just proxy the request.
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) != 3 || parts[0] != "" || !strings.HasPrefix(parts[1], esFlowPrefix) || parts[2] != esSearch {
			// Not a request for flow logs. Proxy.
			log.Debug("Not an elastic flow logs search -  proxying request")
			h.ServeHTTP(w, req)
			return
		}

		// Extract the PIP parameters from the request if present.
		params, err := ExtractPolicyImpactParamsFromRequest(parts[1], req)
		if err != nil {
			log.Infof("Error extracting policy impact parameters: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if params == nil {
			// No params were returned this is not a PIP request, proxy the request  by calling directly through to the
			// child handler.
			log.Debug("Not a policy impact request - proxying elastic request")
			h.ServeHTTP(w, req)
			return
		}

		// Check permissions for the policy actions being previewed.
		if stat, err := checkPolicyActionsPermissions(params.ResourceActions, req, authz); err != nil {
			log.Infof("Not permitting user actions (code=%d): %v", stat, err)
			http.Error(w, err.Error(), stat)
			return
		}

		// Use PIP to calculate the flows and package up the flows for the response. The child handler is not invoked
		// as PIP takes over the processing of the request.
		log.Debug("Policy Impact Permissions OK - getting flows")
		if flows, err := p.GetFlows(context.TODO(), params); err != nil {
			log.WithError(err).Info("Error getting flows")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if flowsJson, err := json.Marshal(flows); err != nil {
			log.WithError(err).Error("Error converting flows to JSON")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if _, err = w.Write(flowsJson); err != nil {
			log.WithError(err).Infof("Error writing JSON flows to HTTP stream: %v", flows)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug("Policy Impact request processed successfully")
	})
}

// ExtractPolicyImpactParamsFromRequest will extract a PolicyImpactParams object if it exists
// in the request (resourceActions) It will also modify the request removing the
// resourceActions from the request body
func ExtractPolicyImpactParamsFromRequest(index string, req *http.Request) (p *pip.PolicyImpactParams, err error) {

	// Read the body data.
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Info("Error reading request body")
		return nil, err
	}

	// If we later return without returning params, reset the request body.
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
		log.WithError(err).Info("Error unmarshaling query")
		return nil, fmt.Errorf("invalid elasticsearch query syntax: %v", err)
	}

	// Extract and remove the resourceActions value if present, if not present just exit immediately.
	resourceActionsRaw, ok := reqRaw["resourceActions"]
	if !ok {
		return nil, nil
	}
	log.Debugf("Policy Impact request found: %s", string(resourceActionsRaw))

	// Extract the start and end time of the query.
	queryRaw, ok := reqRaw["query"]
	q := query{}
	if err = json.Unmarshal(queryRaw, &q); err != nil {
		log.WithError(err).Info("Error unmarshaling query")
		return nil, fmt.Errorf("invalid elasticsearch query syntax: %v", err)
	}
	log.Debugf("Extracted raw query: %s", string(queryRaw))

	now := time.Now()
	var fromTime, toTime *time.Time
	for _, e := range q.Bool.Must {
		if e.Range.EndTime.GTE != nil && e.Range.EndTime.LTE != nil {
			if fromTime, err = ParseElasticsearchTime(now, e.Range.EndTime.GTE); err != nil {
				return nil, fmt.Errorf("invalid time format in query: %s", *e.Range.EndTime.GTE)
			}
			if toTime, err = ParseElasticsearchTime(now, e.Range.EndTime.LTE); err != nil {
				return nil, fmt.Errorf("invalid time format in query: %s", *e.Range.EndTime.LTE)
			}

			log.Debugf("Extracted request time range: %v -> %v", fromTime, toTime)
			break
		}
	}

	// Delete resourceActions param and rebuild the request body without it.
	delete(reqRaw, "resourceActions")
	nb, err := json.Marshal(reqRaw)
	if err != nil {
		log.WithError(err).Error("Error marshaling query with pip params removed")
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(nb))
	req.ContentLength = int64(len(nb))

	// It's a pip request, parse the resourceActions which should be a slice of pip.ResourceChange struct.
	pipParms := &pip.PolicyImpactParams{
		Query:         elastic.RawElasticQuery(queryRaw),
		FromTime:      fromTime,
		ToTime:        toTime,
		DocumentIndex: index,
	}
	err = json.Unmarshal([]byte(resourceActionsRaw), &pipParms.ResourceActions)
	if err != nil {
		log.WithError(err).Debug("Error unmarshaling pip actions")
		return nil, fmt.Errorf("invalid resource actions syntax: %v", err)
	}

	return pipParms, nil
}

// checkPolicyActionsPermissions checks whether the action in each resource update is allowed.
func checkPolicyActionsPermissions(actions []pip.ResourceChange, req *http.Request, authz K8sAuthInterface) (status int, err error) {
	factory := NewStandardPolicyImpactRbacHelperFactory(authz)
	rbac := factory.NewPolicyImpactRbacHelper(req)
	for _, action := range actions {
		if action.Resource == nil {
			return http.StatusBadRequest, fmt.Errorf("invalid resource actions syntax: resource is missing from request")
		}
		if err := validateAction(action.Action); err != nil {
			return http.StatusBadRequest, err
		}
		if stat, err := rbac.CheckCanPreviewPolicyAction(action.Action, action.Resource); err != nil {
			return stat, err
		}
	}
	return 0, nil
}

// validateAction checks that the action in a resource update is one of the expected actions. Any deviation from these
// actions is considered a bad request (even if it is strictly a valid k8s action).
func validateAction(action string) error {
	switch strings.ToLower(action) {
	case "create", "update", "delete":
		return nil
	}
	return fmt.Errorf("invalid action '%s' in preview request", action)
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
func ParseElasticsearchTime(now time.Time, tstr *string) (*time.Time, error) {
	if tstr == nil {
		return nil, nil
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
			return &now, nil
		}

		// Time string has section after the subtraction sign. We currently support minutes (m), hours (h) and days (d).
		clog.Debugf("Time string in now-x format; x=%s", parts[1])
		dur := strings.TrimSpace(parts[1])
		if dur == "0" {
			// 0 does not need units, so this also means now.
			clog.Debug("Zero delta - time is now")
			return &now, nil
		} else if len(dur) < 2 {
			// We need at least two values for the unit and the value
			clog.Debug("Error parsing duration string, unrecognised unit of time")
			return nil, errors.New("error parsing time in query - not a supported format")
		}

		// Last letter indicates the units.
		var mul time.Duration
		switch dur[len(dur)-1:] {
		case "m":
			mul = time.Minute
		case "h":
			mul = time.Hour
		case "d":
			// A day isn't necessarily 24hr, but this should be a good enough approximation for now.
			//TODO(rlb): If we really want to support the ES date math format then this'll need more work.
			mul = 24 * time.Hour
		default:
			clog.Debug("Error parsing duration string, unrecognised unit of time")
			return nil, errors.New("error parsing time in query - not a supported format")
		}

		// First digits indicates the multiplier.
		if val, err := strconv.ParseUint(strings.TrimSpace(dur[:len(dur)-1]), 10, 64); err != nil {
			clog.WithError(err).Debug("Error parsing duration string")
			return nil, err
		} else {
			t := now.Add(-(time.Duration(val) * mul))
			return &t, nil
		}
	}

	// Not now-X format, parse as RFC3339.
	if t, err := time.Parse(time.RFC3339, *tstr); err == nil {
		clog.Debug("Time is in valid RFC3339 format")
		return &t, nil
	} else {
		clog.Debug("Time format is not recognized")
		return nil, err
	}
}
