// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	defaultTimeout = 60 * time.Second
)

// EventHandler handles event bulk requests for deleting and dimssing events.
func EventHandler(esClientFactory lmaelastic.ClusterContextClientFactory) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse http request body into bulk request.
		params, err := parseEventRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// perform elastic bulk actions.
		// only delete and dismiss actions are supported for events.
		resp, err := processEventRequest(r, esClientFactory, params)
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
func processEventRequest(
	r *http.Request,
	esClientFactory lmaelastic.ClusterContextClientFactory,
	params *v1.BulkEventRequest,
) (*v1.BulkEventResponse, error) {
	// create a context with timeout to ensure we don't block for too long.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), defaultTimeout)
	defer cancelWithTimeout()

	esClient, err := esClientFactory.ClientForCluster(params.ClusterName)
	if err != nil {
		log.WithError(err).Error("failed to create Elastic client from factory")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	resp, err := processBulkEventRequest(ctx, esClient, params)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}
	return resp, nil
}

func processBulkEventRequest(ctx context.Context, esClient lmaelastic.Client, params *v1.BulkEventRequest) (*v1.BulkEventResponse, error) {
	bulkService := elastic.NewBulkService(esClient.Backend()).Refresh("wait_for")
	if bulkService == nil {
		err := errors.New("failed to create bulk service for events")
		log.WithError(err).Error("failed to process bulk events request")
		return nil, err
	}

	if params.Delete != nil {
		for _, item := range params.Delete.Items {
			bulkService.Add(elastic.NewBulkDeleteRequest().Index(item.Index).Id(item.ID))
		}
	}
	if params.Dismiss != nil {
		for _, item := range params.Dismiss.Items {
			bulkService.Add(elastic.NewBulkUpdateRequest().Index(item.Index).Id(item.ID).Doc(map[string]bool{"dismissed": true}))
		}
	}

	response, err := bulkService.Do(ctx)
	if err != nil {
		log.WithError(err).Error("failed to process bulk events request")
		return nil, err
	}

	resp := v1.BulkEventResponse{
		Errors: response.Errors,
		Took:   response.Took,
	}

	items := make([]*elastic.BulkResponseItem, 0)
	items = append(items, response.Deleted()...)
	items = append(items, response.Updated()...)
	resp.Items = make([]v1.BulkEventResponseItem, len(items))
	for i, item := range items {
		resp.Items[i].Index = item.Index
		resp.Items[i].ID = item.Id
		resp.Items[i].Result = item.Result
		resp.Items[i].Status = item.Status
		if item.Error != nil {
			resp.Items[i].Error = &v1.BulkEventErrorDetails{
				Type:   item.Error.Type,
				Reason: item.Error.Reason,
			}
		}
	}

	return &resp, nil
}
