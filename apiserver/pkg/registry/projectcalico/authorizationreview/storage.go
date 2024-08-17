// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package authorizationreview

import (
	"context"
	"sort"

	"github.com/projectcalico/calico/apiserver/pkg/rbac"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type REST struct {
	calculator rbac.Calculator
}

// EmptyObject returns an empty instance
func (r *REST) New() runtime.Object {
	return &v3.AuthorizationReview{}
}

func (r *REST) Destroy() {

}

// Takes the userinfo that the authn delegate has put into the context and returns it.
func (r *REST) Create(ctx context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	in := obj.(*v3.AuthorizationReview)
	out := &v3.AuthorizationReview{
		TypeMeta:   in.TypeMeta,
		ObjectMeta: in.ObjectMeta,
		Spec:       in.Spec,
	}

	var userInfo user.Info

	if in.Spec.User != "" {
		// Extract user from spec
		userInfo = &user.DefaultInfo{
			Name:   in.Spec.User,
			UID:    in.Spec.UID,
			Groups: in.Spec.Groups,
		}
	} else {
		// Extract user info from the request context.
		var ok bool
		userInfo, ok = request.UserFrom(ctx)
		if !ok {
			return out, nil
		}
	}

	// Expand the request into a set of ResourceVerbs as input to the RBAC calculator.
	rvs := []rbac.ResourceVerbs{}
	for _, ra := range in.Spec.ResourceAttributes {
		if len(ra.Verbs) == 0 || len(ra.Resources) == 0 {
			continue
		}

		verbs := make([]rbac.Verb, len(ra.Verbs))
		for i := range ra.Verbs {
			verbs[i] = rbac.Verb(ra.Verbs[i])
		}

		for _, r := range ra.Resources {
			rvs = append(rvs, rbac.ResourceVerbs{
				ResourceType: rbac.ResourceType{
					APIGroup: ra.APIGroup,
					Resource: r,
				},
				Verbs: verbs,
			})
		}
	}

	// Calculate the set of permissions.
	results, err := r.calculator.CalculatePermissions(userInfo, rvs)
	if err != nil {
		return nil, err
	}

	// Transfer the results to the status. Sort the results to ensure deterministic data. Start by ordering the
	// resource type info.
	rts := make([]rbac.ResourceType, 0, len(results))
	for rt := range results {
		rts = append(rts, rt)
	}
	sort.Slice(rts, func(i, j int) bool {
		if rts[i].APIGroup < rts[j].APIGroup {
			return true
		} else if rts[i].APIGroup > rts[j].APIGroup {
			return false
		}
		return rts[i].Resource < rts[j].Resource
	})

	// Grab the results for each resource type.
	for _, rt := range rts {
		vms := results[rt]

		res := v3.AuthorizedResourceVerbs{
			APIGroup: rt.APIGroup,
			Resource: rt.Resource,
		}

		// Order the verbs.
		verbs := make([]string, 0, len(vms))
		for v := range vms {
			verbs = append(verbs, string(v))
		}
		sort.Strings(verbs)

		for _, v := range verbs {
			// Grab the authorization matches for the verb and order them before adding to the status.
			ms := vms[rbac.Verb(v)]
			var rgs []v3.AuthorizedResourceGroup

			sort.Slice(ms, func(i, j int) bool {
				if ms[i].Namespace < ms[j].Namespace {
					return true
				} else if ms[i].Namespace > ms[j].Namespace {
					return false
				}
				if ms[i].Tier < ms[j].Tier {
					return true
				} else if ms[i].Tier > ms[j].Tier {
					return false
				}
				return ms[i].UISettingsGroup < ms[j].UISettingsGroup
			})

			for _, m := range ms {
				rgs = append(rgs, v3.AuthorizedResourceGroup{
					Tier:            m.Tier,
					Namespace:       m.Namespace,
					UISettingsGroup: m.UISettingsGroup,
					ManagedCluster:  m.ManagedCluster,
				})
			}
			res.Verbs = append(res.Verbs, v3.AuthorizedResourceVerb{
				Verb:           string(v),
				ResourceGroups: rgs,
			})
		}

		out.Status.AuthorizedResourceVerbs = append(out.Status.AuthorizedResourceVerbs, res)
	}

	return out, nil
}

func (r *REST) GetSingularName() string {
	return "authorizationreview"
}

func (r *REST) NamespaceScoped() bool {
	return false
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &v3.AuthorizationReviewList{}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(calculator rbac.Calculator) *REST {
	return &REST{calculator: calculator}
}
