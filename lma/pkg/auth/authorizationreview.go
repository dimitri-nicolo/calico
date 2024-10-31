// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package auth

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/projectcalico/calico/lma/pkg/k8s"
)

// PerformAuthorizationReviewWithContext performs an authorization review with context.
func PerformAuthorizationReviewWithContext(
	ctx context.Context, client k8s.ClientSet, attr []v3.AuthorizationReviewResourceAttributes,
) ([]v3.AuthorizedResourceVerbs, error) {
	return performAuthorizationReview(
		ctx,
		client,
		v3.AuthorizationReview{Spec: v3.AuthorizationReviewSpec{
			ResourceAttributes: attr,
		}},
	)
}

// PerformAuthorizationReviewWithUser function performs an authorization review for a specific user by creating  the
// AuthorizationReview resource with userinfo and passing that to performAuthorizationReview using the background context.
// This function is used when application doesn't have impersonate rbac to pass in the userinfo via the context.
func PerformAuthorizationReviewWithUser(usr user.Info, client k8s.ClientSet, attr []v3.AuthorizationReviewResourceAttributes,
) ([]v3.AuthorizedResourceVerbs, error) {
	return performAuthorizationReview(
		context.Background(),
		client,
		v3.AuthorizationReview{Spec: v3.AuthorizationReviewSpec{
			ResourceAttributes: attr,
			User:               usr.GetName(),
			Groups:             usr.GetGroups(),
			Extra:              usr.GetExtra(),
		}},
	)
}

// performAuthorizationReview performs an authorization review.
func performAuthorizationReview(ctx context.Context, client k8s.ClientSet, authReview v3.AuthorizationReview) ([]v3.AuthorizedResourceVerbs, error) {
	ar, err := client.ProjectcalicoV3().AuthorizationReviews().Create(
		ctx,
		&authReview,
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

// The set of authorization resource attributes that are required for filtering the various elasticsearch logs.
var authReviewAttrListEndpoints = []v3.AuthorizationReviewResourceAttributes{{
	APIGroup: "projectcalico.org",
	Resources: []string{
		"hostendpoints", "networksets", "globalnetworksets", "networkpolicies", "globalnetworkpolicies",
		"packetcaptures",
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

// PerformUserAuthorizationReviewForLogs performs an authorization review impersonating the user.
//
// This function requests a set of permissions for the various endpoint types and policy types, used for filtering
// flow logs and other elastic logs.
func PerformUserAuthorizationReviewForLogs(ctx context.Context, csFactory k8s.ClientSetFactory, user user.Info, cluster string) ([]v3.AuthorizedResourceVerbs, error) {
	cs, err := csFactory.NewClientSetForUser(user, cluster)
	if err != nil {
		return nil, err
	}

	return PerformAuthorizationReviewWithContext(ctx, cs, authReviewAttrListEndpoints)
}
