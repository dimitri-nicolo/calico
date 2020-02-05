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
)

const (
	sourceAggregationName = "source_namespaces"
	destAggregationName   = "dest_namespaces"
)

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

		response, err := getNamespacesFromElastic(params, esClient)
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

func buildAggregations(params *FlowLogNamespaceParams) (*elastic.TermsAggregation, *elastic.TermsAggregation) {
	baseAgg := elastic.NewTermsAggregation().
		Exclude("-").
		Size(int(params.Limit))
	if params.Prefix != "" {
		baseAgg = baseAgg.Include(params.Prefix)
	}

	sourceAggregation := *baseAgg.Field("source_namespace")
	destAggregation := *baseAgg.Field("dest_namespace")
	return &sourceAggregation, &destAggregation
}

func getNamespacesFromElastic(params *FlowLogNamespaceParams, esClient lmaelastic.Client) ([]Namespace, error) {
	// form query
	query := buildESQuery(params)
	sourceAggregation, destAggregation := buildAggregations(params)
	index := getClusterFlowIndex(params.ClusterName)

	// perform Aggregated ES query
	searchQuery := esClient.Backend().Search().
		Index(index).
		Query(query).
		Size(0)
	searchQuery = searchQuery.Aggregation(sourceAggregationName, sourceAggregation).Aggregation(destAggregationName, destAggregation)
	searchResult, err := esClient.Do(context.Background(), searchQuery)

	if err != nil {
		return nil, err
	}

	if searchResult.Aggregations == nil {
		return []Namespace{}, nil
	}

	sourceAggregationBuckets, sourceFound := searchResult.Aggregations.Terms(sourceAggregationName)
	destAggregationBuckets, destFound := searchResult.Aggregations.Terms(destAggregationName)

	if !sourceFound && !destFound {
		return []Namespace{}, nil
	}

	buckets := make([]*elastic.AggregationBucketKeyItem, 0)
	if sourceFound {
		buckets = append(buckets, sourceAggregationBuckets.Buckets...)
	}

	if destFound {
		buckets = append(buckets, destAggregationBuckets.Buckets...)
	}

	// extract unique namespaces from buckets
	uniqueNamespaces := make(map[string]bool)
	namespaces := make([]Namespace, 0)

	for _, bucket := range buckets {
		namespaceInf := bucket.Key
		namespace := namespaceInf.(string)
		if _, exists := uniqueNamespaces[namespace]; !exists {
			uniqueNamespaces[namespace] = true
			namespaceComponent := Namespace{Name: namespace}
			namespaces = append(namespaces, namespaceComponent)
			if len(namespaces) == int(params.Limit) {
				break
			}
		}
	}

	return namespaces, nil
}
