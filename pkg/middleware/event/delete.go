// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"errors"
	"net/http"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/httputils"
)

func EventDeleteHandler(esClientFactory lmaelastic.ClusterContextClientFactory) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse http request body into bulk request.
		params, err := parseDeleteRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// delete
		resp, err := delete(esClientFactory, params, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

func parseDeleteRequest(w http.ResponseWriter, r *http.Request) (*v1.BulkRequest, error) {
	// events handler
	if r.Method != http.MethodDelete {
		log.WithError(middleware.ErrInvalidMethod).Infof("Invalid http method %s for /events.", r.Method)

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	// Decode the http request body into the struct.
	var params v1.BulkRequest
	params.DefaultParams()

	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating event delete request.")
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

func delete(
	esClientFactory lmaelastic.ClusterContextClientFactory,
	params *v1.BulkRequest,
	r *http.Request,
) (*v1.BulkResponse, error) {
	// Create a context with timeout to ensure we don't block for too long.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), params.Timeout.Duration)
	defer cancelWithTimeout()

	esClient, err := esClientFactory.ClientForCluster(params.ClusterName)
	if err != nil {
		log.WithError(err).Error("failed to create Elastic factory from factory")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	var resp v1.BulkResponse
	afterFn := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
		resp.Errors = response.Errors
		resp.Took = response.Took

		deletedItems := response.Deleted()
		resp.Items = make([]v1.BulkResponseItem, len(deletedItems))
		for i, item := range deletedItems {
			resp.Items[i].Index = item.Index
			resp.Items[i].ID = item.Id
			resp.Items[i].Result = item.Result
			resp.Items[i].Status = item.Status
			if item.Error != nil {
				resp.Items[i].Error = &v1.BulkErrorDetails{
					Type:   item.Error.Type,
					Reason: item.Error.Reason,
				}
			}
		}
	}

	if err := esClient.BulkProcessorInitialize(ctx, afterFn); err != nil {
		log.WithError(err).Error("failed to initialize bulk processor for event delete requests")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	for _, item := range params.Items {
		if err := esClient.DeleteBulkSecurityEvent(item.ID); err != nil {
			log.WithError(err).Warnf("failed to add event delete request to bulk processor for event id=%s", item.ID)
		}
	}
	if err := esClient.BulkProcessorClose(); err != nil {
		log.WithError(err).Error("failed to flush or close bulk processor for event delete requests")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	return &resp, nil
}
