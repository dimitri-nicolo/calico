// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package exceptions

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	v1 "github.com/projectcalico/calico/ui-apis/pkg/apis/v1"
	"github.com/projectcalico/calico/ui-apis/pkg/middleware"
)

// EventExceptionsHandler handles requests related to security-event exceptions.
func EventExceptionsHandler(authReview middleware.AuthorizationReview, k8sClientSetFactory lmak8s.ClientSetFactory, lsclient client.Client) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the request user
		user, ok := request.UserFrom(r.Context())
		if !ok {
			httputils.EncodeError(w, &httputils.HttpStatusError{
				Status: http.StatusUnauthorized,
				Msg:    "failed to extract user from request",
				Err:    nil,
			})
			return
		}

		// Get the request cluster.
		cluster := middleware.MaybeParseClusterNameFromRequest(r)

		// Get clientSet for the request user
		logrus.WithField("cluster", cluster).Debug("Cluster ID from request")
		k8sClient, err := k8sClientSetFactory.NewClientSetForUser(user, cluster)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// Initialize eventExceptions object that handles logic for the request
		ee := eventExceptions{
			alertExceptions: k8sClient.ProjectcalicoV3().AlertExceptions(),
			eventsProvider:  lsclient.Events(cluster),
		}

		handleExceptionRequest(w, r, &ee)
	})
}

func handleExceptionRequest(w http.ResponseWriter, r *http.Request, eventExceptions EventExceptions) {
	// Validate http method.
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodDelete {
		logrus.WithError(middleware.ErrInvalidMethod).Info("Invalid http method.")

		httputils.EncodeError(w, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		})
		return
	}

	// Parse request body onto event-exceptions parameters. If an error occurs while decoding define an http
	// error and return.
	var params v1.EventException

	if r.Method == http.MethodPost || r.Method == http.MethodDelete {
		// Decode the http request body into the struct.
		if err := httputils.Decode(w, r, &params); err != nil {
			var mr *httputils.HttpStatusError
			if errors.As(err, &mr) {
				logrus.WithError(mr.Err).Info(mr.Msg)
			} else {
				logrus.WithError(mr.Err).Info("Error parsing event exceptions request.")
				err = &httputils.HttpStatusError{
					Status: http.StatusBadRequest,
					Msg:    http.StatusText(http.StatusInternalServerError),
					Err:    err,
				}
			}
			httputils.EncodeError(w, err)
			return
		}
	}

	// Create a context with timeout to ensure we don't block for too long with this query.
	// This releases timer resources if the operation completes before the timeout.
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()

	// Handle the query
	switch r.Method {
	case http.MethodGet:
		var results []*v1.EventException
		results, err := eventExceptions.List(ctx)
		if err != nil {
			httputils.EncodeError(w, err)
		} else {
			httputils.Encode(w, results)
		}
	case http.MethodPost:
		var eventException *v1.EventException
		eventException, err := eventExceptions.Create(ctx, &params)
		if err != nil {
			httputils.EncodeError(w, err)
		} else {
			httputils.Encode(w, eventException)
		}
	case http.MethodDelete:
		err := eventExceptions.Delete(ctx, &params)
		if err != nil {
			httputils.EncodeError(w, err)
		}
	}
}
