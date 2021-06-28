// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package auth

import (
	"context"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/lma/pkg/k8s"
)

// PerformAuthorizationReview performs an authorization review.
func PerformAuthorizationReview(
	ctx context.Context, client k8s.ClientSet, attr []v3.AuthorizationReviewResourceAttributes,
) ([]v3.AuthorizedResourceVerbs, error) {
	ar, err := client.ProjectcalicoV3().AuthorizationReviews().Create(
		ctx,
		&apiv3.AuthorizationReview{Spec: v3.AuthorizationReviewSpec{
			ResourceAttributes: attr,
		}},
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, err
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		if j, err := json.Marshal(ar.Status); err == nil {
			log.Debugf("Authorization matrix: %s", j)
		}
	}

	return ar.Status.AuthorizedResourceVerbs, nil
}

var (
	// The set of authorization resource attributes that are required for filtering the various elasticsearch logs.
	authReviewAttrListEndpoints = []v3.AuthorizationReviewResourceAttributes{{
		APIGroup: "projectcalico.org",
		Resources: []string{
			"hostendpoints", "networksets", "globalnetworksets", "networkpolicies", "globalnetworkpolicies",
		},
		Verbs: []string{"list"},
	}, {
		APIGroup:  "",
		Resources: []string{"pods", "nodes", "events"},
		Verbs:     []string{"list"},
	}, {
		APIGroup:  "networking.k8s.io",
		Resources: []string{"networkpolicies"},
		Verbs:     []string{"list"},
	}}
)

// PerformUserAuthorizationReviewForElasticLogs performs an authorization review on behalf of the user specified in the
// HTTP request.
//
// This function requests a set of permissions for the various endpoint types and policy types, used for filtering
// flow logs and other elastic logs.
func PerformUserAuthorizationReviewForElasticLogs(
	ctx context.Context, csFactory k8s.ClientSetFactory, req *http.Request, cluster string,
) ([]v3.AuthorizedResourceVerbs, error) {
	cs, err := csFactory.NewClientSetForUserRequest(req, cluster)
	if err != nil {
		return nil, err
	}

	return PerformAuthorizationReview(ctx, cs, authReviewAttrListEndpoints)
}
