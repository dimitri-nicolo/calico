// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package elasticsearchconfiguration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"
	esusers "github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type reconciler struct {
	clusterName string
	// ownerReference is used to store the "owner" of this reconciler. If the owner has changed that signals the user
	// credential secrets should be rotated. It's valid to have an empty owner reference.
	ownerReference   string
	management       bool
	managementK8sCLI kubernetes.Interface
	managedK8sCLI    kubernetes.Interface
	esK8sCLI         relasticsearch.RESTClient
	esHash           string
	esClientBuilder  elasticsearch.ClientBuilder
	esCLI            elasticsearch.Client
}

// Reconcile makes sure that the managed cluster this is running for has all the configuration needed for it's components
// to access elasticsearch. If the managed cluster this is running for is actually a management cluster, then the secret
// for the elasticsearch public certificate and the ConfigMap containing elasticsearch configuration are not copied over
func (c *reconciler) Reconcile(name types.NamespacedName) error {
	reqLogger := log.WithFields(map[string]interface{}{
		"cluster": c.clusterName,
		"key":     name,
	})
	reqLogger.Info("Reconciling Elasticsearch credentials")

	currentESHash, err := c.esK8sCLI.CalculateTigeraElasticsearchHash()
	if err != nil {
		return err
	}

	if c.esHash != currentESHash {
		// Only reconcile the roles Elasticsearch has been changes in a way that may have wiped out the roles, or if
		// this is the first time Reconcile has run
		if err := c.reconcileRoles(); err != nil {
			return err
		}

		c.esHash = currentESHash
	}

	if err := c.reconcileUsers(reqLogger); err != nil {
		return err
	}

	if !c.management {
		if err := c.reconcileConfigMap(); err != nil {
			return err
		}

		if err := c.reconcileCASecrets(); err != nil {
			return err
		}
	}

	reqLogger.Info("Finished reconciling Elasticsearch credentials")

	return nil
}

func (c *reconciler) reconcileRoles() error {
	esCLI, err := c.getOrInitializeESClient()
	if err != nil {
		return err
	}

	roles := users.GetAuthorizationRoles(c.clusterName)
	return esCLI.CreateRoles(roles...)
}

// reconcileConfigMap copies the tigera-secure-elasticsearch ConfigMap in the management cluster to the managed cluster,
// changing the clusterName data value to the cluster name this ConfigMap is being copied to
func (c *reconciler) reconcileConfigMap() error {
	configMap, err := c.managementK8sCLI.CoreV1().ConfigMaps(resource.OperatorNamespace).Get(context.Background(), resource.ElasticsearchConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cp := resource.CopyConfigMap(configMap)
	cp.Data["clusterName"] = c.clusterName
	if err := resource.WriteConfigMapToK8s(c.managedK8sCLI, cp); err != nil {
		return err
	}
	return nil
}

// reconcileCASecrets copies the tigera-secure-es-http-certs-public and tigera-secure-kb-http-certs-public secrets from
// the management cluster to the managed cluster
func (c *reconciler) reconcileCASecrets() error {
	for _, secretName := range []string{resource.ElasticsearchCertSecret, resource.KibanaCertSecret} {
		secret, err := c.managementK8sCLI.CoreV1().Secrets(resource.OperatorNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if err := resource.WriteSecretToK8s(c.managedK8sCLI, resource.CopySecret(secret)); err != nil {
			return err
		}
	}

	return nil
}

// reconcileUsers makes sure that all the necessary users exist for a managed cluster in elasticsearch and that the managed
// cluster has access to those users via secrets
func (c *reconciler) reconcileUsers(reqLogger *log.Entry) error {
	staleOrMissingUsers, err := c.missingOrStaleUsers()
	if err != nil {
		return err
	}

	for username, user := range staleOrMissingUsers {
		reqLogger.Infof("creating user %s", username)
		if err := c.createUser(username, user); err != nil {
			return err
		}
	}

	return nil
}

// createUser creates the given elasticsearch user in elasticsearch and creates a secret in the managed cluster containing
// that users credentials
func (c *reconciler) createUser(username esusers.ElasticsearchUserName, esUser elasticsearch.User) error {
	esCLI, err := c.getOrInitializeESClient()
	if err != nil {
		return err
	}

	userPassword, err := randomPassword(16)
	if err != nil {
		return err
	}
	esUser.Password = userPassword
	if err := esCLI.CreateUser(esUser); err != nil {
		return err
	}

	changeHash, err := c.calculateUserChangeHash(esUser)
	if err != nil {
		return err
	}

	return resource.WriteSecretToK8s(c.managedK8sCLI, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-elasticsearch-access", username),
			Namespace: resource.OperatorNamespace,
			Labels: map[string]string{
				UserChangeHashLabel:        changeHash,
				ElasticsearchUserNameLabel: string(username),
			},
		},
		Data: map[string][]byte{
			"username": []byte(esUser.Username),
			"password": []byte(esUser.Password),
		},
	})
}

// missingOrStaleUsers returns a map of all the users that are missing from the cluster or have mismatched elasticsearch
// hashes (indicating that elasticsearch changed in a way that requires user credential recreation)
func (c *reconciler) missingOrStaleUsers() (map[esusers.ElasticsearchUserName]elasticsearch.User, error) {
	esUsers := esusers.ElasticsearchUsers(c.clusterName, c.management)
	secretsList, err := c.managedK8sCLI.CoreV1().Secrets(resource.OperatorNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: ElasticsearchUserNameLabel})
	if err != nil {
		return nil, err
	}

	for _, secret := range secretsList.Items {
		username := esusers.ElasticsearchUserName(secret.Labels[ElasticsearchUserNameLabel])
		if user, exists := esUsers[username]; exists {
			userHash, err := c.calculateUserChangeHash(user)
			if err != nil {
				return nil, err
			}
			if secret.Labels[UserChangeHashLabel] == userHash {
				delete(esUsers, username)
			}
		}
	}

	return esUsers, nil
}

func (c *reconciler) calculateUserChangeHash(user elasticsearch.User) (string, error) {
	return resource.CreateHashFromObject([]interface{}{c.esHash, c.ownerReference, user.Roles})
}

func (c *reconciler) getOrInitializeESClient() (elasticsearch.Client, error) {
	if c.esCLI == nil {
		var err error

		c.esCLI, err = c.esClientBuilder.Build()
		if err != nil {
			return nil, err
		}
	}

	return c.esCLI, nil
}

func randomPassword(length int) (string, error) {
	byts := make([]byte, length)
	_, err := rand.Read(byts)

	return base64.URLEncoding.EncodeToString(byts), err
}

// CleanUpESUserSecrets removes elasticsearch user secrets by label from the operator namespace.
// If Elasticsearch is removed, the secrets present in the tigera-operator namespace should expire.
func CleanUpESUserSecrets(clientset kubernetes.Interface) error {
	log.Info("removing expired elasticsearch secrets")
	// If no secrets are found, no 404/NotFound is returned when using labels.
	return clientset.CoreV1().Secrets(resource.OperatorNamespace).DeleteCollection(
		context.Background(),
		metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: ElasticsearchUserNameLabel,
		})
}
