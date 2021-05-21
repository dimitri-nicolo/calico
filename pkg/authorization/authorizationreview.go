// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package authorization

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/es-proxy/pkg/k8s"
)

// PerformAuthorizationReview performs an authorization review.
func PerformAuthorizationReview(
	ctx context.Context, client k8s.ClientSet, attr []apiv3.AuthorizationReviewResourceAttributes,
) ([]apiv3.AuthorizedResourceVerbs, error) {
	ar, err := client.AuthorizationReviews().Create(
		ctx,
		&v3.AuthorizationReview{Spec: apiv3.AuthorizationReviewSpec{
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
