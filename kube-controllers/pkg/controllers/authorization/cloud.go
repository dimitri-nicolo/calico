// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package authorization

import (
	"fmt"
	"os"
	"strings"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	esusers "github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
)

var tenantID = os.Getenv("ELASTIC_INDEX_TENANT_ID")

var resourceNameToElasticsearchRole = map[string]string{
	"flows":      formatRoleName(esusers.ElasticsearchRoleNameFlowsViewer),
	"audit*":     formatRoleName(esusers.ElasticsearchRoleNameAuditViewer),
	"audit_ee":   formatRoleName(esusers.ElasticsearchRoleNameAuditEEViewer),
	"audit_kube": formatRoleName(esusers.ElasticsearchRoleNameAuditKubeViewer),
	"events":     formatRoleName(esusers.ElasticsearchRoleNameEventsViewer),
	"dns":        formatRoleName(esusers.ElasticsearchRoleNameDNSViewer),
	"l7":         formatRoleName(esusers.ElasticsearchRoleNameL7Viewer),
}

var resourceNameToGlobalElasticsearchRoles = map[string]string{
	"kibana_login":            formatRoleName(esusers.ElasticsearchRoleNameKibanaViewer),
	"elasticsearch_superuser": formatRoleName(esusers.ElasticsearchRoleNameKibanaViewer),
	"kibana_admin":            formatRoleName(esusers.ElasticsearchRoleNameKibanaAdmin),
}

func formatRoleName(name string) string {
	if tenantID != "" {
		return fmt.Sprintf("%s_%s", name, tenantID)
	} else {
		return name
	}
}

// resync removes all elasticsearch native users with prefix `tigera-k8s` that doesn't have an entry in user cache
// and also for every oidc user in cache it creates/overwrites corresponding elasticsearch native users.
// This is the Cloud/Tesla variant of this function which ignores any users that do not have the tenantID suffix
// in their role names to avoid overwriting another tenants oidc users.
func (n *nativeUserSynchronizer) resync() error {
	if tenantID != "" {
		users, err := n.esCLI.GetUsers()
		if err != nil {
			return err
		}

		for _, user := range users {
			if strings.HasPrefix(user.Username, nativeUserPrefix) {
				rolesNames := user.RoleNames()
				// Skip deleting this user if it does not contain roles specific to this tenant.
				if !strings.HasSuffix(rolesNames[0], tenantID) {
					continue
				}
				subjectID := strings.TrimPrefix(user.Username, fmt.Sprintf("%s-", nativeUserPrefix))
				if !n.userCache.Exists(subjectID) {
					if err := n.esCLI.DeleteUser(elasticsearch.User{Username: user.Username}); err != nil {
						return err
					}
				}
			}
		}

		subjects := n.userCache.SubjectIDs()
		return n.synchronizeOIDCUsers(subjects)
	}

	return n.eeResync()
}

func (n *nativeUserSynchronizer) deleteEsUsers(esUsers map[string]elasticsearch.User) error {
	if tenantID != "" {
		for _, esUser := range esUsers {
			rolesNames := esUser.RoleNames()
			// Skip deleting this user if it does not contain roles specific to this tenant.
			if len(rolesNames) != 0 && !strings.HasSuffix(rolesNames[0], tenantID) {
				continue
			}
			if err := n.esCLI.DeleteUser(esUser); err != nil {
				return err
			}
		}
		return nil
	}

	return n.eeDeleteEsUsers(esUsers)
}
