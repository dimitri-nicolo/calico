// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package bgp

import (
	"context"
	"errors"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	LogPath     = "/bgp/logs"
	LogPathBulk = "/bgp/logs/bulk"
)

type bgp struct {
	logs bapi.BGPBackend
}

func New(logs bapi.BGPBackend) *bgp {
	return &bgp{
		logs: logs,
	}
}

func (h bgp) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     LogPathBulk,
			Handler: h.Bulk(),
		},
		{
			Method:  "POST",
			URL:     LogPath,
			Handler: h.GetLogs(),
		},
	}
}

// Bulk handles bulk ingestion requests to add logs, typically from fluentd.
func (h bgp) Bulk() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logs, err := handler.DecodeAndValidateBulkParams[v1.BGPLog](w, req)
		if err != nil {
			log.WithError(err).Error("Failed to decode/validate request parameters")
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
			log.WithError(err).Error("Failed to ingest BGP logs")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}
		logrus.Debugf("%s response is: %+v", LogPathBulk, response)
		httputils.Encode(w, response)
	}
}

func (h bgp) GetLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.BGPLogParams](w, req)
		if err != nil {
			log.WithError(err).Error("Failed to decode/validate request parameters")
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
		response, err := h.logs.List(ctx, clusterInfo, reqParams)
		if err != nil {
			log.WithError(err).Error("Failed to list BGP logs")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		log.Debugf("%s response is: %+v", LogPath, response)
		httputils.Encode(w, response)
	}
}
