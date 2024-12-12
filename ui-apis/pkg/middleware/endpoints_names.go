package middleware

import (
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/httputils"
	querycacheclient "github.com/projectcalico/calico/queryserver/pkg/querycache/client"
	queryserverclient "github.com/projectcalico/calico/queryserver/queryserver/client"
)

type EndpointsNamesRequest struct {
	// ClusterName defines the name of the cluster a connection will be performed on.
	ClusterName string `json:"cluster"`
}

type EndPointName struct {
	Pod       string `json:"pod,omitempty" validate:"omitempty"`
	Namespace string `json:"namespace,omitempty" validate:"omitempty"`
}

// EndpointsNamesHandler returns a list of endpoints names (pod and namespace).
// It's intended to get the full list in one query, in order to provide data to autocomplete endpoints search input.
// This was scale tested (up to 15k pods) and and works as expected.
// If the response is too big, we will get an error from the Query server and the UI can handle it.
func EndpointsNamesHandler(authreview AuthorizationReview, qsConfig *queryserverclient.QueryServerConfig) http.Handler {
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

		// There is no requirements from the UI team for any payload that would allow some filtering.
		// If this is ever needed, the feedback is that we should probably revisit the whole approach.
		qsReqParams := &querycacheclient.QueryEndpointsReqBody{}

		// Parse request body (want to validate there is no undesirable fields).
		_, err := ParseBody[EndpointsNamesRequest](w, r)
		if err != nil {
			logrus.WithError(err).Error("call to ParseBody failed.")
			httputils.EncodeError(w, err)
			return
		}

		// Parse cluster name
		clusterName := MaybeParseClusterNameFromRequest(r)

		// set appropriate token
		qsConfig.QueryServerToken = r.Header.Get("Authorization")[7:]

		// Create queryserverClient client.
		queryserverClient, err := queryserverclient.NewQueryServerClient(qsConfig)
		if err != nil {
			logrus.WithError(err).Error("call to create NewQueryServerClient failed.")
			httputils.EncodeError(w, err)
			return
		}

		// Get all endpoints names.
		endpointNames := []EndPointName{}
		qsEndpointsResp, err := queryserverClient.SearchEndpoints(qsConfig, qsReqParams, clusterName)
		if err != nil {
			httputils.EncodeError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    "failed to get Endpoints from queryserver",
				Err:    errors.New("failed to get Endpoints from queryserver"),
			})
			return
		}

		for _, endpoint := range qsEndpointsResp.Items {
			endpointNames = append(endpointNames, EndPointName{Pod: endpoint.Pod, Namespace: endpoint.Namespace})
		}

		httputils.Encode(w, endpointNames)
	})
}
