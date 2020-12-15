// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package authorization

import (
	"fmt"
	"strings"

	"github.com/projectcalico/kube-controllers/pkg/strutil"

	esusers "github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/kube-controllers/pkg/rbaccache"
	rbacv1 "k8s.io/api/rbac/v1"

	log "github.com/sirupsen/logrus"
)

const (
	roleMappingPrefix = "tigera-k8s"

	resourceUpdated = "updated"
	resourceDeleted = "deleted"
)

var resourceNameToElasticsearchRole = map[string]string{
	"flows":      esusers.ElasticsearchRoleNameFlowsViewer,
	"audit*":     esusers.ElasticsearchRoleNameAuditViewer,
	"audit_ee":   esusers.ElasticsearchRoleNameAuditEEViewer,
	"audit_kube": esusers.ElasticsearchRoleNameAuditKubeViewer,
	"events":     esusers.ElasticsearchRoleNameEventsViewer,
	"dns":        esusers.ElasticsearchRoleNameDNSViewer,
	"l7":         esusers.ElasticsearchRoleNameL7Viewer,
}

var resourceNameToGlobalElasticsearchRoles = map[string]string{
	"kibana_login":            esusers.ElasticsearchRoleNameKibanaViewer,
	"elasticsearch_superuser": esusers.ElasticsearchRoleNameSuperUser,
	"kibana_admin":            esusers.ElasticsearchRoleNameKibanaAdmin,
}

type resourceUpdate struct {
	typ      string
	name     string
	resource interface{}
}

type esRoleMappingSynchronizer struct {
	stopChan        chan chan struct{}
	roleCache       rbaccache.ClusterRoleCache
	esCLI           elasticsearch.Client
	resourceUpdates chan resourceUpdate
	usernamePrefix  string
	groupPrefix     string
}

// synchronizeRoleMappings watches for updates over the resourceUpdates channel, attempts to update the ClusterRoleCache
// with the updates (either "updated" / "deleted" ClusterRole / ClusterRoleBinding) and if the cache was updated it synchronizes
// the ClusterRole (either the ClusterRole sent over the resourceUpdates channel or the one bound the the ClusterRoleBinding
// send over the channel) associated Elasticsearch role mapping. The ClusterRole to Elasticsearch role mapping synchronization
// may create / update or delete the role mapping for a ClusterRole. The Elasticsearch role mapping for a ClusterRole is
// deleted is delete when the ClusterRole has been delete, the rule for the lma.tigera.io API group is removed from the
// ClusterRole, or if  the ClusterRole no longer has an associated ClusterRoleBinding with a subject of type "User" or "Group".
func (r *esRoleMappingSynchronizer) synchronizeRoleMappings() {
	for {
		select {
		case notify, ok := <-r.stopChan:
			if !ok {
				return
			}

			close(notify)
			return
		case update, ok := <-r.resourceUpdates:
			if !ok {
				return
			}
			// clusterRoleName is retrieved differently depending on if the resource is a ClusterRole or a ClusterRole
			// binding, but is needed for the role mapping synchronization after the switch
			var clusterRoleName string

			cacheUpdated := true

			switch update.resource.(type) {
			case *rbacv1.ClusterRole:
				clusterRoleName = update.name

				switch update.typ {
				case resourceUpdated:
					cacheUpdated = r.roleCache.AddClusterRole(update.resource.(*rbacv1.ClusterRole))
				case resourceDeleted:
					cacheUpdated = r.roleCache.RemoveClusterRole(update.name)
				default:
					log.Errorf("unknown resource update type %s", update.typ)
					continue
				}
			case *rbacv1.ClusterRoleBinding:
				switch update.typ {
				case resourceUpdated:
					clusterRoleBinding := update.resource.(*rbacv1.ClusterRoleBinding)
					clusterRoleName = clusterRoleBinding.RoleRef.Name

					cacheUpdated = r.roleCache.AddClusterRoleBinding(clusterRoleBinding)
				case resourceDeleted:
					clusterRoleName = r.roleCache.ClusterRoleNameForBinding(update.name)
					if clusterRoleName == "" {
						log.Errorf("couldn't find ClusterRole for ClusterRoleBinding %s", update.name)
						continue
					}
					cacheUpdated = r.roleCache.RemoveClusterRoleBinding(update.name)
				default:
					log.Errorf("unknown resource update type %s", update.typ)
					continue
				}
			}

			if cacheUpdated {
				if err := r.synchronizeElasticsearchMapping(clusterRoleName); err != nil {
					//TODO we might want to try requeueing the failed updates
					log.WithError(err).Errorf("failed to synchronize ClusterRole %s", clusterRoleName)
				}
			}
		}
	}

}

// stop sends a signal to the stopChan which stops the loop running in synchronizeRoleMappings. This function blocks until
// it receives confirm
func (r *esRoleMappingSynchronizer) stop() {
	done := make(chan struct{})
	r.stopChan <- done
	<-done
	close(r.stopChan)
}

// syncBoundClusterRoles retrieves the ClusterRoles from the cache that have have an associated ClusterRole with rule for
// the lma.tigera.io API group and a ClusterRoleBinding with a User or Group subject kind and creates / overwrites the Elasticsearch
// mapping for the ClusterRole.
func (r *esRoleMappingSynchronizer) syncBoundClusterRoles() error {
	rolesWithBindings := r.roleCache.ClusterRoleNamesWithBindings()
	for _, clusterRoleName := range rolesWithBindings {
		if err := r.synchronizeElasticsearchMapping(clusterRoleName); err != nil {
			return err
		}
	}

	return nil
}

// removeStaleMappings removes any Elasticsearch role mapping that does not have an associate ClusterRole with rules for
// the lma.tigera.io API group or a ClusterRoleBinding with a User or Group subject kind.
func (r *esRoleMappingSynchronizer) removeStaleMappings() error {
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

	return nil
}

func (r *esRoleMappingSynchronizer) synchronizeElasticsearchMapping(clusterRoleName string) error {
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

func rulesToElasticsearchRoles(rules ...rbacv1.PolicyRule) []string {
	var esRoles []string

	for _, rule := range rules {
		var baseESRoles []string

		if len(rule.ResourceNames) == 0 {
			// Even though an empty resource name list is supposed to signal that all resource names are to be applied,
			// we still do not want to automatically give access to kibana or assign somebody superuser privileges. This
			// would be a very unexpected privilege escalation for users migrating to a KubeControllers version that does
			// that.
			baseESRoles = []string{
				esusers.ElasticsearchRoleNameFlowsViewer,
				esusers.ElasticsearchRoleNameAuditViewer,
				esusers.ElasticsearchRoleNameEventsViewer,
				esusers.ElasticsearchRoleNameDNSViewer,
				esusers.ElasticsearchRoleNameL7Viewer,
			}
		}

		var globalRoles []string
		for _, resourceName := range rule.ResourceNames {
			if esRole, exists := resourceNameToElasticsearchRole[resourceName]; exists {
				baseESRoles = append(baseESRoles, esRole)
			} else if esRole, exists := resourceNameToGlobalElasticsearchRoles[resourceName]; exists {
				globalRoles = append(globalRoles, esRole)
			}
		}

		if strutil.InList(rbacv1.ResourceAll, rule.Resources) {
			esRoles = baseESRoles
		} else {
			for _, clusterName := range rule.Resources {
				// Don't add cluster name suffix for the global roles
				for _, baseESRole := range baseESRoles {
					esRoles = append(esRoles, fmt.Sprintf("%s_%s", baseESRole, clusterName))
				}
			}
		}

		esRoles = append(esRoles, globalRoles...)
	}

	return esRoles
}

func (r *esRoleMappingSynchronizer) createRoleMapping(name string, users, groups, roles []string) elasticsearch.RoleMapping {
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
