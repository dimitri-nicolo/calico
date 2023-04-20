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
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
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
		err = backendutils.CleanupIndices(context.Background(), esClient, cluster)
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
			Stage:      kaudit.StageResponseComplete,
			Level:      kaudit.LevelRequestResponse,
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
			Stage:      kaudit.StageResponseComplete,
			Level:      kaudit.LevelRequestResponse,
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

func TestAuditLogFiltering(t *testing.T) {
	type testCase struct {
		Name   string
		Params v1.AuditLogParams

		// Configuration for which logs are expected to match.
		ExpectLog1 bool
		ExpectLog2 bool
		ExpectKube bool

		// Whether to perform an equality comparison on the returned
		// logs. Can be useful for tests where stats differ.
		SkipComparison bool

		// Whether or not to filter based on time range.
		AllTime bool
	}

	numExpected := func(tc testCase) int {
		num := 0
		if tc.ExpectLog1 {
			num++
		}
		if tc.ExpectLog2 {
			num++
		}
		if tc.ExpectKube {
			num++
		}
		return num
	}

	testcases := []testCase{
		{
			Name: "should query both logs",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
			},
			ExpectLog1: true,
			ExpectLog2: true,
		},
		{
			Name: "should filter based on type",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeKube,
			},
			ExpectLog1: false,
			ExpectLog2: false,
			ExpectKube: true,
		},
		{
			Name: "should filter based on kind",
			Params: v1.AuditLogParams{
				Kinds: []v1.Kind{v1.KindNetworkPolicy},
				Type:  v1.AuditLogTypeEE,
			},
			ExpectLog1: true,
			ExpectLog2: false,
		},
		{
			Name: "should filter based on name",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
				ObjectRefs: []v1.ObjectReference{
					{Name: "np-1"},
				},
			},
			ExpectLog1: true,
			ExpectLog2: false,
		},
		{
			Name: "should filter based on multiple names",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
				ObjectRefs: []v1.ObjectReference{
					{Name: "np-1"},
					{Name: "gnp-1"},
				},
			},
			ExpectLog1: true,
			ExpectLog2: true,
		},
		{
			Name: "should filter based on author",
			Params: v1.AuditLogParams{
				Authors: []string{"garfunkel"},
				Type:    v1.AuditLogTypeEE,
			},
			ExpectLog1: true,
			ExpectLog2: false,
		},
		{
			Name: "should filter based on namespace",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
				ObjectRefs: []v1.ObjectReference{
					{Namespace: "default"},
				},
			},
			ExpectLog1: true,
			ExpectLog2: false,
		},
		{
			Name: "should filter based on global namespaces",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
				ObjectRefs: []v1.ObjectReference{
					{Namespace: "-"},
				},
			},
			ExpectLog1: false,
			ExpectLog2: true,
		},
		{
			Name: "should filter based on multiple namespaces",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeAny,
				ObjectRefs: []v1.ObjectReference{
					{Namespace: "default"},
					{Namespace: "calico-system"},
				},
			},
			ExpectLog1: true,
			ExpectLog2: false,
			ExpectKube: true,
		},
		{
			Name: "should filter based on API group",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
				ObjectRefs: []v1.ObjectReference{
					{APIGroup: "projectcalico.org"},
				},
			},
			ExpectLog1: true,
			ExpectLog2: true,
		},
		{
			Name: "should filter based on API group and version",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeEE,
				ObjectRefs: []v1.ObjectReference{
					{
						APIGroup:   "projectcalico.org",
						APIVersion: "v4",
					},
				},
			},
			ExpectLog1: false,
			ExpectLog2: true,
		},
		{
			Name: "should filter based on response code",
			Params: v1.AuditLogParams{
				Type:          v1.AuditLogTypeEE,
				ResponseCodes: []int32{201},
			},
			ExpectLog1: false,
			ExpectLog2: true,
		},
		{
			Name: "should support returning both kube and EE audit logs at once",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeAny,
			},
			ExpectLog1: true,
			ExpectLog2: true,
			ExpectKube: true,
		},
		{
			Name: "should support queries that don't include a time range",
			Params: v1.AuditLogParams{
				Type: v1.AuditLogTypeAny,
			},
			AllTime:    true,
			ExpectLog1: true,
			ExpectLog2: true,
			ExpectKube: true,
		},
		{
			Name: "should support matching on Level",
			Params: v1.AuditLogParams{
				Type:   v1.AuditLogTypeEE,
				Levels: []kaudit.Level{kaudit.LevelRequestResponse},
			},
			AllTime:    true,
			ExpectLog1: true,
			ExpectLog2: false,
		},
		{
			Name: "should support matching on Stage",
			Params: v1.AuditLogParams{
				Type:   v1.AuditLogTypeEE,
				Stages: []kaudit.Stage{kaudit.StageResponseComplete},
			},
			AllTime:    true,
			ExpectLog1: true,
			ExpectLog2: false,
		},
	}

	// Run each testcase both as a multi-tenant scenario, as well as a single-tenant case.
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		for _, testcase := range testcases {
			// Each testcase creates multiple audit logs, and then uses
			// different filtering parameters provided in the params
			// to query one or more audit logs.
			name := fmt.Sprintf("%s (tenant=%s)", testcase.Name, tenant)
			t.Run(name, func(t *testing.T) {
				defer setupTest(t)()

				clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}

				// Time that the logs occur.
				logTime := time.Unix(1, 0)

				// Set the time range for the test. We set this per-test
				// so that the time range captures the windows that the logs
				// are created in.
				tr := &lmav1.TimeRange{}
				tr.From = logTime.Add(-1 * time.Millisecond)
				tr.To = logTime.Add(1 * time.Millisecond)
				testcase.Params.QueryParams.TimeRange = tr

				// The object that audit log is for.
				obj := v3.NetworkPolicy{
					TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
				}
				objRaw, err := json.Marshal(obj)
				require.NoError(t, err)

				// Create log #1.
				a1 := v1.AuditLog{
					Event: kaudit.Event{
						TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "audit.k8s.io/v1"},
						AuditID:    types.UID("audit-log-one"),
						Stage:      kaudit.StageResponseComplete,
						Level:      kaudit.LevelRequestResponse,
						RequestURI: "/apis/v3/projectcalico.org",
						Verb:       "PUT",
						User: authnv1.UserInfo{
							Username: "garfunkel",
							UID:      "1234",
							Extra:    map[string]authnv1.ExtraValue{"extra": authnv1.ExtraValue([]string{"value"})},
						},
						ImpersonatedUser: &authnv1.UserInfo{
							Username: "impuser",
							UID:      "impuid",
							Groups:   []string{"g1"},
						},
						SourceIPs: []string{"1.2.3.4"},
						UserAgent: "user-agent",
						ObjectRef: &kaudit.ObjectReference{
							Resource:   "networkpolicies",
							Name:       "np-1",
							Namespace:  "default",
							APIGroup:   "projectcalico.org",
							APIVersion: "v3",
						},
						ResponseStatus: &metav1.Status{
							Code: 200,
						},
						RequestObject: &runtime.Unknown{
							Raw:         objRaw,
							ContentType: runtime.ContentTypeJSON,
						},
						ResponseObject: &runtime.Unknown{
							Raw:         objRaw,
							ContentType: runtime.ContentTypeJSON,
						},
						RequestReceivedTimestamp: metav1.NewMicroTime(logTime),
						StageTimestamp:           metav1.NewMicroTime(logTime),
						Annotations:              map[string]string{"brick": "red"},
					},
					Name: testutils.StringPtr("ee-any"),
				}

				// The object that audit log is for.
				obj = v3.NetworkPolicy{
					TypeMeta: metav1.TypeMeta{Kind: "GlobalNetworkPolicy", APIVersion: "projectcalico.org/v4"},
				}
				objRaw2, err := json.Marshal(obj)
				require.NoError(t, err)

				// Create log #2.
				a2 := v1.AuditLog{
					Event: kaudit.Event{
						TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "audit.k8s.io/v1"},
						AuditID:    types.UID("audit-log-two"),
						Stage:      kaudit.StageRequestReceived,
						Level:      kaudit.LevelRequest,
						RequestURI: "/apis/v3/projectcalico.org",
						Verb:       "PUT",
						User: authnv1.UserInfo{
							Username: "oates",
							UID:      "0987",
							Extra:    map[string]authnv1.ExtraValue{"extra": authnv1.ExtraValue([]string{"value"})},
						},
						ImpersonatedUser: &authnv1.UserInfo{
							Username: "impuser",
							UID:      "impuid",
							Groups:   []string{"g1"},
						},
						SourceIPs: []string{"1.2.3.4"},
						UserAgent: "user-agent",
						ObjectRef: &kaudit.ObjectReference{
							Resource:   "globalnetworkpolicies",
							Name:       "gnp-1",
							Namespace:  "",
							APIGroup:   "projectcalico.org",
							APIVersion: "v4",
						},
						ResponseStatus: &metav1.Status{
							Code: 201,
						},
						RequestObject: &runtime.Unknown{
							Raw:         objRaw2,
							ContentType: runtime.ContentTypeJSON,
						},
						ResponseObject: &runtime.Unknown{
							Raw:         objRaw2,
							ContentType: runtime.ContentTypeJSON,
						},
						RequestReceivedTimestamp: metav1.NewMicroTime(logTime),
						StageTimestamp:           metav1.NewMicroTime(logTime),
						Annotations:              map[string]string{"brick": "red"},
					},
					Name: testutils.StringPtr("ee-any"),
				}

				response, err := b.Create(ctx, v1.AuditLogTypeEE, clusterInfo, []v1.AuditLog{a1, a2})
				require.NoError(t, err)
				require.Equal(t, []v1.BulkError(nil), response.Errors)
				require.Equal(t, 0, response.Failed)

				// Also create a Kube audit log.
				ds := apps.DaemonSet{
					TypeMeta: metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
				}
				dsRaw, err := json.Marshal(ds)
				require.NoError(t, err)

				a3 := v1.AuditLog{
					Event: kaudit.Event{
						TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "audit.k8s.io/v1"},
						AuditID:    types.UID("some-uuid-most-likely"),
						Stage:      kaudit.StageResponseComplete,
						Level:      kaudit.LevelRequestResponse,
						RequestURI: "/apis/v1/namespaces",
						Verb:       "GET",
						User: authnv1.UserInfo{
							Username: "prince",
							UID:      "uid",
							Extra:    map[string]authnv1.ExtraValue{"extra": authnv1.ExtraValue([]string{"value"})},
						},
						ImpersonatedUser: &authnv1.UserInfo{
							Username: "impuser",
							UID:      "impuid",
							Groups:   []string{"g1"},
						},
						SourceIPs: []string{"1.2.3.4"},
						UserAgent: "user-agent",
						ObjectRef: &kaudit.ObjectReference{
							Resource:   "daemonsets",
							Name:       "calico-node",
							Namespace:  "calico-system",
							APIGroup:   "apps",
							APIVersion: "v1",
						},
						ResponseStatus: &metav1.Status{},
						RequestObject: &runtime.Unknown{
							Raw:         dsRaw,
							ContentType: runtime.ContentTypeJSON,
						},
						ResponseObject: &runtime.Unknown{
							Raw:         dsRaw,
							ContentType: runtime.ContentTypeJSON,
						},
						RequestReceivedTimestamp: metav1.NewMicroTime(logTime),
						StageTimestamp:           metav1.NewMicroTime(logTime),
						Annotations:              map[string]string{"brick": "red"},
					},
					Name: testutils.StringPtr("any"),
				}

				resp, err := b.Create(ctx, v1.AuditLogTypeKube, clusterInfo, []v1.AuditLog{a3})
				require.NoError(t, err)
				require.Equal(t, 0, len(resp.Errors))

				err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_audit_*")
				require.NoError(t, err)

				// Query for audit logs.
				r, err := b.List(ctx, clusterInfo, &testcase.Params)
				require.NoError(t, err)
				require.Len(t, r.Items, numExpected(testcase))
				require.Nil(t, r.AfterKey)
				require.Empty(t, err)

				// Querying with another tenant ID should result in zero results.
				r2, err := b.List(ctx, bapi.ClusterInfo{Cluster: cluster, Tenant: "bad-actor"}, &testcase.Params)
				require.NoError(t, err)
				require.Len(t, r2.Items, 0)

				if testcase.SkipComparison {
					return
				}

				// Assert that the correct logs are returned.
				if testcase.ExpectLog1 {
					require.Contains(t, r.Items, a1)
				}
				if testcase.ExpectLog2 {
					require.Contains(t, r.Items, a2)
				}
				if testcase.ExpectKube {
					require.Contains(t, r.Items, a3)
				}
			})
		}
	}
}
