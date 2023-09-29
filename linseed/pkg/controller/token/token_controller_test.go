// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package token_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/utils"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"

	"github.com/projectcalico/calico/linseed/pkg/controller/token"

	"github.com/stretchr/testify/require"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	projectcalicov3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	"github.com/projectcalico/calico/lma/pkg/k8s"
)

var (
	cs            clientset.Interface
	ctx           context.Context
	privateKey    *rsa.PrivateKey
	factory       *k8s.MockClientSetFactory
	mockK8sClient *k8sfake.Clientset
	mockClientSet clientSetSet

	tenantName string

	nilUserPtr *user.DefaultInfo

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

	tenantName = "bogustenant"

	nilUserPtr = nil

	// Set up a mock client set factory for the tests.
	factory = k8s.NewMockClientSetFactory(t)

	return func() {
		logCancel()
		cancel()
	}
}

func TestOptions(t *testing.T) {
	t.Run("Should reject invalid user info with no name", func(t *testing.T) {
		uis := []token.UserInfo{{Name: "", Namespace: defaultNamespace}}
		opt := token.WithUserInfos(uis)
		err := opt(nil)
		require.Error(t, err)
	})

	t.Run("Should reject invalid user info with no namespace", func(t *testing.T) {
		uis := []token.UserInfo{{Name: "service", Namespace: ""}}
		opt := token.WithUserInfos(uis)
		err := opt(nil)
		require.Error(t, err)
	})

	t.Run("Should make a new controller when correct options are given", func(t *testing.T) {
		defer setup(t)()
		opts := []token.ControllerOption{
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),
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
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

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
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

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
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),

			// Reconcile quickly, so that we can verify the secret isn't updated
			// across several reconciles.
			token.WithReconcilePeriod(50 * time.Millisecond),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

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
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),

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
		factory.On("Impersonate", nilUserPtr).Return(factory)

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
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),

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
		factory.On("Impersonate", nilUserPtr).Return(factory)

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

	t.Run("should not retry indefinitely", func(t *testing.T) {
		// If the controller fails to create a secret, it should retry a few times and then give up.
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

		// Configure the mock client to fail to create secrets, and keep track of the number of attempts.
		mu := sync.Mutex{}
		count := 0
		increment := func() {
			mu.Lock()
			defer mu.Unlock()
			count += 1
		}
		callsEqual := func(expected int) bool {
			mu.Lock()
			defer mu.Unlock()
			return count == expected
		}

		mockK8sClient.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor("create", "secrets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			increment()
			return true, &corev1.Secret{}, fmt.Errorf("Error creating secret")
		})

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithExpiry(30 * time.Minute),
			token.WithReconcilePeriod(1 * time.Minute),
			token.WithK8sClient(mockK8sClient),

			// Set a small initial retry period so that we exaust the retries quickly.
			token.WithBaseRetryPeriod(1 * time.Millisecond),
			token.WithMaxRetries(5),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory. We have one clientset for each managed cluster.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// We should expect 6 total attempts - 5 retries and 1 initial attempt.
		require.Eventually(t, func() bool {
			return callsEqual(6)
		}, 5*time.Second, 10*time.Millisecond)
		for i := 0; i < 5; i++ {
			require.True(t, callsEqual(6))
			time.Sleep(250 * time.Millisecond)
		}
	})

	t.Run("handle simultaneous periodic and triggered reconciles", func(t *testing.T) {
		defer setup(t)()

		// Add two managed clusters.
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

		mc2 := v3.ManagedCluster{}
		mc2.Name = "test-managed-cluster-2"
		mc2.Status.Conditions = []v3.ManagedClusterStatusCondition{
			{
				Type:   v3.ManagedClusterStatusTypeConnected,
				Status: v3.ManagedClusterStatusValueTrue,
			},
		}
		_, err = cs.ProjectcalicoV3().ManagedClusters().Create(ctx, &mc2, v1.CreateOptions{})
		require.NoError(t, err)

		// Configure the client to error on attempts to create secrets in the second managed cluster. Because this is constantly erroring,
		// it will result in the kickChan trigger being called repeatedly.
		mockK8sClient2 := k8sfake.NewSimpleClientset()
		mockClientSet2 := clientSetSet{mockK8sClient2, cs}
		mockK8sClient2.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor("create", "secrets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &corev1.Secret{}, fmt.Errorf("Error creating secret")
		})

		// Make a new controller.
		opts := []token.ControllerOption{
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),

			// Configure tokens to expire after 500ms. This means we should see several updates
			// over the course of this test.
			token.WithExpiry(500 * time.Millisecond),

			// Set the reconcile period to be very small so that the controller acts faster than
			// the expiry time of the tokens it creates.
			token.WithReconcilePeriod(100 * time.Millisecond),

			// Set the retry period to be smaller than either, so that we are constantly triggering
			// the kick channel.
			token.WithBaseRetryPeriod(50 * time.Millisecond),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		// Set the mock client set as the return value for the factory. We have one clientset for each managed cluster.
		factory.On("NewClientSetForApplication", mc.Name).Return(&mockClientSet, nil)
		factory.On("NewClientSetForApplication", mc2.Name).Return(&mockClientSet2, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		// Expect a token to have been generated for the first cluster.
		// This happens asynchronously, so we need to wait for the controller to finish processing.
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

	t.Run("verify VoltronLinseedCert propagation from management cluster to managed cluster due to periodic update", func(t *testing.T) {
		defer setup(t)()

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

		operatorNS := "test-operator-ns"
		err = os.Setenv("MANAGEMENT_OPERATOR_NS", operatorNS)
		require.NoError(t, err)

		voltronLinseedSecret := corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      resource.VoltronLinseedPublicCert,
				Namespace: operatorNS,
			},
		}
		secretsToCopy := []corev1.Secret{
			voltronLinseedSecret,
		}

		opts := []token.ControllerOption{
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),
			token.WithReconcilePeriod(1 * time.Second),
			token.WithSecretsToCopy(secretsToCopy),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		managedClientSet := clientSetSet{
			k8sfake.NewSimpleClientset(),
			fake.NewSimpleClientset(),
		}

		factory.On("NewClientSetForApplication", mc.Name).Return(&managedClientSet, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

		createdSecret, err := mockK8sClient.CoreV1().Secrets(operatorNS).Create(ctx, &voltronLinseedSecret, v1.CreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, createdSecret)

		// Reconcile
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		managedOperatorNS, err := utils.FetchOperatorNamespace(managedClientSet)
		require.NoError(t, err)

		secretCreated := func() bool {
			_, err = managedClientSet.CoreV1().Secrets(managedOperatorNS).Get(ctx, resource.VoltronLinseedPublicCert, v1.GetOptions{})
			return err == nil
		}
		require.Eventually(t, secretCreated, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("verify VoltronLinseedCert propagation from management cluster to managed cluster due to secret update", func(t *testing.T) {
		defer setup(t)()

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

		operatorNS := "test-operator-ns"
		err = os.Setenv("MANAGEMENT_OPERATOR_NS", operatorNS)
		require.NoError(t, err)

		voltronLinseedSecret := corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      resource.VoltronLinseedPublicCert,
				Namespace: operatorNS,
			},
			StringData: map[string]string{
				"key": "original-data",
			},
		}
		secretsToCopy := []corev1.Secret{
			voltronLinseedSecret,
		}

		opts := []token.ControllerOption{
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithK8sClient(mockK8sClient),
			token.WithReconcilePeriod(24 * time.Hour), // Make update period long enough that we're guaranteed not to trigger it during test
			token.WithSecretsToCopy(secretsToCopy),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		managedClientSet := clientSetSet{
			k8sfake.NewSimpleClientset(),
			fake.NewSimpleClientset(),
		}

		factory.On("NewClientSetForApplication", mc.Name).Return(&managedClientSet, nil)
		factory.On("Impersonate", nilUserPtr).Return(factory)

		createdSecret, err := mockK8sClient.CoreV1().Secrets(operatorNS).Create(ctx, &voltronLinseedSecret, v1.CreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, createdSecret)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		managedOperatorNS, err := utils.FetchOperatorNamespace(managedClientSet)
		require.NoError(t, err)

		// The controller will eventually cause the VoltronLinseedPublicCert to get copied into the managed cluster by
		// way of the ManagedCluster creation update. Wait for this to occur then update the data in the secret to make
		// sure we update correctly based on changes to the secret itself.
		originalSecretCreated := func() bool {
			_, err = managedClientSet.CoreV1().Secrets(managedOperatorNS).Get(ctx, resource.VoltronLinseedPublicCert, v1.GetOptions{})
			return err == nil
		}
		require.Eventually(t, originalSecretCreated, 5*time.Second, 100*time.Millisecond)

		// Update voltronLinseedSecret to trigger copy process
		updatedVoltronLinseedSecretData := "updated-data"
		updatedVoltronLinseedSecret := voltronLinseedSecret.DeepCopy()
		updatedVoltronLinseedSecret.StringData["key"] = updatedVoltronLinseedSecretData
		updatedSecret, err := mockK8sClient.CoreV1().Secrets(operatorNS).Update(ctx, updatedVoltronLinseedSecret, v1.UpdateOptions{})
		require.NoError(t, err)
		require.NotNil(t, updatedSecret)

		// Now verify that voltronLinseedSecret has been copied with updated data
		secretUpdated := func() bool {
			updatedSecret, err = managedClientSet.CoreV1().Secrets(managedOperatorNS).Get(ctx, resource.VoltronLinseedPublicCert, v1.GetOptions{})
			return updatedSecret.StringData["key"] == updatedVoltronLinseedSecretData
		}
		require.Eventually(t, secretUpdated, 5*time.Second, 100*time.Millisecond)
	})
}

func TestMultiTenant(t *testing.T) {
	t.Run("verify Impersonation headers are added", func(t *testing.T) {
		defer setup(t)()

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

		impersonationInfo := user.DefaultInfo{
			Name: tenantName,
			Groups: []string{
				serviceaccount.AllServiceAccountsGroup,
				"system:authenticated",
				fmt.Sprintf("%s%s", serviceaccount.ServiceAccountGroupPrefix, "tigera-elasticsearch"),
			},
		}

		operatorNS := "test-operator-ns"
		err = os.Setenv("MANAGEMENT_OPERATOR_NS", operatorNS)
		require.NoError(t, err)

		opts := []token.ControllerOption{
			token.WithCalicoClient(cs),
			token.WithPrivateKey(privateKey),
			token.WithIssuer(issuer),
			token.WithIssuerName(issuer),
			token.WithUserInfos([]token.UserInfo{{Name: defaultServiceName, Namespace: defaultNamespace}}),
			token.WithFactory(factory),
			token.WithTenant(tenantName),
			token.WithK8sClient(mockK8sClient),
			token.WithImpersonation(&impersonationInfo),
		}
		controller, err := token.NewController(opts...)
		require.NoError(t, err)
		require.NotNil(t, controller)

		managedClientSet := clientSetSet{
			k8sfake.NewSimpleClientset(),
			fake.NewSimpleClientset(),
		}

		factory.On("NewClientSetForApplication", mc.Name).Return(&managedClientSet, nil)
		factory.On("Impersonate", &impersonationInfo).Return(factory)

		// Reconcile.
		stopCh := make(chan struct{})
		defer close(stopCh)
		go controller.Run(stopCh)

		time.Sleep(5 * time.Second)
		// Verify that "NewClientSetForApplication" and "Impersonate" have been called at least once. We only really
		// care about "Impersonate" for the purposes of this particular test.
		factory.AssertExpectations(t)
	})
}

type clientSetSet struct {
	kubernetes.Interface
	Calico clientset.Interface
}

func (c *clientSetSet) ProjectcalicoV3() projectcalicov3.ProjectcalicoV3Interface {
	return c.Calico.ProjectcalicoV3()
}
