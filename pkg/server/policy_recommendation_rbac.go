// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	esprox "github.com/tigera/es-proxy/pkg/middleware"
	authzv1 "k8s.io/api/authorization/v1"
)

const (
	lmaGroup = "lma.tigera.io"
)

type PolicyRecommendationRbacHelper interface {
	CanGetPolicyRecommendation() (bool, error)
}

type policyRecommendationRbacHelper struct {
	Request *http.Request
	k8sAuth K8sAuthInterface
}

// NewPolicyRecommendationRbacHelper returns a new initialized policyRecommendationRbacHelper.
func (f *standardRbacHelperFactory) NewPolicyRecommendationRbacHelper(req *http.Request) PolicyRecommendationRbacHelper {
	return &policyRecommendationRbacHelper{
		Request: req,
		k8sAuth: f.auth,
	}
}

// CanGetPolicyRecommendation returns true if the caller is allowed to Get a policy recommendation.
func (pr *policyRecommendationRbacHelper) CanGetPolicyRecommendation() (bool, error) {
	stat, err := pr.validatePermissions()
	switch stat {
	case 0:
		log.WithField("stat", stat).Info("Policy recommendation request authorized")
		return true, nil
	case http.StatusForbidden:
		log.WithField("stat", stat).WithError(err).Info("Policy Recommendation request forbidden - not authorized")
		return false, nil
	}
	log.WithField("stat", stat).WithError(err).Info("Error authorizing")
	return false, err
}

func (pr *policyRecommendationRbacHelper) validatePermissions() (int, error) {
	// Check permissions against our custom resource for verifying the appropriate permissions
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    lmaGroup,
		Resource: "index",
		Name:     "flows",
	}

	if stat, err := pr.checkAuthorized(*resAtr); err != nil {
		return stat, fmt.Errorf("Not authorized to get flow logs")
	}

	// Check permissions against our custom resource for verifying the appropriate permissions
	resAtr = &authzv1.ResourceAttributes{
		Verb:     "create",
		Group:    lmaGroup,
		Resource: "policyrecommendation",
	}

	if stat, err := pr.checkAuthorized(*resAtr); err != nil {
		return stat, fmt.Errorf("Not authorized to create policies through policyrecommendation")
	}

	// Authorized for all actions on all resources required
	return 0, nil
}

// checkAuthorized returns true if the request is allowed for the resources decribed in provieded attributes
func (pr *policyRecommendationRbacHelper) checkAuthorized(atr authzv1.ResourceAttributes) (int, error) {

	ctx := esprox.NewContextWithReviewResource(pr.Request.Context(), &atr)
	req := pr.Request.WithContext(ctx)

	return pr.k8sAuth.Authorize(req)
}
