// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/handler"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

type flows struct {
	backend bapi.DNSFlowBackend
}

func NewFlows(backend bapi.DNSFlowBackend) *flows {
	return &flows{
		backend: backend,
	}
}

func (n flows) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": n.Serve(),
	}
}

func (n flows) URL() string {
	return baseURL
}

func (n flows) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.DNSFlowParams](w, req)
		if err != nil {
			log.WithError(err).Error("Failed to decode/validate request parameters")
			var httpErr *httputils.HttpStatusError
			if errors.As(err, &httpErr) {
				httputils.JSONError(w, httpErr, httpErr.Status)
			} else {
				httputils.JSONError(w, &httputils.HttpStatusError{
					Err:    err,
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
			log.WithError(err).Error("Failed to list flows")
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusInternalServerError)
			return
		}

		log.Debugf("%s response is: %+v", baseURL, response)
		httputils.Encode(w, response)
	}
}
