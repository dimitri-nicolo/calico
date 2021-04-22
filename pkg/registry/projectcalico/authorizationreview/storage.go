// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package authorizationreview

import (
	"context"
	"sort"

	calico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
	"github.com/projectcalico/apiserver/pkg/rbac"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type REST struct {
	calculator rbac.Calculator
}

// EmptyObject returns an empty instance
func (r *REST) New() runtime.Object {
	return &calico.AuthorizationReview{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.AuthorizationReviewList{}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(calculator rbac.Calculator) *REST {
	return &REST{calculator: calculator}
}

// Necessary to satisfy generated informers, but not intended for real use.
func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return NewList(), nil
}

// Necessary to satisfy generated informers, but not intended for real use.
func (r *REST) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	return watch.NewEmptyWatch(), nil
}

// Takes the userinfo that the authn delegate has put into the context and returns it.
func (r *REST) Create(ctx context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	in := obj.(*calico.AuthorizationReview)
	out := &calico.AuthorizationReview{
		TypeMeta:   in.TypeMeta,
		ObjectMeta: in.ObjectMeta,
		Spec:       in.Spec,
	}

	// Extract user info from the request.
	user, ok := request.UserFrom(ctx)
	if !ok {
		return out, nil
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
	results, err := r.calculator.CalculatePermissions(user, rvs)
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
				return ms[i].Tier < ms[j].Tier
			})

			for _, m := range ms {
				rgs = append(rgs, v3.AuthorizedResourceGroup{
					Tier:      m.Tier,
					Namespace: m.Namespace,
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

func (r *REST) NamespaceScoped() bool {
	return false
}
