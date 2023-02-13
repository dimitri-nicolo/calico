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

func NewEvents(backend bapi.EventsBackend) *events {
	return &events{
		backend: backend,
	}
}

type events struct {
	backend bapi.EventsBackend
}

func (n events) APIS() []handler.API {
	return []handler.API{
		{
			// Base URL queries for events.
			Method:  "POST",
			URL:     baseURL,
			Handler: n.List(),
		},
		{
			// Bulk creation for events.
			Method:  "POST",
			URL:     baseURL + "/bulk",
			Handler: n.Bulk(),
		},
	}
}

func (n events) List() http.HandlerFunc {
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
		response, err := n.backend.List(ctx, clusterInfo, *reqParams)
		if err != nil {
			logrus.WithError(err).Error("Failed to list events")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		logrus.Debugf("%s response is: %+v", baseURL, response)
		httputils.Encode(w, response)
	}
}

func (n events) Bulk() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logs, err := handler.DecodeAndValidateBulkParams[v1.Event](w, req)
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

		response, err := n.backend.Create(ctx, clusterInfo, logs)
		if err != nil {
			logrus.WithError(err).Error("Failed to ingest events")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}
		logrus.Debugf("Bulk response is: %+v", response)
		httputils.Encode(w, response)
	}
}
