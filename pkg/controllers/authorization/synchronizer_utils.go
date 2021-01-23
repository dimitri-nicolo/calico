// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package authorization

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch/userscache"
	"github.com/projectcalico/kube-controllers/pkg/rbaccache"
	"github.com/projectcalico/kube-controllers/pkg/resource"

	rbacv1 "k8s.io/api/rbac/v1"

	esusers "github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"
	"github.com/projectcalico/kube-controllers/pkg/strutil"
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
				// Don't add cluster name suffix for the global roles.
				for _, baseESRole := range baseESRoles {
					esRoles = append(esRoles, fmt.Sprintf("%s_%s", baseESRole, clusterName))
				}
			}
		}

		esRoles = append(esRoles, globalRoles...)
	}

	return esRoles
}

// initializeRolesCache creates and fills the rbaccache.ClusterRoleCache with the available ClusterRoles and ClusterRoleBindings.
func initializeRolesCache(k8sCLI kubernetes.Interface) (rbaccache.ClusterRoleCache, error) {
	ctx := context.Background()

	clusterRolesCache := rbaccache.NewClusterRoleCache([]string{rbacv1.UserKind, rbacv1.GroupKind}, []string{"lma.tigera.io"})

	clusterRoleBindings, err := k8sCLI.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoles, err := k8sCLI.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, binding := range clusterRoleBindings.Items {
		clusterRolesCache.AddClusterRoleBinding(&binding)
	}

	for _, role := range clusterRoles.Items {
		clusterRolesCache.AddClusterRole(&role)
	}

	return clusterRolesCache, nil
}

// initializeOIDCUserCache creates and fills the userscache.OIDCUserCache with available data in ConfigMap.
func initializeOIDCUserCache(k8sCLI kubernetes.Interface) (userscache.OIDCUserCache, error) {
	ctx := context.Background()

	configMap, err := k8sCLI.CoreV1().ConfigMaps(resource.TigeraElasticsearchNamespace).Get(ctx, resource.OIDCUsersConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	userCache := userscache.NewOIDCUserCache()

	oidcUsers, err := configMapDataToOIDCUsers(configMap.Data)
	if err != nil {
		log.WithError(err).Errorf("error getting OIDC users from ConfigMap")
		return userCache, nil
	}
	userCache.UpdateOIDCUsers(oidcUsers)

	return userCache, nil
}

// configMapDataToOIDCUsers extracts and returns the userscache.OIDCUser from ConfigMap Data.
func configMapDataToOIDCUsers(data map[string]string) (map[string]userscache.OIDCUser, error) {
	var users = make(map[string]userscache.OIDCUser)
	for subjectID, jsonStr := range data {
		var user userscache.OIDCUser
		if err := json.Unmarshal([]byte(jsonStr), &user); err != nil {
			return users, err
		}
		users[subjectID] = user
	}
	return users, nil
}
