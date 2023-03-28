package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// TODO: Update other typed handlers to use this.
//
// GenericHandler implements a basic HTTP handler that allows us to have a common
// implementation for most APIs that use common verbs - create / bulk / etc.
type GenericHandler[T any, P RequestParams, B BulkRequestParams] struct {
	ListFn   func(context.Context, bapi.ClusterInfo, *P) (*v1.List[T], error)
	CreateFn func(context.Context, bapi.ClusterInfo, []B) (*v1.BulkResponse, error)
}

func (h GenericHandler[T, P, B]) List() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		f := logrus.Fields{
			"path":   req.URL.Path,
			"method": req.Method,
		}
		logCtx := logrus.WithFields(f)

		// Decore the request body, which contains parameters for the request.
		params, err := DecodeAndValidateReqParams[P](w, req)
		if err != nil {
			logCtx.WithError(err).Error("Failed to decode/validate request parameters")
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

		// Get the timeout from the request, and use it to build a context.
		timeout, err := Timeout(w, req)
		if err != nil {
			httputils.JSONError(w, &v1.HTTPError{
				Msg:    err.Error(),
				Status: http.StatusBadRequest,
			}, http.StatusBadRequest)
		}
		ctx, cancel := context.WithTimeout(req.Context(), timeout.Duration)
		defer cancel()

		// Get cluster and tenant information, which will have been populated by middleware.
		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}

		// Perform the request
		response, err := h.ListFn(ctx, clusterInfo, params)
		if err != nil {
			logCtx.WithError(err).Error("Error performing list")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}
		logCtx.Debugf("Response is: %+v", response)
		httputils.Encode(w, response)
	}
}

func (h GenericHandler[T, P, B]) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		f := logrus.Fields{
			"path":   req.URL.Path,
			"method": req.Method,
		}
		logCtx := logrus.WithFields(f)

		data, err := DecodeAndValidateBulkParams[B](w, req)
		if err != nil {
			logCtx.WithError(err).Error("Failed to decode/validate request parameters")
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

		// Bulk creation requests don't include a timeout, so use the default.
		ctx, cancel := context.WithTimeout(context.Background(), v1.DefaultTimeOut)
		defer cancel()
		clusterInfo := bapi.ClusterInfo{
			Cluster: middleware.ClusterIDFromContext(req.Context()),
			Tenant:  middleware.TenantIDFromContext(req.Context()),
		}

		// Call the creation function.
		response, err := h.CreateFn(ctx, clusterInfo, data)
		if err != nil {
			logCtx.WithError(err).Error("Error performing bulk ingestion")
			httputils.JSONError(w, &v1.HTTPError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}, http.StatusInternalServerError)
			return
		}
		logCtx.Debugf("Response is: %+v", response)
		httputils.Encode(w, response)
	}
}
