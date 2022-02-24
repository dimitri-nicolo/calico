// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/httputils"
)

const (
	defaultTimeout = 60 * time.Second
)

// EventBulkHandler handles event bulk requests for deleting and dimssing events.
func EventBulkHandler(esClientFactory lmaelastic.ClusterContextClientFactory) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse http request body into bulk request.
		params, err := parseEventBulkRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// perform elastic bulk actions.
		// only delete and dismiss actions are supported for events.
		resp, err := processEventBulkRequest(r, esClientFactory, params)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

// parseEventBulkRequest extracts bulk parameters from the request body and validates them.
func parseEventBulkRequest(w http.ResponseWriter, r *http.Request) (*v1.BulkEventRequest, error) {
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

	// Set cluster name to default: "cluster", if empty.
	if params.ClusterName == "" {
		params.ClusterName = middleware.MaybeParseClusterNameFromRequest(r)
	}

	return &params, nil
}

// processEventBulkRequest translates bulk parameters to Elastic bulk requests and return responses.
func processEventBulkRequest(
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

	resp, err := processBulkRequest(ctx, esClient, params)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}
	return resp, nil
}

func processBulkRequest(ctx context.Context, esClient lmaelastic.Client, params *v1.BulkEventRequest) (*v1.BulkEventResponse, error) {
	var resp v1.BulkEventResponse
	afterFn := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
		resp.Errors = response.Errors
		resp.Took = response.Took

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
	}

	if err := esClient.BulkProcessorInitialize(ctx, afterFn); err != nil {
		log.WithError(err).Error("failed to initialize bulk processor for events")
		return nil, err
	}

	if params.Delete != nil {
		for _, item := range params.Delete.Items {
			if err := esClient.DeleteBulkSecurityEvent(item.ID); err != nil {
				log.WithError(err).Warnf("failed to add event delete request to bulk processor for event %s", item.ID)
			}
		}
	}
	if params.Dismiss != nil {
		for _, item := range params.Dismiss.Items {
			if err := esClient.DismissBulkSecurityEvent(item.ID); err != nil {
				log.WithError(err).Warnf("failed to add event dismiss request to bulk processor for event %s", item.ID)
			}
		}
	}

	if err := esClient.BulkProcessorClose(); err != nil {
		log.WithError(err).Error("failed to flush or close bulk processor for events")
		return nil, err
	}

	return &resp, nil
}
