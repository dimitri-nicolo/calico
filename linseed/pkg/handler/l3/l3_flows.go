// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3

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

type NetworkFlows struct {
	backend bapi.FlowBackend
}

func NewNetworkFlows(backend bapi.FlowBackend) *NetworkFlows {
	return &NetworkFlows{
		backend: backend,
	}
}

func (n NetworkFlows) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": n.Serve(),
	}
}

func (n NetworkFlows) URL() string {
	return "/flows/network"
}

func (n NetworkFlows) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.L3FlowParams](w, req)

		if err != nil {
			log.WithError(err).Error("failed to decode/validate request parameters")
			var httpErr *httputils.HttpStatusError
			if errors.As(err, &httpErr) {
				httputils.JSONError(w, httpErr, httpErr.Status)
			}
			return
		}

		if reqParams.QueryParams.Timeout == nil {
			reqParams.QueryParams.Timeout = &metav1.Duration{Duration: v1.DefaultTimeOut}
		}

		// List flows from backend
		ctx, cancel := context.WithTimeout(context.Background(), reqParams.QueryParams.Timeout.Duration)
		defer cancel()
		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}
		flows, err := n.backend.List(ctx, clusterInfo, *reqParams)
		if err != nil {
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusInternalServerError)
			return
		}

		response := v1.L3FlowResponse{L3Flows: flows}

		httputils.Encode(w, response)
	}
}
