package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	elastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	lmaauth "github.com/tigera/lma/pkg/auth"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/rbac"
)

const (
	sourceAggregationName = "source_namespace"
	destAggregationName   = "dest_namespace"
	compAggregationName   = "source_dest_namespaces"
)

var namespaceGatheringTimeout = 10 * time.Second

type FlowLogNamespaceParams struct {
	Limit         int32    `json:"limit"`
	Actions       []string `json:"actions"`
	ClusterName   string   `json:"cluster"`
	Prefix        string   `json:"prefix"`
	Unprotected   bool     `json:"unprotected"`
	StartDateTime string   `json:"startDateTime"`
	EndDateTime   string   `json:"endDateTime"`

	// Parsed parameters
	startDateTimeESParm interface{}
	endDateTimeESParm   interface{}
}

type Namespace struct {
	Name string `json:"name"`
}

func FlowLogNamespaceHandler(auth lmaauth.K8sAuthInterface, esClient lmaelastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// validate request
		params, err := validateFlowLogNamespacesRequest(req)
		if err != nil {
			log.WithError(err).Info("Error validating request")
			switch err {
			case errInvalidMethod:
				http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			case errParseRequest:
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errInvalidAction:
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			}
			return
		}

		rbacHelper := rbac.NewCachedFlowHelper(&userAuthorizer{k8sAuth: auth, userReq: req})
		response, err := getNamespacesFromElastic(params, esClient, rbacHelper)
		if err != nil {
			log.WithError(err).Info("Error getting namespaces from elastic")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
		}

		// return namespace components array
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.WithError(err).Info("Encoding namespaces array failed")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
			return
		}
		return
	})
}

// TODO: Commonize this since this appears to be a copy of validateFlowLogNamesRequest
func validateFlowLogNamespacesRequest(req *http.Request) (*FlowLogNamespaceParams, error) {
	// Validate http method
	if req.Method != http.MethodGet {
		return nil, errInvalidMethod
	}

	// extract params from request
	url := req.URL.Query()
	limit, err := extractLimitParam(url)
	if err != nil {
		return nil, errParseRequest
	}
	actions := lowerCaseParams(url["actions"])
	cluster := strings.ToLower(url.Get("cluster"))
	prefix := strings.ToLower(url.Get("prefix"))
	unprotected := false
	if unprotectedValue := url.Get("unprotected"); unprotectedValue != "" {
		if unprotected, err = strconv.ParseBool(unprotectedValue); err != nil {
			return nil, errParseRequest
		}
	}

	startDateTimeString := url.Get("startDateTime")
	endDateTimeString := url.Get("endDateTime")

	// Parse the start/end time to validate the format. We don't need the resulting time struct.
	now := time.Now()
	_, startDateTimeESParm, err := ParseElasticsearchTime(now, &startDateTimeString)
	if err != nil {
		log.WithError(err).Info("Error extracting start date time")
		return nil, errParseRequest
	}
	_, endDateTimeESParm, err := ParseElasticsearchTime(now, &endDateTimeString)
	if err != nil {
		log.WithError(err).Info("Error extracting end date time")
		return nil, errParseRequest
	}

	params := &FlowLogNamespaceParams{
		Actions:             actions,
		Limit:               limit,
		ClusterName:         cluster,
		Prefix:              prefix,
		Unprotected:         unprotected,
		StartDateTime:       startDateTimeString,
		EndDateTime:         endDateTimeString,
		startDateTimeESParm: startDateTimeESParm,
		endDateTimeESParm:   endDateTimeESParm,
	}

	// Check whether the params are provided in the request and set default values if not
	if params.ClusterName == "" {
		params.ClusterName = "cluster"
	}
	if params.Prefix != "" {
		params.Prefix = fmt.Sprintf("%s.*", params.Prefix)
	}
	valid := validateActions(params.Actions)
	if !valid {
		return nil, errInvalidAction
	}
	valid = validateActionsAndUnprotected(params.Actions, params.Unprotected)
	if !valid {
		return nil, errInvalidActionUnprotected
	}

	return params, nil
}

func buildESQuery(params *FlowLogNamespaceParams) *elastic.BoolQuery {
	query := elastic.NewBoolQuery()
	var termFilterValues []interface{}
	if len(params.Actions) == 0 {
		return query
	}

	for _, action := range params.Actions {
		termFilterValues = append(termFilterValues, action)
	}
	query = query.Filter(elastic.NewTermsQuery("action", termFilterValues...))

	if params.Unprotected {
		query = query.Filter(UnprotectedQuery())
	}

	if params.StartDateTime != "" {
		startFilter := elastic.NewRangeQuery("start_time").Gt(params.StartDateTime)
		query = query.Filter(startFilter)
	}
	if params.EndDateTime != "" {
		endFilter := elastic.NewRangeQuery("end_time").Lt(params.EndDateTime)
		query = query.Filter(endFilter)
	}

	return query
}

func buildAggregation(params *FlowLogNamespaceParams, afterKey map[string]interface{}) *elastic.CompositeAggregation {
	// Need to request double the limit in order to guarantee the number of unique terms
	// e.g. Ep1 -> Ep2, Ep1 -> Ep3, Ep2 -> Ep1, Ep3 -> Ep1 has endpoints Ep1, Ep2, Ep3 for 4 flows.
	// For X aggregated flows, the smallest amount of unique endpoints we can have is X/2 + 1
	baseAgg := elastic.NewCompositeAggregation().Size(int(params.Limit) * 2).AggregateAfter(afterKey)

	sourceNameAgg := elastic.NewCompositeAggregationTermsValuesSource(sourceAggregationName).Field(sourceAggregationName)
	destNameAgg := elastic.NewCompositeAggregationTermsValuesSource(destAggregationName).Field(destAggregationName)

	return baseAgg.Sources(sourceNameAgg, destNameAgg)
}

func validNamespaces(params *FlowLogNamespaceParams, name string, rbacHelper rbac.FlowHelper) bool {
	// Filter out names based on the search prefix if it exists
	if params.Prefix != "" {
		return strings.HasPrefix(name, params.Prefix)
	}

	// Filter out an empty namespace which signifies that
	// this log was for the global namespace
	if name == "-" {
		return false
	}

	valid, err := validateNamespaceRBAC(name, rbacHelper)
	if err != nil {
		log.WithError(err).Debugf("Error attempting to validate namespace RBAC for namespace: %s", name)
	}

	return valid
}

func validateNamespaceRBAC(namespace string, rbacHelper rbac.FlowHelper) (bool, error) {
	// Check if the user has access to all namespaces first
	if allowed, err := rbacHelper.IncludeNamespace(""); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}

	allowed, err := rbacHelper.IncludeNamespace(namespace)
	if err != nil {
		return false, err
	}

	return allowed, err
}

func getNamespacesFromElastic(params *FlowLogNamespaceParams, esClient lmaelastic.Client, rbacHelper rbac.FlowHelper) ([]Namespace, error) {
	uniqueNamespaces := make(map[string]struct{})
	namespaces := make([]Namespace, 0)
	var afterKey map[string]interface{}
	filtered := make(map[string]struct{})

	// Form the query
	query := buildESQuery(params)

	// Set up the timeout
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), namespaceGatheringTimeout)
	defer cancel()

	// Keep sampling until the limit has been hit
	for len(namespaces) < int(params.Limit) {
		var namespaceAggregationItems *elastic.AggregationBucketCompositeItems
		select {
		case <-ctxWithTimeout.Done():
			return namespaces, ctxWithTimeout.Err()
		default:
			var namespacesFound bool

			// Build the aggregation
			compAggregation := buildAggregation(params, afterKey)
			index := getClusterFlowIndex(params.ClusterName)

			// perform Aggregated ES query
			searchQuery := esClient.Backend().Search().
				Index(index).
				Query(query).
				Size(0)
			searchQuery = searchQuery.Aggregation(compAggregationName, compAggregation)
			searchResult, err := esClient.Do(ctxWithTimeout, searchQuery)

			if err != nil {
				return nil, err
			}

			if searchResult.Aggregations == nil {
				return []Namespace{}, nil
			}

			namespaceAggregationItems, namespacesFound = searchResult.Aggregations.Composite(compAggregationName)

			if !namespacesFound {
				return []Namespace{}, nil
			}
		}

		// Extract unique namespaces from buckets
		for _, bucket := range namespaceAggregationItems.Buckets {
			srcNamespaceInf := bucket.Key[sourceAggregationName]
			destNamespaceInf := bucket.Key[destAggregationName]
			srcNamespace := srcNamespaceInf.(string)
			destNamespace := destNamespaceInf.(string)
			if _, seen := filtered[srcNamespace]; !seen && validNamespaces(params, srcNamespace, rbacHelper) {
				if _, exists := uniqueNamespaces[srcNamespace]; !exists {
					uniqueNamespaces[srcNamespace] = struct{}{}
					namespaceComponent := Namespace{Name: srcNamespace}
					namespaces = append(namespaces, namespaceComponent)
					if len(namespaces) == int(params.Limit) {
						break
					}
				}
			} else if !seen {
				// Cache namespaces that did not pass validation so we do not
				// need to reevaluate them
				filtered[srcNamespace] = struct{}{}
			}
			// TODO: This logic can be simplified with the use of the common Set in libcalico
			if _, seen := filtered[destNamespace]; !seen && validNamespaces(params, destNamespace, rbacHelper) {
				if _, exists := uniqueNamespaces[destNamespace]; !exists {
					uniqueNamespaces[destNamespace] = struct{}{}
					namespaceComponent := Namespace{Name: destNamespace}
					namespaces = append(namespaces, namespaceComponent)
					if len(namespaces) == int(params.Limit) {
						break
					}
				}
			} else if !seen {
				// Cache namespaces that did not pass validation so we do not
				// need to reevaluate them
				filtered[destNamespace] = struct{}{}
			}
		}

		// Check to see if there are any more results. If there
		// are no more results, quit the loop.
		// If there are less than the requested amount of results,
		// there should be no more results to give.
		if len(namespaceAggregationItems.Buckets) < int(params.Limit)*2 {
			break
		}

		// Set the after key for the next iteration
		afterKey = namespaceAggregationItems.AfterKey
	}

	return namespaces, nil
}
