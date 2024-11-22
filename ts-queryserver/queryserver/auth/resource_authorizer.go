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
	IsAuthorized(res api.Resource, verb, tier string) bool
}
type permission struct {
	APIGroupsPermissions map[string]ResourcePermissions // apiGroup string --> ResourcePermission
}

type ResourcePermissions map[string]VerbPermissions          // resource name string -->
type VerbPermissions map[string][]v3.AuthorizedResourceGroup // verb string --> []ResourceGroup

// IsAuthorized is checking if current users' permissions allows either of the verbs passed in the param on the resource passed in.
func (p *permission) IsAuthorized(res api.Resource, verb, tier string) bool {
	if rscMap, ok := p.APIGroupsPermissions[getAPIGroup(res.GetObjectKind().GroupVersionKind().Group)]; ok {
		if verbsMap, ok := rscMap[convertV1KindToResourceType(res.GetObjectKind().GroupVersionKind().Kind,
			res.GetObjectMeta().GetName())]; ok {
			if authorizedRscGrps, ok := verbsMap[verb]; ok {
				for _, r := range authorizedRscGrps {
					// check namespace and tier
					if isNamespaceAllowed(r, res) && isTierAllowed(r, tier) {
						return true
					}
				}
			}
		}
	}
	return false
}

func isNamespaceAllowed(authorizedRscGrp v3.AuthorizedResourceGroup, res api.Resource) bool {

	return authorizedRscGrp.Namespace == "" || (authorizedRscGrp.Namespace == res.GetObjectMeta().GetNamespace())

}

func isTierAllowed(authorizedRscGrp v3.AuthorizedResourceGroup, tier string) bool {
	return authorizedRscGrp.Tier == "" || authorizedRscGrp.Tier == tier
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
	case "globalnetworsets", "globalnetworset":
		return "globalnetworsets"
	case "networksets", "networkset":
		return "networksets"
	case "tiers", "tier":
		return "tiers"
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

func convertAuthorizationReviewStatusToPermissions(authorizedResourceVerbs []v3.AuthorizedResourceVerbs) (Permission, error) {
	permMap := permission{
		APIGroupsPermissions: map[string]ResourcePermissions{},
	}
	for _, rAtt := range authorizedResourceVerbs {
		if _, ok := permMap.APIGroupsPermissions[rAtt.APIGroup]; !ok {
			permMap.APIGroupsPermissions[rAtt.APIGroup] = ResourcePermissions{}
		}
		if _, ok := permMap.APIGroupsPermissions[rAtt.APIGroup][rAtt.Resource]; !ok {
			permMap.APIGroupsPermissions[rAtt.APIGroup][rAtt.Resource] = map[string][]v3.AuthorizedResourceGroup{}
		}
		for _, verb := range rAtt.Verbs {
			for _, rg := range verb.ResourceGroups {
				if _, ok := permMap.APIGroupsPermissions[rAtt.APIGroup][rAtt.Resource][verb.Verb]; !ok {
					permMap.APIGroupsPermissions[rAtt.APIGroup][rAtt.Resource][verb.Verb] = make([]v3.AuthorizedResourceGroup, 0)
				}
				resourceGroups := permMap.APIGroupsPermissions[rAtt.APIGroup][rAtt.Resource][verb.Verb]
				resourceGroups = append(resourceGroups, rg)
				permMap.APIGroupsPermissions[rAtt.APIGroup][rAtt.Resource][verb.Verb] = resourceGroups
			}
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
