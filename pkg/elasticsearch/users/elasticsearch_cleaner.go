// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package users

import (
	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	log "github.com/sirupsen/logrus"
	"regexp"
)

// ESCleaner deletes residue elastic users and roles associated with
// managed clusters
type ESCleaner struct {
	esCLI elasticsearch.Client
}

// NewEsCleaner creates a new EsCleaner
func NewEsCleaner(esCLI elasticsearch.Client) *ESCleaner {
	return &ESCleaner{esCLI}
}

// DeleteResidueUsers deletes all elasticsearch users associated with a clusterName
func (c *ESCleaner) DeleteResidueUsers(clusterName string) {
	log.Infof("Deleting ES Users for managed cluster %s", clusterName)

	// Ignore public users here since they were not created in Elasticsearch.
	esUsers, _ := ElasticsearchUsers(clusterName, false)
	var usersToBeDeleted []elasticsearch.User
	var rolesToBeDeleted = make(map[elasticsearch.Role]bool)

	add := func(usersToBeDeleted []elasticsearch.User, user elasticsearch.User,
		rolesToBeDeleted map[elasticsearch.Role]bool) ([]elasticsearch.User, map[elasticsearch.Role]bool) {
		usersToBeDeleted = append(usersToBeDeleted, user)
		for _, role := range user.Roles {
			if role.Name != "watcher_admin" {
				rolesToBeDeleted[role] = true
			}
		}
		return usersToBeDeleted, rolesToBeDeleted
	}

	for _, user := range esUsers {
		usersToBeDeleted, rolesToBeDeleted = add(usersToBeDeleted, user, rolesToBeDeleted)
	}

	c.delete(usersToBeDeleted, rolesToBeDeleted)
}

// DeleteAllResidueUsers deletes all elasticsearch users that have not been removed when
// a cluster was deleted
func (c *ESCleaner) DeleteAllResidueUsers(registeredManagedClusters map[string]bool) error {
	log.Infof("Deleting ES Users that have not been deleted when a cluster was removed")

	// Fetch all the users from Elastic
	esUsers, err := c.esCLI.GetUsers()
	if err != nil {
		return err
	}

	// Extract managed clusters that have still users in Elastic
	registeredEsClusters := make(map[string]bool)
	usersPatterns := buildManagedUserPattern()
	for _, user := range esUsers {
		cluster, found := c.matches(user.Username, usersPatterns)
		if found {
			registeredEsClusters[cluster] = true
		}
	}

	for k := range registeredEsClusters {
		_, found := registeredManagedClusters[k]
		if !found {
			c.DeleteResidueUsers(k)
		}
	}

	return nil
}

func (c *ESCleaner) matches(userName string, usersPatterns []*regexp.Regexp) (string, bool) {
	for _, pattern := range usersPatterns {
		match := pattern.FindAllStringSubmatch(userName, -1)
		if len(match) == 1 && len(match[0]) == 2 {
			return match[0][1], true
		}
	}

	return "", false
}

func (c *ESCleaner) delete(usersToBeDeleted []elasticsearch.User, rolesToBeDeleted map[elasticsearch.Role]bool) {
	for _, user := range usersToBeDeleted {
		log.Infof("Deleting user %s", user.Username)
		if err := c.esCLI.DeleteUser(user); err != nil {
			log.Errorf("Failed to delete user %s from elastic due to %s", user.Username, err)
		}
	}

	for role := range rolesToBeDeleted {
		log.Infof("Deleting role %s", role.Name)
		if err := c.esCLI.DeleteRole(role); err != nil {
			log.Errorf("Failed to delete role %s from elastic due to %s", role.Name, err)
		}
	}
}