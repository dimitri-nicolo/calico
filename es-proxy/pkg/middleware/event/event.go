// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"errors"
	"net/http"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"

	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// EventHandler handles event bulk requests for deleting and dimssing events.
func EventHandler(lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse http request body into bulk request.
		params, err := parseEventRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// Perform bulk actions - only delete and dismiss actions are supported for events.
		resp, err := processEventRequest(r, lsclient, params)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

// parseEventRequest extracts bulk parameters from the request body and validates them.
func parseEventRequest(w http.ResponseWriter, r *http.Request) (*v1.BulkEventRequest, error) {
	// events handler
	if r.Method != http.MethodPost {
		log.WithError(middleware.ErrInvalidMethod).Infof("Invalid http method %s for /events/bulk.", r.Method)

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	// Decode the http request body into the struct.
	var params v1.BulkEventRequest

	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating event bulk requests.")
			return nil, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	// validate BulkEventRequestItem index and id
	var items []v1.BulkEventRequestItem
	if params.Delete != nil {
		items = append(items, params.Delete.Items...)
	}
	if params.Dismiss != nil {
		items = append(items, params.Dismiss.Items...)
	}

	for _, item := range items {
		if item.ID == "" || item.Index == "" {
			return nil, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    middleware.ErrParseRequest.Error(),
				Err:    middleware.ErrParseRequest,
			}
		}
	}

	// Set cluster name to default: "cluster", if empty.
	if params.ClusterName == "" {
		params.ClusterName = middleware.MaybeParseClusterNameFromRequest(r)
	}

	return &params, nil
}

// processEventRequest translates bulk parameters to Elastic bulk requests and return responses.
func processEventRequest(r *http.Request, lsclient client.Client, params *v1.BulkEventRequest) (*v1.BulkEventResponse, error) {
	// create a context with timeout to ensure we don't block for too long.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), middleware.DefaultRequestTimeout)
	defer cancelWithTimeout()

	resp := v1.BulkEventResponse{}
	var dismissResp, delResp *lapi.BulkResponse
	var err error

	// We don't actually perform the delete and dismiss together. The UI only sends
	// one of these at a time anyway. We will handle a request that has both set, though.
	if params.Delete != nil {
		eventsToDelete := []lapi.Event{}
		for _, item := range params.Delete.Items {
			eventsToDelete = append(eventsToDelete, lapi.Event{ID: item.ID})
		}
		delResp, err = lsclient.Events(params.ClusterName).Delete(ctx, eventsToDelete)
		if err != nil {
			return nil, err
		}
	}
	if params.Dismiss != nil {
		eventsToDismiss := []lapi.Event{}
		for _, item := range params.Delete.Items {
			eventsToDismiss = append(eventsToDismiss, lapi.Event{ID: item.ID})
		}
		dismissResp, err = lsclient.Events(params.ClusterName).Dismiss(ctx, eventsToDismiss)
		if err != nil {
			return nil, err
		}
	}

	// Populate bulk response errors.
	resp.Errors = resp.Errors || len(dismissResp.Errors) > 0
	resp.Errors = resp.Errors || len(delResp.Errors) > 0

	// For legacy reasons, the UI expects elastic.BulkResponseItems.
	// Ideally we swich the UI so we don't need this conversion.
	items := make([]*elastic.BulkResponseItem, 0)
	for _, d := range append(delResp.Deleted, dismissResp.Updated...) {
		item := elastic.BulkResponseItem{
			Id:    d.ID,
			Index: "tigera_secure_ee_events",
		}
		switch d.Status {
		case lapi.StatusOK:
			item.Status = 200
			item.Result = "OK"
		default:
			item.Status = 500
			item.Result = "Failed"
		}
		items = append(items, &item)
	}

	return &resp, nil
}
