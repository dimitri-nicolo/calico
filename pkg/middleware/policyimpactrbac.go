package middleware

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	authzv1 "k8s.io/api/authorization/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/resources"
)

type PolicyImpactRbacHelperFactory interface {
	NewPolicyImpactRbacHelper(*http.Request) PolicyImpactRbacHelper
}

type standardPolicyImpactRbacHelperFactor struct {
	auth K8sAuthInterface
}

func (s *standardPolicyImpactRbacHelperFactor) NewPolicyImpactRbacHelper(req *http.Request) PolicyImpactRbacHelper {
	return &policyImpactRbacHelper{
		Request: req,
		k8sAuth: s.auth,
	}
}

func NewStandardPolicyImpactRbacHelperFactory(auth K8sAuthInterface) PolicyImpactRbacHelperFactory {
	return &standardPolicyImpactRbacHelperFactor{auth: auth}
}

type PolicyImpactRbacHelper interface {
	CheckCanPreviewPolicyAction(action string, policy resources.Resource) (status int, err error)
}

// policyImpactRbacHelper is used by a single API request to to determine if a user can
// view and modify a policy
type policyImpactRbacHelper struct {
	Request *http.Request
	k8sAuth K8sAuthInterface
}

// CheckCanPreviewPolicyAction returns true if the user can perform the preview action on the requested
// policy. If the user is not permitted, an error detailing the reason is returned.
func (h *policyImpactRbacHelper) CheckCanPreviewPolicyAction(action string, policy resources.Resource) (status int, err error) {
	// We must be able to perform the action we are attempting to preview.
	return h.checkCanPerformPolicyAction(action, policy)
}

func (h *policyImpactRbacHelper) checkCanPerformPolicyAction(verb string, res resources.Resource) (status int, err error) {
	rid := resources.GetResourceID(res)
	clog := log.WithFields(log.Fields{
		"verb":     verb,
		"resource": rid,
	})
	clog.Debug("Checking policy action permissions")

	// Get the resource helper from the resource type meta. If the resource helper is not found then this is not a
	// resource type we support.
	//
	// Note that the PIP policy calculator supports more than just Calico and Kubernetes resources types - it can take
	// updates of any resource that is used by the policy calculator, hence the code here is pretty generic.
	rh := resources.GetResourceHelperByTypeMeta(rid.TypeMeta)
	if rh == nil {
		// This is not a resource type we support, so deny the operation.
		clog.Warning("Resource type is not supported for preview action")
		return http.StatusBadRequest, fmt.Errorf("resource type '" + rid.Kind + "' is not supported for impact preview")
	}

	// If this is a Calico tiered policy then extract the tier since we need that to perform some more complicated
	// authz on top of the default authz that we'll perform for *all* resource types.
	tier, err := getTier(rid, res)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// Always perform the default authorization check on the resource. We do this for *all* resource types. This checks
	// whether the user can perform the requested action on the resource, and is the only authz check we need to do for
	// Kubernetes policies and other non-Calico tiered policy resource types.
	resAtr := &authzv1.ResourceAttributes{
		Verb:      verb,
		Group:     res.GetObjectKind().GroupVersionKind().Group,
		Resource:  rh.Plural(),
		Name:      rid.Name,
		Namespace: rid.Namespace,
	}

	if stat, err := h.isAuthorized(*resAtr); err != nil {
		return stat, fmt.Errorf("not authorized to " + verb + " " + rid.String())
	}

	if tier == "" {
		// There is no tier, this means the resource is not a Calico tiered policy resource (i.e. it's neither a Calico
		// NetworkPolicy nor GlobalNetworkPolicy). Since we have already performed authz checks on the resource above,
		// there is nothing else to do here.
		clog.Debug("Action authorized for non-tiered policy")
		return 0, nil
	}

	// This is a Calico tiered policy. We need to perform three additional checks that can further restrict the users
	// access to the policy:
	// - User has read access to the tier
	// - User either has:
	//   - Wildcarded access to tiered-policies in the tier
	//   - Access to the specific tiered-policy in the tier
	// These checks are not performed on Kubernetes network policies or any other Kubernetes resource. Note that a
	// different resource kind is used to determine access to "tiered-policy" - this pseudo resource is the same as the
	// underlying resource kind but prefixed with "tier.", i.e. "tier.networkpolicies" and "tier.globalnetworkpolicies".
	//
	// Start by checking tier read access.
	resAtr = &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    v3.Group,
		Resource: "tiers",
		Name:     tier,
	}

	if stat, err := h.isAuthorized(*resAtr); err != nil {
		// return http.StatusForbidden, fmt.Errorf("not authorized to " + verb + " " + rid.String() + ": user cannot get tier " + tier)
		return stat, err
	}

	// Authorized for tier access, check wildcard policy access in this tier.
	resAtr = &authzv1.ResourceAttributes{
		Verb:      verb,
		Group:     v3.Group,
		Resource:  "tier." + rh.Plural(),
		Name:      tier + ".*",
		Namespace: rid.Namespace,
	}
	if stat, err := h.isAuthorized(*resAtr); err != nil {
		if stat != http.StatusForbidden {
			return stat, err
		}
	} else {
		clog.Debug("Action authorized for all policies of this type in this tier and namespace")
		return 0, nil
	}

	// Not authorized for wildcard policy access in this tier.
	// before we return 'unauthorized, check access for the specific policy.
	resAtr = &authzv1.ResourceAttributes{
		Verb:      verb,
		Group:     v3.Group,
		Resource:  "tier." + rh.Plural(),
		Name:      rid.Name,
		Namespace: rid.Namespace,
	}
	if stat, err := h.isAuthorized(*resAtr); err != nil {
		return stat, err
		// stat, fmt.Errorf("not authorized to " + verb + " " + rid.String())
	}
	// Action not authorized on this tiered policy.
	clog.Debug("Action not authorized for tiered policy")
	return 0, nil
}

// isAuthorized returns true if the request is allowed for the resources decribed in the attributes
func (h *policyImpactRbacHelper) isAuthorized(atr authzv1.ResourceAttributes) (int, error) {
	ctx := NewContextWithReviewResource(h.Request.Context(), &atr)
	req := h.Request.WithContext(ctx)

	return h.k8sAuth.Authorize(req)
}

// getTier extracts the tier from a Calico tiered policy. If the resource is not a Calico tiered policy an empty string
// and no error are returned. If an invalid tier is found then an error is returned.
func getTier(rid v3.ResourceID, res resources.Resource) (tier string, err error) {
	switch np := res.(type) {
	case *v3.NetworkPolicy:
		tier = np.Spec.Tier
	case *v3.GlobalNetworkPolicy:
		tier = np.Spec.Tier
	default:
		// Not a calico tiered policy, so return nothing.
		return "", nil
	}

	// Sanity check the tier in the spec matches the tier in the name. This has already been done by the unmarshaling
	// of the resource request, but better safe than sorry.
	if tier == "" || strings.Contains(tier, ".") || !strings.HasPrefix(rid.Name, tier+".") {
		return "", fmt.Errorf("policy name " + rid.String() + " is not correct for the configured tier '" + tier + "'")
	}
	return tier, nil
}
