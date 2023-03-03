// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package audit

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/sirupsen/logrus"
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

		// Create a pager to use.
		pager := client.NewListPager[v1.AuditLog](params)

		logs, err := doAuditSearch(ctx, cluster, lsclient, pager)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// Write the response.
		httputils.Encode(w, logs)
	})
}

func parseRequest(w http.ResponseWriter, r *http.Request) (*v1.AuditLogParams, string, error) {
	type auditRequest struct {
		v1.AuditLogParams `json:",inline"`
		Cluster           string `json:"cluster"`
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

	if params.Cluster == "" {
		params.Cluster = datastore.DefaultCluster
	}

	// Verify required fields.
	if params.Type == "" {
		return nil, "", &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "Missing log type parameter",
		}
	}

	return &params.AuditLogParams, params.Cluster, nil
}

func doAuditSearch(ctx context.Context, cluster string, lsc client.Client, pager client.ListPager[v1.AuditLog]) ([]v1.AuditLog, error) {
	allLogs := []v1.AuditLog{}
	pages, errors := pager.Stream(ctx, lsc.AuditLogs(cluster).List)

	for page := range pages {
		allLogs = append(allLogs, page.Items...)
	}

	if err, ok := <-errors; ok {
		return nil, err
	}
	return allLogs, nil
}
