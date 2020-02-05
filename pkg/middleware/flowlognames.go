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
	srcDestNamesAggName    = "source_dest_name_aggrs"
	sourceAggName          = "source_name_aggr"
	destAggName            = "dest_name_aggr"
	flowLogEndpointTypeNs  = "ns"
	flowLogEndpointTypeWep = "wep"
	flowLogEndpointTypeHep = "hep"
)

var namesGatheringTimeout = 10 * time.Second

type FlowLogNamesParams struct {
	Limit         int32    `json:"limit"`
	Actions       []string `json:"actions"`
	ClusterName   string   `json:"cluster"`
	Namespace     string   `json:"namespace"`
	Prefix        string   `json:"prefix"`
	Unprotected   bool     `json:"unprotected"`
	StartDateTime string   `json:"startDateTime"`
	EndDateTime   string   `json:"endDateTime"`

	// Parsed parameters
	startDateTimeESParm interface{}
	endDateTimeESParm   interface{}

	// The following are parameters set by the handling, not the request
	filterNamespaces []NamespacePermissions
}

type NamespacePermissions struct {
	Namespace string
	// TODO: Look into changing this into a bitwise set of values so checking can just be as simple as a bitwise AND or OR.
	EndpointTypes map[string]struct{}
}

func ConvertNamesParamsToNamespacesParams(params *FlowLogNamesParams) *FlowLogNamespaceParams {
	return &FlowLogNamespaceParams{
		Limit:         params.Limit,
		Actions:       params.Actions,
		ClusterName:   params.ClusterName,
		Prefix:        params.Prefix,
		Unprotected:   params.Unprotected,
		StartDateTime: params.StartDateTime,
		EndDateTime:   params.EndDateTime,
	}
}

func FlowLogNamesHandler(auth lmaauth.K8sAuthInterface, esClient lmaelastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Validate request
		params, err := validateFlowLogNamesRequest(req)
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

		// Validate RBAC
		params, err = buildAllowedEndpointsByNamespace(params, esClient, rbacHelper)
		if err != nil {
			log.WithError(err).Info("Error validating RBAC for the request")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
		}

		response, err := getNamesFromElastic(params, esClient)
		if err != nil {
			log.WithError(err).Info("Error getting names from elastic")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
		}

		// return array of strings with unique names
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.WithError(err).Info("Encoding names array failed")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// TODO: Commonize this since this appears to be a copy of validateFlowLogNamespacesRequest
func validateFlowLogNamesRequest(req *http.Request) (*FlowLogNamesParams, error) {
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
	namespace := strings.ToLower(url.Get("namespace"))
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

	params := &FlowLogNamesParams{
		Actions:             actions,
		Limit:               limit,
		ClusterName:         cluster,
		Prefix:              prefix,
		Namespace:           namespace,
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

// buildAllowedEndpointsByNamespace queries all namespaces and then uses RBAC to filter out which resources are
// allowed to be seen within each namespace.
func buildAllowedEndpointsByNamespace(params *FlowLogNamesParams, esClient lmaelastic.Client, rbacHelper rbac.FlowHelper) (*FlowLogNamesParams, error) {
	// Query namespaces from Elastic first. This is needed for doing
	// RBAC filtering on namespace permissions.

	// Check permissions across all namespaces
	allNamespacesAllowedEPs, err := allowedEndpointTypes("", rbacHelper)
	if err != nil {
		return params, err
	}

	// An empty namespace value from the params means that we will try to return flows from as many namespaces as allowed.
	if params.Namespace == "" {
		// Check if global endpoint types are allowed
		globalAllowedEndpoints, err := globalAllowedEndpointTypes(rbacHelper)
		if err != nil {
			return params, err
		}

		// Add in a blank namespace permission if viewing all endpoints over all namespaces and globally is allowed.
		if allAllowedForContext(globalAllowedEndpoints) && allAllowedForContext(allNamespacesAllowedEPs) {
			params.filterNamespaces = append(params.filterNamespaces, NamespacePermissions{})
			return params, nil
		}

		// Not all endpoint types are allowed over all namespaces and global resources.
		// Query each specific namespace to verify what endpoint types are allowed.
		nsParams := ConvertNamesParamsToNamespacesParams(params)
		nsResponse, err := getNamespacesFromElastic(nsParams, esClient, rbacHelper)
		if err != nil {
			log.WithError(err).Info("Error getting namespaces from elastic for a name request")
			return params, err
		}

		// Filter out namespaces based on RBAC and add them to the params
		for _, namespace := range nsResponse {
			allowedEndpoints, err := allowedEndpointTypes(namespace.Name, rbacHelper)
			if err != nil {
				return params, err
			}

			// Consolidate permissions specified for the namespace with permissions over all namespaces
			for allEpType, _ := range allNamespacesAllowedEPs {
				// If an endpoint type is allowed across all namespaces it
				// is allowed for this particular namespace.
				allowedEndpoints[allEpType] = struct{}{}
			}

			if len(allowedEndpoints) > 0 {
				nsPerms := NamespacePermissions{
					Namespace:     namespace.Name,
					EndpointTypes: allowedEndpoints,
				}
				params.filterNamespaces = append(params.filterNamespaces, nsPerms)
			}
		}

		// Add in the global namespace "-" if allowed
		if len(globalAllowedEndpoints) > 0 {
			globalPerms := NamespacePermissions{
				Namespace:     "-",
				EndpointTypes: globalAllowedEndpoints,
			}
			params.filterNamespaces = append(params.filterNamespaces, globalPerms)
		}
	} else {
		// Check if the specified namespace is allowed
		var allowedEndpoints map[string]struct{}
		if params.Namespace != "-" {
			allowedEndpoints, err = allowedEndpointTypes(params.Namespace, rbacHelper)
			if err != nil {
				return params, err
			}
		} else {
			allowedEndpoints, err = globalAllowedEndpointTypes(rbacHelper)
		}

		// Consolidate permissions specified for the namespace with permissions over all namespaces
		for allEpType, _ := range allNamespacesAllowedEPs {
			if _, exist := allowedEndpoints[allEpType]; !exist {
				// If an endpoint type is allowed across all namespaces but not specified for
				// this particular namespace, allow that endpoint type
				allowedEndpoints[allEpType] = struct{}{}
			}
		}

		if len(allowedEndpoints) == 0 {
			// Nothing is allowed for a specified endpoint. Raise an error.
			return params, fmt.Errorf("Not authorized to access namespace: %s", params.Namespace)
		} else {
			nsPerms := NamespacePermissions{
				Namespace:     params.Namespace,
				EndpointTypes: allowedEndpoints,
			}
			params.filterNamespaces = append(params.filterNamespaces, nsPerms)
		}
	}

	return params, nil
}

func allowedEndpointTypes(namespace string, rbacHelper rbac.FlowHelper) (map[string]struct{}, error) {
	allowedEndpointTypes := make(map[string]struct{})
	if allowPods, err := rbacHelper.CanListPods(namespace); err != nil {
		log.WithError(err).Infof("Error checking pod list permissions for namespace: %s", namespace)
		return allowedEndpointTypes, err
	} else if allowPods {
		allowedEndpointTypes[flowLogEndpointTypeWep] = struct{}{}
	}

	if allowNetSets, err := rbacHelper.CanListNetworkSets(namespace); err != nil {
		log.WithError(err).Infof("Error checking network set list permissions for namesapce: %s", namespace)
		return allowedEndpointTypes, err
	} else if allowNetSets {
		allowedEndpointTypes[flowLogEndpointTypeNs] = struct{}{}
	}

	return allowedEndpointTypes, nil
}

func globalAllowedEndpointTypes(rbacHelper rbac.FlowHelper) (map[string]struct{}, error) {
	globalAllowedEndpointTypes := make(map[string]struct{})
	if globalAllowHep, err := rbacHelper.CanListHostEndpoints(); err != nil {
		log.WithError(err).Info("Error checking host endpoint permissions")
		return globalAllowedEndpointTypes, err
	} else if globalAllowHep {
		globalAllowedEndpointTypes[flowLogEndpointTypeHep] = struct{}{}
	}

	if globalAllowNs, err := rbacHelper.CanListGlobalNetworkSets(); err != nil {
		log.WithError(err).Info("Error checking global network set permissions")
		return globalAllowedEndpointTypes, err
	} else if globalAllowNs {
		globalAllowedEndpointTypes[flowLogEndpointTypeNs] = struct{}{}
	}

	return globalAllowedEndpointTypes, nil
}

// allAllowedForContext returns true if every endpoint type is present for a set in either
// the global or namespaced context.
// This means having "wep" and "ns" in the namespaced case or "hep" and "ns" in the global case.
func allAllowedForContext(allowedTypes map[string]struct{}) bool {
	if _, ok := allowedTypes[flowLogEndpointTypeNs]; !ok {
		return false
	}

	// Only need 1 of wep or hep to be allowed.
	// "wep" means that the permissions being checked on in a namespaced context.
	// "hep" means that the permissions being checked are in a global context.
	_, wepOk := allowedTypes[flowLogEndpointTypeWep]
	_, hepOk := allowedTypes[flowLogEndpointTypeHep]
	if !wepOk && !hepOk {
		return false
	}

	return true
}

func buildNamesQuery(params *FlowLogNamesParams) *elastic.BoolQuery {
	var termFilterValues []interface{}
	query := elastic.NewBoolQuery()
	nestedQuery := elastic.NewBoolQuery()
	if len(params.Actions) > 0 {
		for _, action := range params.Actions {
			termFilterValues = append(termFilterValues, action)
		}
		nestedQuery = nestedQuery.Filter(elastic.NewTermsQuery("action", termFilterValues...))
	}
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

	if len(params.filterNamespaces) > 0 {
		// If a specific namespace is not specified, then add the allowed namespaces to the filter
		filters := []elastic.Query{}
		for _, perms := range params.filterNamespaces {
			// If a blank namespace permission is the only namespace allowed, then this
			// means that all endpoints, over all namespaces and globally, are allowed to be seen.
			if len(perms.EndpointTypes) == 0 && perms.Namespace == "" {
				continue
			}
			// If both weps and networksets are allowed, then no need to add more complex queries
			// If both heps and networksets are allowed in the global case, we need to add
			// endpoint types to the query in order to exclude flows of to endpoints of type
			// "net" (internet) since those are invalid.
			if len(perms.EndpointTypes) == 2 && perms.Namespace != "-" {
				filters = append(filters, elastic.NewTermQuery("source_namespace", perms.Namespace), elastic.NewTermQuery("dest_namespace", perms.Namespace))
			} else {
				// Add queries that filter for the namespace AND the endpoint type
				srcTypeQueries := []elastic.Query{}
				destTypeQueries := []elastic.Query{}
				for key, _ := range perms.EndpointTypes {
					srcTypeQueries = append(srcTypeQueries, elastic.NewTermQuery("source_type", key))
					destTypeQueries = append(destTypeQueries, elastic.NewTermQuery("dest_type", key))
				}
				nestedSrcTypeQuery := elastic.NewBoolQuery().Should(srcTypeQueries...)
				nestedDestTypeQuery := elastic.NewBoolQuery().Should(destTypeQueries...)
				srcQuery := elastic.NewBoolQuery().Must(elastic.NewTermQuery("source_namespace", perms.Namespace), nestedSrcTypeQuery)
				destQuery := elastic.NewBoolQuery().Must(elastic.NewTermQuery("dest_namespace", perms.Namespace), nestedDestTypeQuery)
				filters = append(filters, srcQuery, destQuery)
			}
		}
		nestedQuery = nestedQuery.Should(filters...)
	}
	query = query.Filter(nestedQuery)

	return query
}

func buildNameAggregation(params *FlowLogNamesParams, afterKey map[string]interface{}) *elastic.CompositeAggregation {
	// Need to request double the limit in order to guarantee the number of unique terms
	// e.g. Ep1 -> Ep2, Ep1 -> Ep3, Ep2 -> Ep1, Ep3 -> Ep1 has endpoints Ep1, Ep2, Ep3 for 4 flows.
	// For X aggregated flows, the smallest amount of unique endpoints we can have is X/2 + 1
	baseAgg := elastic.NewCompositeAggregation().Size(int(params.Limit) * 2).AggregateAfter(afterKey)

	sourceNameAgg := elastic.NewCompositeAggregationTermsValuesSource(sourceAggName).Field(sourceAggName)
	destNameAgg := elastic.NewCompositeAggregationTermsValuesSource(destAggName).Field(destAggName)

	// Add source and destination namespaces if the namespace is specified so that the results
	// can be filtered for the correct namespace later.
	if params.Namespace != "" {
		// sourceAggregationName and destAggregationName are defined for use with the flowlognamespaces endpoint
		sourceNamespaceAgg := elastic.NewCompositeAggregationTermsValuesSource(sourceAggregationName).Field(sourceAggregationName)
		destNamespaceAgg := elastic.NewCompositeAggregationTermsValuesSource(destAggregationName).Field(destAggregationName)
		baseAgg = baseAgg.Sources(sourceNamespaceAgg, destNamespaceAgg)
	}

	return baseAgg.Sources(sourceNameAgg, destNameAgg)
}

func validName(params *FlowLogNamesParams, name, namespace string) bool {
	// Filter out names based on the search prefix if it exists
	if params.Prefix != "" {
		return strings.HasPrefix(name, params.Prefix)
	}

	// Filter out the names based on if the namespace matches the specified namespace
	if params.Namespace != "" {
		return params.Namespace == namespace
	}

	return true
}

func getNamesFromElastic(params *FlowLogNamesParams, esClient lmaelastic.Client) ([]string, error) {
	uniqueNames := make(map[string]struct{})
	names := make([]string, 0)
	var afterKey map[string]interface{}
	filtered := make(map[string]struct{})

	// Form the query
	query := buildNamesQuery(params)

	// Set up the timeout
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), namesGatheringTimeout)
	defer cancel()

	// Keep sampling until the limit has been hit
	for len(names) < int(params.Limit) {
		var nameAggregationItems *elastic.AggregationBucketCompositeItems
		select {
		case <-ctxWithTimeout.Done():
			return names, ctxWithTimeout.Err()
		default:
			var namesFound bool

			// Build the aggregation
			compNameAggregation := buildNameAggregation(params, afterKey)
			index := getClusterFlowIndex(params.ClusterName)

			// perform Aggregated ES query
			searchQuery := esClient.Backend().Search().
				Index(index).
				Query(query).
				Size(0)
			searchQuery = searchQuery.Aggregation(srcDestNamesAggName, compNameAggregation)
			searchResult, err := esClient.Do(ctxWithTimeout, searchQuery)

			if err != nil {
				return nil, err
			}

			if searchResult.Aggregations == nil {
				return names, nil
			}

			nameAggregationItems, namesFound = searchResult.Aggregations.Composite(srcDestNamesAggName)

			if !namesFound {
				return names, nil
			}
		}

		// Extract unique names from buckets
		for _, bucket := range nameAggregationItems.Buckets {
			srcNameInf := bucket.Key[sourceAggName]
			destNameInf := bucket.Key[destAggName]
			srcName := srcNameInf.(string)
			destName := destNameInf.(string)
			srcNamespaceInf := bucket.Key[sourceAggregationName]
			destNamespaceInf := bucket.Key[destAggregationName]
			srcNamespace := srcNamespaceInf.(string)
			destNamespace := destNamespaceInf.(string)
			if _, seen := filtered[srcName]; !seen && validName(params, srcName, srcNamespace) {
				if _, exists := uniqueNames[srcName]; !exists {
					uniqueNames[srcName] = struct{}{}
					names = append(names, srcName)
					if len(names) == int(params.Limit) {
						break
					}
				}
			} else if !seen {
				// Cache names that did not pass validation so we do not
				// need to reevaluate them
				filtered[srcName] = struct{}{}
			}
			// TODO: This logic can be simplified with the use of the common Set in libcalico
			if _, seen := filtered[destName]; !seen && validName(params, destName, destNamespace) {
				if _, exists := uniqueNames[destName]; !exists {
					uniqueNames[destName] = struct{}{}
					names = append(names, destName)
					if len(names) == int(params.Limit) {
						break
					}
				}
			} else if !seen {
				// Cache names that did not pass validation so we do not
				// need to reevaluate them
				filtered[destName] = struct{}{}
			}
		}

		// Check to see if there are any more results. If there
		// are no more results, quit the loop.
		// If there are less than the requested amount of results,
		// there should be no more results to give.
		if len(nameAggregationItems.Buckets) < int(params.Limit)*2 {
			break
		}

		// Set the after key for the next iteration
		afterKey = nameAggregationItems.AfterKey
	}

	return names, nil
}
