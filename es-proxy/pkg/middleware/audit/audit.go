// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package audit

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

func NewHandler(lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the request.
		params, cluster, err := parseRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		items, err := lsclient.AuditLogs(cluster).List(ctx, params)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// Write the response.
		httputils.Encode(w, items)
	})
}

func parseRequest(w http.ResponseWriter, r *http.Request) (*v1.AuditLogParams, string, error) {
	type auditRequest struct {
		v1.AuditLogParams `json:",inline"`
		Page              int `json:"page"`
	}

	params := auditRequest{}
	if err := httputils.Decode(w, r, &params); err != nil {
		var e *httputils.HttpStatusError
		if errors.As(err, &e) {
			logrus.WithError(e.Err).Info(e.Msg)
			return nil, "", e
		} else {
			logrus.WithError(e.Err).Info("Error validating process requests.")
			return nil, "", &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	// Ideally, clients don't know the syntax of the after key, but
	// for paged lists we currently need this.
	params.SetAfterKey(map[string]interface{}{
		"startFrom": params.Page * params.MaxPageSize,
	})

	// Verify required fields.
	if params.Type == "" {
		return nil, "", &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "Missing log type parameter",
		}
	}

	// Extract the cluster ID header.
	cluster := middleware.MaybeParseClusterNameFromRequest(r)

	return &params.AuditLogParams, cluster, nil
}
