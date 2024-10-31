// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package userscache

import (
	"github.com/projectcalico/calico/kube-controllers/pkg/strutil"
	"github.com/projectcalico/calico/kube-controllers/pkg/utils"
)

// OIDCUserCache caches OIDC user information like subject, username and groups.
type OIDCUserCache interface {
	// UpdateOIDCUsers adds/updates the cache and returns list of subjectIDs updated,
	// where key for given data is OIDC user's subject ID and value is string representation of the OIDCUser.
	UpdateOIDCUsers(oidcUsers map[string]OIDCUser) []string

	// DeleteOIDCUser deletes a subjectID from cache and returns false if subjectID doesn't exists in the cache for deletions.
	DeleteOIDCUser(subjectID string) bool

	// SubjectIDToUserOrGroups returns list of username and groups that subjectID belongs to in cache.
	SubjectIDToUserOrGroups(subjectID string) []string

	// UserOrGroupToSubjectIDs returns list of subjectIDs the given username or group belongs to.
	UserOrGroupToSubjectIDs(userOrGroup string) []string

	// SubjectIDs returns list of all cached SubjectIDs.
	SubjectIDs() []string

	// Exists returns true if subjectID is in cache.
	Exists(subjectID string) bool
}

func NewOIDCUserCache() OIDCUserCache {
	return &oidcUserCache{
		subjectIDToUserAndGroups:  make(map[string][]string),
		userAndGroupsToSubjectIDs: make(map[string][]string),
	}
}

type oidcUserCache struct {
	subjectIDToUserAndGroups  map[string][]string
	userAndGroupsToSubjectIDs map[string][]string
}

type OIDCUser struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

// UpdateOIDCUsers adds/updates the cache and returns list of subjectIDs updated,
// where key for given data is OIDC user's subject ID and value is string representation of the OIDCUser.
func (cache *oidcUserCache) UpdateOIDCUsers(oidcUsers map[string]OIDCUser) []string {
	var subjectIDsUpdated []string

	for subjectID, user := range oidcUsers {
		usernameAndGroups := append(user.Groups, user.Username)
		if cache.updateOIDCUser(subjectID, usernameAndGroups) {
			subjectIDsUpdated = append(subjectIDsUpdated, subjectID)
		}
	}

	return subjectIDsUpdated
}

// DeleteOIDCUser deletes a subjectID from cache and returns false if subjectID doesn't exists in the cache for deletions.
func (cache *oidcUserCache) DeleteOIDCUser(subjectID string) bool {
	usersOrGroups, ok := cache.subjectIDToUserAndGroups[subjectID]
	if !ok {
		return false
	}
	delete(cache.subjectIDToUserAndGroups, subjectID)
	for _, usersOrGroup := range usersOrGroups {
		ids, ok := cache.userAndGroupsToSubjectIDs[usersOrGroup]
		if ok {
			cache.userAndGroupsToSubjectIDs[usersOrGroup] = utils.DeleteFromArray(ids, subjectID)
		}
	}
	return true
}

// SubjectIDToUserOrGroups returns list of username and groups that subjectID belongs to in cache.
func (cache *oidcUserCache) SubjectIDToUserOrGroups(subject string) []string {
	return cache.subjectIDToUserAndGroups[subject]
}

// UserOrGroupToSubjectIDs returns list of subjectIDs the given username or group belongs to.
func (cache *oidcUserCache) UserOrGroupToSubjectIDs(userOrGroup string) []string {
	return cache.userAndGroupsToSubjectIDs[userOrGroup]
}

// SubjectIDs returns list of all cached SubjectIDs.
func (cache *oidcUserCache) SubjectIDs() []string {
	var subIDs = make([]string, 0, len(cache.subjectIDToUserAndGroups))
	for subID := range cache.subjectIDToUserAndGroups {
		subIDs = append(subIDs, subID)
	}
	return subIDs
}

// Exists returns true if subjectID is in cache.
func (cache *oidcUserCache) Exists(subject string) bool {
	_, ok := cache.subjectIDToUserAndGroups[subject]
	return ok
}

// updateOIDCUser returns true if it adds/updates the cache for oidc user.
func (cache *oidcUserCache) updateOIDCUser(subjectID string, nameAndGroups []string) bool {
	cacheUpdated := false
	cachedNameAndGroups := cache.subjectIDToUserAndGroups[subjectID]

	// cache.subjectIDToUserAndGroups for given subjectID is updated with give nameAndGroups
	cache.subjectIDToUserAndGroups[subjectID] = nameAndGroups

	// If cachedNameAndGroups has an entry not in new nameAndGroups, remove that from cache.userAndGroupsToSubjectIDs
	for _, item := range cachedNameAndGroups {
		if !strutil.InList(item, nameAndGroups) {
			subjectIDsToUpdate := cache.userAndGroupsToSubjectIDs[item]
			cache.userAndGroupsToSubjectIDs[item] = utils.DeleteFromArray(subjectIDsToUpdate, subjectID)
			cacheUpdated = true
		}
	}

	// Add new values into cache.userAndGroupsToSubjectIDs
	for _, item := range nameAndGroups {
		cache.userAndGroupsToSubjectIDs[item] = append(cache.userAndGroupsToSubjectIDs[item], subjectID)
		cacheUpdated = true
	}

	return cacheUpdated
}
