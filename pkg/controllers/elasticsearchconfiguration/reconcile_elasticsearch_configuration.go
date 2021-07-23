// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package elasticsearchconfiguration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	esusers "github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	relasticsearch "github.com/projectcalico/kube-controllers/pkg/resource/elasticsearch"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	// This value is used in calculateUserChangeHash() to force ES users to be considered 'stale' and re-created in case there
	// is version skew between the Managed and Management clusters. The value can be bumped anytime we change something about
	// the way ES credentials work and need to re-create them.
	EsUserCredentialsSchemaVersion = "1"

	// Mark any secret containing credentials for ES gateway with this label key/value. This will allow ES gateway watch only the
	// releveant secrets it needs.
	ESGatewaySelectorLabel      = "esgateway.tigera.io/secrets"
	ESGatewaySelectorLabelValue = "credentials"
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
		// Only reconcile the roles if Elasticsearch has been changed in a way that may have wiped out the roles, or if
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

	roles := esusers.GetAuthorizationRoles(c.clusterName)
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

// reconcileCASecrets copies tigera-secure-es-gateway-http-certs-public from the management cluster to the managed cluster.
func (c *reconciler) reconcileCASecrets() error {
	secret, err := c.managementK8sCLI.CoreV1().Secrets(resource.OperatorNamespace).Get(context.Background(), resource.ESGatewayCertSecret, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err := resource.WriteSecretToK8s(c.managedK8sCLI, resource.CopySecret(secret)); err != nil {
		return err
	}
	// To support older Managed clusters we need to also create the tigera-secure-es-http-certs-public secret containing the same cert
	// so that components configured to mount the old secret can still reach Elasticsearch in the Management cluster.
	secret.ObjectMeta.Name = resource.ElasticsearchCertSecret
	if err := resource.WriteSecretToK8s(c.managedK8sCLI, resource.CopySecret(secret)); err != nil {
		return err
	}
	return nil
}

// reconcileUsers makes sure that all the necessary users exist for a managed cluster in elasticsearch and that the managed
// cluster has access to those users via secrets
func (c *reconciler) reconcileUsers(reqLogger *log.Entry) error {
	staleOrMissingPrivateUsers, staleOrMissingPublicUsers, err := c.missingOrStaleUsers()
	if err != nil {
		return err
	}

	for username, user := range staleOrMissingPrivateUsers {
		reqLogger.Infof("creating user %s", username)
		if err := c.createUser(username, user, true); err != nil {
			return err
		}
	}

	for username, user := range staleOrMissingPublicUsers {
		reqLogger.Infof("creating user %s", username)
		if err := c.createUser(username, user, false); err != nil {
			return err
		}
	}

	return nil
}

// createUser creates the given Elasticsearch user in Elasticsearch if passed a private user and creates a secret containing that users credentials.
// Secrets containing private user credentials (real Elasticsearch credentials) can only be created in the Elasticsearch namespace
// in the Management cluster. Secrets containing public user credentials are created in the Operator namespace in either the Managed or
// Management cluster, as well as in the Elasticsearch namespace in the Management cluster. These public users are not actual Elasticsearch users.
// They are used by ES Gateway to authenticate components attempting to communicate with Elasticsearch and to then swap in credentials for real Elasticsearch users.
func (c *reconciler) createUser(username esusers.ElasticsearchUserName, esUser elasticsearch.User, elasticsearchUser bool) error {
	userPassword, err := randomPassword(16)
	if err != nil {
		return err
	}
	esUser.Password = userPassword

	changeHash, err := c.calculateUserChangeHash(esUser)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s-elasticsearch-access", string(username))
	data := map[string][]byte{
		"username": []byte(esUser.Username),
		"password": []byte(esUser.Password),
	}

	if elasticsearchUser {
		// Only private users are created in Elasticsearch.
		esCLI, err := c.getOrInitializeESClient()
		if err != nil {
			return err
		}
		if err := esCLI.CreateUser(esUser); err != nil {
			return err
		}

		if c.management {
			name = fmt.Sprintf("%s-elasticsearch-access-gateway", string(username))
		} else {
			name = fmt.Sprintf("%s-%s-elasticsearch-access-gateway", string(username), c.clusterName)
		}

		// Set required labels for the user secret.
		labels := map[string]string{
			UserChangeHashLabel:        changeHash,
			ElasticsearchUserNameLabel: string(username),
			ESGatewaySelectorLabel:     ESGatewaySelectorLabelValue,
		}

		// Create the user secret in the Management cluster Elasticsearch namespace.
		return writeUserSecret(name, resource.TigeraElasticsearchNamespace, labels, c.managementK8sCLI, data)
	}

	// Set required labels for the user secret. We leave out the ES Gateway label initially here because the first write
	// below is to the Operator namespace, which doesn't require this label.
	labels := map[string]string{
		UserChangeHashLabel:        changeHash,
		ElasticsearchUserNameLabel: string(username),
	}

	// First create the secret in the Operator namespace in the appropriate cluster. If this is a Management cluster,
	// c.managedK8sCLI actually contains a client for the Management cluster.
	if err := writeUserSecret(name, resource.OperatorNamespace, labels, c.managedK8sCLI, data); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(esUser.Password), bcrypt.MinCost)
	if err != nil {
		return err
	}
	data["password"] = hash

	if c.management {
		name = fmt.Sprintf("%s-gateway-verification-credentials", string(username))
	} else {
		name = fmt.Sprintf("%s-%s-gateway-verification-credentials", string(username), c.clusterName)
	}

	// Now insert the additional label for ES Gateway (so that it knows to include this secret in its watch).
	labels[ESGatewaySelectorLabel] = ESGatewaySelectorLabelValue

	// Next create the user secret with the username and password hashed in the Management cluster Elasticsearch namespace.
	return writeUserSecret(name, resource.TigeraElasticsearchNamespace, labels, c.managementK8sCLI, data)
}

// missingOrStaleUsers returns 2 maps, the first containing private users and the second containing public users that are
// missing from the cluster or have mismatched elasticsearch hashes (indicating that elasticsearch changed in a way that requires user credential recreation).
func (c *reconciler) missingOrStaleUsers() (map[esusers.ElasticsearchUserName]elasticsearch.User, map[esusers.ElasticsearchUserName]elasticsearch.User, error) {
	privateEsUsers, publicEsUsers := esusers.ElasticsearchUsers(c.clusterName, c.management)

	publicSecretsList, err := c.managedK8sCLI.CoreV1().Secrets(resource.OperatorNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: ElasticsearchUserNameLabel})
	if err != nil {
		return nil, nil, err
	}

	privateSecretsList, err := c.managementK8sCLI.CoreV1().Secrets(resource.TigeraElasticsearchNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: ElasticsearchUserNameLabel})
	if err != nil {
		return nil, nil, err
	}

	for _, secret := range publicSecretsList.Items {
		username := esusers.ElasticsearchUserName(secret.Labels[ElasticsearchUserNameLabel])
		if user, exists := publicEsUsers[username]; exists {
			userHash, err := c.calculateUserChangeHash(user)
			if err != nil {
				return nil, nil, err
			}
			if secret.Labels[UserChangeHashLabel] == userHash {
				delete(publicEsUsers, username)
			}
		}
	}

	for _, secret := range privateSecretsList.Items {
		if strings.HasSuffix(secret.Name, "gateway-verification-credentials") {
			break
		}
		username := esusers.ElasticsearchUserName(secret.Labels[ElasticsearchUserNameLabel])
		if user, exists := privateEsUsers[username]; exists {
			userHash, err := c.calculateUserChangeHash(user)
			if err != nil {
				return nil, nil, err
			}
			if secret.Labels[UserChangeHashLabel] == userHash {
				delete(privateEsUsers, username)
			}
		}
	}

	return privateEsUsers, publicEsUsers, nil
}

func (c *reconciler) calculateUserChangeHash(user elasticsearch.User) (string, error) {
	return resource.CreateHashFromObject([]interface{}{c.esHash, c.ownerReference, user.Roles, EsUserCredentialsSchemaVersion})
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

func writeUserSecret(name, namespace string, labels map[string]string, client kubernetes.Interface, data map[string][]byte) error {
	return resource.WriteSecretToK8s(client, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	})
}
