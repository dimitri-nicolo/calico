// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package authorization

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/calico/kube-controllers/pkg/rbaccache"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
)

// roleMappingSynchronizer is an implementation of k8sRBACSynchronizer interface, to keep Elasticsearch role mappings updated.
type roleMappingSynchronizer struct {
	roleCache      rbaccache.ClusterRoleCache
	esCLI          elasticsearch.Client
	usernamePrefix string
	groupPrefix    string
}

func newRoleMappingSynchronizer(stop chan struct{}, esCLI elasticsearch.Client, k8sCLI kubernetes.Interface, usernamePrefix string, groupPrefix string) k8sRBACSynchronizer {
	var clusterRoleCache rbaccache.ClusterRoleCache
	var err error
	ctx := context.Background()

	log.Debug("Deleting ConfigMap and Secret for OIDC Elasticsearch native users.")
	if err := retryUntilNotFound(func() error { return deleteOIDCUserConfigMap(context.Background(), k8sCLI) }); err != nil {
		log.WithError(err).Errorf("failed to delete %s ConfigMap", resource.OIDCUsersConfigMapName)
		return nil
	}

	if err := retryUntilNotFound(func() error { return deleteOIDCUsersEsSecret(context.Background(), k8sCLI) }); err != nil {
		log.WithError(err).Errorf("failed to delete %s Secret", resource.OIDCUsersEsSecreteName)
		return nil
	}

	log.Debug("Initializing ClusterRole cache.")
	// Initialize the cache so resync can calculate what Elasticsearch role mappings or native users should be removed and created/overwritten.
	stopped := retry(stop, 5*time.Second, "failed to initialize cache", func() error {
		clusterRoleCache, err = initializeRolesCache(ctx, k8sCLI)
		return err
	})
	if stopped {
		return nil
	}
	log.Info("Synchronizing K8s RBAC with Elasticsearch role mappings")
	return createRoleMappingSynchronizer(clusterRoleCache, esCLI, usernamePrefix, groupPrefix)
}

func createRoleMappingSynchronizer(roleCache rbaccache.ClusterRoleCache,
	esCLI elasticsearch.Client,
	usernamePrefix string,
	groupPrefix string) k8sRBACSynchronizer {
	return &roleMappingSynchronizer{
		roleCache:      roleCache,
		esCLI:          esCLI,
		usernamePrefix: usernamePrefix,
		groupPrefix:    groupPrefix,
	}
}

// resync removes any Elasticsearch role mapping that does not have an associate ClusterRole with rules for
// the lma.tigera.io API group or a ClusterRoleBinding with a User or Group subject kind, and it also
// retrieves the ClusterRoles from the cache that have an associated ClusterRole with rule for
// the lma.tigera.io API group and a ClusterRoleBinding with a User or Group subject kind and creates / overwrites the Elasticsearch
// mapping for the ClusterRole.
func (r *roleMappingSynchronizer) resync() error {
	rolesWithBindings := r.roleCache.ClusterRoleNamesWithBindings()

	existingMappings := make(map[string]struct{})
	for _, name := range rolesWithBindings {
		existingMappings[fmt.Sprintf("%s-%s", roleMappingPrefix, name)] = struct{}{}
	}

	esRoleMappings, err := r.esCLI.GetRoleMappings()
	if err != nil {
		return err
	}

	for _, esRoleMapping := range esRoleMappings {
		if strings.HasPrefix(esRoleMapping.Name, roleMappingPrefix) {
			if _, exists := existingMappings[esRoleMapping.Name]; !exists {
				if deleted, err := r.esCLI.DeleteRoleMapping(esRoleMapping.Name); err != nil {
					if deleted {
						log.Infof("Deleted stale role mapping %s", esRoleMapping.Name)
					}
					return err
				}
			}
		}
	}

	for _, clusterRoleName := range rolesWithBindings {
		if err := r.synchronizeElasticsearchMapping(clusterRoleName); err != nil {
			return err
		}
	}
	return nil
}

// synchronize updates the cache with changes to k8s resource and also updates affected role mapping in Elasticsearch.
func (r *roleMappingSynchronizer) synchronize(update resourceUpdate) error {
	var clusterRoleNamesUpdated []string
	switch update.resource.(type) {
	case *rbacv1.ClusterRole:
		clusterRole := update.resource.(*rbacv1.ClusterRole)
		clusterRoleName := update.name
		switch update.typ {
		case resourceUpdated:
			if r.roleCache.AddClusterRole(clusterRole) {
				clusterRoleNamesUpdated = []string{clusterRole.Name}
			}
		case resourceDeleted:
			if r.roleCache.RemoveClusterRole(clusterRoleName) {
				clusterRoleNamesUpdated = []string{clusterRoleName}
			}
		default:
			log.Errorf("unknown resource update type %s", update.typ)
		}

	case *rbacv1.ClusterRoleBinding:
		clusterRoleBinding := update.resource.(*rbacv1.ClusterRoleBinding)
		clusterRoleBindingName := update.name
		switch update.typ {
		case resourceUpdated:
			if r.roleCache.AddClusterRoleBinding(clusterRoleBinding) {
				clusterRoleNamesUpdated = []string{clusterRoleBinding.RoleRef.Name}
			}
		case resourceDeleted:
			role := r.roleCache.ClusterRoleNameForBinding(clusterRoleBindingName)
			if role == "" {
				log.Errorf("couldn't find ClusterRole for ClusterRoleBinding %s", clusterRoleBindingName)
				return nil
			}
			if r.roleCache.RemoveClusterRoleBinding(clusterRoleBindingName) {
				clusterRoleNamesUpdated = []string{role}
			}
		default:
			log.Errorf("unknown resource update type %s", update.typ)
		}

	default:
		log.Errorf("unknown resource update type %s", update.typ)
	}

	if len(clusterRoleNamesUpdated) != 0 {
		for _, clusterRoleName := range clusterRoleNamesUpdated {
			if err := r.synchronizeElasticsearchMapping(clusterRoleName); err != nil {
				return err
			}
		}
	}
	return nil
}

// synchronizeElasticsearchMapping accepts ClusterRole for which Elasticsearch role mapping needs to be updated.
// The ClusterRole to Elasticsearch role mapping synchronization may create / update or delete the role mapping for a ClusterRole.
// The Elasticsearch role mapping for a ClusterRole is deleted when the ClusterRole has been delete,
// the rule for the lma.tigera.io API group is removed from the ClusterRole, or if
// the ClusterRole no longer has an associated ClusterRoleBinding with a subject of type "User" or "Group".
func (r *roleMappingSynchronizer) synchronizeElasticsearchMapping(clusterRoleName string) error {
	log.Debugf("Reconciling %s", clusterRoleName)
	var users, groups []string

	for _, subject := range r.roleCache.ClusterRoleSubjects(clusterRoleName, rbacv1.UserKind) {
		users = append(users, subject.Name)
	}

	for _, subject := range r.roleCache.ClusterRoleSubjects(clusterRoleName, rbacv1.GroupKind) {
		groups = append(groups, subject.Name)
	}

	var esRoles []string

	rules := r.roleCache.ClusterRoleRules(clusterRoleName)
	if len(rules) > 0 {
		esRoles = rulesToElasticsearchRoles(rules...)
	}

	roleMappingName := fmt.Sprintf("%s-%s", roleMappingPrefix, clusterRoleName)
	if (len(users)+len(groups) == 0) || len(esRoles) == 0 {
		deleted, err := r.esCLI.DeleteRoleMapping(roleMappingName)
		if deleted {
			log.Infof("Deleted role mapping %s", roleMappingName)
		}

		return err
	}

	log.Infof("Creating role mapping %#v for users - %#v groups - %#v esRoles - %#v", roleMappingName, users, groups, esRoles)
	mapping := r.createRoleMapping(roleMappingName, users, groups, esRoles)
	if err := r.esCLI.CreateRoleMapping(mapping); err != nil {
		return err
	}

	log.Infof("Updated role mapping %s", roleMappingName)

	return nil
}

func (r *roleMappingSynchronizer) createRoleMapping(name string, users, groups, roles []string) elasticsearch.RoleMapping {
	var rules []elasticsearch.Rule
	for _, user := range users {
		user := strings.TrimPrefix(user, r.usernamePrefix)
		rules = append(rules,
			elasticsearch.Rule{
				Field: map[string]string{
					"username": user,
				},
			})
	}

	for _, group := range groups {
		group := strings.TrimPrefix(group, r.groupPrefix)
		rules = append(rules,
			elasticsearch.Rule{
				Field: map[string]string{
					"groups": group,
				},
			})
	}

	return elasticsearch.RoleMapping{
		Name:  name,
		Roles: roles,
		Rules: map[string][]elasticsearch.Rule{
			"any": rules,
		},
		Enabled: true,
	}
}
