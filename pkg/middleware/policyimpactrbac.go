package middleware

import (
	"fmt"
	"net/http"

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

	//to preview anything we must be able to view the policy
	canViewPolicy, err := h.canPerformPolicyAction("get", policy)
	if err != nil {
		return false, err
	}
	if !canViewPolicy {
		return false, nil
	}

	//we also must be able to perform the action we are attempting to preview
	return h.canPerformPolicyAction(action, policy)
}

func (h *policyImpactRbacHelper) canPerformPolicyAction(verb string, policy resources.Resource) (bool, error) {
	log.Debugf("Checking policy action permissions %v %v", verb, policy.GetObjectMeta().GetName())

	group := policy.GetObjectKind().GroupVersionKind().Group
	kind := policy.GetObjectKind().GroupVersionKind().Kind
	name := policy.GetObjectMeta().GetName()
	namespace := policy.GetObjectMeta().GetNamespace()

	var res string
	if group == "networking.k8s.io" && kind == "NetworkPolicy" {
		//k8s network policies
		res = "networkpolicies"
	} else if group == "projectcalico.org" && kind == "NetworkPolicy" {
		//calico network policy
		res = "tier.networkpolicies"
		np := policy.(*v3.NetworkPolicy)
		name = fmt.Sprintf("%v.*", np.Spec.Tier)
	} else if group == "projectcalico.org" && kind == "GlobalNetworkPolicy" {
		//calico global network policy
		res = "tier.globalnetworkpolicies"
		name = ""
		namespace = ""
	}

	log.WithFields(log.Fields{
		"verb":      verb,
		"group":     group,
		"kind":      kind,
		"resource":  res,
		"name":      name,
		"namespace": namespace,
	}).Debug("CAN-I")

	resAtr := &authzv1.ResourceAttributes{
		Verb:      verb,
		Group:     group,
		Resource:  res,
		Name:      name,
		Namespace: namespace,
	}
	return h.checkAuthorized(*resAtr)
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
