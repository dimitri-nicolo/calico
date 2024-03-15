// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"

	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// EventStatisticsHandler handles event statistics requests.
func EventStatisticsHandler(lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse http request body into bulk request.
		params, err := parseEventStatisticsRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// create a context with timeout to ensure we don't block for too long.
		ctx, cancelWithTimeout := context.WithTimeout(r.Context(), middleware.DefaultRequestTimeout)
		defer cancelWithTimeout()

		// Get cluster name
		clusterName := middleware.MaybeParseClusterNameFromRequest(r)

		// Perform statistics request
		resp, err := lsclient.Events(clusterName).Statistics(ctx, *params)

		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

// parseEventStatisticsRequest extracts statistics parameters from the request body and validates them.
func parseEventStatisticsRequest(w http.ResponseWriter, r *http.Request) (*lapi.EventStatisticsParams, error) {
	// events handler
	if r.Method != http.MethodPost {
		logrus.WithError(middleware.ErrInvalidMethod).Infof("Invalid http method %s for /events/statistics.", r.Method)

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	// Decode the http request body into the struct.
	var params lapi.EventStatisticsParams

	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			logrus.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			logrus.WithError(mr.Err).Info("Error validating event statistics requests.")
			return nil, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	return &params, nil
}
