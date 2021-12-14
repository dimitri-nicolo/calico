// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// +build !tesla

package users

import (
	"fmt"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
)

func indexPattern(prefix, cluster, suffix string) string {
	return fmt.Sprintf("%s.%s%s", prefix, cluster, suffix)
}

func formatRoleName(name, cluster string) string {
	if cluster == "*" {
		return name
	}

	return fmt.Sprintf("%s_%s", name, cluster)
}

func formatName(name ElasticsearchUserName, clusterName string, management, secureSuffix bool) string {
	var formattedName string
	if management {
		formattedName = string(name)
	} else {
		formattedName = fmt.Sprintf("%s-%s", string(name), clusterName)
	}
	if secureSuffix {
		formattedName = fmt.Sprintf("%s-%s", formattedName, ElasticsearchSecureUserSuffix)
	}
	return formattedName
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
