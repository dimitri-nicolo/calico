package middleware

import (
	"net/http"
	"strings"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/compliance/pkg/resources"
	authzv1 "k8s.io/api/authorization/v1"
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
	CanPreviewPolicyAction(action string, policy resources.Resource) (bool, error)
}

// policyImpactRbacHelper is used by a single API request to to determine if a user can
// view and modify a policy
type policyImpactRbacHelper struct {
	Request *http.Request
	k8sAuth K8sAuthInterface
}

// CanPreviewPolicyAction returns true if the user can perform the preview action on the requested
// policy. Regardless of action ability to view the policy is also required
func (h *policyImpactRbacHelper) CanPreviewPolicyAction(action string, policy resources.Resource) (bool, error) {

	log.Debugf("Checking policy impact permissions %v %v", action, policy.GetObjectMeta().GetName())

	// To preview anything we must be able to view the policy
	canViewPolicy, err := h.canPerformPolicyAction("get", policy)
	if err != nil {
		return false, err
	}
	if !canViewPolicy {
		return false, nil
	}

	// We also must be able to perform the action we are attempting to preview.
	return h.canPerformPolicyAction(action, policy)
}

func (h *policyImpactRbacHelper) canPerformPolicyAction(verb string, res resources.Resource) (bool, error) {
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
		return false, nil
	}

	// If this is a Calico tiered policy then extract the tier since we need that to perform some more complicated
	// authz on top of the default authz that we'll perform for *all* resource types. We extract the tier from the
	// policy name rather than from the Spec, since with the latter approach it would be possible to fool the checks by
	// simply not specifying the Spec.Tier field value.
	var tier string
	switch res.(type) {
	case *v3.NetworkPolicy, *v3.GlobalNetworkPolicy:
		// Split the name by ".". The tier is the first part - so there should be at least two parts (a string cannot
		// be split into 0 parts - so check if there is only one part), and the tier should not be blank.
		parts := strings.Split(rid.Name, ".")
		tier = parts[0]
		if len(parts) == 1 || tier == "" {
			clog.Warning("Resource name is not valid for resource type")
			return false, nil
		}
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
	if auth, err := h.checkAuthorized(*resAtr); err != nil {
		return false, err
	} else if !auth {
		clog.Debug("Action not authorized")
		return false, nil
	}

	if tier == "" {
		// There is no tier, this means the resource is not a Calico tiered policy resource (i.e. it's neither a Calico
		// NetworkPolicy nor GlobalNetworkPolicy). Since we have already performed authz checks on the resource above,
		// there is nothing else to do here.
		clog.Debug("Action authorized for non-tiered policy")
		return true, nil
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
	if auth, err := h.checkAuthorized(*resAtr); err != nil {
		return false, err
	} else if !auth {
		clog.Debug("Action not authorized for the tier")
		return false, nil
	}

	// Authorized for tier access, check wildcard policy access in this tier.
	resAtr = &authzv1.ResourceAttributes{
		Verb:      verb,
		Group:     v3.Group,
		Resource:  "tier." + rh.Plural(),
		Name:      tier + ".*",
		Namespace: rid.Namespace,
	}
	if auth, err := h.checkAuthorized(*resAtr); err != nil {
		return false, err
	} else if auth {
		clog.Debug("Action authorized for all policies of this type in this tier and namespace")
		return true, nil
	}

	// Not authorized for wildcard policy access in this tier, checking access for the specific policy.
	resAtr = &authzv1.ResourceAttributes{
		Verb:      verb,
		Group:     v3.Group,
		Resource:  "tier." + rh.Plural(),
		Name:      rid.Name,
		Namespace: rid.Namespace,
	}
	if auth, err := h.checkAuthorized(*resAtr); err != nil {
		return false, err
	} else if auth {
		clog.Debug("Action authorized for specific policy")
		return true, nil
	}

	// Action not authorized on this tiered policy.
	clog.Debug("Action not authorized for tiered policy")
	return false, nil
}

// checkAuthorized returns true if the request is allowed for the resources decribed in the attributes
func (h *policyImpactRbacHelper) checkAuthorized(atr authzv1.ResourceAttributes) (bool, error) {

	ctx := NewContextWithReviewResource(h.Request.Context(), &atr)
	req := h.Request.WithContext(ctx)

	stat, err := h.k8sAuth.Authorize(req)
	//we check stat first because in this case err contains details if the auth failed
	//but it won't be nil when we have a statusForbidden which is fine
	switch stat {
	case 0:
		log.WithField("stat", stat).Info("Request authorized")
		return true, nil
	case http.StatusForbidden:
		log.WithField("stat", stat).WithError(err).Info("Forbidden - not authorized")
		return false, nil
	}
	log.WithField("stat", stat).WithError(err).Info("Error authorizing")
	return false, err
}
