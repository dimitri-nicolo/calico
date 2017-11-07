package util

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/filters"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

const (
	policyDelim = "."
	defaultTier = "default"
)

// TODO: Remove this. Its purely for debugging purposes.
func logAuthorizerAttributes(requestAttributes authorizer.Attributes) {
	glog.Infof("Authorizer APIGroup: %s", requestAttributes.GetAPIGroup())
	glog.Infof("Authorizer APIVersion: %s", requestAttributes.GetAPIVersion())
	glog.Infof("Authorizer Name: %s", requestAttributes.GetName())
	glog.Infof("Authorizer Namespace: %s", requestAttributes.GetNamespace())
	glog.Infof("Authorizer Resource: %s", requestAttributes.GetResource())
	glog.Infof("Authorizer Subresource: %s", requestAttributes.GetSubresource())
	glog.Infof("Authorizer User: %s", requestAttributes.GetUser())
	glog.Infof("Authorizer Verb: %s", requestAttributes.GetVerb())
}

func setTierSelector(options *metainternalversion.ListOptions) {
	options.FieldSelector = fields.SelectorFromSet(map[string]string{"spec.tier": defaultTier})
}

func GetTierNameFromSelector(options *metainternalversion.ListOptions) (string, error) {
	if options.FieldSelector != nil {
		requirements := options.FieldSelector.Requirements()
		for _, requirement := range requirements {
			if requirement.Field == "spec.tier" {
				return requirement.Value, nil
			}
		}
	}

	if options.LabelSelector != nil {
		requirements, _ := options.LabelSelector.Requirements()
		for _, requirement := range requirements {
			if requirement.Key() == "projectcalico.org/tier" {
				if len(requirement.Values()) > 1 {
					return "", fmt.Errorf("multi-valued selector not supported")
				}
				tierName, ok := requirement.Values().PopAny()
				if ok {
					return tierName, nil
				}
			}
		}
	}

	// Reaching here implies tier is 'default' and hasn't been explicitly set as part of the selectors.
	setTierSelector(options)
	return defaultTier, nil
}

// Check the user is allowed to "get" the tier.
// This is required to be allowed to perform actions on policies.
func AuthorizeTierOperation(ctx genericapirequest.Context, authz authorizer.Authorizer, tierName string) error {
	if authz == nil {
		glog.Infof("Authorization disabled for testing purposes")
		return nil
	}
	attributes, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return err
	}
	attrs := authorizer.AttributesRecord{}
	attrs.APIGroup = attributes.GetAPIGroup()
	attrs.APIVersion = attributes.GetAPIVersion()
	attrs.Name = tierName
	attrs.Resource = "tiers"
	attrs.User = attributes.GetUser()
	attrs.Verb = "get"
	attrs.ResourceRequest = attributes.IsResourceRequest()
	attrs.Path = "/apis/projectcalico.org/v2/tiers/" + tierName
	glog.Infof("Tier Auth Attributes for the given Policy")
	logAuthorizerAttributes(attrs)
	authorized, reason, err := authz.Authorize(attrs)
	if err != nil {
		return err
	}
	if !authorized {
		if reason == "" {
			reason = fmt.Sprintf("(Forbidden) Policy operation is associated with tier %s. "+
				"User \"%s\" cannot get tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)",
				tierName, attrs.User.GetName())
		}
		return errors.NewForbidden(calico.Resource("tiers"), tierName, fmt.Errorf(reason))

	}
	return nil
}

func GetTierPolicy(policyName string) (string, string) {
	policySlice := strings.Split(policyName, policyDelim)
	if len(policySlice) < 2 {
		return "default", policySlice[0]
	}
	return policySlice[0], policySlice[1]
}
