// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package authorizer

import (
	"fmt"
	"sync"

	"github.com/golang/glog"

	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"

	"k8s.io/apimachinery/pkg/api/errors"
	k8sauth "k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/filters"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

type TierAuthorizer interface {
	// AuthorizeTierOperation checks whether the request is for a tiered policy, and if so checks
	// whether the user us authorized to perform the operation. Returns a Forbidden error  if the
	// operation is not authorized.
	AuthorizeTierOperation(ctx genericapirequest.Context, policyName string, tierName string) error
}

type authorizer struct {
	k8sauth.Authorizer
}

// Returns a new TierAuthorizer that uses the provided standard authorizer to perform the underlying
// lookups.
func NewTierAuthorizer(a k8sauth.Authorizer) TierAuthorizer {
	return &authorizer{a}
}

// AuthorizeTierOperation implements the TierAuthorizer interface.
func (a *authorizer) AuthorizeTierOperation(
	ctx genericapirequest.Context,
	policyName string,
	tierName string,
) error {
	if a.Authorizer == nil {
		glog.V(4).Info("No authorizer - allow operation")
		return nil
	}

	attrs, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		glog.Errorf("Unable to extract authorizer attributes: %s", err)
		return err
	}

	// Log the original authorizer attributes.
	logAuthorizerAttributes(attrs)

	if !isTieredPolicy(attrs) {
		// This is not a tiered policy resource request, so exit. RBAC control will be entirely
		// handled by the default processing.
		glog.V(4).Info("Operation is not for Calico tiered policy - defer to standard RBAC only")
		return nil
	}

	// Perform tier authorization.
	if err := a.authorizeTieredPolicy(attrs, policyName, tierName); err != nil {
		glog.V(4).Infof("Operation on Calico tiered policy is forbidden: %v", err)
		return err
	}

	return nil
}

// isTieredPolicy returns true if the attributes indicate this is a request for a tiered policy.
func isTieredPolicy(attr k8sauth.Attributes) bool {
	if !attr.IsResourceRequest() {
		return false
	}

	switch attr.GetResource() {
	case "networkpolicies", "globalnetworkpolicies", "stagednetworkpolicies", "stagedglobalnetworkpolicies":
		return true
	}

	glog.V(4).Infof("Is not Calico policy type: %s", attr.GetResource())
	return false
}

// authorizeTieredPolicy performs the multi-stage tier authorization for the policy request.
func (a *authorizer) authorizeTieredPolicy(attributes k8sauth.Attributes, policyName, tierName string) error {
	// We need to check whether the user is authorized to perform the action on the tier.<resourcetype>
	// resource, with a resource name of either:
	// - <tier>.*         (this is the wildcard syntax for any Calico policy within a tier)
	// - <tier>.<policy>  (this checks for a specific policy and tier, or fully wildcarded policy and tier)
	// *and* has GET access for the tier.
	// These requests can be performed in parallel.
	wg := sync.WaitGroup{}
	wg.Add(3)

	// Query GET access for the tier.
	var decisionGetTier k8sauth.Decision
	go func() {
		defer wg.Done()
		attrs := k8sauth.AttributesRecord{
			User:            attributes.GetUser(),
			Verb:            "get",
			Namespace:       "",
			APIGroup:        attributes.GetAPIGroup(),
			APIVersion:      attributes.GetAPIVersion(),
			Resource:        "tiers",
			Subresource:     "",
			Name:            tierName,
			ResourceRequest: true,
			Path:            "/apis/projectcalico.org/v3/tiers/" + tierName,
		}

		glog.V(4).Infof("Checking authorization using tier resource type (user can get tier)")
		logAuthorizerAttributes(attrs)
		decisionGetTier, _, _ = a.Authorizer.Authorize(attrs)
	}()

	// Query required access to the tiered policy resource or tier wildcard resource.
	var decisionPolicy, decisionTierWildcard k8sauth.Decision
	var pathPrefix string
	tierScopedResource := "tier." + attributes.GetResource()
	if attributes.GetNamespace() == "" {
		pathPrefix = "/apis/projectcalico.org/v3/" + tierScopedResource
	} else {
		pathPrefix = "/apis/projectcalico.org/v3/namespaces/" + attributes.GetNamespace() + "/" + tierScopedResource
	}
	go func() {
		defer wg.Done()
		path := pathPrefix
		if attributes.GetName() != "" {
			path = pathPrefix + "/" + attributes.GetName()
		}
		attrs := k8sauth.AttributesRecord{
			User:            attributes.GetUser(),
			Verb:            attributes.GetVerb(),
			Namespace:       attributes.GetNamespace(),
			APIGroup:        attributes.GetAPIGroup(),
			APIVersion:      attributes.GetAPIVersion(),
			Resource:        tierScopedResource,
			Subresource:     attributes.GetSubresource(),
			Name:            attributes.GetName(),
			ResourceRequest: true,
			Path:            path,
		}

		glog.V(4).Infof("Checking authorization using tier scoped resource type (policy name match)")
		logAuthorizerAttributes(attrs)
		decisionPolicy, _, _ = a.Authorizer.Authorize(attrs)
	}()
	go func() {
		defer wg.Done()
		name := tierName + ".*"
		path := pathPrefix + "/" + name
		attrs := k8sauth.AttributesRecord{
			User:            attributes.GetUser(),
			Verb:            attributes.GetVerb(),
			Namespace:       attributes.GetNamespace(),
			APIGroup:        attributes.GetAPIGroup(),
			APIVersion:      attributes.GetAPIVersion(),
			Resource:        tierScopedResource,
			Subresource:     attributes.GetSubresource(),
			Name:            name,
			ResourceRequest: true,
			Path:            path,
		}

		glog.V(4).Infof("Checking authorization using tier scoped resource type (tier name match)")
		logAuthorizerAttributes(attrs)
		decisionTierWildcard, _, _ = a.Authorizer.Authorize(attrs)
	}()

	// Wait for the requests to complete.
	wg.Wait()

	// If the user has GET access to the tier and either the policy match or tier wildcard match are authorized
	// then allow the request.
	if decisionGetTier == k8sauth.DecisionAllow &&
		(decisionPolicy == k8sauth.DecisionAllow || decisionTierWildcard == k8sauth.DecisionAllow) {
		glog.Infof("Operation allowed")
		return nil
	}

	// Request is forbidden.
	reason := forbiddenMessage(attributes, tierName, decisionGetTier)
	return errors.NewForbidden(calico.Resource(attributes.GetResource()), policyName, fmt.Errorf(reason))
}

// forbiddenMessage crafts the appropriate tier-specific forbidden message. This is largely copied
// from k8s.io/apiserver/pkg/endpoints/handlers/responsewriters/errors.go
func forbiddenMessage(attributes k8sauth.Attributes, tierName string, decisionGetTier k8sauth.Decision) string {
	username := ""
	if user := attributes.GetUser(); user != nil {
		username = user.GetName()
	}

	resource := attributes.GetResource()
	if group := attributes.GetAPIGroup(); len(group) > 0 {
		resource = resource + "." + group
	}
	if subresource := attributes.GetSubresource(); len(subresource) > 0 {
		resource = resource + "/" + subresource
	}

	var msg string
	if ns := attributes.GetNamespace(); len(ns) > 0 {
		msg = fmt.Sprintf("User %q cannot %s %s in tier %q and namespace %q", username, attributes.GetVerb(), resource, tierName, ns)
	} else {
		msg = fmt.Sprintf("User %q cannot %s %s in tier %q", username, attributes.GetVerb(), resource, tierName)
	}

	// If the user does not have get access to the tier, append additional text to the message.
	if decisionGetTier != k8sauth.DecisionAllow {
		msg += " (user cannot get tier)"
	}
	return msg
}

// logAuthorizerAttributes logs out the auth attributes.
func logAuthorizerAttributes(requestAttributes k8sauth.Attributes) {
	if glog.V(4) {
		glog.Infof("Authorizer APIGroup: %s", requestAttributes.GetAPIGroup())
		glog.Infof("Authorizer APIVersion: %s", requestAttributes.GetAPIVersion())
		glog.Infof("Authorizer Name: %s", requestAttributes.GetName())
		glog.Infof("Authorizer Namespace: %s", requestAttributes.GetNamespace())
		glog.Infof("Authorizer Resource: %s", requestAttributes.GetResource())
		glog.Infof("Authorizer Subresource: %s", requestAttributes.GetSubresource())
		glog.Infof("Authorizer User: %s", requestAttributes.GetUser())
		glog.Infof("Authorizer Verb: %s", requestAttributes.GetVerb())
		glog.Infof("Authorizer Path: %s", requestAttributes.GetPath())
	}
}
