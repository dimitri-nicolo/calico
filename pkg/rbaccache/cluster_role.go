// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package rbaccache

import (
	"github.com/projectcalico/kube-controllers/pkg/strutil"
	rbacv1 "k8s.io/api/rbac/v1"
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

	// ClusterRoleRules gets the rules for the ClusterRole with given name
	ClusterRoleRules(clusterRoleName string) []rbacv1.PolicyRule

	// ClusterRoleNameForBinding retrieves the ClusterRole name bound to the ClusterRoleBinding with the given name.
	ClusterRoleNameForBinding(clusterRoleBindingName string) string

	// ClusterRoleNamesWithBindings retrieves a list of ClusterRole names from the cache that have associated ClusterRoleBindings
	// in the cache, i.e. a ClusterRole with name "role" was added via AddClusterRole and a ClusterRoleBinding was added
	// via AddClusterRoleBinding.
	ClusterRoleNamesWithBindings() []string
}

func NewClusterRoleCache(subjectsToCache []string, apiGroupRulesToCache []string) ClusterRoleCache {
	return &clusterRoleCache{
		subjectsToCache:      subjectsToCache,
		apiGroupRulesToCache: apiGroupRulesToCache,
		entryCache:           make(map[string]*clusterRoleCacheEntry),
		bindingToRoleCache:   make(map[string]string),
	}
}

type clusterRoleCache struct {
	subjectsToCache      []string
	apiGroupRulesToCache []string
	entryCache           map[string]*clusterRoleCacheEntry
	bindingToRoleCache   map[string]string
}

func (cache *clusterRoleCache) getOrCreateCacheEntry(roleName string) *clusterRoleCacheEntry {
	var cacheEntry *clusterRoleCacheEntry
	var exist bool

	if cacheEntry, exist = cache.entryCache[roleName]; !exist {
		cacheEntry = new(clusterRoleCacheEntry)
		cacheEntry.subjects = make(map[string]map[string][]rbacv1.Subject)

		cache.entryCache[roleName] = cacheEntry
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
		cacheEntry.rules = rules
	}

	return cacheUpdated
}

func (cache *clusterRoleCache) RemoveClusterRole(clusterRoleName string) bool {
	if _, exists := cache.entryCache[clusterRoleName]; exists {
		delete(cache.entryCache, clusterRoleName)
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
		delete(cache.entryCache[role].subjects, clusterRoleBindingName)
		delete(cache.bindingToRoleCache, clusterRoleBindingName)

		return true
	}

	return false
}

func (cache *clusterRoleCache) ClusterRoleSubjects(clusterRoleName string, subjectName string) []rbacv1.Subject {
	if entryCache, exists := cache.entryCache[clusterRoleName]; exists {
		var subjects []rbacv1.Subject
		for _, subjectMap := range entryCache.subjects {
			if _, exists := subjectMap[subjectName]; exists {
				subjects = append(subjects, subjectMap[subjectName]...)
			}
		}
		return subjects
	}

	return []rbacv1.Subject{}
}

func (cache *clusterRoleCache) ClusterRoleRules(clusterRoleName string) []rbacv1.PolicyRule {
	if entryCache, exists := cache.entryCache[clusterRoleName]; exists {
		return entryCache.rules
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
	for name, role := range cache.entryCache {
		// if any entry has rules, that means the ClusterRole is in the cache, and if it has subjects it means it has
		// an appropriate ClusterRoleBinding
		if len(role.subjects) > 0 && len(role.rules) > 0 {
			rolesWithBindings = append(rolesWithBindings, name)
		}
	}

	return rolesWithBindings
}

type clusterRoleCacheEntry struct {
	subjects map[string]map[string][]rbacv1.Subject
	rules    []rbacv1.PolicyRule
}
