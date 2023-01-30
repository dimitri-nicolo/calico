// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package audit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kaudit "k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/kubernetes/pkg/apis/apps"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/audit"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// TestCreateAuditLog tests running a real elasticsearch query to create a log.
func TestCreateAuditLog(t *testing.T) {
	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)

	// Instantiate a backend.
	b := audit.NewBackend(client)

	clusterInfo := bapi.ClusterInfo{
		Cluster: "testcluster",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_audit_kube.%s", clusterInfo.Cluster)).Do(ctx)

	// The DaemonSet that this audit log is for.
	ds := apps.DaemonSet{
		TypeMeta: metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
	}
	dsRaw, err := json.Marshal(ds)
	require.NoError(t, err)

	f := kaudit.Event{
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
	}

	// Create the event in ES.
	err = b.Create(ctx, clusterInfo, []kaudit.Event{f})
	require.NoError(t, err)

	// Refresh the index.
	err = testutils.RefreshIndex(ctx, client, fmt.Sprintf("tigera_secure_ee_audit_kube.%s", clusterInfo.Cluster))
	require.NoError(t, err)

	// List the event, assert that it matches the one we just wrote.
	results, err := b.List(ctx, clusterInfo, v1.AuditLogParams{})
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	require.Equal(t, f, results[0])

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_audit_kube.%s", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}
