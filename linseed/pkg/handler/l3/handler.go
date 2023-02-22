// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3

import (
	"context"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/handler"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	FlowPath    = "/flows"
	LogPath     = "/flows/logs"
	LogPathBulk = "/flows/logs/bulk"
)

type Flows struct {
	flows bapi.FlowBackend
	logs  bapi.FlowLogBackend
}

func New(flows bapi.FlowBackend, logs bapi.FlowLogBackend) handler.Handler {
	return &Flows{
		flows: flows,
		logs:  logs,
	}
}

func (h Flows) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     FlowPath,
			Handler: h.Flows(),
		},
		{
			Method:  "POST",
			URL:     LogPath,
			Handler: h.GetLogs(),
		},
		{
			Method:  "POST",
			URL:     LogPathBulk,
			Handler: h.Bulk(),
		},
	}
}

func (h Flows) Flows() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.L3FlowParams](w, req)
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

		// List flows from backend
		ctx, cancel := context.WithTimeout(context.Background(), reqParams.Timeout.Duration)
		defer cancel()
		response, err := h.flows.List(ctx, clusterInfo, *reqParams)
		if err != nil {
			logrus.WithError(err).Error("Failed to list flows")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		logrus.Debugf("Flow response is: %+v", response)
		httputils.Encode(w, response)
	}
}

func (h Flows) GetLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.FlowLogParams](w, req)
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
		response, err := h.logs.List(ctx, clusterInfo, *reqParams)
		if err != nil {
			logrus.WithError(err).Error("Failed to list flow logs")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		logrus.Debugf("FlowLog response is: %+v", response)
		httputils.Encode(w, response)
	}
}

// Bulk handles bulk ingestion requests to add flow logs, typically from fluentd.
func (h Flows) Bulk() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logs, err := handler.DecodeAndValidateBulkParams[v1.FlowLog](w, req)
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

		response, err := h.logs.Create(ctx, clusterInfo, logs)
		if err != nil {
			logrus.WithError(err).Error("Failed to ingest flow logs")
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
