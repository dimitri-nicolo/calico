package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8srequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/compliance/pkg/datastore"
	elasticvariant "github.com/tigera/es-proxy/pkg/elastic"
	pippkg "github.com/tigera/es-proxy/pkg/pip"
	lmaauth "github.com/tigera/lma/pkg/auth"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	lmaindex "github.com/tigera/lma/pkg/elastic/index"
	"github.com/tigera/lma/pkg/rbac"
	"github.com/tigera/lma/pkg/timeutils"
)

type FlowLogsParams struct {
	ClusterName          string          `json:"cluster"`
	Limit                int32           `json:"limit"`
	SourceType           []string        `json:"srcType"`
	SourceLabels         []LabelSelector `json:"srcLabels"`
	DestType             []string        `json:"dstType"`
	DestLabels           []LabelSelector `json:"dstLabels"`
	StartDateTime        string          `json:"startDateTime"`
	EndDateTime          string          `json:"endDateTime"`
	Actions              []string        `json:"actions"`
	Namespace            string          `json:"namespace"`
	SourceDestNamePrefix string          `json:"srcDstNamePrefix"`
	PolicyPreviews       []PolicyPreview `json:"policyPreviews"`
	Unprotected          bool            `json:"unprotected"`
	ImpactedOnly         bool            `json:"impactedOnly"`

	// Parsed timestamps
	startDateTime       *time.Time
	endDateTime         *time.Time
	startDateTimeESParm interface{}
	endDateTimeESParm   interface{}
}

type LabelSelector struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type PolicyPreview struct {
	Verb          string             `json:"verb"`
	NetworkPolicy resources.Resource `json:"networkPolicy"`
}

// policyPreviewTrial is used to temporarily unmarshal the PolicyPreviews so that we can extract the TypeMeta from
// the resource definition.
type policyPreviewTrial struct {
	NetworkPolicy metav1.TypeMeta `json:"networkPolicy"`
}

// Defined an alias for the ResourceChange so that we can json unmarshal it from the PolicyPreviews.UnmarshalJSON
// without causing recursion (since aliased types do not inherit methods).
type AliasedPolicyPreview *PolicyPreview

// UnmarshalJSON allows unmarshalling of a PolicyPreviews from JSON bytes. This is required because the Resource
// field is an interface, and so it needs to be set with a concrete type before it can be unmarshalled.
func (c *PolicyPreview) UnmarshalJSON(b []byte) error {
	// Unmarshal into the "trial" struct that allows us to easily extract the TypeMeta of the resource.
	var r policyPreviewTrial
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	c.NetworkPolicy = resources.NewResource(r.NetworkPolicy)

	// Decode the policy preview JSON data. We should fail if there are unhandled fields in the request. Validation of
	// the actual data is done within PIP as part of the xrefcache population.
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(AliasedPolicyPreview(c)); err != nil {
		return err
	}
	if decoder.More() {
		return errPreviewResourceExtraData
	}

	// If this is a Calico tiered network policy, configure an empty tier to be default and verify the name matches
	// the tier.
	var tier *string
	var name string
	switch np := c.NetworkPolicy.(type) {
	case *v3.NetworkPolicy:
		tier = &np.Spec.Tier
		name = np.Name
	case *v3.GlobalNetworkPolicy:
		tier = &np.Spec.Tier
		name = np.Name
	default:
		// Not a calico tiered policy, so just exit now, no need to do the extra checks.
		return nil
	}

	// Calico tiered policy. The tier in the spec should also be the prefix of the policy name.
	if *tier == "" {
		// The tier is not set, so set it to be default.
		*tier = "default"
	}
	if !strings.HasPrefix(name, *tier+".") {
		return errors.New("policy name '" + name + "' is not correct for the configured tier '" + *tier + "'")
	}
	return nil
}

// A handler for the /flowLogs endpoint, uses url parameters to build an elasticsearch query,
// executes it and returns the results.
func FlowLogsHandler(k8sClientFactory datastore.ClusterCtxK8sClientFactory, esClient lmaelastic.Client, pip pippkg.PIP) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Validate Request
		params, err := validateFlowLogsRequest(req)
		if err != nil {
			log.WithError(err).Info("Error validating flowLogs request")
			switch err {
			case ErrInvalidMethod:
				http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			case ErrParseRequest:
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

		k8sCli, err := k8sClientFactory.ClientSetForCluster(params.ClusterName)
		if err != nil {
			log.WithError(err).Error("failed to get k8s cli")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		user, ok := k8srequest.UserFrom(req.Context())
		if !ok {
			log.Error("user not found in context")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		flowHelper := rbac.NewCachedFlowHelper(user, lmaauth.NewRBACAuthorizer(k8sCli))
		flowFilter := lmaelastic.NewFlowFilterUserRBAC(flowHelper)

		var response interface{}
		var stat int
		if len(params.PolicyPreviews) == 0 {
			response, stat, err = getFlowLogsFromElastic(flowFilter, params, esClient)
		} else {
			rbacHelper := NewPolicyImpactRbacHelper(user, lmaauth.NewRBACAuthorizer(k8sCli))
			response, stat, err = getPIPFlowLogsFromElastic(flowFilter, params, pip, rbacHelper)
		}

		if err != nil {
			log.WithError(err).Info("Error getting search results from elastic")
			http.Error(w, err.Error(), stat)
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
		return nil, ErrInvalidMethod
	}

	// extract params from request
	url := req.URL.Query()
	cluster := strings.ToLower(url.Get("cluster"))
	limit, err := extractLimitParam(url)
	if err != nil {
		log.WithError(err).Info("Error extracting limit")
		return nil, ErrParseRequest
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
	startDateTimeString := url.Get("startDateTime")
	endDateTimeString := url.Get("endDateTime")
	actions := lowerCaseParams(url["actions"])
	namespace := strings.ToLower(url.Get("namespace"))
	srcDstNamePrefix := strings.ToLower(url.Get("srcDstNamePrefix"))
	policyPreviews, err := getPolicyPreviews(url["policyPreview"])
	if err != nil {
		log.WithError(err).Info("Error extracting policyPreview")
		return nil, ErrParseRequest
	}
	unprotected := false
	if unprotectedValue := url.Get("unprotected"); unprotectedValue != "" {
		if unprotected, err = strconv.ParseBool(unprotectedValue); err != nil {
			return nil, ErrParseRequest
		}
	}
	impactedOnly := false
	if impactedOnlyValue := url.Get("impactedOnly"); impactedOnlyValue != "" {
		if impactedOnly, err = strconv.ParseBool(impactedOnlyValue); err != nil {
			return nil, ErrParseRequest
		}
	}

	now := time.Now()
	startDateTime, startDateTimeESParm, err := timeutils.ParseTime(now, &startDateTimeString)
	if err != nil {
		log.WithError(err).Info("Error extracting start date time")
		return nil, ErrParseRequest
	}
	endDateTime, endDateTimeESParm, err := timeutils.ParseTime(now, &endDateTimeString)
	if err != nil {
		log.WithError(err).Info("Error extracting end date time")
		return nil, ErrParseRequest
	}

	params := &FlowLogsParams{
		ClusterName:          cluster,
		Limit:                limit,
		SourceType:           srcType,
		SourceLabels:         srcLabels,
		DestType:             dstType,
		DestLabels:           dstLabels,
		StartDateTime:        startDateTimeString,
		EndDateTime:          endDateTimeString,
		Actions:              actions,
		Namespace:            namespace,
		SourceDestNamePrefix: srcDstNamePrefix,
		PolicyPreviews:       policyPreviews,
		ImpactedOnly:         impactedOnly,
		Unprotected:          unprotected,
		startDateTime:        startDateTime,
		endDateTime:          endDateTime,
		startDateTimeESParm:  startDateTimeESParm,
		endDateTimeESParm:    endDateTimeESParm,
	}

	if params.ClusterName == "" {
		params.ClusterName = "cluster"
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
	actionsValid := validateActions(params.Actions)
	if !actionsValid {
		return nil, errInvalidAction
	}
	valid := validateActionsAndUnprotected(params.Actions, params.Unprotected)
	if !valid {
		return nil, errInvalidActionUnprotected
	}
	policyPreviewValid := validatePolicyPreviews(params.PolicyPreviews)
	if !policyPreviewValid {
		return nil, errInvalidPolicyPreview
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
	if params.startDateTimeESParm != nil || params.endDateTimeESParm != nil {
		filter := elastic.NewRangeQuery("end_time")
		if params.startDateTimeESParm != nil {
			filter = filter.Gte(params.startDateTimeESParm)
		}
		if params.endDateTimeESParm != nil {
			filter = filter.Lt(params.endDateTimeESParm)
		}
		filters = append(filters, filter)
	}

	if params.Unprotected {
		filters = append(filters, UnprotectedQuery())
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

// IMPORTANT: This function does not create the correct Elasticsearch query from the label selector and needs to be redone.
// It tries to find docs that have labels name "<key>.<operator>.<value>" which could be "name!=foobar" or "namecontainsfoobar".
// The only successful queries generated by this function are those with an operator of "=".
//
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

// This method will take a look at the request parameters made to the /flowLogs endpoint and return the results from elastic.
func getFlowLogsFromElastic(flowFilter lmaelastic.FlowFilter, params *FlowLogsParams, esClient lmaelastic.Client) (interface{}, int, error) {
	query := buildFlowLogsQuery(params)
	index := lmaindex.FlowLogs().GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))
	result, err := lmaelastic.GetCompositeAggrFlows(
		context.TODO(), 60*time.Second, esClient, query, index, flowFilter, params.Limit)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return result, http.StatusOK, nil
}

func getPIPParams(params *FlowLogsParams) *pippkg.PolicyImpactParams {
	query := buildFlowLogsQuery(params)
	index := lmaindex.FlowLogs().GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))

	// Convert the input format to the PIP format.
	// TODO(rlb): We don't need both formats, and the PIP format has the more generically named "Resource" field rather
	//            than limiting to network policies.
	resourceChanges := make([]pippkg.ResourceChange, len(params.PolicyPreviews))
	for i, pp := range params.PolicyPreviews {
		resourceChanges[i] = pippkg.ResourceChange{
			Action:   pp.Verb,
			Resource: pp.NetworkPolicy,
		}
	}

	return &pippkg.PolicyImpactParams{
		Query:           query,
		DocumentIndex:   index,
		ClusterName:     params.ClusterName,
		ResourceActions: resourceChanges,
		Limit:           params.Limit,
		ImpactedOnly:    params.ImpactedOnly,
		FromTime:        params.startDateTime,
		ToTime:          params.endDateTime,
	}
}

// This method will take a look at the request parameters made to the /flowLogs endpoint with pip settings,
// verify RBAC based on the previewed settings and return the results from elastic.
func getPIPFlowLogsFromElastic(flowFilter lmaelastic.FlowFilter, params *FlowLogsParams, pip pippkg.PIP, rbacHelper PolicyImpactRbacHelper) (interface{}, int, error) {

	// Check a NP is supplied in every request.
	if len(params.PolicyPreviews) == 0 {
		// Expect the policy preview to contain a network policy.
		return nil, http.StatusBadRequest, errors.New("no network policy specified in preview request")
	}
	for i := range params.PolicyPreviews {
		if params.PolicyPreviews[i].NetworkPolicy == nil {
			return nil, http.StatusBadRequest, errors.New("no network policy specified in preview request")
		}
	}

	// This is a PIP request. Extract the PIP parameters.
	pipParams := getPIPParams(params)

	// Check for RBAC
	for _, action := range pipParams.ResourceActions {
		if action.Resource == nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid resource actions syntax: resource is missing from request")
		}
		if err := validateAction(action.Action); err != nil {
			return nil, http.StatusBadRequest, err
		}
		if authorized, err := rbacHelper.CheckCanPreviewPolicyAction(action.Action, action.Resource); err != nil {
			return nil, http.StatusInternalServerError, err
		} else if !authorized {
			return nil, http.StatusForbidden, fmt.Errorf("Forbidden")
		}
	}

	// Fetch results from Elasticsearch
	response, err := pip.GetFlows(context.TODO(), pipParams, flowFilter)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return response, http.StatusOK, nil
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
