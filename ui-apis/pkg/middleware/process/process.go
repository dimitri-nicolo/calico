// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package process

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	v1 "github.com/projectcalico/calico/ui-apis/pkg/apis/v1"
	"github.com/projectcalico/calico/ui-apis/pkg/middleware"
)

// ProcessHandler handles process instance requests from manager dashboard.
func ProcessHandler(authReview middleware.AuthorizationReview, lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, err := parseProcessRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		resp, err := processProcessRequest(params, authReview, lsclient, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

// parseProcessRequest extracts parameters from the request body and validates them.
func parseProcessRequest(w http.ResponseWriter, r *http.Request) (*v1.ProcessRequest, error) {
	// Validate http method.
	if r.Method != http.MethodPost {
		log.WithError(middleware.ErrInvalidMethod).Info("Invalid http method.")

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	// Decode the http request body into the struct.
	var params v1.ProcessRequest

	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating process requests.")
			return nil, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	// Set cluster name to default: "cluster", if empty.
	if params.ClusterName == "" {
		params.ClusterName = middleware.MaybeParseClusterNameFromRequest(r)
	}

	return &params, nil
}

// processProcessRequest translates process request parameters to Linseed queries and returns responses.
func processProcessRequest(
	request *v1.ProcessRequest,
	authReview middleware.AuthorizationReview,
	lsclient client.Client,
	r *http.Request,
) (*v1.ProcessResponse, error) {
	// create a context with timeout to ensure we don't block for too long.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), middleware.DefaultRequestTimeout)
	defer cancelWithTimeout()

	// Build list params.
	params := lapi.ProcessParams{}
	params.TimeRange = request.TimeRange
	params.Selector = request.Selector

	verbs, err := authReview.PerformReview(ctx, request.ClusterName)
	if err != nil {
		return nil, err
	}
	params.Permissions = verbs

	// Perform paginated list.
	pager := client.NewListPager[lapi.ProcessInfo](&params)
	pages, errors := pager.Stream(ctx, lsclient.Processes(request.ClusterName).List)

	processes := []lapi.ProcessInfo{}
	for page := range pages {
		processes = append(processes, page.Items...)
	}

	if err, ok := <-errors; ok {
		return nil, err
	}

	return &v1.ProcessResponse{
		Processes: processes,
	}, nil
}
