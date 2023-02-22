// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package processes

import (
	"context"
	"errors"
	"net/http"

	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	ProcPath = "/processes"
)

type procHandler struct {
	processes bapi.ProcessBackend
}

func New(procs bapi.ProcessBackend) handler.Handler {
	return &procHandler{
		processes: procs,
	}
}

func (h procHandler) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     ProcPath,
			Handler: h.List(),
		},
	}
}

func (h procHandler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.ProcessParams](w, req)
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

		// List from backend
		ctx, cancel := context.WithTimeout(context.Background(), reqParams.Timeout.Duration)
		defer cancel()
		response, err := h.processes.List(ctx, clusterInfo, reqParams)
		if err != nil {
			logrus.WithError(err).Error("Failed to list process information")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		logrus.Debugf("%s response is: %+v", ProcPath, response)
		httputils.Encode(w, response)
	}
}
