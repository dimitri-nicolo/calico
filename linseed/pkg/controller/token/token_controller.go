// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package token

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/SermoDigital/jose/jws"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calico "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/client/informers_generated/externalversions"
	v3listers "github.com/tigera/api/pkg/client/listers_generated/projectcalico/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/lma/pkg/k8s"
)

const (
	LinseedIssuer        string = "linseed.tigera.io"
	defaultTokenLifetime        = 24 * time.Hour
)

type Controller interface {
	Run(<-chan struct{})
}

type ControllerOption func(*controller) error

func WithPrivateKey(k *rsa.PrivateKey) ControllerOption {
	return func(c *controller) error {
		c.privateKey = k
		return nil
	}
}

// WithClient configures the Calico client used to access managed cluster resources.
func WithClient(cs calico.Interface) ControllerOption {
	return func(c *controller) error {
		c.managementClient = cs
		return nil
	}
}

func WithTenant(tenant string) ControllerOption {
	return func(c *controller) error {
		c.tenant = tenant
		return nil
	}
}

// WithIssuer sets the issuer of the generated tokens.
func WithIssuer(iss string) ControllerOption {
	return func(c *controller) error {
		c.issuer = iss
		return nil
	}
}

// WithIssuerName sets the name of the token issuer, used when generating
// names for token secrets in managed clusters.
func WithIssuerName(name string) ControllerOption {
	return func(c *controller) error {
		c.issuerName = name
		return nil
	}
}

// WithExpiry sets the duration that generated tokens should be valid for.
func WithExpiry(d time.Duration) ControllerOption {
	return func(c *controller) error {
		c.expiry = d
		return nil
	}
}

// WithFactory sets the factory to use for generating per-cluster clients.
func WithFactory(f k8s.ClientSetFactory) ControllerOption {
	return func(c *controller) error {
		c.factory = f
		return nil
	}
}

type UserInfo struct {
	Name      string
	Namespace string
}

// WithUserInfos sets the users in each managed cluster that this controller
// should generate tokens for.
func WithUserInfos(s []UserInfo) ControllerOption {
	return func(c *controller) error {
		for _, sa := range s {
			if sa.Name == "" {
				return fmt.Errorf("missing Name field in UserInfo")
			}
			if sa.Namespace == "" {
				return fmt.Errorf("missing Namespace field in UserInfo")
			}

		}
		c.userInfos = s
		return nil
	}
}

func WithReconcilePeriod(t time.Duration) ControllerOption {
	return func(c *controller) error {
		c.reconcilePeriod = &t
		return nil
	}
}

// WithBaseRetryPeriod sets the base retry period for retrying failed operations.
// The actual retry period is calculated as baseRetryPeriod * 2^retryCount.
func WithBaseRetryPeriod(t time.Duration) ControllerOption {
	return func(c *controller) error {
		c.baseRetryPeriod = &t
		return nil
	}
}

func WithMaxRetries(n int) ControllerOption {
	return func(c *controller) error {
		c.maxRetries = &n
		return nil
	}
}

func WithHealthReport(reportHealth func(*health.HealthReport)) ControllerOption {
	return func(c *controller) error {
		c.reportHealth = reportHealth
		return nil
	}
}

func NewController(opts ...ControllerOption) (Controller, error) {
	c := &controller{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	// Default anything not set.
	if c.reconcilePeriod == nil {
		d := 60 * time.Minute
		c.reconcilePeriod = &d
	}
	if c.baseRetryPeriod == nil {
		d := 1 * time.Second
		c.baseRetryPeriod = &d
	}
	if c.maxRetries == nil {
		n := 20
		c.maxRetries = &n
	}

	// Verify necessary options set.
	if c.managementClient == nil {
		return nil, fmt.Errorf("must provide a management cluster calico client")
	}
	if c.privateKey == nil {
		return nil, fmt.Errorf("must provide a private key")
	}
	if c.issuer == "" {
		return nil, fmt.Errorf("must provide an issuer")
	}
	if c.issuerName == "" {
		return nil, fmt.Errorf("must provide an issuer name")
	}
	if len(c.userInfos) == 0 {
		return nil, fmt.Errorf("must provide at least one user info")
	}
	if c.factory == nil {
		return nil, fmt.Errorf("must provide a clientset factory")
	}
	return c, nil
}

type controller struct {
	// Input configuration.
	privateKey       *rsa.PrivateKey
	tenant           string
	issuer           string
	issuerName       string
	managementClient calico.Interface
	expiry           time.Duration
	reconcilePeriod  *time.Duration
	baseRetryPeriod  *time.Duration
	maxRetries       *int
	reportHealth     func(*health.HealthReport)
	factory          k8s.ClientSetFactory

	// userInfos in the managed cluster that we should provision tokens for.
	userInfos []UserInfo
}

func (c *controller) Run(stopCh <-chan struct{}) {
	// TODO: Support multiple copies of this running.

	// Start a watch on ManagedClusters, wait for it to sync, and then proceed.
	// We'll trigger events whenever a new cluster is added, causing us to check whether
	// we need to provision token secrets in that cluster.
	logrus.Info("Starting token controller")

	// Create an informer for watching managed clusters.
	factory := externalversions.NewSharedInformerFactory(c.managementClient, 0)
	managedClusterInformer := factory.Projectcalico().V3().ManagedClusters().Informer()

	// Make channels for sending updates.
	kickChan := make(chan string, 100)
	defer close(kickChan)

	// Register handlers for events.
	handler := cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {},
		AddFunc: func(obj interface{}) {
			if mc, ok := obj.(*v3.ManagedCluster); ok && isConnected(mc) {
				kickChan <- mc.Name
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			if mc, ok := obj.(*v3.ManagedCluster); ok && isConnected(mc) {
				kickChan <- mc.Name
			}
		},
	}
	_, err := managedClusterInformer.AddEventHandler(handler)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to add event handler")
	}

	// Start the informer.
	logrus.Info("Waiting for token controller to sync")
	go managedClusterInformer.Run(stopCh)
	for !managedClusterInformer.HasSynced() {
		time.Sleep(1 * time.Second)
	}
	logrus.Info("Token controller has synced with ManagedCluster API")

	// Start the token manager.
	c.ManageTokens(
		stopCh,
		kickChan,
		v3listers.NewManagedClusterLister(managedClusterInformer.GetIndexer()),
	)
}

func isConnected(mc *v3.ManagedCluster) bool {
	for _, s := range mc.Status.Conditions {
		if s.Type == v3.ManagedClusterStatusTypeConnected {
			return s.Status == v3.ManagedClusterStatusValueTrue
		}
	}
	logrus.WithField("cluster", mc.Name).Debug("ManagedCluster is not connected")
	return false
}

func (c *controller) ManageTokens(stop <-chan struct{}, kickChan chan string, lister v3listers.ManagedClusterLister) {
	defer logrus.Info("Token manager shutting down")

	// Local helper function for reconciling.
	reconcile := func(clusterName string) error {
		if clusterName == "" {
			logrus.Warn("No cluster name given")
			return nil
		}
		logrus.WithField("cluster", clusterName).Info("Reconciling tokens for cluster")
		client, err := c.factory.NewClientSetForApplication(clusterName)
		if err != nil {
			logrus.WithError(err).Warn("failed to get client for cluster")
			return err
		}
		err = c.reconcileTokens(clusterName, client)
		if err != nil {
			logrus.WithError(err).Warn("Error reconciling tokens")
			return err
		}
		logrus.WithField("cluster", clusterName).Debug("Token reconciliation complete")
		return nil
	}

	ticker := time.After(*c.reconcilePeriod)
	rc := NewRetryCalculator(*c.baseRetryPeriod, *c.maxRetries)

	// Main loop.
	for {
		select {
		case <-stop:
			return
		case <-ticker:
			logrus.Info("Reconciling all clusters tokens")

			// Start a new ticker.
			ticker = time.After(*c.reconcilePeriod)

			// Get all clusters.
			items, err := lister.List(labels.Everything())
			if err != nil {
				logrus.WithError(err).Error("Failed to list managed clusters")
				continue
			}

			for _, mc := range items {
				if isConnected(mc) {
					if err = reconcile(mc.Name); err != nil {
						logrus.WithError(err).WithField("cluster", mc.Name).Warn("Error reconciling cluster")
					}
				}
			}
			if c.reportHealth != nil {
				c.reportHealth(&health.HealthReport{Live: true, Ready: true})
			}
		case name := <-kickChan:
			if name != "" {
				if err := reconcile(name); err != nil {
					// Check if we should retry this cluster.
					retry, dur := rc.duration(name)
					if !retry {
						logrus.WithError(err).WithField("cluster", name).Warn("Giving up on cluster")
						continue
					}

					// Schedule a retry.
					go func(n string, d time.Duration, ch chan string) {
						logrus.WithError(err).WithField("wait", d).WithField("cluster", name).Info("Scheduling retry for failed sync")
						time.Sleep(d)

						// Use select to prevent accidentally sending to a closed channel if the controller initiated a shut down
						// while this routine was sleeping.
						select {
						case <-stop:
						default:
							ch <- n
						}
					}(name, dur, kickChan)
				}
			}
		}
	}
}

func NewRetryCalculator(start time.Duration, maxRetries int) *retryCalculator {
	return &retryCalculator{
		startDuration:      start,
		maxRetries:         maxRetries,
		outstandingRetries: map[string]time.Duration{},
		numRetries:         map[string]int{},
	}
}

type retryCalculator struct {
	startDuration      time.Duration
	outstandingRetries map[string]time.Duration
	numRetries         map[string]int
	maxRetries         int
}

// duration returns the next duration to use when retrying the given key.
// after a max number of retries, it will return (false, 0) to indicate that we should give up.
func (r *retryCalculator) duration(key string) (bool, time.Duration) {
	if r.numRetries[key] >= r.maxRetries {
		// Give up.
		delete(r.numRetries, key)
		delete(r.outstandingRetries, key)
		return false, 0 * time.Second
	}
	r.numRetries[key]++

	if d, ok := r.outstandingRetries[key]; ok {
		// Double the duration, up to a maximum of 1 minute.
		d = d * 2
		if d > 1*time.Minute {
			d = 1 * time.Minute
		}
		r.outstandingRetries[key] = d
		return true, d
	} else {
		// First time we've seen this key.
		d = r.startDuration
		r.outstandingRetries[key] = d
		return true, d
	}
}

// reconcileTokens reconciles tokens. This is a hack and should be moved to its own location.
func (c *controller) reconcileTokens(cluster string, managedClient kubernetes.Interface) error {
	for _, user := range c.userInfos {
		f := logrus.Fields{
			"cluster": cluster,
			"tenant":  c.tenant,
			"service": user.Name,
		}
		log := logrus.WithFields(f)

		// First, check if token exists. If it does, we don't need to do anything.
		tokenName := c.tokenNameForService(user.Name)
		if update, err := c.needsUpdate(log, managedClient, tokenName, user.Namespace); err != nil {
			log.WithError(err).Error("error checking token")
			return err
		} else if !update {
			log.Debug("Token does not need to be updated")
			continue
		}

		// Token needs to be created or updated.
		token, err := c.createToken(c.tenant, cluster, user)
		if err != nil {
			return err
		}

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tokenName,
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"token": token,
			},
		}

		if err := resource.WriteSecretToK8s(managedClient, resource.CopySecret(&secret)); err != nil {
			return err
		}
		log.WithField("name", secret.Name).Info("Created/updated token secret")
	}
	return nil
}

func (c *controller) tokenNameForService(service string) string {
	// Secret names should be identified by:
	// - The issuer of the token
	// - The service the token is being created for
	return fmt.Sprintf("%s-%s-token", service, c.issuerName)
}

func (c *controller) needsUpdate(log *logrus.Entry, cs kubernetes.Interface, name, namespace string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cm, err := cs.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		// Error querying the token.
		return false, err
	} else if errors.IsNotFound(err) {
		// No token exists.
		return true, nil
	} else {
		// Validate the token to make sure it was signed by us.
		tokenBytes := []byte(cm.Data["token"])
		_, err = jwt.ParseWithClaims(string(tokenBytes), &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			return c.privateKey.Public(), nil
		})
		if err != nil {
			// If the token is not signed by us, we should replace it. This covers two cases:
			// - User has manually specified a new invalid token in the secret.
			// - We're using a new cert to sign tokens, invalidating any and all tokens that we
			//   had previously distributed to clients.
			log.WithError(err).Warn("Could not authenticate token")
			return true, nil
		}

		// Parse the token to get its expiry.
		tkn, err := jws.ParseJWT(tokenBytes)
		if err != nil {
			log.WithError(err).Warn("failed to parse token")
			return true, nil
		}
		expiry, exists := tkn.Claims().Expiration()
		if !exists {
			log.Info("token has no expiration data present")
			return true, nil
		}

		// Refresh the token if the time between the expiry and now
		// is less than 2/3 of the total expiry time.
		dur := 2 * c.expiry / 3
		if time.Until(expiry) < dur {
			log.Info("token needs to be refreshed")
			return true, nil
		}

	}
	return false, nil
}

func (c *controller) createToken(tenant, cluster string, user UserInfo) ([]byte, error) {
	tokenLifetime := c.expiry
	if tokenLifetime == 0 {
		tokenLifetime = defaultTokenLifetime
	}
	expirationTime := time.Now().Add(tokenLifetime)

	// Subject is a combination of tenantID, clusterID, and service name.
	subj := fmt.Sprintf("%s:%s:%s:%s", tenant, cluster, user.Namespace, user.Name)

	claims := &jwt.RegisteredClaims{
		Subject:   subj,
		Issuer:    c.issuer,
		Audience:  jwt.ClaimStrings{c.issuerName},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(c.privateKey)
	if err != nil {
		return nil, err
	}
	return []byte(tokenString), err
}

func ParseSubjectLinseed(subject string) (tenant, cluster, namespace, name string, err error) {
	splits := strings.Split(subject, ":")
	if len(splits) != 4 {
		return "", "", "", "", fmt.Errorf("bad subject")
	}
	return splits[0], splits[1], splits[2], splits[3], nil
}

// ParseClaimsLinseed implements ClaimParser for token claims generated by Linseed.
func ParseClaimsLinseed(claims jwt.Claims) (*user.DefaultInfo, error) {
	reg, ok := claims.(*jwt.RegisteredClaims)
	if !ok {
		logrus.WithField("claims", claims).Warn("given claims were not a RegisteredClaims")
		return nil, fmt.Errorf("invalid claims given")
	}
	_, _, namespace, name, err := ParseSubjectLinseed(reg.Subject)
	if err != nil {
		return nil, err
	}
	return &user.DefaultInfo{Name: fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)}, nil
}
