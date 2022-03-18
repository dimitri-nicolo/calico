// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package users

import (
	"fmt"
	"os"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
)

var tenantID = os.Getenv("ELASTIC_INDEX_TENANT_ID")

func indexPattern(prefix, cluster, suffix string) string {
	if tenantID != "" {
		return fmt.Sprintf("%s.%s.%s%s", prefix, tenantID, cluster, suffix)
	}

	return eeIndexPattern(prefix, cluster, suffix)
}

func formatRoleName(name, cluster string) string {
	if tenantID != "" {
		if cluster == "*" {
			return fmt.Sprintf("%s_%s", name, tenantID)
		}

		return fmt.Sprintf("%s_%s_%s", name, tenantID, cluster)
	}

	return eeFormatRoleName(name, cluster)
}

func formatName(name ElasticsearchUserName, clusterName string, management, secureSuffix bool) string {
	if tenantID != "" {
		var formattedName string
		if management {
			formattedName = string(name)
		} else {
			formattedName = fmt.Sprintf("%s-%s", string(name), clusterName)
		}
		if secureSuffix {
			formattedName = fmt.Sprintf("%s-%s-%s", formattedName, tenantID, ElasticsearchSecureUserSuffix)
		}
		return formattedName
	}

	return eeFormatName(name, clusterName, management, secureSuffix)
}

func GetGlobalAuthorizationRoles() []elasticsearch.Role {
	if tenantID != "" {
		return []elasticsearch.Role{{
			Name: fmt.Sprintf("%s_%s", ElasticsearchRoleNameKibanaViewer, tenantID),
			Definition: &elasticsearch.RoleDefinition{
				Indices: []elasticsearch.RoleIndex{{
					Names:      []string{indexPattern("tigera_secure_ee_*", "*", ".*")},
					Privileges: []string{"all"},
				}},
				Applications: []elasticsearch.Application{{
					Application: "kibana-.kibana",
					Privileges: []string{
						"feature_discover.read",
						"feature_visualize.read",
						"feature_dashboard.read",
						"space_all",
					},
					Resources: []string{fmt.Sprintf("space:%s", tenantID)},
				}},
			},
		}}
	}

	return eeGetGlobalAuthorizationRoles()
}
