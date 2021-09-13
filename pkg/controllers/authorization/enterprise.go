// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// +build !tesla

package authorization

import (
	"fmt"
	"strings"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	esusers "github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"
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

// resync removes all elasticsearch native users with prefix `tigera-k8s` that doesn't have an entry in user cache
// and also for every oidc user in cache it creates/overwrites corresponding elasticsearch native users.
func (n *nativeUserSynchronizer) resync() error {
	users, err := n.esCLI.GetUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		if strings.HasPrefix(user.Username, nativeUserPrefix) {
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

func (n *nativeUserSynchronizer) deleteEsUsers(esUsers map[string]elasticsearch.User) error {
	for _, esUser := range esUsers {
		if err := n.esCLI.DeleteUser(esUser); err != nil {
			return err
		}
	}
	return nil
}
