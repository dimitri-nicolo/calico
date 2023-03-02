// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"net/http"

	log "github.com/sirupsen/logrus"

	"k8s.io/apiserver/pkg/endpoints/request"

	libcalv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
)

type AuthorizationReview interface {
	PerformReview(ctx context.Context, cluster string) ([]libcalv3.AuthorizedResourceVerbs, error)
}

// The user authentication review struct implementing the authentication review interface.
type userAuthorizationReview struct {
	csf lmak8s.ClientSetFactory
}

// NewAuthorizationReview creates an implementation of the AuthorizationReview.
func NewAuthorizationReview(csFactory lmak8s.ClientSetFactory) AuthorizationReview {
	return &userAuthorizationReview{csf: csFactory}
}

// PerformReviewForElasticLogs performs an authorization review on behalf of the user specified in
// the HTTP request.
//
// This function wraps lma's PerformUserAuthorizationReviewForElasticLogs.
func (a userAuthorizationReview) PerformReview(
	ctx context.Context, cluster string,
) ([]libcalv3.AuthorizedResourceVerbs, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		// There should be user info on the request context. If not this is is server error since an earlier handler
		// should have authenticated.
		log.Debug("No user information on request")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    "No user information on request",
		}
	}

	verbs, err := lmaauth.PerformUserAuthorizationReviewForLogs(ctx, a.csf, user, cluster)
	return verbs, err
}
