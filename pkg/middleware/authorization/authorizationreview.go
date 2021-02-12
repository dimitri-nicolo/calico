// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package authorization

import (
	"context"
	"encoding/json"
	"net/http"

	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/es-proxy/pkg/middleware/k8s"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// Unexported types avoids context key collisions.
type key int

const (
	keyVerbs key = iota
)

func GetAuthorizedResourceVerbs(req *http.Request) []apiv3.AuthorizedResourceVerbs {
	return req.Context().Value(keyVerbs).([]apiv3.AuthorizedResourceVerbs)
}

// AuthorizationReviewHandler is a handler used to perform an authorization review.
//
// The result is added to the request context, and may be extracted using the GetAuthorizedResourceVerbs function.
func AuthorizationReviewHandler(attr []apiv3.AuthorizationReviewResourceAttributes, h http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		k8sCli := k8s.GetClientSetUserFromContext(req.Context())
		if k8sCli == nil {
			log.Panic("no k8s user clientset in context, handler is missing from handler chain")
			return
		}

		ar, err := k8sCli.AuthorizationReviews().Create(req.Context(),
			&v3.AuthorizationReview{Spec: apiv3.AuthorizationReviewSpec{
				ResourceAttributes: attr,
			}},
			metav1.CreateOptions{},
		)
		if err != nil {
			log.WithError(err).Error("failed to authenticate")
			http.Error(w, "Failed to authenticate", http.StatusBadGateway)
			return
		}

		if log.IsLevelEnabled(log.DebugLevel) {
			if j, err := json.Marshal(ar.Status); err == nil {
				log.Debugf("Authorization matrix: %s", j)
			}
		}

		cxt := context.WithValue(req.Context(), keyVerbs, ar.Status.AuthorizedResourceVerbs)
		req = req.WithContext(cxt)

		// Chain to the next handler.
		h.ServeHTTP(w, req)
	})
}
