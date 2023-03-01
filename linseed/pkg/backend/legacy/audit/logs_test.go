// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package audit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/projectcalico/calico/linseed/pkg/testutils"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kaudit "k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/kubernetes/pkg/apis/apps"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/audit"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	b       bapi.AuditBackend
	ctx     context.Context
	cluster string
)

// setupTest runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupTest(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	client = lmaelastic.NewWithClient(esClient)
	cache := templates.NewTemplateCache(client, 1, 0)

	// Instantiate a backend.
	b = audit.NewBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = backendutils.RandomClusterName()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)

	// Function contains teardown logic.
	return func() {
		err = backendutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_audit_kube.%s", cluster))
		require.NoError(t, err)
		err = backendutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_audit_ee.%s", cluster))
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

// TestCreateKubeAuditLog tests running a real elasticsearch query to create a kube audit log.
func TestCreateKubeAuditLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// The DaemonSet that this audit log is for.
	ds := apps.DaemonSet{
		TypeMeta: metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
	}
	dsRaw, err := json.Marshal(ds)
	require.NoError(t, err)

	f := v1.AuditLog{
		Event: kaudit.Event{
			TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "audit.k8s.io/v1"},
			AuditID:    types.UID("some-uuid-most-likely"),
			Stage:      kaudit.StageRequestReceived,
			RequestURI: "/apis/v1/namespaces",
			Verb:       "GET",
			User: authnv1.UserInfo{
				Username: "user",
				UID:      "uid",
				Extra:    map[string]authnv1.ExtraValue{"extra": authnv1.ExtraValue([]string{"value"})},
			},
			ImpersonatedUser: &authnv1.UserInfo{
				Username: "impuser",
				UID:      "impuid",
				Groups:   []string{"g1"},
			},
			SourceIPs:      []string{"1.2.3.4"},
			UserAgent:      "user-agent",
			ObjectRef:      &kaudit.ObjectReference{},
			ResponseStatus: &metav1.Status{},
			RequestObject: &runtime.Unknown{
				Raw:         dsRaw,
				ContentType: runtime.ContentTypeJSON,
			},
			ResponseObject: &runtime.Unknown{
				Raw:         dsRaw,
				ContentType: runtime.ContentTypeJSON,
			},
			RequestReceivedTimestamp: metav1.NewMicroTime(time.Now().Add(-5 * time.Second)),
			StageTimestamp:           metav1.NewMicroTime(time.Now()),
			Annotations:              map[string]string{"brick": "red"},
		},
		Name: testutils.StringPtr("any"),
	}

	// Create the event in ES.
	resp, err := b.Create(ctx, v1.AuditLogTypeKube, clusterInfo, []v1.AuditLog{f})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.Errors))

	// Refresh the index.
	err = backendutils.RefreshIndex(ctx, client, fmt.Sprintf("tigera_secure_ee_audit_kube.%s.*", clusterInfo.Cluster))
	require.NoError(t, err)

	// List the event, assert that it matches the one we just wrote.
	results, err := b.List(ctx, clusterInfo, &v1.AuditLogParams{Type: v1.AuditLogTypeKube})
	require.NoError(t, err)
	require.Equal(t, 1, len(results.Items))

	// MicroTime doesn't JSON serialize and deserialize properly, so we need to force the results to
	// match here. When you serialize and deserialize a MicroTime, the microsecond precision is lost
	// and so the resulting objects do not match.
	f.RequestReceivedTimestamp = results.Items[0].RequestReceivedTimestamp
	f.StageTimestamp = results.Items[0].StageTimestamp
	require.Equal(t, f, results.Items[0])
}

// TestCreateEEAuditLog tests running a real elasticsearch query to create a EE audit log.
func TestCreateEEAuditLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// The NetworkSet that this audit log is for.
	obj := v3.GlobalNetworkSet{
		TypeMeta: metav1.TypeMeta{Kind: "GlobalNetworkSet", APIVersion: "projectcalico.org/v3"},
	}
	objRaw, err := json.Marshal(obj)
	require.NoError(t, err)

	f := v1.AuditLog{
		Event: kaudit.Event{
			TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "audit.k8s.io/v1"},
			AuditID:    types.UID("some-uuid-most-likely"),
			Stage:      kaudit.StageRequestReceived,
			RequestURI: "/apis/v3/projectcalico.org",
			Verb:       "PUT",
			User: authnv1.UserInfo{
				Username: "user",
				UID:      "uid",
				Extra:    map[string]authnv1.ExtraValue{"extra": authnv1.ExtraValue([]string{"value"})},
			},
			ImpersonatedUser: &authnv1.UserInfo{
				Username: "impuser",
				UID:      "impuid",
				Groups:   []string{"g1"},
			},
			SourceIPs:      []string{"1.2.3.4"},
			UserAgent:      "user-agent",
			ObjectRef:      &kaudit.ObjectReference{},
			ResponseStatus: &metav1.Status{},
			RequestObject: &runtime.Unknown{
				Raw:         objRaw,
				ContentType: runtime.ContentTypeJSON,
			},
			ResponseObject: &runtime.Unknown{
				Raw:         objRaw,
				ContentType: runtime.ContentTypeJSON,
			},
			RequestReceivedTimestamp: metav1.NewMicroTime(time.Now().Add(-5 * time.Second)),
			StageTimestamp:           metav1.NewMicroTime(time.Now()),
			Annotations:              map[string]string{"brick": "red"},
		},
		Name: testutils.StringPtr("ee-any"),
	}

	// Create the event in ES.
	resp, err := b.Create(ctx, v1.AuditLogTypeEE, clusterInfo, []v1.AuditLog{f})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.Errors))

	// Refresh the index.
	err = backendutils.RefreshIndex(ctx, client, fmt.Sprintf("tigera_secure_ee_audit_ee.%s.*", clusterInfo.Cluster))
	require.NoError(t, err)

	// List the event, assert that it matches the one we just wrote.
	results, err := b.List(ctx, clusterInfo, &v1.AuditLogParams{Type: v1.AuditLogTypeEE})
	require.NoError(t, err)
	require.Equal(t, 1, len(results.Items))

	// MicroTime doesn't JSON serialize and deserialize properly, so we need to force the results to
	// match here. When you serialize and deserialize a MicroTime, the microsecond precision is lost
	// and so the resulting objects do not match.
	f.RequestReceivedTimestamp = results.Items[0].RequestReceivedTimestamp
	f.StageTimestamp = results.Items[0].StageTimestamp
	require.Equal(t, f, results.Items[0])
}
