// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package dns

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDNSLogs(backend bapi.DNSLogBackend) handler.Handler {
	return &dnsLogs{
		backend: backend,
	}
}

type dnsLogs struct {
	backend bapi.DNSLogBackend
}

func (n dnsLogs) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     "/flows/dns/logs/bulk",
			Handler: n.Bulk(),
		},
		{
			Method:  "POST",
			URL:     "/flows/dns/logs",
			Handler: n.GetLogs(),
		},
	}
}

func (n dnsLogs) URL() string {
	return fmt.Sprintf("%s/logs", baseURL)
}

func (b dnsLogs) GetLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		reqParams, err := handler.DecodeAndValidateReqParams[v1.DNSLogParams](w, req)
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
		response, err := b.backend.List(ctx, clusterInfo, *reqParams)
		if err != nil {
			log.WithError(err).Error("Failed to list DNS logs")
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusInternalServerError)
			return
		}

		log.Debugf("DNSLog response is: %+v", response)
		httputils.Encode(w, response)
	}
}

// Bulk handles bulk ingestion requests to add logs, typically from fluentd.
func (b dnsLogs) Bulk() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logs, err := handler.DecodeAndValidateBulkParams[v1.DNSLog](w, req)
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

		ctx, cancel := context.WithTimeout(context.Background(), v1.DefaultTimeOut)
		defer cancel()
		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}

		response, err := b.backend.Create(ctx, clusterInfo, logs)
		if err != nil {
			log.WithError(err).Error("Failed to ingest DNS logs")
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusInternalServerError)
			return
		}
		log.Debugf("Bulk response is: %+v", response)
		httputils.Encode(w, response)
	}
}
