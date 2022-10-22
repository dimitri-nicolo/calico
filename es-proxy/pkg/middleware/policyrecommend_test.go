// Copyright (c) 2019, 2022 Tigera, Inc. All rights reserved.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmaerror "github.com/projectcalico/calico/lma/pkg/api"
	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/policyrec"
	lmapolicyrec "github.com/projectcalico/calico/lma/pkg/policyrec"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
)

const recommendURLPath = "/recommend"

// Given a source reported flow from deployment app1 to endpoint nginx on port 80,
// the engine should return a policy selecting app1 to nginx, to port 80.
var (
	destPort       = uint16(80)
	destPortInRule = numorstring.SinglePort(destPort)

	protoInRule = numorstring.ProtocolFromString("TCP")

	app1Dep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app1",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app1",
			},
		},
	}
	app1Rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app1-abcdef",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app1",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "app1",
				},
			},
		},
	}

	app2Dep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app2",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app2",
			},
		},
	}
	app2Rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app2-abcdef",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app2",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "app2",
				},
			},
		},
	}

	app3Dep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app3",
			Namespace: "namespace2",
			Labels: map[string]string{
				"app": "app3",
			},
		},
	}
	app3Rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app3-abcdef",
			Namespace: "namespace2",
			Labels: map[string]string{
				"app": "app3",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "app3",
				},
			},
		},
	}

	nginxDep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	}
	nginxRs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-12345",
			Namespace: "namespace1",
			Labels: map[string]string{
				"app": "app1",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "nginx",
				},
			},
		},
	}

	nginx2Dep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx2",
			Namespace: "namespace2",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	}
	nginx2Rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx2-12345",
			Namespace: "namespace2",
			Labels: map[string]string{
				"app": "app3",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "nginx2",
				},
			},
		},
	}

	nginx3Dep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx3",
			Namespace: "namespace2",
			Labels: map[string]string{
				"app": "nginx3",
			},
		},
	}
	nginx3Rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx3-12345",
			Namespace: "namespace2",
			Labels: map[string]string{
				"app": "app3",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "nginx3",
				},
			},
		},
	}
	namespace1Namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace1",
		},
	}
	namespace2Namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace2",
		},
	}

	app1Query = &policyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "app1-abcdef-*",
		Namespace:    "namespace1",
	}

	nginxQuery = &policyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "nginx-12345-*",
		Namespace:    "namespace1",
	}

	namespace1Query = &policyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "",
		Namespace:    "namespace1",
	}

	namespace2Query = &policyrec.PolicyRecommendationParams{
		StartTime:    "now-1h",
		EndTime:      "now",
		EndpointName: "",
		Namespace:    "namespace2",
	}

	app1Policy = &v3.StagedNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindStagedNetworkPolicy,
			APIVersion: v3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default.app1",
			Namespace: "namespace1",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "default",
			Types:        []v3.PolicyType{v3.PolicyTypeEgress},
			Selector:     "app == 'app1'",
			Egress: []v3.Rule{
				v3.Rule{
					Action:   v3.Allow,
					Protocol: &protoInRule,
					Destination: v3.EntityRule{
						Selector: "app == 'nginx'",
						Ports:    []numorstring.Port{destPortInRule},
					},
				},
			},
		},
	}

	nginxPolicy = &v3.StagedNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindStagedNetworkPolicy,
			APIVersion: v3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default.nginx",
			Namespace: "namespace1",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "default",
			Types:        []v3.PolicyType{v3.PolicyTypeIngress},
			Selector:     "app == 'nginx'",
			Ingress: []v3.Rule{
				v3.Rule{
					Action:   v3.Allow,
					Protocol: &protoInRule,
					Source: v3.EntityRule{
						Selector: "app == 'app1'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{destPortInRule},
					},
				},
			},
		},
	}

	protoIPIP = numorstring.ProtocolFromString("ipip")
	protoTCP  = numorstring.ProtocolFromString("TCP")
	protoUDP  = numorstring.ProtocolFromString("UDP")

	namespace1Policy = &v3.StagedNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindStagedNetworkPolicy,
			APIVersion: v3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default.namespace1-policy",
			Namespace: "namespace1",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "default",
			Types:        []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Selector:     "projectcalico.org/name == 'namespace1'",
			Egress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protoTCP,
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace1'",
						Ports:             []numorstring.Port{numorstring.SinglePort(8080)},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protoTCP,
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						Ports:             []numorstring.Port{numorstring.SinglePort(80), numorstring.SinglePort(8091)},
					},
				},
			},
			Ingress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protoIPIP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{numorstring.SinglePort(50)},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protoUDP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{numorstring.SinglePort(40)},
					},
				},
			},
		},
	}

	namespace2Policy = &v3.StagedNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       v3.KindStagedNetworkPolicy,
			APIVersion: v3.GroupVersionCurrent,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default.namespace2-policy",
			Namespace: "namespace2",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "default",
			Types:        []v3.PolicyType{v3.PolicyTypeEgress},
			Selector:     "projectcalico.org/name == 'namespace2'",
			Egress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protoUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace1'",

						Ports: []numorstring.Port{numorstring.SinglePort(40)},
					},
				},
			},
			Ingress: nil,
		},
	}

	app1ToNginxFlows = []*elastic.CompositeAggregationBucket{
		&elastic.CompositeAggregationBucket{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				elastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				elastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				elastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				elastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				elastic.CompositeAggregationSourceValue{Name: "reporter", Value: "src"},
				elastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		&elastic.CompositeAggregationBucket{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				elastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				elastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				elastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				elastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				elastic.CompositeAggregationSourceValue{Name: "reporter", Value: "dst"},
				elastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
	}

	app1ToNginxEgressFlows = []*elastic.CompositeAggregationBucket{
		&elastic.CompositeAggregationBucket{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				elastic.CompositeAggregationSourceValue{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "source_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				elastic.CompositeAggregationSourceValue{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				elastic.CompositeAggregationSourceValue{Name: "dest_namespace", Value: "namespace1"},
				elastic.CompositeAggregationSourceValue{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				elastic.CompositeAggregationSourceValue{Name: "proto", Value: "6"},
				elastic.CompositeAggregationSourceValue{Name: "dest_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_ip", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "source_port", Value: ""},
				elastic.CompositeAggregationSourceValue{Name: "dest_port", Value: 80.0},
				elastic.CompositeAggregationSourceValue{Name: "reporter", Value: "src"},
				elastic.CompositeAggregationSourceValue{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": &elastic.AggregatedTerm{
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
	}

	flowsForNamespaceTest = []*elastic.CompositeAggregationBucket{
		// flow-1
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace1"},
				{Name: "source_name_aggr", Value: "app1-abcdef-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace2"},
				{Name: "dest_name_aggr", Value: "nginx2-12345-*"},
				{Name: "proto", Value: "6"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 80.0},
				{Name: "reporter", Value: "src"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app1": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx2": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		// flow-2
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace1"},
				{Name: "source_name_aggr", Value: "app2-abcdef-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace1"},
				{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				{Name: "proto", Value: "6"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 8080.0},
				{Name: "reporter", Value: "src"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app2": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		// flow-3
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace1"},
				{Name: "source_name_aggr", Value: "app2-abcdef-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace1"},
				{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				{Name: "proto", Value: "6"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 8080.0},
				{Name: "reporter", Value: "dst"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app2": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		// flow-4
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace2"},
				{Name: "source_name_aggr", Value: "nginx3-12345-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace1"},
				{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				{Name: "proto", Value: "4"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 50.0},
				{Name: "reporter", Value: "dst"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx3": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		// flow-5
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace2"},
				{Name: "source_name_aggr", Value: "app3-abcdef-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace1"},
				{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				{Name: "proto", Value: "17"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 40.0},
				{Name: "reporter", Value: "src"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app3": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		// flow-6
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace2"},
				{Name: "source_name_aggr", Value: "app3-abcdef-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace1"},
				{Name: "dest_name_aggr", Value: "nginx-12345-*"},
				{Name: "proto", Value: "17"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 40.0},
				{Name: "reporter", Value: "dst"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app3": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
		// flow-7
		{
			CompositeAggregationKey: []elastic.CompositeAggregationSourceValue{
				{Name: "source_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "source_namespace", Value: "namespace1"},
				{Name: "source_name_aggr", Value: "nginx-12345-*"},
				{Name: "dest_type", Value: api.FlowLogEndpointTypeWEP},
				{Name: "dest_namespace", Value: "namespace2"},
				{Name: "dest_name_aggr", Value: "nginx2-12345-*"},
				{Name: "proto", Value: "6"},
				{Name: "dest_ip", Value: ""},
				{Name: "source_ip", Value: ""},
				{Name: "source_port", Value: ""},
				{Name: "dest_port", Value: 8091.0},
				{Name: "reporter", Value: "src"},
				{Name: "action", Value: "allow"},
			},
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"source_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=app3": 1,
					},
				},
				"dest_labels": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"app=nginx": 1,
					},
				},
				"policies": {
					DocCount: 1,
					Buckets: map[interface{}]int64{
						"0|__PROFILE__|__PROFILE__.kns.namespace1|allow|0": 1,
					},
				},
			},
		},
	}
)

var _ = Describe("Policy Recommendation", func() {
	var (
		fakeKube           k8s.Interface
		ec                 *fakeAggregator
		mockRBACAuthorizer *lmaauth.MockRBACAuthorizer
	)
	BeforeEach(func() {
		fakeKube = fake.NewSimpleClientset(namespace1Namespace, namespace2Namespace, app1Dep, app1Rs, app2Dep, app2Rs, app3Dep, app3Rs, nginxDep,
			nginxRs, nginx2Dep, nginx2Rs, nginx3Dep, nginx3Rs)
		ec = newFakeAggregator()
		mockRBACAuthorizer = new(lmaauth.MockRBACAuthorizer)
	})
	DescribeTable("Recommend policies for matching flows and endpoint",
		func(queryResults []*elastic.CompositeAggregationBucket, queryError error,
			query *policyrec.PolicyRecommendationParams,
			expectedResponse *PolicyRecommendationResponse, statusCode int) {

			mockClientSet := datastore.NewClientSet(fakeKube, nil)

			mockK8sClientFactory := new(datastore.MockClusterCtxK8sClientFactory)
			mockK8sClientFactory.On("RBACAuthorizerForCluster", mock.Anything).Return(mockRBACAuthorizer, nil)
			mockK8sClientFactory.On("ClientSetForCluster", mock.Anything).Return(mockClientSet, nil)

			By("Initializing the engine") // Tempted to say "Start your engines!"
			hdlr := PolicyRecommendationHandler(mockK8sClientFactory, mockClientSet, ec)

			jsonQuery, err := json.Marshal(query)
			Expect(err).To(BeNil())

			req, err := http.NewRequest(http.MethodPost, recommendURLPath, bytes.NewBuffer(jsonQuery))
			Expect(err).To(BeNil())

			mockRBACAuthorizer.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

			// add a bogus user
			req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))

			By("setting up next results")
			ec.setNextResults(queryResults)
			ec.setNextError(queryError)
			// Always allow

			w := httptest.NewRecorder()
			hdlr.ServeHTTP(w, req)
			Expect(err).To(BeNil())

			if statusCode != http.StatusOK {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				recResponse, err := ioutil.ReadAll(w.Body)
				Expect(err).NotTo(HaveOccurred())
				errorBody := &lmaerror.Error{}
				err = json.Unmarshal(recResponse, errorBody)
				Expect(err).To(BeNil())
				Expect(errorBody.Code).To(Equal(statusCode))
				Expect(errorBody.Feature).To(Equal(lmaerror.PolicyRec))
				return
			}

			recResponse, err := ioutil.ReadAll(w.Body)
			Expect(err).NotTo(HaveOccurred())

			actualRec := &PolicyRecommendationResponse{}
			err = json.Unmarshal(recResponse, actualRec)
			Expect(err).To(BeNil())

			if expectedResponse == nil {
				Expect(actualRec).To(BeNil())
			} else {
				Expect(actualRec).ToNot(BeNil())
				Expect(actualRec).To(Equal(expectedResponse))
			}
		},
		Entry("for source endpoint", app1ToNginxFlows, nil,
			app1Query,
			&PolicyRecommendationResponse{
				Recommendation: &policyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						app1Policy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
			}, http.StatusOK),
		Entry("for destination endpoint", app1ToNginxFlows, nil,
			nginxQuery,
			&PolicyRecommendationResponse{
				Recommendation: &policyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						nginxPolicy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
			}, http.StatusOK),
		Entry("for destination endpoint with egress only flows - no rules will be computed", app1ToNginxEgressFlows, nil,
			nginxQuery, nil, http.StatusInternalServerError),
		Entry("for unknown endpoint", []*elastic.CompositeAggregationBucket{}, nil,
			&policyrec.PolicyRecommendationParams{
				StartTime:    "now-1h",
				EndTime:      "now",
				EndpointName: "idontexist-*",
				Namespace:    "default",
			}, nil, http.StatusNotFound),
		Entry("for query that errors out - invalid time parameters", nil, fmt.Errorf("Elasticsearch error"),
			&policyrec.PolicyRecommendationParams{
				StartTime:    "now",
				EndTime:      "now-1h",
				EndpointName: "someendpoint-*",
				Namespace:    "default",
			}, nil, http.StatusInternalServerError),
	)

	DescribeTable("Namespace policy - recommend policies for matching flows and namespace",
		func(queryResults []*elastic.CompositeAggregationBucket, queryError error,
			query *policyrec.PolicyRecommendationParams,
			expectedResponse *PolicyRecommendationResponse, statusCode int) {

			mockClientSet := datastore.NewClientSet(fakeKube, nil)

			mockK8sClientFactory := new(datastore.MockClusterCtxK8sClientFactory)
			mockK8sClientFactory.On("RBACAuthorizerForCluster", mock.Anything).Return(mockRBACAuthorizer, nil)
			mockK8sClientFactory.On("ClientSetForCluster", mock.Anything).Return(mockClientSet, nil)

			By("Initializing the engine") // Tempted to say "Start your engines!"
			hdlr := PolicyRecommendationHandler(mockK8sClientFactory, mockClientSet, ec)

			jsonQuery, err := json.Marshal(query)
			Expect(err).To(BeNil())

			req, err := http.NewRequest(http.MethodPost, recommendURLPath, bytes.NewBuffer(jsonQuery))
			Expect(err).To(BeNil())

			mockRBACAuthorizer.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

			// add a bogus user
			req = req.WithContext(request.WithUser(req.Context(), &user.DefaultInfo{}))

			By("setting up next results")
			ec.setNextResults(queryResults)
			ec.setNextError(queryError)
			// Always allow

			w := httptest.NewRecorder()
			hdlr.ServeHTTP(w, req)
			Expect(err).To(BeNil())

			if statusCode != http.StatusOK {
				Expect(w.Code).To(Equal(http.StatusNotFound))
				recResponse, err := ioutil.ReadAll(w.Body)
				Expect(err).NotTo(HaveOccurred())
				errorBody := &lmaerror.Error{}
				err = json.Unmarshal(recResponse, errorBody)
				Expect(err).To(BeNil())
				Expect(errorBody.Code).To(Equal(statusCode))
				Expect(errorBody.Feature).To(Equal(lmaerror.PolicyRec))
				return
			}

			recResponse, err := ioutil.ReadAll(w.Body)
			Expect(err).NotTo(HaveOccurred())

			actualRec := &PolicyRecommendationResponse{}
			err = json.Unmarshal(recResponse, actualRec)
			Expect(err).To(BeNil())

			if expectedResponse == nil {
				Expect(actualRec).To(BeNil())
			} else {
				Expect(actualRec).ToNot(BeNil())
				for i, gnp := range actualRec.GlobalNetworkPolicies {
					Expect(gnp).To(lmapolicyrec.MatchPolicy(expectedResponse.GlobalNetworkPolicies[i]))
				}
				for i, np := range actualRec.NetworkPolicies {
					Expect(np).To(lmapolicyrec.MatchPolicy(expectedResponse.NetworkPolicies[i]))
				}
			}
		},
		Entry("policy for namespace1", flowsForNamespaceTest, nil, namespace1Query,
			&PolicyRecommendationResponse{
				Recommendation: &policyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						namespace1Policy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
			}, http.StatusOK),
		Entry("policy for namespace2", flowsForNamespaceTest, nil, namespace2Query,
			&PolicyRecommendationResponse{
				Recommendation: &policyrec.Recommendation{
					NetworkPolicies: []*v3.StagedNetworkPolicy{
						namespace2Policy,
					},
					GlobalNetworkPolicies: []*v3.StagedGlobalNetworkPolicy{},
				},
			}, http.StatusOK),
	)
})

// fakeAggregator is a test utility that implements the CompositeAggregator interface
type fakeAggregator struct {
	nextResults []*elastic.CompositeAggregationBucket
	nextError   error
}

func newFakeAggregator() *fakeAggregator {
	return &fakeAggregator{}
}

func (fa *fakeAggregator) SearchCompositeAggregations(
	context.Context, *elastic.CompositeAggregationQuery, elastic.CompositeAggregationKey,
) (<-chan *elastic.CompositeAggregationBucket, <-chan error) {
	dataChan := make(chan *elastic.CompositeAggregationBucket, len(fa.nextResults))
	errorChan := make(chan error, 1)
	go func() {
		defer func() {
			close(dataChan)
			close(errorChan)
		}()
		if fa.nextError != nil {
			errorChan <- fa.nextError
			return
		}
		for _, result := range fa.nextResults {
			dataChan <- result
		}
	}()
	return dataChan, errorChan
}

func (fa *fakeAggregator) setNextResults(cab []*elastic.CompositeAggregationBucket) {
	fa.nextResults = cab
}

func (fa *fakeAggregator) setNextError(err error) {
	fa.nextError = err
}

// // Test Utilities

// // MatchPolicy is a convenience function that returns a policyMatcher for matching
// // policies in a Gomega assertion.
// func MatchPolicy(expected interface{}) *policyMatcher {
// 	log.Debugf("Creating policy matcher")
// 	return &policyMatcher{expected: expected}
// }

// // policyMatcher implements the GomegaMatcher interface to match policies.
// type policyMatcher struct {
// 	expected interface{}
// }

// func (pm *policyMatcher) Match(actual interface{}) (success bool, err error) {
// 	// We expect to only handle pointer to TSEE NetworkPolicy for now.
// 	// TODO(doublek): Support for other policy resources should be added here.
// 	switch actualPolicy := actual.(type) {
// 	case *v3.StagedNetworkPolicy:
// 		expectedPolicy := pm.expected.(*v3.StagedNetworkPolicy)
// 		success = expectedPolicy.GroupVersionKind().Kind == actualPolicy.GroupVersionKind().Kind &&
// 			expectedPolicy.GroupVersionKind().Version == actualPolicy.GroupVersionKind().Version &&
// 			expectedPolicy.GetName() == actualPolicy.GetName() &&
// 			expectedPolicy.GetNamespace() == actualPolicy.GetNamespace() &&
// 			expectedPolicy.Spec.Tier == actualPolicy.Spec.Tier &&
// 			expectedPolicy.Spec.Order == actualPolicy.Spec.Order &&
// 			reflect.DeepEqual(expectedPolicy.Spec.Types, actualPolicy.Spec.Types) &&
// 			matchSelector(expectedPolicy.Spec.Selector, actualPolicy.Spec.Selector) &&
// 			matchRules(expectedPolicy.Spec.Ingress, actualPolicy.Spec.Ingress) &&
// 			matchRules(expectedPolicy.Spec.Egress, actualPolicy.Spec.Egress)
// 	default:
// 		// TODO(doublek): Remove this after testing the test.
// 		log.Debugf("Default case")

// 	}
// 	return
// }

// func matchSelector(actual, expected string) bool {
// 	// Currently only matches &&-ed selectors.
// 	// TODO(doublek): Add support for ||-ed selectors as well.
// 	actualSelectors := strings.Split(actual, " && ")
// 	expectedSelectors := strings.Split(expected, " && ")
// 	as := set.FromArray(actualSelectors)
// 	es := set.FromArray(expectedSelectors)
// 	es.Iter(func(item string) error {
// 		if as.Contains(item) {
// 			as.Discard(item)
// 			return set.RemoveItem
// 		}
// 		return nil
// 	})
// 	log.Debugf("\nActual %+v\nExpected %+v\n", actual, expected)
// 	if es.Len() != 0 || as.Len() != 0 {
// 		return false
// 	}
// 	return true
// }

// func matchRules(actual, expected []v3.Rule) bool {
// 	// TODO(doublek): Make sure there aren't any extra rules left over in either params.
// NEXTRULE:
// 	for _, actualRule := range actual {
// 		for i, expectedRule := range expected {
// 			if matchSingleRule(actualRule, expectedRule) {
// 				expected = append(expected[:i], expected[i+1:]...)
// 				continue NEXTRULE
// 			}
// 		}
// 		log.Debugf("\nDidn't find a match for rule\n\t%+v", actualRule)
// 		return false
// 	}
// 	if len(expected) != 0 {
// 		log.Debugf("\nDidn't find matching actual rules\n\t%+v for  expected rules\n\t%+v\n", actual, expected)
// 		return false
// 	}
// 	return true
// }

// func matchSingleRule(actual, expected v3.Rule) bool {
// 	return matchEntityRule(actual.Source, expected.Source) &&
// 		matchEntityRule(actual.Destination, expected.Destination) &&
// 		actual.Protocol.String() == expected.Protocol.String()
// }

// func matchEntityRule(actual, expected v3.EntityRule) bool {
// 	match := set.FromArray(actual.Nets).ContainsAll(set.FromArray(expected.Nets)) &&
// 		set.FromArray(actual.Ports).ContainsAll(set.FromArray(expected.Ports)) &&
// 		matchSelector(actual.Selector, expected.Selector) &&
// 		matchSelector(actual.NamespaceSelector, expected.NamespaceSelector) &&
// 		set.FromArray(actual.NotNets).ContainsAll(set.FromArray(expected.NotNets))
// 	if actual.ServiceAccounts != nil && expected.ServiceAccounts != nil {
// 		return match &&
// 			set.FromArray(actual.ServiceAccounts.Names).ContainsAll(set.FromArray(expected.ServiceAccounts.Names)) &&
// 			matchSelector(actual.ServiceAccounts.Selector, expected.ServiceAccounts.Selector)
// 	}
// 	return match
// }

// func (pm *policyMatcher) FailureMessage(actual interface{}) (message string) {
// 	message = fmt.Sprintf("Expected\n\t%#v\nto match\n\t%#v", actual, pm.expected)
// 	return
// }

// func (pm *policyMatcher) NegatedFailureMessage(actual interface{}) (message string) {
// 	message = fmt.Sprintf("Expected\n\t%#v\nnot to match\n\t%#v", actual, pm.expected)
// 	return
// }
