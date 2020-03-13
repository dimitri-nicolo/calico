// Copyright (c) 2020 Tigera Inc. All rights reserved.

package rbac

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/tigera/intrusion-detection/controller/pkg/feeds/puller"
)

func TestRestrictedSecretsClient(t *testing.T) {
	tcs := []struct {
		name     string
		expected bool
	}{
		{"default-token-6p2tr", false},
		{"intrusion-detection-controller-token-dcqvv", false},
		{"tigera-ee-installer-elasticsearch-access", false},
		{"tigera-ee-intrusion-detection-elasticsearch-access", false},
		{"tigera-pull-secret", false},
		{"tigera-secure-es-http-certs-public", false},
		{"tigera-secure-kb-http-certs-public", false},
		{"alertmanager-calico-node-alertmanager", false},
		{"calico-monitoring-tigera-kibana-kibana-user", false},
		{"calico-prometheus-operator-token-rxd2t", false},
		{"cnx-manager-tls", false},
		{"cnx-manager-token-g8fjg", false},
		{"cnx-pull-secret", false},
		{"default-token-2prsq", false},
		{"elastic-compliance-user", false},
		{"elastic-curator-user", false},
		{"elastic-ee-intrusion-detection", false},
		{"elastic-fluentd-user", false},
		{"elastic-operator-token-gk2dv", false},
		{"intrusion-detection-controller-token-h74cd", false},
		{"prometheus-calico-node-prometheus", false},
		{"prometheus-calico-node-prometheus-tls-assets", false},
		{"prometheus-token-m9kdl", false},
		{"tigera-compliance-benchmarker-token-wm9rx", false},
		{"tigera-compliance-controller-token-qgh4n", false},
		{"tigera-compliance-reporter-token-vflnz", false},
		{"tigera-compliance-server-token-cxjnk", false},
		{"tigera-compliance-snapshotter-token-28shc", false},
		{"tigera-elasticsearch-es-dg4vrgmqtp-certs", false},
		{"tigera-elasticsearch-es-dg4vrgmqtp-config", false},
		{"tigera-elasticsearch-es-elastic-user", false},
		{"tigera-elasticsearch-es-http-ca-internal", false},
		{"tigera-elasticsearch-es-http-certs-internal", false},
		{"tigera-elasticsearch-es-http-certs-public", false},
		{"tigera-elasticsearch-es-internal-users", false},
		{"tigera-elasticsearch-es-transport-ca-internal", false},
		{"tigera-elasticsearch-es-transport-certs-public", false},
		{"tigera-elasticsearch-es-xpack-file-realm", false},
		{"tigera-es-config", false},
		{"tigera-fluentd-token-phd7w", false},
		{"tigera-kibana-kb-config", false},
		{"tigera-kibana-kb-es-ca", false},
		{"tigera-kibana-kb-http-ca-internal", false},
		{"tigera-kibana-kb-http-certs-internal", false},
		{"tigera-kibana-kb-http-certs-public", false},
		{"tigera-kibana-kibana-user", false},
		{"webhook-server-secret", false},
		{"ok", true},
	}

	r := RestrictedSecretsClient{&puller.MockSecrets{}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(r.isPermitted(tc.name)).Should(Equal(tc.expected))
		})
	}

	g := NewWithT(t)
	_, err := r.Get("tigera-pull-secret", metav1.GetOptions{})
	g.Expect(err).Should(HaveOccurred())

	_, err = r.Get("ok", metav1.GetOptions{})
	g.Expect(err).ShouldNot(HaveOccurred())

	_, err = r.List(metav1.ListOptions{})
	g.Expect(err).Should(HaveOccurred())

	_, err = r.Watch(metav1.ListOptions{})
	g.Expect(err).Should(HaveOccurred())

	_, err = r.Create(nil)
	g.Expect(err).Should(HaveOccurred())

	_, err = r.Update(nil)
	g.Expect(err).Should(HaveOccurred())

	err = r.Delete("foo", nil)
	g.Expect(err).Should(HaveOccurred())

	err = r.DeleteCollection(nil, metav1.ListOptions{})
	g.Expect(err).Should(HaveOccurred())

	_, err = r.Patch("foo", types.JSONPatchType, nil)
	g.Expect(err).Should(HaveOccurred())
}
