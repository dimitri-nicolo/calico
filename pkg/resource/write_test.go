// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package resource_test

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("ConfigMap", func() {
	It("Creates the config map when it doesn't exist", func() {
		cli := fake.NewSimpleClientset()
		Expect(resource.WriteConfigMapToK8s(cli, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestName",
				Namespace: "TestNamespace",
			},
		})).ShouldNot(HaveOccurred())

		_, err := cli.CoreV1().ConfigMaps("TestNamespace").Get(context.Background(), "TestName", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Updates the config map when it exists", func() {
		cli := fake.NewSimpleClientset()
		Expect(resource.WriteConfigMapToK8s(cli, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestName",
				Namespace: "TestNamespace",
			},
			Data: map[string]string{
				"key": "value",
			},
		})).ShouldNot(HaveOccurred())

		cm, err := cli.CoreV1().ConfigMaps("TestNamespace").Get(context.Background(), "TestName", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cm.Data["key"]).Should(Equal("value"))

		Expect(resource.WriteConfigMapToK8s(cli, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestName",
				Namespace: "TestNamespace",
			},
			Data: map[string]string{
				"key": "newvalue",
			},
		})).ShouldNot(HaveOccurred())

		cm, err = cli.CoreV1().ConfigMaps("TestNamespace").Get(context.Background(), "TestName", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cm.Data["key"]).Should(Equal("newvalue"))
	})
})

var _ = Describe("Secret", func() {
	It("Creates the Secret when it doesn't exist", func() {
		cli := fake.NewSimpleClientset()
		Expect(resource.WriteSecretToK8s(cli, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestName",
				Namespace: "TestNamespace",
			},
		})).ShouldNot(HaveOccurred())

		_, err := cli.CoreV1().Secrets("TestNamespace").Get(context.Background(), "TestName", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Updates the Secret when it exists", func() {
		cli := fake.NewSimpleClientset()
		Expect(resource.WriteSecretToK8s(cli, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestName",
				Namespace: "TestNamespace",
			},
			Data: map[string][]byte{
				"key": []byte("value"),
			},
		})).ShouldNot(HaveOccurred())

		s, err := cli.CoreV1().Secrets("TestNamespace").Get(context.Background(), "TestName", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(s.Data["key"]).Should(Equal([]byte("value")))

		Expect(resource.WriteSecretToK8s(cli, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestName",
				Namespace: "TestNamespace",
			},
			Data: map[string][]byte{
				"key": []byte("newvalue"),
			},
		})).ShouldNot(HaveOccurred())

		s, err = cli.CoreV1().Secrets("TestNamespace").Get(context.Background(), "TestName", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(s.Data["key"]).Should(Equal([]byte("newvalue")))
	})
})
