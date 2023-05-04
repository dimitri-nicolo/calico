// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package token_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	projectcalicov3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	"github.com/projectcalico/calico/linseed/pkg/controller/token"
	"github.com/projectcalico/calico/lma/pkg/k8s"
)

var (
	cs            clientset.Interface
	ctx           context.Context
	privateKey    *rsa.PrivateKey
	factory       *k8s.MockClientSetFactory
	mockK8sClient *k8sfake.Clientset
	mockClientSet clientSetSet

	// Default values for tokens to be created in the tests.
	issuer             string = "testissuer"
	defaultServiceName string = "servicename"
	defaultNamespace   string = "default"
	tokenName          string = "servicename-testissuer-token"
)

func setup(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)

	// Create fake client sets.
	cs = fake.NewSimpleClientset()

	// Generate a private key for the tests.
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Set up expected mock calls. We expect a clientset to be generated for the
	// managed cluster which will be used to check and create a secret.
	mockK8sClient = k8sfake.NewSimpleClientset()
	mockClientSet = clientSetSet{mockK8sClient, cs}

	// Set up a mock client set factory for the tests.
	factory = k8s.NewMockClientSetFactory(t)

	return func() {
		logCancel()
		cancel()
	}
}

func TestOptions(t *testing.T) {
	t.Run("Should reject invalid user info with no name", func(t *testing.T) {
		defer setup(t)()

		uis := []token.UserInfo{{Name: "", Namespace: defaultNamespace}}
		opt := token.WithUserInfos(uis)
		err := opt(nil)
		require.Error(t, err)
	})

	t.Run("Should reject invalid user info with no namespace", func(t *testing.T) {
		defer setup(t)()

		uis := []token.UserInfo{{Name: "service", Namespace: ""}}
		opt := token.WithUserInfos(uis)
		err := opt(nil)
		require.Error(t, err)
	})

	t.Run("Should make a new controller when correct options are given", func(t *testing.T) {
		defer setup(t)()

		opts := []token.ControllerOption{
			token.WithClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)
	})
}

func TestMainlineFunction(t *testing.T) {
	t.Run("provision a secret for a service in a connected managed cluster", func(t *testing.T) {
		defer setup(t)()

		// Add a managed cluster.
		mc := v3.ManagedCluster{}
		mc.Name = "test-managed-cluster"
		mc.Status.Conditions = []v3.ManagedClusterStatusCondition{
			{
				Type:   v3.ManagedClusterStatusTypeConnected,
				Status: v3.ManagedClusterStatusValueTrue,
			},
		}
		_, err := cs.ProjectcalicoV3().ManagedClusters().Create(ctx, &mc, v1.CreateOptions{})
		require.NoError(t, err)

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// Expect a token to have been generated. This happens asynchronously, so we need
		// to wait for the controller to finish processing.
		var secret *corev1.Secret
		secretCreated := func() bool {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			return err == nil
		}
		require.Eventually(t, secretCreated, 5*time.Second, 100*time.Millisecond)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, defaultNamespace, secret.Namespace)
	})

	t.Run("provision a secret for a service when a managed cluster becomes connected", func(t *testing.T) {
		defer setup(t)()

		// Add a managed cluster - start it off as not connected. We will expect no secret
		// in this case. Then, we'll connect the cluster and make sure a secret is created.
		mc := v3.ManagedCluster{}
		mc.Name = "test-managed-cluster"
		mc.Status.Conditions = []v3.ManagedClusterStatusCondition{
			{
				Type:   v3.ManagedClusterStatusTypeConnected,
				Status: v3.ManagedClusterStatusValueFalse,
			},
		}
		_, err := cs.ProjectcalicoV3().ManagedClusters().Create(ctx, &mc, v1.CreateOptions{})
		require.NoError(t, err)

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// No token should be created yet.
		var secret *corev1.Secret
		secretCreated := func() bool {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			return !errors.IsNotFound(err)
		}
		for i := 0; i < 5; i++ {
			require.False(t, secretCreated())
			require.Nil(t, secret)
			time.Sleep(1 * time.Second)
		}

		// Mark the cluster as connected. This should eventually trigger creation of the secret.
		mc.Status.Conditions[0].Status = v3.ManagedClusterStatusValueTrue
		_, err = cs.ProjectcalicoV3().ManagedClusters().Update(ctx, &mc, v1.UpdateOptions{})
		require.NoError(t, err)
		require.Eventually(t, secretCreated, 5*time.Second, 100*time.Millisecond)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, defaultNamespace, secret.Namespace)
	})

	t.Run("skip updating an already valid token", func(t *testing.T) {
		defer setup(t)()

		// Add a managed cluster.
		mc := v3.ManagedCluster{}
		mc.Name = "test-managed-cluster"
		mc.Status.Conditions = []v3.ManagedClusterStatusCondition{
			{
				Type:   v3.ManagedClusterStatusTypeConnected,
				Status: v3.ManagedClusterStatusValueTrue,
			},
		}
		_, err := cs.ProjectcalicoV3().ManagedClusters().Create(ctx, &mc, v1.CreateOptions{})
		require.NoError(t, err)

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),

			// Reconcile quickly, so that we can verify the secret isn't updated
			// across several reconciles.
			token.WithReconcilePeriod(50 * time.Millisecond),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// Expect a token to have been generated. This happens asynchronously, so we need
		// to wait for the controller to finish processing.
		var secret *corev1.Secret
		secretCreated := func() bool {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			return err == nil
		}
		require.Eventually(t, secretCreated, 5*time.Second, 100*time.Millisecond)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, defaultNamespace, secret.Namespace)

		// The token should remain the same across multiple reconciles, since it is still valid.
		oldSecret := *secret
		for i := 0; i < 5; i++ {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			require.NoError(t, err)
			require.NotNil(t, secret)
			require.Equal(t, oldSecret, *secret, "Secret changed unexpectedly")
			time.Sleep(1 * time.Second)
		}
	})

	t.Run("update an existing token that isn't valid", func(t *testing.T) {
		defer setup(t)()

		// Add a managed cluster.
		mc := v3.ManagedCluster{}
		mc.Name = "test-managed-cluster"
		mc.Status.Conditions = []v3.ManagedClusterStatusCondition{
			{
				Type:   v3.ManagedClusterStatusTypeConnected,
				Status: v3.ManagedClusterStatusValueTrue,
			},
		}
		_, err := cs.ProjectcalicoV3().ManagedClusters().Create(ctx, &mc, v1.CreateOptions{})
		require.NoError(t, err)

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),

			// Set the reconcile period to be very small so that the controller can reconcile
			// the changes we make to the token. Ideally, the controller would be watching
			// the secret and we wouldn't need this.
			token.WithReconcilePeriod(10 * time.Millisecond),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// Expect a token to have been generated. This happens asynchronously, so we need
		// to wait for the controller to finish processing.
		var secret *corev1.Secret
		secretCreated := func() bool {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			return err == nil
		}
		require.Eventually(t, secretCreated, 5*time.Second, 100*time.Millisecond)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, defaultNamespace, secret.Namespace)

		// Modify the token so that it's no longer valid. The controller should notice that the token is
		// invalid and replace it.
		invalidSecret := *secret
		invalidSecret.Data["token"] = []byte(fmt.Sprintf("%s-modified", invalidSecret.Data["token"]))
		_, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Update(ctx, &invalidSecret, v1.UpdateOptions{})
		require.NoError(t, err)

		// Eventually the secret should be updated back to a valid token by the controller's normal reconcile loop.
		secretUpdated := func() bool {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			if err != nil {
				return false
			}
			if reflect.DeepEqual(*secret, invalidSecret) {
				return false
			}
			return true
		}
		require.Eventually(t, secretUpdated, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("update secrets before they expire", func(t *testing.T) {
		defer setup(t)()

		// Add a managed cluster.
		mc := v3.ManagedCluster{}
		mc.Name = "test-managed-cluster"
		mc.Status.Conditions = []v3.ManagedClusterStatusCondition{
			{
				Type:   v3.ManagedClusterStatusTypeConnected,
				Status: v3.ManagedClusterStatusValueTrue,
			},
		}
		_, err := cs.ProjectcalicoV3().ManagedClusters().Create(ctx, &mc, v1.CreateOptions{})
		require.NoError(t, err)

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),

			// Configure tokens to expire after 50ms. This means we should see several updates
			// over the course of this test.
			token.WithExpiry(50 * time.Millisecond),

			// Set the reconcile period to be very small so that the controller acts faster than
			// the expiry time of the tokens it creates.
			token.WithReconcilePeriod(10 * time.Millisecond),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// Expect a token to have been generated. This happens asynchronously, so we need
		// to wait for the controller to finish processing.
		var secret *corev1.Secret
		secretCreated := func() bool {
			secret, err = mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			return err == nil
		}
		require.Eventually(t, secretCreated, 5*time.Second, 100*time.Millisecond)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, defaultNamespace, secret.Namespace)

		// Eventually the secret should be updated to a new token due to the approaching expiry.
		secretUpdated := func() bool {
			newSecret, err := mockK8sClient.CoreV1().Secrets(defaultNamespace).Get(ctx, tokenName, v1.GetOptions{})
			if err != nil {
				return false
			}
			if reflect.DeepEqual(secret, newSecret) {
				return false
			}
			return true
		}
		require.Eventually(t, secretUpdated, 5*time.Second, 50*time.Millisecond)
	})
}

type clientSetSet struct {
	kubernetes.Interface
	Calico clientset.Interface
}

func (c *clientSetSet) ProjectcalicoV3() projectcalicov3.ProjectcalicoV3Interface {
	return c.Calico.ProjectcalicoV3()
}
