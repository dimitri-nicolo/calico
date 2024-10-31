// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package rbaccache

import (
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/projectcalico/calico/kube-controllers/pkg/strutil"
	"github.com/projectcalico/calico/kube-controllers/pkg/utils"
)

// ClusterRoleCache is an interface which caches ClusterRole information along with their bound ClusterRoleBindings.
type ClusterRoleCache interface {
	// AddClusterRole adds the given ClusterRole to the cache. Implementations may filter what ClusterRoles will actually
	// be cached, so true is returned if the ClusterRole is added to the cache, and false otherwise.
	AddClusterRole(clusterRole *rbacv1.ClusterRole) bool

	// AddClusterRoleBinding updates the cache with the given ClusterRoleBinding information. Implementations may filter
	// what ClusterRoleBindings are eligible for storage, so true is return if the ClusterRoleBinding was cached and false
	// otherwise
	AddClusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) bool

	// RemoveClusterRole removes the ClusterRole with the given name from the cache. If the cache was updated, true is returned,
	// otherwise false is returned
	RemoveClusterRole(clusterRoleName string) bool

	// RemoveClusterRoleBinding removes the ClusterRoleBinding with the given name from the cache. If the cache was updated,
	// true is returned, otherwise false is returned
	RemoveClusterRoleBinding(clusterRoleBindingName string) bool

	// ClusterRoleSubjects gets the subjects of all ClusterRoleBindings bound to the ClusterRole with the name
	// clusterRoleName
	ClusterRoleSubjects(clusterRoleName string, kind string) []rbacv1.Subject

	// ClusterRoleRules gets the rules for the ClusterRole with given name.
	ClusterRoleRules(clusterRoleName string) []rbacv1.PolicyRule

	// ClusterRoleNameForBinding retrieves the ClusterRole name bound to the ClusterRoleBinding with the given name.
	ClusterRoleNameForBinding(clusterRoleBindingName string) string

	// ClusterRoleNamesWithBindings retrieves a list of ClusterRole names from the cache that have associated ClusterRoleBindings
	// in the cache, i.e. a ClusterRole with name "role" was added via AddClusterRole and a ClusterRoleBinding was added
	// via AddClusterRoleBinding.
	ClusterRoleNamesWithBindings() []string

	// ClusterRoleNamesForSubjectName retrieves list of ClusterRole names from the cache that have associated ClusterRoleBindings
	// with give rbacv1.Subject name.
	// When OIDCUsersConfigMapName is updated, this will be used to get ClusterRole for the oidc subject ID's username and groups.
	ClusterRoleNamesForSubjectName(rbacSubjectName string) []string

	// ClusterRoleBindingsForClusterRole retrieves list of ClusterRoleBindings for the give ClusterRoleName name
	// When ClusterRole is updated, the ClusterRoleBindings returned by this will be used in finding the oidc users affected
	// by the change in this ClusterRole.
	ClusterRoleBindingsForClusterRole(clusterRoleName string) []string

	// SubjectNamesForBinding retrieves list of rbacv1.Subject name for the given ClusterRoleBinding.
	SubjectNamesForBinding(clusterRoleBindingName string) []string
}

func NewClusterRoleCache(subjectsToCache []string, apiGroupRulesToCache []string) ClusterRoleCache {
	return &clusterRoleCache{
		subjectsToCache:       subjectsToCache,
		apiGroupRulesToCache:  apiGroupRulesToCache,
		roleEntryCache:        make(map[string]*clusterRoleCacheEntry),
		bindingToRoleCache:    make(map[string]string),
		bindingToSubjectNames: make(map[string][]string),
		subjectNameToBindings: make(map[string][]string),
	}
}

type clusterRoleCache struct {
	subjectsToCache       []string
	apiGroupRulesToCache  []string
	roleEntryCache        map[string]*clusterRoleCacheEntry
	bindingToRoleCache    map[string]string
	bindingToSubjectNames map[string][]string
	subjectNameToBindings map[string][]string
}

func (cache *clusterRoleCache) getOrCreateCacheEntry(roleName string) *clusterRoleCacheEntry {
	var cacheEntry *clusterRoleCacheEntry
	var exist bool

	if cacheEntry, exist = cache.roleEntryCache[roleName]; !exist {
		cacheEntry = new(clusterRoleCacheEntry)
		cacheEntry.subjects = make(map[string]map[string][]rbacv1.Subject)

		cache.roleEntryCache[roleName] = cacheEntry
	}

	return cacheEntry
}

func (cache *clusterRoleCache) ClusterRoleNameForBinding(bindingName string) string {
	return cache.bindingToRoleCache[bindingName]
}

func (cache *clusterRoleCache) AddClusterRole(clusterRole *rbacv1.ClusterRole) bool {
	cacheUpdated := false
	var rules []rbacv1.PolicyRule
	for _, rule := range clusterRole.Rules {
		for _, apiGroup := range rule.APIGroups {
			if len(cache.apiGroupRulesToCache) == 0 || strutil.InList(apiGroup, cache.apiGroupRulesToCache) {
				rules = append(rules, rule)

				cacheUpdated = true
			}
		}
	}

	if len(rules) > 0 {
		cacheEntry := cache.getOrCreateCacheEntry(clusterRole.Name)
		cacheEntry.exists = true
		cacheEntry.rules = rules
	}

	return cacheUpdated
}

func (cache *clusterRoleCache) RemoveClusterRole(clusterRoleName string) bool {
	if _, exists := cache.roleEntryCache[clusterRoleName]; exists {
		// If there's still subjects (i.e. role bindings) don't remove the cluster role from the cache, otherwise
		// we 1. lose the role binding cache if the cluster role is re added and 2. cause panics in some of the
		// cluster role binding handling code.
		if len(cache.roleEntryCache[clusterRoleName].subjects) == 0 {
			delete(cache.roleEntryCache, clusterRoleName)
		} else {
			cache.roleEntryCache[clusterRoleName].exists = false
			cache.roleEntryCache[clusterRoleName].rules = nil
		}

		return true
	}

	return false
}

func (cache *clusterRoleCache) AddClusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) bool {
	cacheUpdated := false
	roleName := clusterRoleBinding.RoleRef.Name

	subjectMap := make(map[string][]rbacv1.Subject)
	for _, subject := range clusterRoleBinding.Subjects {
		if len(cache.subjectsToCache) == 0 || strutil.InList(subject.Kind, cache.subjectsToCache) {
			subjectMap[subject.Kind] = append(subjectMap[subject.Kind], subject)

			cache.subjectNameToBindings[subject.Name] = append(cache.subjectNameToBindings[subject.Name], clusterRoleBinding.Name)
			cache.bindingToSubjectNames[clusterRoleBinding.Name] = append(cache.bindingToSubjectNames[clusterRoleBinding.Name], subject.Name)
		}
	}

	if len(subjectMap) > 0 {
		cacheEntry := cache.getOrCreateCacheEntry(roleName)
		cacheEntry.subjects[clusterRoleBinding.Name] = subjectMap

		cacheUpdated = true
		cache.bindingToRoleCache[clusterRoleBinding.Name] = roleName
	}

	return cacheUpdated
}

func (cache *clusterRoleCache) RemoveClusterRoleBinding(clusterRoleBindingName string) bool {
	if role, exists := cache.bindingToRoleCache[clusterRoleBindingName]; exists {
		delete(cache.roleEntryCache[role].subjects, clusterRoleBindingName)
		delete(cache.bindingToRoleCache, clusterRoleBindingName)

		rbacSubjectNamesForBinding := cache.bindingToSubjectNames[clusterRoleBindingName]
		delete(cache.bindingToSubjectNames, clusterRoleBindingName)

		// remove the binding from cache.subjectNameToBindings
		for _, subjectName := range rbacSubjectNamesForBinding {
			cache.subjectNameToBindings[subjectName] = utils.DeleteFromArray(cache.subjectNameToBindings[subjectName], clusterRoleBindingName)
			if len(cache.subjectNameToBindings[subjectName]) == 0 {
				delete(cache.subjectNameToBindings, subjectName)
			}
		}

		// Remove the cluster role entry if there are no more cluster role bindings referencing a role that doesn't
		// exist in k8s.
		if len(cache.roleEntryCache[role].subjects) == 0 && !cache.roleEntryCache[role].exists {
			delete(cache.roleEntryCache, role)
		}
		return true
	}

	return false
}

func (cache *clusterRoleCache) ClusterRoleSubjects(clusterRoleName string, subjectName string) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	if roleEntryCache, exists := cache.roleEntryCache[clusterRoleName]; exists {
		for _, subjectMap := range roleEntryCache.subjects {
			if _, exists := subjectMap[subjectName]; exists {
				subjects = append(subjects, subjectMap[subjectName]...)
			}
		}
	}

	return subjects
}

func (cache *clusterRoleCache) ClusterRoleRules(clusterRoleName string) []rbacv1.PolicyRule {
	if roleEntryCache, exists := cache.roleEntryCache[clusterRoleName]; exists {
		return roleEntryCache.rules
	}

	return []rbacv1.PolicyRule{}
}

// ClusterRoleNamesWithBindings returns the names of all the ClusterRoles in the cache that have ClusterRoleBindings in
// the cache.
//
// Note that the cache filters what ClusterRoles and ClusterRoleBindings may be added to the cache, so just because there
// is a ClusterRoleBinding for a ClusterRole in k8s doesn't mean the cache will store it.
func (cache *clusterRoleCache) ClusterRoleNamesWithBindings() []string {
	var rolesWithBindings []string
	for name, role := range cache.roleEntryCache {
		// if any entry has rules, that means the ClusterRole is in the cache, and if it has subjects it means it has
		// an appropriate ClusterRoleBinding
		if len(role.subjects) > 0 && len(role.rules) > 0 {
			rolesWithBindings = append(rolesWithBindings, name)
		}
	}

	return rolesWithBindings
}

// ClusterRoleNamesForSubjectName retrieves list of ClusterRole names from the cache that have associated ClusterRoleBindings
// with give rbacv1.Subject name.
// When OIDCUsersConfigMapName is updated, this will be used to get ClusterRole for the oidc subject ID's username and groups.
func (cache *clusterRoleCache) ClusterRoleNamesForSubjectName(rbacSubjectName string) []string {
	var clusterRoleNames []string
	bindings, ok := cache.subjectNameToBindings[rbacSubjectName]
	if !ok {
		return clusterRoleNames
	}
	for _, binding := range bindings {
		if val := cache.ClusterRoleNameForBinding(binding); val != "" {
			clusterRoleNames = append(clusterRoleNames, val)
		}
	}
	return clusterRoleNames
}

// ClusterRoleBindingsForClusterRole retrieves list of ClusterRoleBindings for the give ClusterRoleName name
// When ClusterRole is updated, the ClusterRoleBindings returned by this will be used in finding the oidc users affected
// by the change in this ClusterRole.
func (cache *clusterRoleCache) ClusterRoleBindingsForClusterRole(clusterRoleName string) []string {
	var bindings []string
	entry := cache.roleEntryCache[clusterRoleName]
	// if any entry has rules, that means the ClusterRole is in the cache, and if it has subjects it means it has
	// an appropriate ClusterRoleBinding
	if entry != nil && len(entry.subjects) > 0 && len(entry.rules) > 0 {
		for bind := range entry.subjects {
			bindings = append(bindings, bind)
		}
	}
	return bindings
}

// SubjectNamesForBinding retrieves list of rbacv1.Subject name for the given ClusterRoleBinding.
func (cache *clusterRoleCache) SubjectNamesForBinding(clusterRoleBindingName string) []string {
	return cache.bindingToSubjectNames[clusterRoleBindingName]
}

type clusterRoleCacheEntry struct {
	subjects map[string]map[string][]rbacv1.Subject
	rules    []rbacv1.PolicyRule
	// exists represents whether this cluster role exists or not in k8s or was created because a cluster role binding
	// is for a role that doesn't exist.
	exists bool
}
