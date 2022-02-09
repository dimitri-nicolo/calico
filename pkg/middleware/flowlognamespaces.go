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

	"github.com/tigera/compliance/pkg/datastore"
	elasticvariant "github.com/tigera/es-proxy/pkg/elastic"
	lmaauth "github.com/tigera/lma/pkg/auth"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	lmaindex "github.com/tigera/lma/pkg/elastic/index"
	"github.com/tigera/lma/pkg/rbac"
	"github.com/tigera/lma/pkg/timeutils"
)

const (
	namespaceBucketName = "source_dest_namespaces"
	srcNamespaceIdx     = 0
	destNamespaceIdx    = 1
)

var (
	namespaceTimeout = 10 * time.Second

	NamespaceCompositeSources = []lmaelastic.AggCompositeSourceInfo{
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "dest_namespace", Field: "dest_namespace"},
	}
)

type FlowLogNamespaceParams struct {
	Limit         int32    `json:"limit"`
	Actions       []string `json:"actions"`
	ClusterName   string   `json:"cluster"`
	Prefix        string   `json:"prefix"`
	Unprotected   bool     `json:"unprotected"`
	StartDateTime string   `json:"startDateTime"`
	EndDateTime   string   `json:"endDateTime"`
	Strict        bool     `json:"strict"`

	// Parsed timestamps
	startDateTimeESParm interface{}
	endDateTimeESParm   interface{}
}

type Namespace struct {
	Name string `json:"name"`
}

func FlowLogNamespaceHandler(k8sClientFactory datastore.ClusterCtxK8sClientFactory, esClient lmaelastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// validate request
		params, err := validateFlowLogNamespacesRequest(req)
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

		response, err := getNamespacesFromElastic(params, esClient, flowHelper)
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
	})
}

func validateFlowLogNamespacesRequest(req *http.Request) (*FlowLogNamespaceParams, error) {
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

	if params.Prefix != "" {
		query = query.Should(
			elastic.NewPrefixQuery("source_namespace", params.Prefix),
			elastic.NewPrefixQuery("dest_namespace", params.Prefix),
		).MinimumNumberShouldMatch(1)
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

func getNamespacesFromElastic(params *FlowLogNamespaceParams, esClient lmaelastic.Client, rbacHelper rbac.FlowHelper) ([]Namespace, error) {
	// form query
	query := buildESQuery(params)
	index := lmaindex.FlowLogs().GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))

	aggQuery := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   query,
		Name:                    namespaceBucketName,
		AggCompositeSourceInfos: NamespaceCompositeSources,
	}

	ctx, cancel := context.WithTimeout(context.Background(), namespaceTimeout)
	defer cancel()

	// Perform the query with composite aggregation
	rcvdBuckets, rcvdErrors := esClient.SearchCompositeAggregations(ctx, aggQuery, nil)
	nsSet := set.New()
	namespaces := make([]Namespace, 0)
	for bucket := range rcvdBuckets {
		// Pull namespaces out of the buckets
		// The index refers to the order in which the fields are listed in the composite sources
		key := bucket.CompositeAggregationKey
		source_namespace := key[srcNamespaceIdx].String()
		dest_namespace := key[destNamespaceIdx].String()

		// Check if the set length hits the requested limit
		if nsSet.Len() >= int(params.Limit) {
			break
		}

		// Add namespaces to the set
		if params.Strict {
			// If we strictly enforce RBAC, then we will only return namespaces we have RBAC
			// permissions for and match the query parameters.
			if allowedNamespace(params, source_namespace, rbacHelper) && checkNamespaceRBAC(rbacHelper, source_namespace) {
				nsSet.Add(source_namespace)
			}
			if allowedNamespace(params, dest_namespace, rbacHelper) && checkNamespaceRBAC(rbacHelper, dest_namespace) {
				nsSet.Add(dest_namespace)
			}
		} else {
			// If we are not strictly enforcing RBAC, we will return both namespaces as long we
			// have the permissions to view one namespace in the flow and they match the query
			// parameters.
			if checkNamespaceRBAC(rbacHelper, source_namespace) || checkNamespaceRBAC(rbacHelper, dest_namespace) {
				if allowedNamespace(params, source_namespace, rbacHelper) {
					nsSet.Add(source_namespace)
				}
				if allowedNamespace(params, dest_namespace, rbacHelper) {
					nsSet.Add(dest_namespace)
				}
			}
		}
	}

	// Convert the set to the namespace slice
	i := 0
	nsSet.Iter(func(item interface{}) error {
		// Only add items up to the limit
		if i < int(params.Limit) {
			namespaces = append(namespaces, Namespace{Name: item.(string)})
			i++
		}
		return nil
	})

	// Sort the namespaces for nice display purposes
	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})

	// Check for an error after all the namespaces have been processed. This should be fine
	// since an error should stop more buckets from being received.
	if err, ok := <-rcvdErrors; ok {
		log.WithError(err).Warning("Error processing the flow logs for finding valid namespaces")
		return namespaces, err
	}

	return namespaces, nil
}

func allowedNamespace(params *FlowLogNamespaceParams, namespace string, rbacHelper rbac.FlowHelper) bool {
	if params.Prefix != "" && !strings.HasPrefix(namespace, params.Prefix) {
		return false
	}

	return true
}

func checkNamespaceRBAC(rbacHelper rbac.FlowHelper, namespace string) bool {
	var allowed bool
	var err error

	if namespace == "-" {
		// Check the global namespace permissions
		allowed, err = rbacHelper.IncludeGlobalNamespace()
		if err != nil {
			log.WithError(err).Info("Error checking RBAC permissions for the cluster scope")
		}
	} else {
		// Check if the user has access to all namespaces first
		if allowed, err = rbacHelper.IncludeNamespace(""); err != nil {
			log.WithError(err).Info("Error checking namespace RBAC permissions for all namespaces")
			return false
		} else if allowed {
			return true
		}

		// Check the namespace permissions
		allowed, err = rbacHelper.IncludeNamespace(namespace)
		if err != nil {
			log.WithError(err).Info("Error checking namespace RBAC permissions")
		}
	}
	return allowed
}
