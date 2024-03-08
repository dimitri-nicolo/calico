package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	querycacheclient "github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	queryserverclient "github.com/projectcalico/calico/ts-queryserver/queryserver/client"
)

type EndpointsAggregationRequest struct {
	// ClusterName defines the name of the cluster a connection will be performed on.
	ClusterName string `json:"cluster"`

	// QueryServer params
	QueryEndpointsReq querycacheclient.QueryEndpointsReqBody `json:"query_filters" validate:"omitempty"`

	// Policy properties
	PolicyMatch *lapi.PolicyMatch `json:"policy_match" validate:"omitempty"`

	// Time range
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"omitempty"`

	// Timeout for the request. Defaults to 60s.
	Timeout *metav1.Duration `json:"timeout" validate:"omitempty"`
}

// EndpointsAggregationHandler is a handler for /endpoints/aggregation api
//
// returns a http handler function for filtering endpoints by a set of parameters including:
// 1. network traffic (retrieved from flowlogs), and 2. static info (from endpoints, policies,
// nodes, and labels which are retrieved from queryserver cache).
func EndpointsAggregationHandler(authz auth.RBACAuthorizer, authreview AuthorizationReview,
	qsConfig *queryserverclient.QueryServerConfig, lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Validate http method.
		if r.Method != http.MethodPost {
			logrus.WithError(ErrInvalidMethod).Info("Invalid http method.")

			err := &httputils.HttpStatusError{
				Status: http.StatusMethodNotAllowed,
				Msg:    ErrInvalidMethod.Error(),
				Err:    ErrInvalidMethod,
			}

			httputils.EncodeError(w, err)
			return
		}

		// Parse request body.
		endpointsAggregationRequest, err := ParseBody[EndpointsAggregationRequest](w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// Validate parameters.
		err = validateEndpointsAggregationRequest(r, endpointsAggregationRequest)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), endpointsAggregationRequest.Timeout.Duration)
		defer cancel()

		if endpointsAggregationRequest.PolicyMatch != nil {
			// Check authorization for flowlogs
			AuthorizeRequest(authz, endpointsFilteredByFlowLogsHandler(qsConfig, ctx, endpointsAggregationRequest, lsclient, authreview)).ServeHTTP(w, r)
		} else {
			queryserverEndpointsHandler(qsConfig, endpointsAggregationRequest, nil).ServeHTTP(w, r)
		}
	})
}

// endpointsFilteredByFlowLogsHandlers is a handler for searching endpoints
//
// returns a http handler function that filters endpoints in two phases:
// 1. filter endpoints based on flowlogs data,
// 2. filter endpoints via queryserver based on non-network related search params.
// this function is called when search parameters include network related critera.
func endpointsFilteredByFlowLogsHandler(qsConfig *queryserverclient.QueryServerConfig, ctx context.Context,
	params *EndpointsAggregationRequest, lsclient client.Client, authreview AuthorizationReview) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// filter endpoints based on flowlogs (via linseed)
		var endpoints []string
		if params.PolicyMatch != nil {
			var err error

			endpoints, err = getEndpointsFromLinseed(ctx, params, lsclient, authreview)
			if err != nil {
				httputils.EncodeError(w, &httputils.HttpStatusError{
					Status: http.StatusInternalServerError,
					Msg:    "request to get endpoints from flowlogs has failed",
					Err:    errors.New("fetching endpoints from flowlogs has failed"),
				})
				return
			}
		}

		// Filter endpoints by other parameters (via queryserver)
		queryserverEndpointsHandler(qsConfig, params, endpoints).ServeHTTP(w, r)
	})
}

// queryserverEndpointsHandler is a handler for queryserver endpoint search
//
// returns a http handler function that is calling queryserver to filter endpoints based on
// the search parameters provided.
func queryserverEndpointsHandler(qsConfig *queryserverclient.QueryServerConfig, params *EndpointsAggregationRequest,
	endpoints []string) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Filter endpoints by other parameters (via queryserver)
		qsReqParams := buildQueryServerRequestParams(endpoints, params)

		// Create queryserverClient client.
		queryserverClient, err := queryserverclient.NewQueryServerClient(qsConfig)
		if err != nil {
			logrus.WithError(err).Error("failed to create queryserver client")
			httputils.EncodeError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    "something went wrong when creating queryserver client",
				Err:    errors.New("failed to create queryserver client"),
			})
		}

		// call to queryserver client to search endpoints
		resp, err := queryserverClient.SearchEndpoints(qsConfig, qsReqParams, params.ClusterName)
		if err != nil {
			httputils.EncodeError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    "request to filter endpoints has failed",
				Err:    errors.New("filtering endpoints failed"),
			})
		}

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			httputils.EncodeError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    "error preparing the response",
				Err:    errors.New("redirecting response from queryserver failed"),
			})
		}
		resp.Body.Close()
	})
}

// validateEndpointsAggregationRequest validates the request params for /endpoints/aggregation api
//
// return error if an unacceptable set of parameters are provided
func validateEndpointsAggregationRequest(r *http.Request, endpointReq *EndpointsAggregationRequest) error {
	// Set cluster name to default: "cluster", if empty.
	if endpointReq.ClusterName == "" {
		endpointReq.ClusterName = MaybeParseClusterNameFromRequest(r)
	}

	if endpointReq.PolicyMatch != nil {
		// validate policyMatch params and only allow action:"deny"
		if endpointReq.PolicyMatch.Name != nil || endpointReq.PolicyMatch.Namespace != nil || len(endpointReq.PolicyMatch.Tier) > 0 {
			return &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    "policy_match values provided are not supported for this api",
				Err:    errors.New("unsupported parameters are set: \"policy_match\" includes \"name\", \"namespace\" and/or \"tier\""),
			}
		}

		// validate queryserver params to not include endpoints list when policyMatch is provided
		if endpointReq.QueryEndpointsReq.EndpointsList != nil {
			return &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    "both policyMatch and endpointList can not be provided in the same request",
				Err:    errors.New("invalid combination of parameters are provided: \"policy_match\" and / or \"endpointsList\""),
			}
		}

		if endpointReq.PolicyMatch.Action != nil && (*endpointReq.PolicyMatch.Action) != lapi.FlowActionDeny {
			return &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    "policy_match action can only be set to \"deny\"",
				Err:    errors.New("invalid value set for parameter: \"policy_match\".\"action\""),
			}
		}
	}

	if endpointReq.Timeout == nil {
		endpointReq.Timeout = &metav1.Duration{Duration: DefaultRequestTimeout}
	}

	// validate time range and only allow "from"
	// We want to allow user to be able to select using only From the UI
	if endpointReq.TimeRange != nil {
		if endpointReq.TimeRange.To.IsZero() && !endpointReq.TimeRange.From.IsZero() {
			endpointReq.TimeRange.To = time.Now().UTC()
		} else if !endpointReq.TimeRange.To.IsZero() {
			return &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    "time_range \"to\" should not be provided",
				Err:    errors.New("prohibited parameter is set: \"time_range\".\"to\""),
			}
		}
	}

	return nil
}

// buildFlowLogsParams prepares the parameters for flowlog search call to linseed
//
// returns FlowLogParams to be passed to linseed client, and an error.
func buildFlowLogParams(ctx context.Context, authReview AuthorizationReview, params *EndpointsAggregationRequest,
	pageNumber, pageSize int) (*lapi.FlowLogParams, error) {
	fp := &lapi.FlowLogParams{}

	if params.TimeRange != nil {
		fp.SetTimeRange(params.TimeRange)
	}

	// set policy match params.
	if params.PolicyMatch != nil {
		fp.PolicyMatches = []lapi.PolicyMatch{*params.PolicyMatch}
	}

	// Get the user's permissions. We'll pass these to Linseed to filter out logs that
	// the user doens't have permission to view.
	verbs, err := authReview.PerformReview(ctx, params.ClusterName)
	if err != nil {
		return nil, err
	}
	fp.SetPermissions(verbs)

	// Configure pagination, timeout, etc.
	fp.SetTimeout(params.Timeout)

	fp.SetMaxPageSize(pageSize)
	if pageNumber != 0 {
		fp.SetAfterKey(map[string]interface{}{
			"startFrom": pageNumber * (fp.GetMaxPageSize()),
		})
	}

	return fp, nil
}

// getEndpointsFromLinseed extracts list of endpoints from flowlogs.
//
// It calls linseed.FlowLogs().List to search flowlogs
// based on the provided parameters.
// returns a 2-tupple:
// 1. []string that includes all endpoints (src or dst) from flowlogs. Endpoint formatting in the string is compatible
// with endpoints keys in datastore (used in queryserver).
// 2. error
func getEndpointsFromLinseed(ctx context.Context, endpointsAggregationRequest *EndpointsAggregationRequest,
	lsclient client.Client, authreview AuthorizationReview) ([]string, error) {

	endpoints := map[string]int{}
	pageNumber := 0
	pageSize := 1000
	var afterKey map[string]interface{}
	flowLogsParams, err := buildFlowLogParams(ctx, authreview, endpointsAggregationRequest, pageNumber, pageSize)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    "error preparing flowlog search parameters",
			Err:    err,
		}
	}

	// iterate over all the page to get all flowlogs returned by flowlogs search
	for pageNumber == 0 || afterKey != nil {
		listFn := lsclient.FlowLogs(endpointsAggregationRequest.ClusterName).List

		items, err := listFn(ctx, flowLogsParams)
		if err != nil {
			return nil, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    "error performing flowlog search",
				Err:    err,
			}
		}

		for _, item := range items.Items {
			// Add both src and dst as endpoints extracted from flowlogs
			source, dest := extractEndpointsFromFlowLogs(item)
			endpoints[source]++
			endpoints[dest]++
		}
		pageNumber++

		// update flowlog params for next
		afterKey = items.AfterKey
		flowLogsParams.SetAfterKey(items.AfterKey)
	}

	return maps.Keys(endpoints), nil
}

// extractEndpointsFromFlowLogs extracts source and destination endpoints from flowlogs
//
// return src and dst endpoints in the endpoints key format understandable by queryserver
func extractEndpointsFromFlowLogs(item lapi.FlowLog) (string, string) {
	source := buildQueryServerEndpointKeyString(item.Host, item.SourceNamespace, item.SourceName, item.SourceNameAggr)
	dest := buildQueryServerEndpointKeyString(item.Host, item.DestNamespace, item.DestName, item.DestNameAggr)

	return source, dest
}

// buildQueryServerEndpointKeyString is building endpoints key in the format expected by queryserver.
//
// Here is one example of an endpoint key in queryserver:
// WorkloadEndpoint(tigera-fluentd/afra--bz--vaxb--kadm--ms-k8s-fluentd--node--dfpzf-eth0)
// In this code, we create the following string that will be a match for the above endpoint:
//
//	.*tigera-fluentd/afra--bz--vaxb--kadm--ms-k8s-fluentd--node--*
func buildQueryServerEndpointKeyString(host, ns, name, nameaggr string) string {
	if name == "-" {
		return fmt.Sprintf(".*%s/%s-k8s-%s", ns,
			strings.Replace(host, "-", "--", -1),
			strings.Replace(nameaggr, "-", "--", -1))
	} else {
		return fmt.Sprintf(".*%s/%s-k8s-%s", ns,
			strings.Replace(host, "-", "--", -1),
			strings.Replace(name, "-", "--", -1))
	}
}

// buildQueryServerRequestParams prepare the queryserver params
//
// return *querycacheclient.QueryEndpointsReq
func buildQueryServerRequestParams(endpoints []string, params *EndpointsAggregationRequest) *querycacheclient.QueryEndpointsReqBody {
	qsReq := &params.QueryEndpointsReq
	if endpoints != nil {
		qsReq.EndpointsList = endpoints
	}

	return qsReq
}
