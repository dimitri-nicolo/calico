package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	k8srequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	elasticvariant "github.com/projectcalico/calico/es-proxy/pkg/elastic"

	"github.com/projectcalico/calico/lma/pkg/api"
	celastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/rbac"
	"github.com/projectcalico/calico/lma/pkg/timeutils"
)

const (
	HttpErrUnauthorizedFlowAccess = "User is not authorised to view this flow."
)

// flowRequestParams is the representation of the parameters sent to the "flow" endpoint. An http.Request object should
// be validated and parsed with the parseAndValidateFlowRequest function.
//
// Note that if srcType or dstType are global endpoint types, HEP or global NS, the the srcNamespace and / or dstNamespace
// must be "-". srcNamespace and dstNamespace are always required.
type flowRequestParams struct {
	// Required parameters used to uniquely define a "flow".
	clusterName  string
	srcType      api.EndpointType
	srcNamespace string
	srcName      string
	dstType      api.EndpointType
	dstNamespace string
	dstName      string

	// Optional parameters used to filter flow logs evaluated for a "flow".
	srcLabels []LabelSelector
	dstLabels []LabelSelector

	// The format is either RFC3339 or a relative time like now-15m, now-1h.
	startDateTime *time.Time
	endDateTime   *time.Time
}

func (req flowRequestParams) clusterIndex() string {
	return lmaindex.FlowLogs().GetIndex(elasticvariant.AddIndexInfix(req.clusterName))
}

// parseAndValidateFlowRequest parses the fields in the request query, validating that required parameters are set and or the
// correct format, then setting them to the appropriate values in a flowRequest.
//
// Any error returned is of a format and contains information that can be returned in the response body. Any errors that
// are not to be returned in the response are logged as an error.
func parseAndValidateFlowRequest(req *http.Request) (*flowRequestParams, error) {
	var err error
	query := req.URL.Query()

	requiredParams := []string{"srcType", "srcName", "dstType", "dstName", "srcNamespace", "dstNamespace"}
	for _, param := range requiredParams {
		if query.Get(param) == "" {
			return nil, fmt.Errorf("missing required parameter '%s'", param)
		}
	}

	flowParams := &flowRequestParams{
		clusterName:  strings.ToLower(query.Get("cluster")),
		srcType:      api.StringToEndpointType(strings.ToLower(query.Get("srcType"))),
		srcNamespace: strings.ToLower(query.Get("srcNamespace")),
		srcName:      strings.ToLower(query.Get("srcName")),
		dstType:      api.StringToEndpointType(strings.ToLower(query.Get("dstType"))),
		dstNamespace: strings.ToLower(query.Get("dstNamespace")),
		dstName:      strings.ToLower(query.Get("dstName")),
	}

	if flowParams.clusterName == "" {
		flowParams.clusterName = datastore.DefaultCluster
	}

	if flowParams.srcType == api.EndpointTypeInvalid {
		return nil, fmt.Errorf("srcType value '%s' is not a valid endpoint type", flowParams.srcType)
	} else if flowParams.dstType == api.EndpointTypeInvalid {
		return nil, fmt.Errorf("dstType value '%s' is not a valid endpoint type", flowParams.dstType)
	}

	if dateTimeStr := query.Get("startDateTime"); len(dateTimeStr) > 0 {
		flowParams.startDateTime, _, err = timeutils.ParseTime(time.Now(), &dateTimeStr)
		if err != nil {
			errMsg := fmt.Sprintf("failed to parse 'startDateTime' value '%s' as RFC3339 datetime or relative time", dateTimeStr)
			return nil, fmt.Errorf(errMsg)
		}
	}

	if dateTimeStr := query.Get("endDateTime"); len(dateTimeStr) > 0 {
		flowParams.endDateTime, _, err = timeutils.ParseTime(time.Now(), &dateTimeStr)
		if err != nil {
			errMsg := fmt.Sprintf("failed to parse 'endDateTime' value '%s' as RFC3339 datetime or relative time", dateTimeStr)
			return nil, fmt.Errorf(errMsg)
		}
	}

	if labels, exists := query["srcLabels"]; exists {
		srcLabels, err := getLabelSelectors(labels)
		if err != nil {
			return nil, err
		}

		flowParams.srcLabels = srcLabels
	}

	if labels, exists := query["dstLabels"]; exists {
		dstLabels, err := getLabelSelectors(labels)
		if err != nil {
			return nil, err
		}

		flowParams.dstLabels = dstLabels
	}

	return flowParams, nil
}

type PolicyReport struct {
	// AllowedFlowPolicies contains the policies from the allowed flow. Policies that the user is not authorized to view
	// are obfuscated.
	AllowedFlowPolicies []*FlowResponsePolicy `json:"allowedFlowPolicies"`

	// DeniedFlowPolicies contains the policies from the denied flow. Policies that the user is not authorized to view
	// are obfuscated.
	DeniedFlowPolicies []*FlowResponsePolicy `json:"deniedFlowPolicies"`
}

// FlowResponse is the response that will be returned json marshaled and written in the flowHandler's ServeHTTP.
type FlowResponse struct {
	// Count is the total number of documents that were included in the flow log.
	Count int64 `json:"count"`

	// DstLabels contains all the labels the flows destination had, if applicable, in the given time frame for the flow query.
	DstLabels FlowResponseLabels `json:"dstLabels"`

	// SrcLabels contains all the labels the flow's source had, if applicable, in the given time frame for the flow query.
	SrcLabels FlowResponseLabels `json:"srcLabels"`

	// SrcPolicyReport contains the policies that were applied and reported by the source of the flow.
	SrcPolicyReport *PolicyReport `json:"srcPolicyReport"`

	// DstPolicyReport contains the policies that were applied and reported by the destination of the flow.
	DstPolicyReport *PolicyReport `json:"dstPolicyReport"`
}

type FlowResponseLabels map[string][]FlowResponseLabelValue

type FlowResponseLabelValue struct {
	Count int64  `json:"count"`
	Value string `json:"value"`
}

type FlowResponsePolicy struct {
	Index        int    `json:"index"`
	Tier         string `json:"tier"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Action       string `json:"action"`
	IsStaged     bool   `json:"isStaged"`
	IsKubernetes bool   `json:"isKubernetes"`
	IsProfile    bool   `json:"isProfile"`
	Count        int64  `json:"count"`
}

type flowHandler struct {
	k8sCliFactory datastore.ClusterCtxK8sClientFactory
	esClient      celastic.Client
}

func NewFlowHandler(esClient celastic.Client, k8sClientFactory datastore.ClusterCtxK8sClientFactory) http.Handler {
	return &flowHandler{
		esClient:      esClient,
		k8sCliFactory: k8sClientFactory,
	}
}

func (handler *flowHandler) ServeHTTP(w http.ResponseWriter, rawRequest *http.Request) {
	log.Debug("GET Flow request received.")

	req, err := parseAndValidateFlowRequest(rawRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.WithField("Request", req).Debug("Request validated.")

	log.Debug("Retrieving user from context.")
	user, ok := k8srequest.UserFrom(rawRequest.Context())
	if !ok {
		log.WithError(err).Error("user not found in context")
		http.Error(w, HttpErrUnauthorizedFlowAccess, http.StatusUnauthorized)
		return
	}
	log.WithField("user", user).Debug("User retrieved from context.")

	authorizer, err := handler.k8sCliFactory.RBACAuthorizerForCluster(req.clusterName)
	if err != nil {
		log.WithError(err).Errorf("failed to get k8s client for cluster %s", req.clusterName)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	flowHelper := rbac.NewCachedFlowHelper(user, authorizer)

	srcAuthorized, err := flowHelper.CanListEndpoint(req.srcType, req.srcNamespace)
	if err != nil {
		log.WithError(err).Error("Failed to check authorization status of flow log")

		switch err.(type) {
		case *rbac.ErrUnknownEndpointType:
			http.Error(w, "Unknown srcType", http.StatusInternalServerError)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		return
	}
	log.Debugf("User has source endpoint authorization: %t", srcAuthorized)

	dstAuthorized, err := flowHelper.CanListEndpoint(req.dstType, req.dstNamespace)
	if err != nil {
		log.WithError(err).Error("Failed to check authorization status of flow log")

		switch err.(type) {
		case *rbac.ErrUnknownEndpointType:
			http.Error(w, "Unknown srcType", http.StatusInternalServerError)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		return
	}
	log.Debugf("User has destination endpoint authorization: %t", srcAuthorized)

	// If the user is not authorized to access the source or destination endpoints then they are not authorized to access
	// the flow.
	if !srcAuthorized && !dstAuthorized {
		http.Error(w, HttpErrUnauthorizedFlowAccess, http.StatusUnauthorized)
		return
	}
	log.Debug("User is authorised to view flow.")

	filters := []elastic.Query{
		elastic.NewTermQuery("source_type", req.srcType),
		elastic.NewTermQuery("source_name_aggr", req.srcName),
		elastic.NewTermQuery("source_namespace", req.srcNamespace),
		elastic.NewTermQuery("dest_type", req.dstType),
		elastic.NewTermQuery("dest_name_aggr", req.dstName),
		elastic.NewTermQuery("dest_namespace", req.dstNamespace),
	}

	if len(req.srcLabels) > 0 {
		filters = append(filters, buildLabelSelectorFilter(req.srcLabels, "source_labels", "source_labels.labels"))
	}

	if len(req.dstLabels) > 0 {
		filters = append(filters, buildLabelSelectorFilter(req.dstLabels, "dest_labels", "dest_labels.labels"))
	}

	if req.startDateTime != nil || req.endDateTime != nil {
		filter := elastic.NewRangeQuery("end_time")
		if req.startDateTime != nil {
			filter = filter.Gte(req.startDateTime.Unix())
		}

		if req.endDateTime != nil {
			filter = filter.Lt(req.endDateTime.Unix())
		}

		filters = append(filters, filter)
	}

	query := elastic.NewBoolQuery().Filter(filters...)

	log.Debug("Querying elasticsearch for flow.")
	esResponse, err := handler.esClient.Backend().Search().
		Index(req.clusterIndex()).
		Size(0).
		Query(query).
		Aggregation("src_policy_report", newReporterPolicyFilterAggregation("src")).
		Aggregation("dest_policy_report", newReporterPolicyFilterAggregation("dst")).
		Aggregation("dest_labels",
			elastic.NewNestedAggregation().Path("dest_labels").
				SubAggregation("by_kvpair", elastic.NewTermsAggregation().Field("dest_labels.labels"))).
		Aggregation("source_labels",
			elastic.NewNestedAggregation().Path("source_labels").
				SubAggregation("by_kvpair", elastic.NewTermsAggregation().Field("source_labels.labels"))).
		Do(context.Background())

	if err != nil {
		log.WithError(err).Error("failed to get flow logs from elasticsearch")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debugf("Total matching flow logs for flow: %d", esResponse.TotalHits())
	if esResponse.TotalHits() == 0 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	response := FlowResponse{
		Count: esResponse.TotalHits(),
	}

	if labelsBucket, found := esResponse.Aggregations.Nested("source_labels"); found {
		response.SrcLabels = getLabelsFromLabelAggregation(labelsBucket)
		log.WithField("labels", response.SrcLabels).Debug("Source labels parsed.")
	}

	if labelsBucket, found := esResponse.Aggregations.Nested("dest_labels"); found {
		response.DstLabels = getLabelsFromLabelAggregation(labelsBucket)
		log.WithField("labels", response.DstLabels).Debug("Destination labels parsed.")
	}

	if srcPolicyReportBucket, found := esResponse.Aggregations.Filter("src_policy_report"); found {
		if policyReport, err := newPolicyReportFromBucket(srcPolicyReportBucket, flowHelper); err != nil {
			log.WithError(err).Error("failed to read the source policy report from the elasticsearch response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			log.WithField("policies", policyReport).Debug("Policies parsed.")
			response.SrcPolicyReport = policyReport
		}
	}

	if dstPolicyReportBucket, found := esResponse.Aggregations.Filter("dest_policy_report"); found {
		if policyReport, err := newPolicyReportFromBucket(dstPolicyReportBucket, flowHelper); err != nil {
			log.WithError(err).Error("failed to read the destination policy report from the elasticsearch response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			log.WithField("policies", policyReport).Debug("Policies parsed.")
			response.DstPolicyReport = policyReport
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("failed to json encode response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func newReporterPolicyFilterAggregation(reporter string) elastic.Aggregation {
	return elastic.NewFilterAggregation().Filter(elastic.NewTermQuery("reporter", reporter)).
		SubAggregation("allowed_flow_policies", newActionPolicyFilterAggregation("allow")).
		SubAggregation("denied_flow_policies", newActionPolicyFilterAggregation("deny"))
}

func newActionPolicyFilterAggregation(action string) elastic.Aggregation {
	return elastic.NewFilterAggregation().Filter(elastic.NewTermQuery("action", action)).
		SubAggregation("policies", elastic.NewNestedAggregation().Path("policies").
			SubAggregation("by_tiered_policy", elastic.NewTermsAggregation().Field("policies.all_policies")))
}

func newPolicyReportFromBucket(policyReportAgg *elastic.AggregationSingleBucket, flowHelper rbac.FlowHelper) (*PolicyReport, error) {
	policyReport := &PolicyReport{}
	if allowedPolicy, found := policyReportAgg.Filter("allowed_flow_policies"); found {
		if policiesBucket, found := allowedPolicy.Nested("policies"); found {
			if policies, err := getPoliciesFromPolicyBucket(policiesBucket, flowHelper); err != nil {
				return nil, err
			} else {
				policyReport.AllowedFlowPolicies = policies
			}
		}
	}
	if deniedPolicy, found := policyReportAgg.Filter("denied_flow_policies"); found {
		if policiesBucket, found := deniedPolicy.Nested("policies"); found {
			if policies, err := getPoliciesFromPolicyBucket(policiesBucket, flowHelper); err != nil {
				return nil, err
			} else {
				policyReport.DeniedFlowPolicies = policies
			}
		}
	}

	// Don't return a policy report if there are neither allowed nor denied policies, since this means a policy report
	// doesn't exist.
	if policyReport.DeniedFlowPolicies == nil && policyReport.AllowedFlowPolicies == nil {
		return nil, nil
	}

	return policyReport, nil
}

// getPoliciesFromPolicyBucket parses the policy logs out from the given AggregationSingleBucket into a FlowResponsePolicy
// that can be sent back in the response. The given flowHelper helps to obfuscate the policy response if the user is not
// authorized to view certain, or all, policies.
func getPoliciesFromPolicyBucket(policiesAggregation *elastic.AggregationSingleBucket, flowHelper rbac.FlowHelper) ([]*FlowResponsePolicy, error) {
	var policies []*FlowResponsePolicy
	if terms, found := policiesAggregation.Terms("by_tiered_policy"); found {
		var obfuscatedPolicy *FlowResponsePolicy
		var policyIdx int

		// Policies aren't necessarily ordered in the flow log, so we parse out the policies from the flow log and sort
		// them first.
		var policyHits api.SortablePolicyHits
		for _, bucket := range terms.Buckets {
			key, ok := bucket.Key.(string)
			if !ok {
				// This means the flow log is invalid so just skip it, otherwise a minor issue with a single flow
				// could completely disable this endpoint.
				log.WithField("key", key).Warning("skipping bucket with non string key type")
				continue
			}

			policyHit, err := api.PolicyHitFromFlowLogPolicyString(key, bucket.DocCount)
			if err != nil {
				// This means the flow log is invalid so just skip it, otherwise a minor issue with a single flow
				// could completely disable this endpoint.
				log.WithError(err).WithField("key", key).Warning("skipping policy that failed to parse")
				continue
			}

			policyHits = append(policyHits, policyHit)
		}

		sort.Sort(policyHits)

		for _, policyHit := range policyHits {
			if canListPolicy, err := flowHelper.CanListPolicy(policyHit); err != nil {
				// An error here may mean that the request needs to be retried, i.e. a temporary error, so we should fail
				// the request so the user knows to try again.
				return nil, err
			} else if canListPolicy {
				if obfuscatedPolicy != nil {
					obfuscatedPolicy.Index = policyIdx
					policies = append(policies, obfuscatedPolicy)

					obfuscatedPolicy = nil
					policyIdx++
				}

				policies = append(policies, &FlowResponsePolicy{
					Index:        policyIdx,
					Action:       string(policyHit.Action()),
					Tier:         policyHit.Tier(),
					Namespace:    policyHit.Namespace(),
					Name:         policyHit.Name(),
					IsStaged:     policyHit.IsStaged(),
					IsKubernetes: policyHit.IsKubernetes(),
					IsProfile:    policyHit.IsProfile(),
					Count:        policyHit.Count(),
				})

				policyIdx++
			} else if policyHit.IsStaged() {
				// Ignore staged policies the use is not authorized to view
				continue
			} else {
				if obfuscatedPolicy != nil {
					obfuscatedPolicy.Action = string(policyHit.Action())
					obfuscatedPolicy.Count += policyHit.Count()
				} else {
					obfuscatedPolicy = &FlowResponsePolicy{
						Namespace: "*",
						Tier:      "*",
						Name:      "*",
						Action:    string(policyHit.Action()),
						Count:     policyHit.Count(),
					}
				}
			}
		}

		if obfuscatedPolicy != nil {
			obfuscatedPolicy.Index = policyIdx
			policies = append(policies, obfuscatedPolicy)
		}
	}

	return policies, nil
}

// getLabelsFromLabelAggregation parses the labels out from the given aggregation and puts them into a map map[string][]FlowResponseLabels
// that can be sent back in the response.
func getLabelsFromLabelAggregation(labelAggregation *elastic.AggregationSingleBucket) FlowResponseLabels {
	labelMap := make(FlowResponseLabels)
	if terms, found := labelAggregation.Terms("by_kvpair"); found {
		for _, bucket := range terms.Buckets {
			key, ok := bucket.Key.(string)
			if !ok {
				log.WithField("key", key).Warning("skipping bucket with non string key type")
				continue
			}

			labelParts := strings.Split(key, "=")
			if len(labelParts) != 2 {
				log.WithField("key", key).Warning("skipping bucket with key with invalid format (format should be 'key=value')")
				continue
			}

			labelName, labelValue := labelParts[0], labelParts[1]
			labelMap[labelName] = append(labelMap[labelName], FlowResponseLabelValue{
				Count: bucket.DocCount,
				Value: labelValue,
			})
		}
	}

	return labelMap
}
