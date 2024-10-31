// Copyright (c) 2022-2024 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	v1 "github.com/projectcalico/calico/ui-apis/pkg/apis/v1"
	"github.com/projectcalico/calico/ui-apis/pkg/middleware"
)

// EventHandler handles event bulk requests for deleting, dismissing and restoring events.
func EventHandler(lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse http request body into bulk request.
		params, err := parseEventRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// Perform bulk actions - only delete, dismiss and restore actions are supported for events.
		start := time.Now()
		resp, err := processEventRequest(r, lsclient, params)
		if resp != nil {
			resp.Took = int(time.Since(start).Milliseconds())
		}
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
		logrus.WithError(middleware.ErrInvalidMethod).Infof("Invalid http method %s for /events/bulk.", r.Method)

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
			logrus.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			logrus.WithError(mr.Err).Info("Error validating event bulk requests.")
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
	if params.Restore != nil {
		items = append(items, params.Restore.Items...)
	}

	for _, item := range items {
		if item.ID == "" {
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
	var delResp, dismissResp, restoreResp *lapi.BulkResponse
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
			logrus.WithError(err).Errorf("Error deleting events")
			return nil, err
		}
	}
	if params.Dismiss != nil {
		eventsToDismiss := []lapi.Event{}
		for _, item := range params.Dismiss.Items {
			eventsToDismiss = append(eventsToDismiss, lapi.Event{
				ID:        item.ID,
				Dismissed: true,
			})
		}
		dismissResp, err = lsclient.Events(params.ClusterName).UpdateDismissFlag(ctx, eventsToDismiss)
		if err != nil {
			logrus.WithError(err).Errorf("Error dismissing events")
			return nil, err
		}
	}
	if params.Restore != nil {
		eventsToRestore := []lapi.Event{}
		for _, item := range params.Restore.Items {
			eventsToRestore = append(eventsToRestore, lapi.Event{
				ID:        item.ID,
				Dismissed: false,
			})
		}
		restoreResp, err = lsclient.Events(params.ClusterName).UpdateDismissFlag(ctx, eventsToRestore)
		if err != nil {
			logrus.WithError(err).Errorf("Error restoring events")
			return nil, err
		}
	}

	if dismissResp != nil && len(dismissResp.Errors) > 0 {
		logrus.Errorf("Error dismissing %d of %d events", len(dismissResp.Errors), len(params.Dismiss.Items))
	}
	if restoreResp != nil && len(restoreResp.Errors) > 0 {
		logrus.Errorf("Error restoring %d of %d events", len(restoreResp.Errors), len(params.Restore.Items))
	}
	if delResp != nil && len(delResp.Errors) > 0 {
		logrus.Errorf("Error deleting %d of %d events", len(delResp.Errors), len(params.Delete.Items))
	}

	// Populate bulk response errors.
	resp.Errors = resp.Errors || (restoreResp != nil && len(restoreResp.Errors) > 0)
	resp.Errors = resp.Errors || (dismissResp != nil && len(dismissResp.Errors) > 0)
	resp.Errors = resp.Errors || (delResp != nil && len(delResp.Errors) > 0)

	// For legacy reasons, the UI expects elastic.BulkResponseItems.
	// Ideally we swich the UI so we don't need this conversion.
	if delResp != nil {
		for _, d := range delResp.Deleted {
			item := v1.BulkEventResponseItem{
				ID:     d.ID,
				Status: d.Status,
			}
			switch d.Status {
			case http.StatusOK:
				item.Result = "deleted"
			default:
				item.Error = &v1.BulkEventErrorDetails{
					Type:   "unknown",
					Reason: "unknown",
				}
			}
			resp.Items = append(resp.Items, item)
		}
	}
	if dismissResp != nil {
		for _, d := range dismissResp.Updated {
			item := v1.BulkEventResponseItem{
				ID:     d.ID,
				Status: d.Status,
			}
			switch d.Status {
			case http.StatusOK:
				item.Result = "updated"
			default:
				item.Error = &v1.BulkEventErrorDetails{
					Type:   "unknown",
					Reason: "unknown",
				}
			}
			resp.Items = append(resp.Items, item)
		}
	}
	if restoreResp != nil {
		for _, d := range restoreResp.Updated {
			item := v1.BulkEventResponseItem{
				ID:     d.ID,
				Status: d.Status,
			}
			switch d.Status {
			case http.StatusOK:
				item.Result = "updated"
			default:
				item.Error = &v1.BulkEventErrorDetails{
					Type:   "unknown",
					Reason: "unknown",
				}
			}
			resp.Items = append(resp.Items, item)
		}
	}
	return &resp, nil
}
