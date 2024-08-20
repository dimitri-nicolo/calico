// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"os"
	"time"
	"unsafe"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
)

var _ = Describe("Linseed out plugin token tests", func() {
	var (
		f                 *os.File
		pluginConfigKeyFn PluginConfigKeyFunc
		serviceAccount    *corev1.ServiceAccount
	)

	BeforeEach(func() {
		var err error
		f, err = os.CreateTemp("", "kubeconfig")
		Expect(err).NotTo(HaveOccurred())

		pluginConfigKeyFn = func(plugin unsafe.Pointer, key string) string {
			if key == "tls.verify" {
				return "true"
			}
			return ""
		}

		serviceAccount = &corev1.ServiceAccount{
			TypeMeta: resources.TypeK8sServiceAccounts,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "noncluster-serviceaccount",
				Namespace: "default",
			},
		}
	})

	Context("Token tests", func() {
		It("should fetch token when the current one is expired", func() {
			_, err := f.WriteString(validKubeconfig)
			f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("KUBECONFIG", f.Name())
			Expect(err).NotTo(HaveOccurred())
			err = os.Setenv("ENDPOINT", "https://1.2.3.4:5678")
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(nil, pluginConfigKeyFn)
			Expect(err).NotTo(HaveOccurred())

			mockClientSet := lmak8s.NewMockClientSet(GinkgoT())
			fakeCoreV1 := fake.NewSimpleClientset().CoreV1()
			_, err = fakeCoreV1.ServiceAccounts("default").Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			mockClientSet.On("CoreV1").Return(fakeCoreV1)
			cfg.clientset = mockClientSet

			cfg.serviceAccountName = serviceAccount.GetName()
			cfg.expiration = time.Now().Add(-2 * tokenExpiration) // must be expired
			cfg.token = "some-token"

			_, err = GetToken(cfg)
			Expect(err).NotTo(HaveOccurred())
			// createToken from corev1 must be called
			Expect(mockClientSet.AssertCalled(GinkgoT(), "CoreV1")).To(BeTrue())
		})

		It("should reuse the token when it is still valid", func() {
			_, err := f.WriteString(validKubeconfig)
			f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("KUBECONFIG", f.Name())
			Expect(err).NotTo(HaveOccurred())
			err = os.Setenv("ENDPOINT", "https://1.2.3.4:5678")
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(nil, pluginConfigKeyFn)
			Expect(err).NotTo(HaveOccurred())

			mockClientSet := lmak8s.NewMockClientSet(GinkgoT())
			fakeCoreV1 := fake.NewSimpleClientset().CoreV1()
			_, err = fakeCoreV1.ServiceAccounts("default").Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			mockClientSet.On("CoreV1").Return(fakeCoreV1)
			cfg.clientset = mockClientSet

			cfg.serviceAccountName = serviceAccount.GetName()
			cfg.expiration = time.Now().Add(1 * time.Hour) // must not be expired
			cfg.token = "some-token"

			token, err := GetToken(cfg)
			Expect(err).NotTo(HaveOccurred())
			// should not call createToken
			Expect(mockClientSet.AssertNotCalled(GinkgoT(), "CoreV1")).To(BeTrue())
			Expect(token).To(Equal("some-token"))
		})

		It("should return error when missing serviceaccount", func() {
			_, err := f.WriteString(validKubeconfig)
			f.Close()
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("KUBECONFIG", f.Name())
			Expect(err).NotTo(HaveOccurred())
			err = os.Setenv("ENDPOINT", "https://1.2.3.4:5678")
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(nil, pluginConfigKeyFn)
			Expect(err).NotTo(HaveOccurred())

			mockClientSet := lmak8s.NewMockClientSet(GinkgoT())
			fakeCoreV1 := fake.NewSimpleClientset().CoreV1()
			_, err = fakeCoreV1.ServiceAccounts("default").Create(context.Background(), serviceAccount, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			mockClientSet.On("CoreV1").Return(fakeCoreV1)
			cfg.clientset = mockClientSet

			cfg.serviceAccountName = "invalid-service-account"
			cfg.expiration = time.Now().Add(-2 * tokenExpiration) // must be expired
			cfg.token = "some-token"

			_, err = GetToken(cfg)
			Expect(err).To(HaveOccurred())
		})
	})
})
