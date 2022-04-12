// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec_test

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/tigera/lma/pkg/policyrec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var (
	depPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abcdefg",
			Namespace: "test-dep-namespace",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "test-app",
				},
			},
		},
	}
	deployment = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-dep-namespace",
		},
	}
	jobPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abcdefg",
			Namespace: "test-job-namespace",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Job",
					Name: "test-app",
				},
			},
		},
	}
	job = &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind: "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-job-namespace",
		},
	}
	dsPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abcdefg",
			Namespace: "test-ds-namespace",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "DaemonSet",
					Name: "test-app",
				},
			},
		},
	}
	ds = &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-ds-namespace",
		},
	}
	rsPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abcdefg",
			Namespace: "test-rs-namespace",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "ReplicaSet",
					Name: "test-app-rs",
				},
			},
		},
	}
	rs = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-rs",
			Namespace: "test-rs-namespace",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "test-app",
				},
			},
		},
	}
	rsDep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-rs-namespace",
		},
	}
	orphanPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abcdefg",
			Namespace: "test-orphan-namespace",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind: "Deployment",
					Name: "test-app",
				},
			},
		},
	}
	alonePod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abcdefg",
			Namespace: "test-alone-namespace",
		},
	}
	wcDep = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-wc-namespace",
		},
	}
)

var _ = Describe("Test Generating Names for Recommended Policies", func() {
	// Define the kubernetes interface
	kube := fake.NewSimpleClientset(depPod, deployment, jobPod, job, dsPod, ds, rsPod, rs, rsDep, orphanPod, alonePod, wcDep)
	DescribeTable("Extracts the query parameters from the request and validates them",
		func(k k8s.Interface, searchName, searchNamespace, expectedName string) {
			params := &policyrec.PolicyRecommendationParams{
				EndpointName: searchName,
				Namespace:    searchNamespace,
			}

			genName := policyrec.GeneratePolicyName(k, params)
			Expect(genName).To(Equal(expectedName))
		},
		// pod -> deployment
		Entry("Given a pod name that has a reference to a deployment, it should return the deployment name", kube, "test-app-abcdefg", "test-dep-namespace", "test-app"),
		// pod -> job
		Entry("Given a pod name that has a reference to a job, it should return the job name", kube, "test-app-abcdefg", "test-job-namespace", "test-app"),
		// pod -> daemonset
		Entry("Given a pod name that has a reference to a daemonset, it should return the daemonset name", kube, "test-app-abcdefg", "test-ds-namespace", "test-app"),
		// pod -> replicaset -> deployment
		Entry("Given a pod name that has a reference to a replicaset which was created by a deployment, it should return the deployment name", kube, "test-app-abcdefg", "test-rs-namespace", "test-app"),
		// something that doesn't exist
		Entry("Given a pod name that has a reference to a deployment that doesn't exist, the non-existing deployment name is returned", kube, "test-app-abcdefg", "test-orphan-namespace", "test-app"),
		// no owner reference
		Entry("Given a pod name that does not have a reference, it should return the pod name", kube, "test-app-abcdefg", "test-alone-namespace", "test-app-abcdefg"),
		// wildcard name -> deployment
		Entry("Given a wildcard name (probably replicaset), it should return the deployment that would create it", kube, "test-app-*", "test-wc-namespace", "test-app"),
	)
})
