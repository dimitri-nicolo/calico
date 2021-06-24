// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"net/http"

	lmaauth "github.com/tigera/lma/pkg/auth"
	lmak8s "github.com/tigera/lma/pkg/k8s"

	libcalv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type AuthorizationReview interface {
	PerformReviewForElasticLogs(
		ctx context.Context, req *http.Request, cluster string,
	) ([]libcalv3.AuthorizedResourceVerbs, error)
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
func (a userAuthorizationReview) PerformReviewForElasticLogs(
	ctx context.Context, req *http.Request, cluster string,
) ([]libcalv3.AuthorizedResourceVerbs, error) {
	verbs, err := lmaauth.PerformUserAuthorizationReviewForElasticLogs(ctx, a.csf, req, cluster)
	return verbs, err
}
