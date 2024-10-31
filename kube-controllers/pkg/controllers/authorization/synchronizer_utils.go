// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package authorization

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	retryr "k8s.io/client-go/util/retry"

	esusers "github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/userscache"
	"github.com/projectcalico/calico/kube-controllers/pkg/rbaccache"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
	"github.com/projectcalico/calico/kube-controllers/pkg/strutil"
)

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
				esusers.ElasticsearchRoleNameWafViewer,
				esusers.ElasticsearchRoleNameRuntimeViewer,
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
func initializeRolesCache(ctx context.Context, k8sCLI kubernetes.Interface) (rbaccache.ClusterRoleCache, error) {

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
func initializeOIDCUserCache(ctx context.Context, k8sCLI kubernetes.Interface) (userscache.OIDCUserCache, error) {
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

// createOIDCUserConfigMap create resource.OIDCUsersConfigMapName if it doesn't exists.
func createOIDCUserConfigMap(ctx context.Context, k8sCLI kubernetes.Interface) error {
	var err error
	if _, err = k8sCLI.CoreV1().ConfigMaps(resource.TigeraElasticsearchNamespace).Get(ctx, resource.OIDCUsersConfigMapName, metav1.GetOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			cm := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.OIDCUsersConfigMapName,
					Namespace: resource.TigeraElasticsearchNamespace,
				},
			}
			if _, err = k8sCLI.CoreV1().ConfigMaps(resource.TigeraElasticsearchNamespace).Create(ctx, cm, metav1.CreateOptions{}); err == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

// deleteOIDCUserConfigMap deletes resource.OIDCUsersConfigMapName if it exists.
func deleteOIDCUserConfigMap(ctx context.Context, k8sCLI kubernetes.Interface) error {
	if err := k8sCLI.CoreV1().ConfigMaps(resource.TigeraElasticsearchNamespace).Delete(ctx, resource.OIDCUsersConfigMapName, metav1.DeleteOptions{}); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// createOIDCUsersEsSecret create resource.OIDCUsersEsSecreteName if it doesn't exists.
func createOIDCUsersEsSecret(ctx context.Context, k8sCLI kubernetes.Interface) error {
	var err error
	if _, err = k8sCLI.CoreV1().Secrets(resource.TigeraElasticsearchNamespace).Get(ctx, resource.OIDCUsersEsSecreteName, metav1.GetOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			secret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      resource.OIDCUsersEsSecreteName,
					Namespace: resource.TigeraElasticsearchNamespace,
				},
			}
			if _, err = k8sCLI.CoreV1().Secrets(resource.TigeraElasticsearchNamespace).Create(ctx, secret, metav1.CreateOptions{}); err == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

// deleteOIDCUsersEsSecret deletes resource.OIDCUsersEsSecreteName if it exists.
func deleteOIDCUsersEsSecret(ctx context.Context, k8sCLI kubernetes.Interface) error {
	if err := k8sCLI.CoreV1().Secrets(resource.TigeraElasticsearchNamespace).Delete(ctx, resource.OIDCUsersEsSecreteName, metav1.DeleteOptions{}); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}
	return nil
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

// retryUntilNotFound retries the given func f based on the DefaultRetry values, stops when f returns nil or NotFound error.
func retryUntilNotFound(f func() error) error {
	return retryr.OnError(retryr.DefaultRetry, func(err error) bool {
		return !kerrors.IsNotFound(err)
	}, func() error { return f() })
}

// retryUntilExists retries the given func f based on the DefaultRetry values, stops retrying if f returns either nil or AlreadyExists error.
func retryUntilExists(f func() error) error {
	return retryr.OnError(retryr.DefaultRetry, func(err error) bool {
		return !kerrors.IsAlreadyExists(err)
	}, func() error { return f() })
}
