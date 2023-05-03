// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package auth

import (
	"context"

	v1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	authnv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
)

type tokenReviewer interface {
	Review(ctx context.Context, spec v1.TokenReviewSpec) (v1.TokenReviewStatus, error)
}

type k8sTokenReviewer struct {
	tri authnv1.TokenReviewInterface
}

func newK8sTokenReviewer(c kubernetes.Interface) *k8sTokenReviewer {
	return &k8sTokenReviewer{tri: c.AuthenticationV1().TokenReviews()}
}

func (k k8sTokenReviewer) Review(ctx context.Context, spec v1.TokenReviewSpec) (v1.TokenReviewStatus, error) {
	result, err := k.tri.Create(ctx, &v1.TokenReview{Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		return v1.TokenReviewStatus{}, err
	}
	return result.Status, nil
}
