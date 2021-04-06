// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package auth

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
)

type contextKey int

const (
	ResourceAttributeKey contextKey = iota
	NonResourceAttributeKey
)

// RBACAuthorizer is an interface who's implementors are used to check if a user is authorised to access given K8s RBAC
// attributes.
type RBACAuthorizer interface {
	Authorize(usr user.Info, resources *authzv1.ResourceAttributes, nonResources *authzv1.NonResourceAttributes) (bool, error)
}

type rbacAuthorizer struct {
	k8sCli k8s.Interface
}

func NewRBACAuthorizer(k8sCli k8s.Interface) RBACAuthorizer {
	return &rbacAuthorizer{
		k8sCli: k8sCli,
	}
}

// Authorize checks if the given user is authorized to access the given resources and non resources. If the user is authorized
// true is returned, if not false is returned. An error that occurred is returned.
func (auth *rbacAuthorizer) Authorize(usr user.Info, resources *authzv1.ResourceAttributes, nonResources *authzv1.NonResourceAttributes) (bool, error) {
	if usr == nil {
		return false, fmt.Errorf("no user available to authorize against")
	}

	if resources == nil && nonResources == nil {
		return false, fmt.Errorf("no resource available to authorize")
	}

	return auth.createSubjectAccessReview(usr, resources, nonResources)
}

// subjectAccessReview creates a authzv1.SubjectAccessReview to check if the given user is authorized to access the given
// authzv1.ResourceAttributes and authzv1.NonResourceAttributes.
//
// If the user is authorized true is returned, if not false is returned. An error that occurred is returned.
func (auth *rbacAuthorizer) createSubjectAccessReview(user user.Info, resource *authzv1.ResourceAttributes, nonResource *authzv1.NonResourceAttributes) (bool, error) {
	sar := authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			ResourceAttributes:    resource,
			NonResourceAttributes: nonResource,
			User:                  user.GetName(),
			Groups:                user.GetGroups(),
			Extra:                 make(map[string]authzv1.ExtraValue),
			UID:                   user.GetUID(),
		},
	}

	for k, v := range user.GetExtra() {
		sar.Spec.Extra[k] = v
	}

	res, err := auth.k8sCli.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), &sar, metav1.CreateOptions{})
	if res != nil {
		log.Debugf("Response to access review: %#v", res.Status)
	}

	if err != nil {
		return false, fmt.Errorf("error performing AccessReview: %v", err)
	}

	return res.Status.Allowed, nil
}

// ExtractRBACFieldsFromContext retrieves the user, authzv1.ResourceAttributes, and authzv1.ResourceNonAttributes from the
// given context, if available.
func ExtractRBACFieldsFromContext(ctx context.Context) (user.Info, *authzv1.ResourceAttributes, *authzv1.NonResourceAttributes) {
	usr, _ := request.UserFrom(ctx)
	res, _ := FromContextGetReviewResource(ctx)
	nonRes, _ := FromContextGetReviewNonResource(ctx)

	return usr, res, nonRes
}

func NewContextWithReviewResource(
	ctx context.Context,
	ra *authzv1.ResourceAttributes,
) context.Context {
	return context.WithValue(ctx, ResourceAttributeKey, ra)
}

func NewContextWithReviewNonResource(
	ctx context.Context,
	ra *authzv1.NonResourceAttributes,
) context.Context {
	return context.WithValue(ctx, NonResourceAttributeKey, ra)
}

// FromContextGetReviewResource retrieves the stored authzv1.ResourceAttributes from the context, if available.
func FromContextGetReviewResource(ctx context.Context) (*authzv1.ResourceAttributes, bool) {
	ra, ok := ctx.Value(ResourceAttributeKey).(*authzv1.ResourceAttributes)
	return ra, ok
}

// FromContextGetReviewResource retrieves the stored authzv1.NonResourceAttributes from the context, if available.
func FromContextGetReviewNonResource(ctx context.Context) (*authzv1.NonResourceAttributes, bool) {
	nra, ok := ctx.Value(NonResourceAttributeKey).(*authzv1.NonResourceAttributes)
	return nra, ok
}
