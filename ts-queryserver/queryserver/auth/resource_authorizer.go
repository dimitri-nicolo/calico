package auth

import (
	"context"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
)

type Permission interface {
	IsAuthorized(res api.Resource, tier *string, verbs []string) bool
}
type permission struct {
	APIGroupsPermissions map[string]ResourcePermissions // apiGroup string --> ResourcePermission
}

type ResourcePermissions map[string]VerbPermissions          // resource name string -->
type VerbPermissions map[string][]v3.AuthorizedResourceGroup // verb string --> []ResourceGroup
type AuthorizationVerb string
type APIGroupResourceName string

func getCombinedName(apiGroup string, resourceName string) APIGroupResourceName {
	return APIGroupResourceName(strings.Join([]string{apiGroup, resourceName}, "/"))
}

// IsAuthorized is checking if current users' permissions allows either of the verbs passed in the param on the resource passed in.
func (p *permission) IsAuthorized(res api.Resource, tier *string, verbs []string) bool {
	combinedName := getCombinedName(
		getAPIGroup(res.GetObjectKind().GroupVersionKind().Group),
		convertV1KindToResourceType(res.GetObjectKind().GroupVersionKind().Kind, res.GetObjectMeta().GetName()))

	if verbsMap, ok := p.APIGroupsResourceNamePermissions[combinedName]; ok {
		for _, v := range verbs {
			if resourceGrps, ok := verbsMap[AuthorizationVerb(v)]; ok {
				for _, resourceGrp := range resourceGrps {
					if resourceGrp.Namespace == "" && resourceGrp.Tier == "" {
						return true
					}
					if resourceGrp.Namespace != "" && resourceGrp.Tier == "" {
						if namespaceMatch(res.GetObjectMeta().GetNamespace(), resourceGrp.Namespace) {
							return true
						}
					}
					if resourceGrp.Namespace == "" && resourceGrp.Tier != "" {
						if tier != nil {
							if tierMatch(*tier, resourceGrp.Tier) {
								return true
							}
						}
					}
					if resourceGrp.Namespace != "" && resourceGrp.Tier != "" {
						if tier != nil {
							if namespaceMatch(res.GetObjectMeta().GetNamespace(), resourceGrp.Namespace) &&
								tierMatch(*tier, resourceGrp.Tier) {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func namespaceMatch(ns1, ns2 string) bool {
	return ns1 == ns2
}

func tierMatch(tier1, tier2 string) bool {
	return tier1 == tier2
}

func getAPIGroup(apigroup string) string {
	return strings.ToLower(apigroup)
}

// convertV1KindToResourceType converts the kind stored in the V1 resource to the actual type present
// in the authorizationreview response.
func convertV1KindToResourceType(kind string, name string) string {
	kind = strings.ToLower(kind)

	// needs to be checked to determine if the policy is of type "Staged"
	if strings.HasPrefix(name, "staged:") && !strings.HasPrefix(kind, "staged") {
		kind = "staged" + kind
	}

	switch kind {
	case "stagedglobalnetworkpolicies", "stagedglobalnetworkpolicy":
		return "stagedglobalnetworkpolicies"
	case "stagednetworkpolicies", "stagednetworkpolicy":
		return "stagednetworkpolicies"
	case "stagedkubernetesnetworkpolicies", "stagedkubernetesnetworkpolicy":
		return "stagedkubernetesnetworkpolicies"
	case "globalnetworkpolicies", "globalnetworkpolicy":
		return "globalnetworkpolicies"
	case "networkpolicies", "networkpolicy":
		return "networkpolicies"
	case "globalnetworksets", "globalnetworkset":
		return "globalnetworksets"
	case "networksets", "networkset":
		return "networksets"
	case "tiers", "tier":
		return "tiers"
	case "pods", "pod":
		return "pods"
	default:
		return kind
	}

}

type Authorizer interface {
	PerformUserAuthorizationReview(ctx context.Context,
		authreviewList []v3.AuthorizationReviewResourceAttributes) (Permission, error)
}

type authorizer struct {
	clientSetFactory k8s.ClientSetFactory
}

func NewAuthorizer(clsfactory k8s.ClientSetFactory) Authorizer {
	return &authorizer{
		clientSetFactory: clsfactory,
	}
}

// PerformUserAuthorizationReview, creates an authorizationreview for the passed in authreviewattributes and
// build permission based on the results.
//
// returns permission, error
func (authz *authorizer) PerformUserAuthorizationReview(ctx context.Context,
	authReviewattributes []v3.AuthorizationReviewResourceAttributes) (Permission, error) {

	user, ok := request.UserFrom(ctx)
	if !ok {
		// There should be user info in the request context. If not this is server error since an earlier handler
		// should have authenticated.
		log.Debug("No user information on request")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    "No user information on request",
		}
	}

	// since each cluster has its own queryserver, we do not need to pass clusterID to get the clientSet.
	cs, err := authz.clientSetFactory.NewClientSetForApplication("")
	if err != nil {
		return nil, err
	}

	// we cannot use the current context to PerformAuthorizationReviewContext because it contains the userInfo and queryserver
	// does not have impersonate rbac to execute calls on behalf of users. However, we still need to run AuthorizationReview
	// for the user. Thus, we use PerformAuthroizationReviewWithUser which is using the background context to execute the request
	// and allows us to pass in user info to be used in the authorization.
	authorizedResourceVerbs, err := auth.PerformAuthorizationReviewWithUser(user, cs, authReviewattributes)

	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    "Unable to perform authorization review",
		}
	}

	return convertAuthorizationReviewStatusToPermissions(authorizedResourceVerbs)
}

// function convertAuthorizationReviewStatusToPermissions converts AuthorizedResourceVerbs to Permission (map of resource groups / name -> verb -> authorizedResourceGroup) for
// faster lookup.
func convertAuthorizationReviewStatusToPermissions(authorizedResourceVerbs []v3.AuthorizedResourceVerbs) (Permission, error) {
	permMap := permission{
		APIGroupsResourceNamePermissions: map[APIGroupResourceName]map[AuthorizationVerb][]v3.AuthorizedResourceGroup{},
	}
	for _, rAtt := range authorizedResourceVerbs {
		combinedName := getCombinedName(rAtt.APIGroup, rAtt.Resource)
		if _, ok := permMap.APIGroupsResourceNamePermissions[combinedName]; !ok {
			permMap.APIGroupsResourceNamePermissions[combinedName] = map[AuthorizationVerb][]v3.AuthorizedResourceGroup{}
		}
		for _, verb := range rAtt.Verbs {
			resourceGroups := []v3.AuthorizedResourceGroup{}
			if _, ok := permMap.APIGroupsResourceNamePermissions[combinedName][AuthorizationVerb(verb.Verb)]; ok {
				resourceGroups = permMap.APIGroupsResourceNamePermissions[combinedName][AuthorizationVerb(verb.Verb)]
			}
			resourceGroups = append(resourceGroups, verb.ResourceGroups...)
			permMap.APIGroupsResourceNamePermissions[combinedName][AuthorizationVerb(verb.Verb)] = resourceGroups
		}
	}

	return &permMap, nil
}

var PolicyAuthReviewAttrList = []v3.AuthorizationReviewResourceAttributes{
	{
		APIGroup: "projectcalico.org",
		Resources: []string{
			"stagednetworkpolicies", "stagedglobalnetworkpolicies", "stagedkubernetesnetworkpolicies",
			"globalnetworkpolicies", "networkpolicies", "networksets", "globalnetworksets",
			"tiers",
		},
		Verbs: []string{"watch", "get", "delete", "create", "update", "list", "patch"},
	},
	{
		APIGroup:  "networking.k8s.io",
		Resources: []string{"networkpolicies"},
		Verbs:     []string{"watch", "get", "delete", "create", "update", "list", "patch"},
	},
}
