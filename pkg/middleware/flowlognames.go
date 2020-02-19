package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	elastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/set"
	lmaauth "github.com/tigera/lma/pkg/auth"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/rbac"
)

const (
	namesBucketName        = "source_dest_name_aggrs"
	flowLogEndpointTypeNs  = "ns"
	flowLogEndpointTypeWep = "wep"
	flowLogEndpointTypeHep = "hep"
	srcEpNamespaceIdx      = 0
	srcEpNameIdx           = 1
	srcEpTypeIdx           = 2
	destEpNamespaceIdx     = 3
	destEpNameIdx          = 4
	destEpTypeIdx          = 5
)

var (
	namesTimeout = 10 * time.Second

	NamesCompositeSources = []lmaelastic.AggCompositeSourceInfo{
		{"source_namespace", "source_namespace"},
		{"source_name_aggr", "source_name_aggr"},
		{"source_type", "source_type"},
		{"dest_namespace", "dest_namespace"},
		{"dest_name_aggr", "dest_name_aggr"},
		{"dest_type", "dest_type"},
	}
)

type FlowLogNamesParams struct {
	Limit         int32    `json:"limit"`
	Actions       []string `json:"actions"`
	ClusterName   string   `json:"cluster"`
	Namespace     string   `json:"namespace"`
	Prefix        string   `json:"prefix"`
	Unprotected   bool     `json:"unprotected"`
	StartDateTime string   `json:"startDateTime"`
	EndDateTime   string   `json:"endDateTime"`
	Strict        bool     `json:"string"`

	// Parsed timestamps
	startDateTimeESParm interface{}
	endDateTimeESParm   interface{}
}

type EndpointInfo struct {
	Namespace string
	Name      string
	Type      string
}

func FlowLogNamesHandler(auth lmaauth.K8sAuthInterface, esClient lmaelastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// validate request
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
		response, err := getNamesFromElastic(params, esClient, rbacHelper)
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
	strict := false
	if strictValue := url.Get("strict"); strictValue != "" {
		if strict, err = strconv.ParseBool(strictValue); err != nil {
			return nil, errParseRequest
		}
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
		Strict:              strict,
	}

	// Check whether the params are provided in the request and set default values if not
	if params.ClusterName == "" {
		params.ClusterName = "cluster"
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

	if params.startDateTimeESParm != nil || params.endDateTimeESParm != nil {
		filter := elastic.NewRangeQuery("end_time")
		if params.startDateTimeESParm != nil {
			filter = filter.Gte(params.startDateTimeESParm)
		}
		if params.endDateTimeESParm != nil {
			filter = filter.Lt(params.endDateTimeESParm)
		}
		query = query.Filter(filter)
	}

	// If both the namespace and prefix are specified, make sure to filter
	// so that at least one of the two endpoints in the flow will match both
	// conditions.
	if params.Namespace != "" && params.Prefix != "" {
		sourceQuery := elastic.NewBoolQuery().Must(
			elastic.NewPrefixQuery("source_name_aggr", params.Prefix),
			elastic.NewTermQuery("source_namespace", params.Namespace),
		)
		destQuery := elastic.NewBoolQuery().Must(
			elastic.NewPrefixQuery("dest_name_aggr", params.Prefix),
			elastic.NewTermQuery("dest_namespace", params.Namespace),
		)
		query = query.Should(sourceQuery, destQuery).MinimumNumberShouldMatch(1)
	} else if params.Prefix != "" {
		query = query.Should(
			elastic.NewPrefixQuery("source_name_aggr", params.Prefix),
			elastic.NewPrefixQuery("dest_name_aggr", params.Prefix),
		).MinimumNumberShouldMatch(1)
	} else if params.Namespace != "" {
		nestedQuery = nestedQuery.
			Should(
				elastic.NewTermQuery("source_namespace", params.Namespace),
				elastic.NewTermQuery("dest_namespace", params.Namespace),
			).
			MinimumNumberShouldMatch(1)
	}
	query = query.Filter(nestedQuery)

	return query
}

func getNamesFromElastic(params *FlowLogNamesParams, esClient lmaelastic.Client, rbacHelper rbac.FlowHelper) ([]string, error) {
	// form query
	query := buildNamesQuery(params)
	index := getClusterFlowIndex(params.ClusterName)

	aggQuery := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   query,
		Name:                    namesBucketName,
		AggCompositeSourceInfos: NamesCompositeSources,
	}

	ctx, cancel := context.WithTimeout(context.Background(), namesTimeout)
	defer cancel()

	// Perform the query with composite aggregation
	rcvdBuckets, rcvdErrors := esClient.SearchCompositeAggregations(ctx, aggQuery, nil)
	nameSet := set.New()
	names := make([]string, 0)
	for bucket := range rcvdBuckets {
		// Pull endpoint names out of the buckets
		// The index refers to the order in which the fields are listed in the composite sources
		key := bucket.CompositeAggregationKey
		source := EndpointInfo{
			Namespace: key[srcEpNamespaceIdx].String(),
			Name:      key[srcEpNameIdx].String(),
			Type:      key[srcEpTypeIdx].String(),
		}
		dest := EndpointInfo{
			Namespace: key[destEpNamespaceIdx].String(),
			Name:      key[destEpNameIdx].String(),
			Type:      key[destEpTypeIdx].String(),
		}

		// Check if the set length hits the requested limit
		if nameSet.Len() >= int(params.Limit) {
			break
		}

		// Add names to the set
		if params.Strict {
			//If we strictly enforce RBAC, then we will only return endpoints we have RBAC
			// permissions for and match the query parameters.
			if allowedName(params, source, rbacHelper) && checkEndpointRBAC(rbacHelper, source) {
				nameSet.Add(source.Name)
			}
			if allowedName(params, dest, rbacHelper) && checkEndpointRBAC(rbacHelper, dest) {
				nameSet.Add(dest.Name)
			}
		} else {
			// If we are not strictly enforcing RBAC, we will return both endpoints as long as we
			// have the RBAC permissions to view one endpoint in a flow and they match the query
			// parameters.
			if checkEndpointRBAC(rbacHelper, source) || checkEndpointRBAC(rbacHelper, dest) {
				if allowedName(params, source, rbacHelper) {
					nameSet.Add(source.Name)
				}
				if allowedName(params, dest, rbacHelper) {
					nameSet.Add(dest.Name)
				}
			}
		}
	}

	// Convert the set to the name slice
	i := 0
	nameSet.Iter(func(item interface{}) error {
		// Only add items up to the limit
		if i < int(params.Limit) {
			names = append(names, item.(string))
			i++
		}
		return nil
	})

	// Sort the names for nice display purposes
	sort.Strings(names)

	// Check for an error after all the namespaces have been processed. This should be fine
	// since an error should stop more buckets from being received.
	if err, ok := <-rcvdErrors; ok {
		log.WithError(err).Warning("Error processing the flow logs for finding valid names")
		return names, err
	}

	return names, nil
}

func allowedName(params *FlowLogNamesParams, ep EndpointInfo, rbacHelper rbac.FlowHelper) bool {
	if params.Prefix != "" && !strings.HasPrefix(ep.Name, params.Prefix) {
		return false
	}

	// If a specific namespace is specified, filter out endpoints that are not from that namespace.
	if params.Namespace != "" && params.Namespace != ep.Namespace {
		return false
	}

	return true
}

func checkEndpointRBAC(rbacHelper rbac.FlowHelper, ep EndpointInfo) bool {
	switch ep.Type {
	case flowLogEndpointTypeNs:
		// Check if this is a global networkset
		if ep.Namespace == "-" {
			allowGlobalNs, err := rbacHelper.CanListGlobalNetworkSets()
			if err != nil {
				log.WithError(err).Info("Error checking global network set list permissions")
			}
			return allowGlobalNs
		} else {
			// Check if access to networksets across all namespaces is granted.
			allowAllNs, err := rbacHelper.CanListNetworkSets("")
			if err != nil {
				log.WithError(err).Info("Error checking networkset list permissions across all namespaces")
			}

			// If access is granted across all namespaces, no need to check specific namespace permissions
			if allowAllNs {
				return allowAllNs
			}

			// Check the permissions against the specific namespace
			allowNs, err := rbacHelper.CanListNetworkSets(ep.Namespace)
			if err != nil {
				log.WithError(err).Infof("Error checking networkset list permissions for namespace %s", ep.Namespace)
			}
			return allowNs
		}
	case flowLogEndpointTypeWep:
		// Check if access to pods across all namespaces is granted
		allowAllWep, err := rbacHelper.CanListPods("")
		if err != nil {
			log.WithError(err).Info("Error checking pod list permissions across all namespaces")
		}

		// If access is granted across all namespaces, no need to check the specific namespace permissions
		if allowAllWep {
			return allowAllWep
		}

		// Check the permissions against the specific namespace
		allowWep, err := rbacHelper.CanListPods(ep.Namespace)
		if err != nil {
			log.WithError(err).Infof("Error checking pod list permissions for namespace %s", ep.Namespace)
		}
		return allowWep
	case flowLogEndpointTypeHep:
		allowHep, err := rbacHelper.CanListHostEndpoints()
		if err != nil {
			log.WithError(err).Info("Error checking host endpoint list permissions")
		}
		return allowHep
	default:
		// This is not a valid endpoint type (external network)
		return false
	}
}
