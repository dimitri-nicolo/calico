// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

type NetworkFlows struct {
	// TODO: Add storage
}

func (n NetworkFlows) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": n.Post(),
	}
}

func (n NetworkFlows) URL() string {
	return "/flows/network"
}

func (n NetworkFlows) Post() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams := v1.L3FlowParams{
			QueryParams: &v1.QueryParams{
				Timeout: &metav1.Duration{Duration: 60 * time.Second},
			},
		}

		// Decode the http request body into the struct.
		if err := httputils.Decode(w, req, &reqParams); err != nil {
			log.WithError(err).Error("failed to decode request parameters")
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusBadRequest)
			return
		}

		// Validate parameters.
		if err := validator.Validate(reqParams); err != nil {
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusBadRequest)
			return
		}

		response := v1.L3Flow{}

		for _, s := range reqParams.Statistics {
			switch s {
			case v1.StatsTypeTraffic:
				response.TrafficStats = &v1.TrafficStats{}
			case v1.StatsTypeTCP:
				response.TCPStats = &v1.TCPStats{}
			case v1.StatsTypeProcess:
				response.ProcessStats = &v1.ProcessStats{}
			case v1.StatsTypeFlowLog:
				response.LogStats = &v1.LogStats{}
			}
		}

		httputils.Encode(w, response)
	}
}
