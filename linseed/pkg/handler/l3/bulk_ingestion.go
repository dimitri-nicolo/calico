// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

type BulkIngestion struct {
	backend bapi.FlowLogBackend
}

func NewBulkIngestion(backend bapi.FlowLogBackend) *BulkIngestion {
	return &BulkIngestion{
		backend: backend,
	}
}

func (b BulkIngestion) SupportedAPIs() map[string]http.Handler {
	return map[string]http.Handler{
		"POST": b.Serve(),
	}
}

func (b BulkIngestion) URL() string {
	return "/bulk/flows/network/logs"
}

func (b BulkIngestion) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logs, err := handler.DecodeAndValidateBulkParams(w, req)

		if err != nil {
			log.WithError(err).Error("failed to decode/validate request parameters")
			var httpErr *httputils.HttpStatusError
			if errors.As(err, &httpErr) {
				httputils.JSONError(w, httpErr, httpErr.Status)
			}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), v1.DefaultTimeOut)
		defer cancel()
		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}

		err = b.backend.Create(ctx, clusterInfo, logs)
		if err != nil {
			httputils.JSONError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
				Err:    err,
			}, http.StatusInternalServerError)
			return
		}
	}
}
