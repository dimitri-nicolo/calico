// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package rbac

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/pkg/registry/rbac/validation"
	rbac_auth "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"

	projectcalicov3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"

	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/resources"
)

// Verb is a bit-wise set of available verbs for Kubernetes RBAC. Use Verbs() to convert to a slice of strings.
type Verb string

const (
	VerbGet    Verb = "get"
	VerbList   Verb = "list"
	VerbUpdate Verb = "update"
	VerbCreate Verb = "create"
	VerbPatch  Verb = "patch"
	VerbDelete Verb = "delete"
	VerbWatch  Verb = "watch"
)

const (
	resourceTiers = "tiers"
)

var (
	AllVerbs = []Verb{
		VerbGet, VerbList, VerbUpdate, VerbCreate, VerbPatch, VerbDelete, VerbWatch,
	}

	// DefaultResources is the standard set of resources required by multiple Calico components.
	DefaultResources = []metav1.TypeMeta{
		resources.TypeCalicoHostEndpoints,
		resources.TypeK8sPods,
		resources.TypeCalicoTiers,
		resources.TypeK8sNetworkPolicies,
		resources.TypeCalicoStagedKubernetesNetworkPolicies,
		resources.TypeCalicoNetworkPolicies,
		resources.TypeCalicoStagedNetworkPolicies,
		resources.TypeCalicoGlobalNetworkPolicies,
		resources.TypeCalicoStagedGlobalNetworkPolicies,
		resources.TypeCalicoNetworkSets,
		resources.TypeCalicoGlobalNetworkSets,
	}

	matchAll = Match{}

	typeMetaToResourceTypes map[metav1.TypeMeta][]ResourceType
	resourceTypeToHelper    map[ResourceType]resources.ResourceHelper
)

func init() {
	// Resource types may exist under multiple api groups (e.g. extensions, networking.k8s.io). Populate our lookups so that
	// we provide the full set of resources in the query.
	typeMetaToResourceTypes = make(map[metav1.TypeMeta][]ResourceType)
	resourceTypeToHelper = make(map[ResourceType]resources.ResourceHelper)

	for _, t := range resources.GetAllResourceHelpers() {
		tm := t.TypeMeta()
		gv, _ := schema.ParseGroupVersion(tm.APIVersion)
		rt := ResourceType{APIGroup: gv.Group, Resource: t.Plural()}
		typeMetaToResourceTypes[tm] = append(typeMetaToResourceTypes[tm], rt)
		resourceTypeToHelper[rt] = t

		for _, dtm := range t.Deprecated() {
			gv, _ := schema.ParseGroupVersion(dtm.APIVersion)
			rt := ResourceType{APIGroup: gv.Group, Resource: t.Plural()}
			typeMetaToResourceTypes[tm] = append(typeMetaToResourceTypes[tm], rt)
			resourceTypeToHelper[rt] = t
		}
	}
}

// Calculator provides methods to determine RBAC permissions for a user.
type Calculator interface {
	CalculatePermissions(user user.Info, rts []ResourceType, verbs []Verb) (Permissions, error)
	CalculatePermissionsForTypeMeta(user user.Info, tms []metav1.TypeMeta, verbs []Verb) (Permissions, error)
}

type Permissions map[ResourceType]map[Verb][]Match

type ResourceType struct {
	APIGroup string
	Resource string
}

func (rt ResourceType) MarshalText() ([]byte, error) {
	return []byte(rt.String()), nil
}

func (rt *ResourceType) UnmarshalText(b []byte) error {
	parts := strings.SplitN(string(b), ".", 2)
	rt.Resource = parts[0]
	if len(parts) == 2 {
		rt.APIGroup = parts[1]
	}
	return nil
}

func (r ResourceType) String() string {
	if r.APIGroup == "" {
		return r.Resource
	}
	return r.Resource + "." + r.APIGroup
}

type Match struct {
	Namespace string `json:"namespace"`
	Tier      string `json:"tier"`
}

func (m Match) String() string {
	return fmt.Sprintf("Match(tier=%s; namespace=%s)", m.Tier, m.Namespace)
}

// NewCalculator creates a new RBAC Calculator.
func NewCalculator(clusterRoleGetter ClusterRoleGetter, clusterRoleBindingLister ClusterRoleBindingLister,
	roleGetter RoleGetter, roleBindingLister RoleBindingLister,
	namespaceLister NamespaceLister, tierLister TierLister) Calculator {

	// Split out the cluster and namespaced rule resolvers - this allows us to perform namespace queries without
	// checking cluster rules every time. For cluster specific rule resolver, use a "no-op" RuleBindingLister - this
	// is a lister that returns no RuleBindings, the upshot is that only ClusterRoleBindings are discovered by the
	// RuleResolver meaning only cluster-scoped rules will be considered.  Similarly, for the namespaced rule resolver,
	// use a no-op ClusterRuleBindingLister - this is a lister that returns no ClusterRuleBindings and so the
	// RuleResolver will only discover namespaced rules (and no cluster-scoped rules).
	clusterRuleResolver := validation.NewDefaultRuleResolver(
		roleGetter, &emptyK8sRoleBindingLister{}, clusterRoleGetter, clusterRoleBindingLister,
	)
	namespacedRuleResolver := validation.NewDefaultRuleResolver(
		roleGetter, roleBindingLister, clusterRoleGetter, &emptyK8sClusterRoleBindingLister{},
	)

	return &calculator{
		namespaceLister:        namespaceLister,
		tierLister:             tierLister,
		clusterRuleResolver:    clusterRuleResolver,
		namespacedRuleResolver: namespacedRuleResolver,
	}

}

// RoleGetter interface is used to get a specific Role.
type RoleGetter interface {
	GetRole(namespace, name string) (*rbacv1.Role, error)
}

// RoleBindingLister interface is used to list all RoleBindings in a specific namespace.
type RoleBindingLister interface {
	ListRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error)
}

// ClusterRoleGetter interface is used to get a specific ClusterRole.
type ClusterRoleGetter interface {
	GetClusterRole(name string) (*rbacv1.ClusterRole, error)
}

// ClusterRoleBindingLister interface is used to list all ClusterRoleBindings.
type ClusterRoleBindingLister interface {
	ListClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error)
}

// NamespaceLister interface is used to list all Namespaces.
type NamespaceLister interface {
	ListNamespaces() ([]*corev1.Namespace, error)
}

// TierLister interface is used to list all Tiers.
type TierLister interface {
	ListTiers() ([]*projectcalicov3.Tier, error)
}

// calculator implements the Calculator interface.
type calculator struct {
	namespaceLister        NamespaceLister
	tierLister             TierLister
	clusterRuleResolver    validation.AuthorizationRuleResolver
	namespacedRuleResolver validation.AuthorizationRuleResolver
}

// CalculatePermissions calculates the RBAC permissions for a specific user and set of resource types.
func (c *calculator) CalculatePermissions(user user.Info, rts []ResourceType, verbs []Verb) (Permissions, error) {
	log.Debugf("CalculatePermissions for %v", user)

	r := c.newUserCalculator(user)
	for _, rt := range rts {
		r.updatePermissions(rt, verbs)
	}

	// Return the authorized verbs for the user and any errors that were hit.
	return r.permissions, utilerrors.Flatten(utilerrors.NewAggregate(r.errors))
}

func (c *calculator) CalculatePermissionsForTypeMeta(user user.Info, tms []metav1.TypeMeta, verbs []Verb) (Permissions, error) {
	log.Debugf("CalculatePermissionsForTypes for %v", user)

	r := c.newUserCalculator(user)
	for _, tm := range tms {
		for _, rt := range typeMetaToResourceTypes[tm] {
			r.updatePermissions(rt, verbs)
		}
	}

	// Return the authorized verbs for the user and any errors that were hit.
	return r.permissions, utilerrors.Flatten(utilerrors.NewAggregate(r.errors))

}

// emptyK8sRoleBindingLister implements the RoleBindingLister interface returning no RoleBindings.
type emptyK8sRoleBindingLister struct{}

func (_ *emptyK8sRoleBindingLister) ListRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error) {
	return nil, nil
}

// emptyK8sClusterRoleBindingLister implements the ClusterRoleBindingLister interface returning no ClusterRoleBindings.
type emptyK8sClusterRoleBindingLister struct{}

func (_ *emptyK8sClusterRoleBindingLister) ListClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error) {
	return nil, nil
}

// newUserCalculator returns a new Calculator accumulator. This is used to gather user specific Calculator permissions.
func (c *calculator) newUserCalculator(user user.Info) *userCalculator {
	return &userCalculator{
		user:        user,
		calculator:  c,
		permissions: make(Permissions),
	}
}

// userCalculator is used to gather user specific Calculator permissions.
type userCalculator struct {
	user              user.Info
	errors            []error
	calculator        *calculator
	allTiers          []string
	gettableTiers     []string
	clusterRules      []rbacv1.PolicyRule
	namespacedRules   map[string][]rbacv1.PolicyRule
	canGetAllTiersVal *bool
	permissions       Permissions
}

// accumulate updates the userCalculator with the permissions for the specific resource and verb.
func (u *userCalculator) getMatches(verb Verb, rt ResourceType, rh resources.ResourceHelper) []Match {
	gv, err := schema.ParseGroupVersion(rh.TypeMeta().APIVersion)
	if err != nil {
		// This is an internal interface and should never fail.
		log.Fatal("Unable to parse APIVersion")
		return nil
	}

	var matches []Match
	match := Match{}
	record := authorizer.AttributesRecord{
		Verb:            string(verb),
		APIGroup:        gv.Group,
		Resource:        rh.RbacPlural(),
		Name:            "",
		ResourceRequest: true,
	}

	// We potentially need to check for a combination of possible RBAC policies:
	// 1.  Cluster scope w/ wildcarded name
	// 2.  Cluster scope w/ specific name (*)
	// 3.  Namespaced w/ wildcarded name  (**)
	// 4.  Namespaced w/ specific name    (***)
	//
	// (*)   Only required for tiers and tiered policies
	// (**)  For tiered policies, this is expanded over specific tiers
	// (***) Only required for tiered policies
	//
	// We always expand out the tiers, so never return a wildcarded tier value for tiers or tiered policies.

	// Check if this rh is allowed cluster-wide with full wildcarded name. If it is then:
	if rbac_auth.RulesAllow(record, u.getClusterRules()...) {
		log.Debug("Full wildcard match")

		if !rh.IsTieredPolicy() {
			// This is not a tiered policy so include the match unchanged.
			log.Debug("Add wildcard namespace match for non-tiered policy")
			matches = append(matches, match)
		} else {
			// This is a tiered policy, expand the results by gettable tier.
			for _, tier := range u.getGettableTiers() {
				log.Debugf("Add wildcard namespace match for tiered-policy: tier=%s", tier)
				match.Tier = tier
				matches = append(matches, match)
			}
		}

		return matches
	}

	// Not allowed cluster-wide with full wildcarded-name. If we are handling tier resources, then do that processing
	// here since we'll want to expand across all gettable tiers, and the processing is different to tiered policies.
	if rh.TypeMeta() == resources.TypeCalicoTiers {
		log.Debug("Check individual gettable tiers")
		if verb == VerbGet {
			// Finesse the get case, since we already have to calculate the gettable tiers.
			for _, tier := range u.getGettableTiers() {
				match.Tier = tier
				matches = append(matches, match)
			}
		} else {
			for _, tier := range u.getAllTiers() {
				record.Name = tier
				if rbac_auth.RulesAllow(record, u.getClusterRules()...) {
					log.Debugf("Rules allow Tier(%s) cluster-wide", tier)
					match.Tier = tier
					matches = append(matches, match)
				}
			}
		}

		// Nothing else to do for tier resources.
		return matches
	}

	// If we are processing a tiered policy, check if the user has cluster-scoped access within the tier. We check this
	// by using a rh name of <tier-name>.*
	//
	// Note that if we do not have cluster-wide access for the tier then we'll need to check per-namespace, so we track
	// tiers that did not match cluster scoped.
	var tiersNoClusterMatch []string
	if rh.IsTieredPolicy() {
		log.Debug("Check cluster-scoped tiered-policy in each gettable tier")

		for _, tier := range u.getGettableTiers() {
			record.Name = tier + ".*"
			if rbac_auth.RulesAllow(record, u.getClusterRules()...) {
				log.Debugf("Rules allow cluster-scoped policy in tier(%s)", tier)
				match.Tier = tier
				matches = append(matches, match)
			} else {
				log.Debugf("Rules do not allow cluster-scoped policy in tier(%s)", tier)
				tiersNoClusterMatch = append(tiersNoClusterMatch, tier)
			}
		}

		if tiersNoClusterMatch == nil {
			// If every tier was specified individually and matched cluster-wide, then there is no point in checking
			// per-namespace.
			log.Debug("All tiers individually matched cluster-wide")
			return matches
		}
	}

	// If the rh is not namespaced then no more checks.
	if !rh.IsNamespaced() {
		log.Debug("Resource is not namespaced, so nothing left to check")
		return matches
	}

	// Now check namespaced matches. We limit this just to the namespaces that the user has rules for.
	for namespace, rules := range u.getNamespacedRules() {
		log.Debugf("Processing rules for namespace %s", namespace)
		match.Namespace = namespace
		match.Tier = ""
		record.Namespace = namespace
		record.Name = ""
		if rbac_auth.RulesAllow(record, rules...) {
			// The user is authorized for full wildcarded names for the rh type in this namespace
			// -  If this is a tiered policy then expand by tier.
			// -  Otherwise, include a single wildcarded name result.
			if !rh.IsTieredPolicy() {
				// This is not a tiered policy so include the match unchanged.
				log.Debug("Add namespaced match for non-tiered policy")
				matches = append(matches, match)
			} else {
				// This is a tiered policy, expand the results by gettable tier.
				log.Debug("Add namespaced match for all tiers for tiered policy")
				for _, tier := range tiersNoClusterMatch {
					log.Debugf("Add namespaced match for tiered policy: tier=%s", tier)
					match.Tier = tier
					matches = append(matches, match)
				}
			}

			// We matched wildcard name, so go to next namespace.
			continue
		}

		// Did not match wildcarded tier, so try individual tiers (that were not matched cluster-wide).
		// [note: tiersNoClusterMatch will be nil if this is not a tiered policy rh]
		for _, tier := range tiersNoClusterMatch {
			record.Name = tier + ".*"
			if rbac_auth.RulesAllow(record, rules...) {
				log.Debugf("Add namespaced match for tiered policy: tier=%s", tier)
				match.Tier = tier
				matches = append(matches, match)
			}
		}
	}

	return matches
}

// canGetAllTiers determines whether the user is able to get all tiers.
func (u *userCalculator) canGetAllTiers() bool {
	if u.canGetAllTiersVal == nil {
		allTiers := rbac_auth.RulesAllow(authorizer.AttributesRecord{
			Verb:            string(VerbGet),
			APIGroup:        libcalicov3.Group,
			Resource:        resourceTiers,
			Name:            "",
			ResourceRequest: true,
		}, u.getClusterRules()...)
		u.canGetAllTiersVal = &allTiers
	}

	return *u.canGetAllTiersVal
}

// getAllTiers returns the current set of configured tiers.
func (u *userCalculator) getAllTiers() []string {
	if u.allTiers == nil {
		if tiers, err := u.calculator.tierLister.ListTiers(); err != nil {
			log.WithError(err).Debug("Failed to list tiers")
			u.errors = append(u.errors, err)
			u.allTiers = make([]string, 0)
		} else {
			for _, tier := range tiers {
				u.allTiers = append(u.allTiers, tier.Name)
			}
		}
	}

	return u.allTiers
}

// getGettableTiers determines which tiers the user is able to get.
func (u *userCalculator) getGettableTiers() []string {
	if u.gettableTiers == nil {
		for _, tier := range u.getAllTiers() {
			if u.canGetAllTiers() || rbac_auth.RulesAllow(authorizer.AttributesRecord{
				Verb:            string(VerbGet),
				APIGroup:        libcalicov3.Group,
				Resource:        resourceTiers,
				Name:            tier,
				ResourceRequest: true,
			}, u.getClusterRules()...) {
				u.gettableTiers = append(u.gettableTiers, tier)
			}
		}
	}

	return u.gettableTiers
}

// getClusterRules returns the cluster rules that apply to the user.
func (u *userCalculator) getClusterRules() []rbacv1.PolicyRule {
	if u.clusterRules == nil {
		if rules, err := u.calculator.clusterRuleResolver.RulesFor(u.user, ""); err != nil {
			log.WithError(err).Debug("Failed to list cluster-wide rules for user")
			u.errors = append(u.errors, err)
			u.clusterRules = make([]rbacv1.PolicyRule, 0)
		} else {
			u.clusterRules = rules
		}
	}

	return u.clusterRules
}

// getNamespacedRules returns the namespaced rules that apply to the user.
func (u *userCalculator) getNamespacedRules() map[string][]rbacv1.PolicyRule {
	if u.namespacedRules == nil {
		u.namespacedRules = make(map[string][]rbacv1.PolicyRule)
		if namespaces, err := u.calculator.namespaceLister.ListNamespaces(); err != nil {
			log.WithError(err).Debug("Failed to list namespaced rules for user")
			u.errors = append(u.errors, err)
		} else {
			for _, n := range namespaces {
				if rules, err := u.calculator.namespacedRuleResolver.RulesFor(u.user, n.Name); err != nil {
					u.errors = append(u.errors, err)
				} else if len(rules) > 0 {
					u.namespacedRules[n.Name] = rules
				}
			}
		}
	}
	return u.namespacedRules
}

// updatePermissions calculates the RBAC permissions for a specific resource type and set of verbs, and updates the
// Permissions response struct.
func (u *userCalculator) updatePermissions(rt ResourceType, verbs []Verb) {
	u.permissions[rt] = make(map[Verb][]Match)
	rh := resourceTypeToHelper[rt]
	if rh == nil {
		u.errors = append(u.errors, fmt.Errorf("Resource type is not handled: %s", rt))
		return
	}

	for _, verb := range verbs {
		log.Debugf("Accumulating results for: %s %s", verb, rt)
		u.permissions[rt][verb] = u.getMatches(verb, rt, rh)
	}

	// The verb "watch" has some additional handling. For our use-case, "watch" is always paired with "list" meaning
	// we only return watch matches where we have an equivalent list match. This ensures a client can use perform a list
	// followed by a watch at the same revision as the list. The tier resource is a special case where we may allow
	// watching specific tiers (and so may expand a wildcarded watch by the gettable tiers.
	if watch := u.permissions[rt][VerbWatch]; len(watch) != 0 {
		// Get the list matches. If list was not requested then get those matches now.
		log.Debug("Process watch results")
		list, ok := u.permissions[rt][VerbList]
		if !ok {
			log.Debug("Calculate list results")
			list = u.getMatches(VerbList, rt, rh)
		}

		if rt.Resource == resourceTiers {
			// If all tiers are watchable and listable then include watch value unchanged. Otherwise we have to take
			// the overlap of the watchable and gettable tiers.
			log.Debug("Processing tier resource")
			listAll := len(list) == 1 && list[0] == matchAll
			watchAll := len(watch) == 1 && watch[0] == matchAll

			if watchAll && !listAll {
				// Can watch all tiers but cannot list them.  Expand the watch based on the gettable tiers. This allows
				// the UI to do a get followed by a watch to track individual tiers if needs be.
				var newWatch []Match
				for _, t := range u.getGettableTiers() {
					newWatch = append(newWatch, Match{Tier: t})
				}
				u.permissions[rt][VerbWatch] = newWatch
			} else if !watchAll {
				// Cannot watch all tiers, so only include those that are also gettable.
				gettable := map[string]bool{}
				for _, t := range u.getGettableTiers() {
					gettable[t] = true
				}
				var newWatch []Match
				for _, m := range watch {
					if gettable[m.Tier] {
						newWatch = append(newWatch, m)
					}
				}
				u.permissions[rt][VerbWatch] = newWatch
			}

			return
		}

		// Check if we need to expand watch results by namespace and filter out results that are not listable.
		// Start by sorting into tier and namespace buckets.
		listByTier := make(map[string]map[string]struct{})
		for _, m := range list {
			namespaces, ok := listByTier[m.Tier]
			if !ok {
				namespaces = make(map[string]struct{})
				listByTier[m.Tier] = namespaces
			}
			namespaces[m.Namespace] = struct{}{}
		}

		var newWatch []Match
		for _, m := range watch {
			// If there is no equivalent tier entry matching list action then don't include the watch action.
			log.Debugf("Processing watch match: %s", m)
			if namespaces, ok := listByTier[m.Tier]; !ok {
				// We have no equivalent tier match in list, so don't include results for this tier.
				log.Debugf("No list permissions")
			} else if _, ok := namespaces[m.Namespace]; ok {
				// The list has the equivalent entry as the watch, include it unchanged.
				log.Debugf("List and watch have same match entry: %s", m)
				newWatch = append(newWatch, m)
			} else if m.Namespace == "" {
				// Watch Namespace match is blank, meaning either the resource is not namespaced, or all namespaces
				// are watchable. In this case copy the results over from the List. The list will either also have
				// a blank namespace, or inidvidually specified namespaces - in which case we'd need to limit the
				// watch to the same list.
				log.Debug("Watch match is wildcarded, include namespaces from list")
				for n := range namespaces {
					m.Namespace = n
					log.Debugf("Include list match: %s", m)
					newWatch = append(newWatch, m)
				}
			} else if _, ok = namespaces[""]; ok {
				// Listable across all namespaces, so just include the watch entry unchanged.
				log.Debugf("List has wildcard match entry, include watch match unchanged: %s", m)
				newWatch = append(newWatch, m)
			}
		}
		u.permissions[rt][VerbWatch] = newWatch
	}
}
