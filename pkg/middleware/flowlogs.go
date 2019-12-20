package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"
	pippkg "github.com/tigera/es-proxy/pkg/pip"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"net/http"
	"strings"
	"time"
)

type FlowLogsParams struct {
	ClusterName          string          `json:"cluster"`
	Limit                int32           `json:"limit"`
	SourceType           []string        `json:"srcType"`
	SourceLabels         []LabelSelector `json:"srcLabels"`
	DestType             []string        `json:"dstType"`
	DestLabels           []LabelSelector `json:"dstLabels"`
	StartDateTime        time.Time       `json:"startDateTime"`
	EndDateTime          time.Time       `json:"endDateTime"`
	Actions              []string        `json:"actions"`
	Namespace            string          `json:"namespace"`
	SourceDestNamePrefix string          `json:"srcDstNamePrefix"`
	PolicyPreview        *PolicyPreview  `json:"policyPreview"`
}

type LabelSelector struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type PolicyPreview struct {
	Verb          string                     `json:"verb"`
	NetworkPolicy libcalicoapi.NetworkPolicy `json:"networkPolicy"`
	ImpactedOnly  bool                       `json:"impactedOnly"`
}

const esflowIndexPrefix = "tigera_secure_ee_flows"

// A handler for the /flowLogs endpoint, uses url parameters to build an elasticsearch query,
// executes it and returns the results.
func FlowLogsHandler(esClient lmaelastic.Client, pip pippkg.PIP) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Validate Request
		params, err := validateFlowLogsRequest(req)
		if err != nil {
			log.WithError(err).Info("Error validating flowLogs request")
			switch err {
			case errInvalidMethod:
				http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			case errParseRequest:
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errInvalidAction:
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			case errInvalidFlowType:
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			case errInvalidLabelSelector:
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			case errInvalidPolicyPreview:
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			}
			return
		}

		response, err := getFLowLogsFromElastic(params, esClient, pip)
		if err != nil {
			log.WithError(err).Info("Error getting search results from elastic")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.WithError(err).Info("Encoding search results failed")
			http.Error(w, errGeneric.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// extracts query parameters from url and validates them
func validateFlowLogsRequest(req *http.Request) (*FlowLogsParams, error) {
	// Validate http method
	if req.Method != http.MethodGet {
		return nil, errInvalidMethod
	}

	// extract params from request
	url := req.URL.Query()
	cluster := strings.ToLower(url.Get("cluster"))
	limit, err := extractLimitParam(url)
	if err != nil {
		log.WithError(err).Info("Error extracting limit")
		return nil, errParseRequest
	}
	srcType := lowerCaseParams(url["srcType"])
	srcLabels, err := getLabelSelectors(url["srcLabels"])
	if err != nil {
		log.WithError(err).Info("Error extracting srcLabels")
		return nil, errParseRequest
	}
	dstType := lowerCaseParams(url["dstType"])
	dstLabels, err := getLabelSelectors(url["dstLabels"])
	if err != nil {
		log.WithError(err).Info("Error extracting dstLabels")
		return nil, errParseRequest
	}
	startDateTime, err := parseAndValidateTime(url.Get("startDateTime"))
	if err != nil {
		log.WithError(err).Info("Error parsing startDateTime")
		return nil, errParseRequest
	}
	endDateTime, err := parseAndValidateTime(url.Get("endDateTime"))
	if err != nil {
		log.WithError(err).Info("Error parsing endDateTime")
		return nil, errParseRequest
	}
	actions := lowerCaseParams(url["actions"])
	namespace := strings.ToLower(url.Get("namespace"))
	srcDstNamePrefix := strings.ToLower(url.Get("srcDstNamePrefix"))
	policyPreview, err := getPolicyPreview(url.Get("policyPreview"))
	if err != nil {
		log.WithError(err).Info("Error extracting policyPreview")
		return nil, errParseRequest
	}
	params := &FlowLogsParams{
		ClusterName:          cluster,
		Limit:                limit,
		SourceType:           srcType,
		SourceLabels:         srcLabels,
		DestType:             dstType,
		DestLabels:           dstLabels,
		StartDateTime:        startDateTime,
		EndDateTime:          endDateTime,
		Actions:              actions,
		Namespace:            namespace,
		SourceDestNamePrefix: srcDstNamePrefix,
		PolicyPreview:        policyPreview,
	}

	if params.ClusterName == "" {
		params.ClusterName = "cluster"
	}
	srcTypeValid := validateFlowTypes(params.SourceType)
	if !srcTypeValid {
		return nil, errInvalidFlowType
	}
	params.SourceType = convertFlowTypes(params.SourceType)
	dstTypeValid := validateFlowTypes(params.DestType)
	if !dstTypeValid {
		return nil, errInvalidFlowType
	}
	params.DestType = convertFlowTypes(params.DestType)
	srcLabelsValid := validateLabelSelector(params.SourceLabels)
	if !srcLabelsValid {
		return nil, errInvalidLabelSelector
	}
	dstLabelsValid := validateLabelSelector(params.DestLabels)
	if !dstLabelsValid {
		return nil, errInvalidLabelSelector
	}
	actionsValid := validateActions(params.Actions)
	if !actionsValid {
		return nil, errInvalidAction
	}
	if params.PolicyPreview != nil {
		policyPreviewValid := validatePolicyPreview(*policyPreview)
		if !policyPreviewValid {
			return nil, errInvalidPolicyPreview
		}
	}

	return params, nil
}

// applies appropriate filters to an elastic.BoolQuery
func buildFlowLogsQuery(params *FlowLogsParams) *elastic.BoolQuery {
	query := elastic.NewBoolQuery()
	var filters []elastic.Query
	if len(params.Actions) > 0 {
		actionsFilter := buildTermsFilter(params.Actions, "action")
		filters = append(filters, actionsFilter)
	}
	if len(params.SourceType) > 0 {
		sourceTypeFilter := buildTermsFilter(params.SourceType, "source_type")
		filters = append(filters, sourceTypeFilter)
	}
	if len(params.DestType) > 0 {
		destTypeFilter := buildTermsFilter(params.DestType, "dest_type")
		filters = append(filters, destTypeFilter)
	}
	if len(params.SourceLabels) > 0 {
		sourceLabelsFilter := buildLabelSelectorFilter(params.SourceLabels, "source_labels",
			"source_labels.labels")
		filters = append(filters, sourceLabelsFilter)
	}
	if len(params.DestLabels) > 0 {
		destLabelsFilter := buildLabelSelectorFilter(params.DestLabels, "dest_labels", "dest_labels.labels")
		filters = append(filters, destLabelsFilter)
	}
	if !params.StartDateTime.IsZero() {
		startFilter := elastic.NewRangeQuery("start_time").Gt(params.StartDateTime.Unix())
		filters = append(filters, startFilter)
	}
	if !params.EndDateTime.IsZero() {
		endFilter := elastic.NewRangeQuery("end_time").Lt(params.EndDateTime.Unix())
		filters = append(filters, endFilter)
	}
	if params.Namespace != "" {
		namespaceFilter := elastic.NewBoolQuery().
			Should(
				elastic.NewTermQuery("source_namespace", params.Namespace),
				elastic.NewTermQuery("dest_namespace", params.Namespace),
			).
			MinimumNumberShouldMatch(1)
		filters = append(filters, namespaceFilter)
	}
	if params.SourceDestNamePrefix != "" {
		namePrefixFilter := elastic.NewBoolQuery().
			Should(
				elastic.NewPrefixQuery("source_name_aggr", params.SourceDestNamePrefix),
				elastic.NewPrefixQuery("dest_name_aggr", params.SourceDestNamePrefix),
			).
			MinimumNumberShouldMatch(1)
		filters = append(filters, namePrefixFilter)
	}
	query = query.Filter(filters...)

	return query
}

func buildTermsFilter(terms []string, termsKey string) *elastic.TermsQuery {
	var termValues []interface{}
	for _, term := range terms {
		termValues = append(termValues, term)
	}
	return elastic.NewTermsQuery(termsKey, termValues...)
}

// Builds a nested filter for LabelSelectors
// The LabelSelector allows a user describe matching the labels in elasticsearch. If multiple values are specified,
// a "terms" query will be created with as follows:
// "terms": {
// "<labelType>.labels": [
//  "<key><operator><value1>",
//  "<key><operator><value2>",
//  ]
// }
// If only one value is specified a "term" query is created as follows:
// "term": {
//  "<labelType>.labels": <key><operator><value>"
// }
func buildLabelSelectorFilter(labelSelectors []LabelSelector, path string, termsKey string) *elastic.NestedQuery {
	var labelValues []interface{}
	var selectorQueries []elastic.Query
	for _, selector := range labelSelectors {
		keyAndOperator := fmt.Sprintf("%s%s", selector.Key, selector.Operator)
		if len(selector.Values) == 1 {
			selectorQuery := elastic.NewTermQuery(termsKey, fmt.Sprintf("%s%s", keyAndOperator, selector.Values[0]))
			selectorQueries = append(selectorQueries, selectorQuery)
		} else {
			for _, value := range selector.Values {
				labelValues = append(labelValues, fmt.Sprintf("%s%s", keyAndOperator, value))
			}
			selectorQuery := elastic.NewTermsQuery(termsKey, labelValues...)
			selectorQueries = append(selectorQueries, selectorQuery)
		}
	}
	return elastic.NewNestedQuery(path, elastic.NewBoolQuery().Filter(selectorQueries...))
}

// if a policy preview is provided use pip to perform the ES query and return flows with the policy applied
// otherwise just perform a regular ES query and return the results
func getFLowLogsFromElastic(params *FlowLogsParams, esClient lmaelastic.Client, pip pippkg.PIP) (interface{}, error) {
	query := buildFlowLogsQuery(params)
	index := getClusterFlowIndex(params.ClusterName)

	if params.PolicyPreview == nil {
		searchQuery := esClient.Backend().Search().
			Index(index).
			Query(query).
			Size(int(params.Limit))
		searchResult, err := esClient.Do(context.Background(), searchQuery)
		if err != nil {
			return nil, err
		}
		return searchResult, nil
	} else {
		policyChange := pippkg.ResourceChange{
			Action:   params.PolicyPreview.Verb,
			Resource: &params.PolicyPreview.NetworkPolicy,
		}

		pipParams := &pippkg.PolicyImpactParams{
			Query:           query,
			DocumentIndex:   index,
			ResourceActions: []pippkg.ResourceChange{policyChange},
			Limit:           params.Limit,
			ImpactedOnly:    params.PolicyPreview.ImpactedOnly,
		}

		pipResults, err := pip.GetFlows(context.TODO(), pipParams)
		if err != nil {
			return nil, err
		}
		return pipResults, nil
	}
}

func getClusterFlowIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esflowIndexPrefix, cluster)
}
