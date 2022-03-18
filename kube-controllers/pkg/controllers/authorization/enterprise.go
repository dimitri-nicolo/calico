// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build !tesla
// +build !tesla

package authorization

import (
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	esusers "github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
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
	return n.eeResync()
}

func (n *nativeUserSynchronizer) deleteEsUsers(esUsers map[string]elasticsearch.User) error {
	return n.eeDeleteEsUsers(esUsers)
}
