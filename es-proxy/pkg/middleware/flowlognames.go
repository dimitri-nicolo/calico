package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	k8srequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/libcalico-go/lib/set"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	elasticvariant "github.com/projectcalico/calico/es-proxy/pkg/elastic"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/rbac"
	"github.com/projectcalico/calico/lma/pkg/timeutils"
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
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "source_name_aggr", Field: "source_name_aggr"},
		{Name: "source_type", Field: "source_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name_aggr", Field: "dest_name_aggr"},
		{Name: "dest_type", Field: "dest_type"},
	}
)

type FlowLogNamesParams struct {
	Limit         int32           `json:"limit"`
	Actions       []string        `json:"actions"`
	ClusterName   string          `json:"cluster"`
	Namespace     string          `json:"namespace"`
	Prefix        string          `json:"prefix"`
	Unprotected   bool            `json:"unprotected"`
	StartDateTime string          `json:"startDateTime"`
	EndDateTime   string          `json:"endDateTime"`
	Strict        bool            `json:"bool"`
	SourceType    []string        `json:"srcType"`
	SourceLabels  []LabelSelector `json:"srcLabels"`
	DestType      []string        `json:"dstType"`
	DestLabels    []LabelSelector `json:"dstLabels"`

	// Parsed timestamps
	startDateTimeESParm interface{}
	endDateTimeESParm   interface{}
}

type EndpointInfo struct {
	Namespace string
	Name      string
	Type      string
}

func FlowLogNamesHandler(k8sClientFactory datastore.ClusterCtxK8sClientFactory, esClient lmaelastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// validate request
		params, err := validateFlowLogNamesRequest(req)
		if err != nil {
			log.WithError(err).Info("Error validating request")
			switch err {
			case ErrInvalidMethod:
				http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			case ErrParseRequest:
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errInvalidAction:
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			}
			return
		}

		k8sCli, err := k8sClientFactory.ClientSetForCluster(params.ClusterName)
		if err != nil {
			log.WithError(err).Error("failed to get k8s cli")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		user, ok := k8srequest.UserFrom(req.Context())
		if !ok {
			log.WithError(err).Error("user not found in context")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		flowHelper := rbac.NewCachedFlowHelper(user, lmaauth.NewRBACAuthorizer(k8sCli))

		response, err := getNamesFromElastic(params, esClient, flowHelper)
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
		return nil, ErrInvalidMethod
	}

	// extract params from request
	url := req.URL.Query()
	limit, err := extractLimitParam(url)
	if err != nil {
		return nil, ErrParseRequest
	}
	actions := lowerCaseParams(url["actions"])
	cluster := strings.ToLower(url.Get("cluster"))
	prefix := strings.ToLower(url.Get("prefix"))
	namespace := strings.ToLower(url.Get("namespace"))
	unprotected := false
	if unprotectedValue := url.Get("unprotected"); unprotectedValue != "" {
		if unprotected, err = strconv.ParseBool(unprotectedValue); err != nil {
			return nil, ErrParseRequest
		}
	}
	startDateTimeString := url.Get("startDateTime")
	endDateTimeString := url.Get("endDateTime")

	// Parse the start/end time to validate the format. We don't need the resulting time struct.
	now := time.Now()
	_, startDateTimeESParm, err := timeutils.ParseTime(now, &startDateTimeString)
	if err != nil {
		log.WithError(err).Info("Error extracting start date time")
		return nil, ErrParseRequest
	}
	_, endDateTimeESParm, err := timeutils.ParseTime(now, &endDateTimeString)
	if err != nil {
		log.WithError(err).Info("Error extracting end date time")
		return nil, ErrParseRequest
	}
	strict := false
	if strictValue := url.Get("strict"); strictValue != "" {
		if strict, err = strconv.ParseBool(strictValue); err != nil {
			return nil, ErrParseRequest
		}
	}

	srcType := lowerCaseParams(url["srcType"])
	srcLabels, err := getLabelSelectors(url["srcLabels"])
	if err != nil {
		log.WithError(err).Info("Error extracting srcLabels")
		return nil, ErrParseRequest
	}
	dstType := lowerCaseParams(url["dstType"])
	dstLabels, err := getLabelSelectors(url["dstLabels"])
	if err != nil {
		log.WithError(err).Info("Error extracting dstLabels")
		return nil, ErrParseRequest
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
		SourceType:          srcType,
		SourceLabels:        srcLabels,
		DestType:            dstType,
		DestLabels:          dstLabels,
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

	srcTypeValid := validateFlowTypes(params.SourceType)
	if !srcTypeValid {
		return nil, errInvalidFlowType
	}
	dstTypeValid := validateFlowTypes(params.DestType)
	if !dstTypeValid {
		return nil, errInvalidFlowType
	}

	srcLabelsValid := validateLabelSelector(params.SourceLabels)
	if !srcLabelsValid {
		return nil, errInvalidLabelSelector
	}
	dstLabelsValid := validateLabelSelector(params.DestLabels)
	if !dstLabelsValid {
		return nil, errInvalidLabelSelector
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

	// Collect all the different filtering queries based on the specified parameters.
	sourceConditions := []elastic.Query{}
	destConditions := []elastic.Query{}
	if params.Prefix != "" {
		sourceConditions = append(sourceConditions, elastic.NewPrefixQuery("source_name_aggr", params.Prefix))
		destConditions = append(destConditions, elastic.NewPrefixQuery("dest_name_aggr", params.Prefix))
	}
	if params.Namespace != "" {
		sourceConditions = append(sourceConditions, elastic.NewTermQuery("source_namespace", params.Namespace))
		destConditions = append(destConditions, elastic.NewTermQuery("dest_namespace", params.Namespace))
	}
	if len(params.SourceType) > 0 {
		sourceConditions = append(sourceConditions, buildTermsFilter(params.SourceType, "source_type"))
	}
	if len(params.SourceLabels) > 0 {
		sourceConditions = append(sourceConditions, buildLabelSelectorFilter(params.SourceLabels, "source_labels", "source_labels.labels"))
	}
	if len(params.DestType) > 0 {
		destConditions = append(destConditions, buildTermsFilter(params.DestType, "dest_type"))
	}
	if len(params.DestLabels) > 0 {
		destConditions = append(destConditions, buildLabelSelectorFilter(params.DestLabels, "dest_labels", "dest_labels.labels"))
	}

	// Use the filtering queries to craft the appropriate source and destination filtering queries
	var sourceQuery, destQuery *elastic.BoolQuery
	if len(sourceConditions) > 0 {
		sourceQuery = elastic.NewBoolQuery()
		for _, query := range sourceConditions {
			sourceQuery = sourceQuery.Must(query)
		}
	}
	if len(destConditions) > 0 {
		destQuery = elastic.NewBoolQuery()
		for _, query := range destConditions {
			destQuery = destQuery.Must(query)
		}
	}

	// Add the source and destination filtering queries to the ES query.
	if sourceQuery != nil && destQuery != nil {
		query = query.Should(sourceQuery, destQuery).MinimumNumberShouldMatch(1)
	} else if sourceQuery != nil {
		query = query.Should(sourceQuery)
	} else if destQuery != nil {
		query = query.Should(destQuery)
	}

	query = query.Filter(nestedQuery)

	return query
}

func getNamesFromElastic(params *FlowLogNamesParams, esClient lmaelastic.Client, rbacHelper rbac.FlowHelper) ([]string, error) {
	// form query
	query := buildNamesQuery(params)
	index := lmaindex.FlowLogs().GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))

	aggQuery := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   query,
		Name:                    namesBucketName,
		AggCompositeSourceInfos: NamesCompositeSources,
	}

	ctx, cancel := context.WithTimeout(context.Background(), namesTimeout)
	defer cancel()

	// Perform the query with composite aggregation
	// TODO: Since composite aggregation is expensive and we only care about
	// correlating fields together in order to do RBAC checks, we should instead
	// precalculate RBAC permissions and use those as filters on the query itself.
	// In this case, there will be four possibilities for return values, source filtered
	// on source, destination filtered on source, source filtered on destination, and
	// destination filtered on destination. We will need to run four separate queries
	// with the precalculated filters to grab these four possible return values and
	// then aggregate the set together in order to get our return values. This should
	// be cheaper since the four queries themselves should be much cheaper to run than
	// the single composite aggregation query.
	rcvdBuckets, rcvdErrors := esClient.SearchCompositeAggregations(ctx, aggQuery, nil)
	nameSet := set.New[string]()
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
	nameSet.Iter(func(item string) error {
		// Only add items up to the limit
		if i < int(params.Limit) {
			names = append(names, item)
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
