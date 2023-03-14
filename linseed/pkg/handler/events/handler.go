// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package events

import (
	"context"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	EventsPath     = "/events"
	EventsPathBulk = "/events/bulk"
)

func New(backend bapi.EventsBackend) *events {
	return &events{
		backend: backend,
	}
}

type events struct {
	backend bapi.EventsBackend
}

func (h events) APIS() []handler.API {
	return []handler.API{
		{
			// Base URL queries for events.
			Method:  "POST",
			URL:     EventsPath,
			Handler: h.List(),
		},
		{
			// Bulk creation for events.
			Method:  "POST",
			URL:     EventsPathBulk,
			Handler: h.Bulk(),
		},
		{
			// Bulk dismissal for events.
			Method:  "PUT",
			URL:     EventsPathBulk,
			Handler: h.Bulk(),
		},
		{
			// Bulk delete for events.
			Method:  "DELETE",
			URL:     EventsPathBulk,
			Handler: h.Bulk(),
		},
	}
}

func (h events) List() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.EventParams](w, req)
		if err != nil {
			logrus.WithError(err).Error("Failed to decode/validate request parameters")
			var httpErr *v1.HTTPError
			if errors.As(err, &httpErr) {
				httputils.JSONError(w, httpErr, httpErr.Status)
			} else {
				httputils.JSONError(w, &v1.HTTPError{
					Msg:    err.Error(),
					Status: http.StatusBadRequest,
				}, http.StatusBadRequest)
			}
			return
		}

		if reqParams.Timeout == nil {
			reqParams.Timeout = &metav1.Duration{Duration: v1.DefaultTimeOut}
		}

		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}

		ctx, cancel := context.WithTimeout(context.Background(), reqParams.Timeout.Duration)
		defer cancel()
		response, err := h.backend.List(ctx, clusterInfo, reqParams)
		if err != nil {
			logrus.WithError(err).Error("Failed to list events")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		logrus.Debugf("%s response is: %+v", EventsPath, response)
		httputils.Encode(w, response)
	}
}

func (h events) Bulk() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		events, err := handler.DecodeAndValidateBulkParams[v1.Event](w, req)
		if err != nil {
			logrus.WithError(err).Error("Failed to decode/validate request parameters")
			var httpErr *v1.HTTPError
			if errors.As(err, &httpErr) {
				httputils.JSONError(w, httpErr, httpErr.Status)
			} else {
				httputils.JSONError(w, &v1.HTTPError{
					Msg:    err.Error(),
					Status: http.StatusBadRequest,
				}, http.StatusBadRequest)
			}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), v1.DefaultTimeOut)
		defer cancel()
		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}

		// The bulk API supports multiple operations. Determine which backend
		// handler to use.
		type bulkHandler func(context.Context, bapi.ClusterInfo, []v1.Event) (*v1.BulkResponse, error)
		var handler bulkHandler
		switch req.Method {
		case http.MethodPost:
			// Create events.
			handler = h.backend.Create
		case http.MethodPut:
			// Dismiss events.
			handler = h.backend.Dismiss
		case http.MethodDelete:
			// Delete events.
			handler = h.backend.Delete
		default:
			// Unsupported method.
			httputils.JSONError(w, &v1.HTTPError{
				Msg:    "unsupported method",
				Status: http.StatusMethodNotAllowed,
			}, http.StatusMethodNotAllowed)
			return
		}

		// Call the chosen handler.
		response, err := handler(ctx, clusterInfo, events)
		if err != nil {
			logrus.WithError(err).Error("Failed to perform bulk action on events")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}
		logrus.Debugf("%s %s response is: %+v", req.Method, EventsPathBulk, response)
		httputils.Encode(w, response)
	}
}
