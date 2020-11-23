// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package users

import (
	"fmt"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
)

const (
	ElasticsearchRoleNameFlowsViewer     = "flows_viewer"
	ElasticsearchRoleNameAuditViewer     = "audit_viewer"
	ElasticsearchRoleNameAuditEEViewer   = "audit_ee_viewer"
	ElasticsearchRoleNameAuditKubeViewer = "audit_kube_viewer"
	ElasticsearchRoleNameEventsViewer    = "events_viewer"
	ElasticsearchRoleNameDNSViewer       = "dns_viewer"
	ElasticsearchRoleNameKibanaAdmin     = "kibana_admin"
	ElasticsearchRoleNameKibanaViewer    = "kibana_viewer"
	ElasticsearchRoleNameL7Viewer        = "l7_viewer"

	ElasticsearchRoleNameSuperUser = "superuser"
)

func GetAuthorizationRoles(clusterName string) []elasticsearch.Role {
	return []elasticsearch.Role{
		{
			Name: formatRoleName(ElasticsearchRoleNameFlowsViewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_flows", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
		{
			Name: formatRoleName(ElasticsearchRoleNameAuditViewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_audit_*", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
		{
			Name: formatRoleName(ElasticsearchRoleNameAuditEEViewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_audit_ee", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
		{
			Name: formatRoleName(ElasticsearchRoleNameAuditKubeViewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_audit_kube", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
		{
			Name: formatRoleName(ElasticsearchRoleNameEventsViewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_events", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
		{
			Name: formatRoleName(ElasticsearchRoleNameDNSViewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_dns", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
		{
			Name: formatRoleName(ElasticsearchRoleNameL7Viewer, clusterName),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_l7", clusterName, ".*")},
					Privileges: []string{"read"},
				}},
			},
		},
	}
}

func GetGlobalAuthorizationRoles() []elasticsearch.Role {
	return []elasticsearch.Role{{
		Name: ElasticsearchRoleNameKibanaViewer,
		Definition: &elasticsearch.RoleDefinition{
			Indices: []elasticsearch.RoleIndex{},
			Applications: []elasticsearch.Application{{
				Application: "kibana-.kibana",
				Privileges: []string{
					"feature_discover.read",
					"feature_visualize.read",
					"feature_dashboard.read",
				},
				Resources: []string{"space:default"},
			}},
		},
	}}
}

func formatRoleName(name, cluster string) string {
	if cluster == "*" {
		return name
	}

	return fmt.Sprintf("%s_%s", name, cluster)
}
