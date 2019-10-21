// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"

	log "github.com/sirupsen/logrus"
)

const wildcardSuffix = "-*"

func GeneratePolicyName(k k8s.Interface, params *PolicyRecommendationParams) string {
	// Checks the owner reference and returns the name of highest owner in the chain.
	// Remove the trailing -* wildcard suffix from the name if it exists.
	name := strings.TrimSuffix(params.EndpointName, wildcardSuffix)
	// TODO: What to do about no namespace for global policies?
	// TODO: What do we do about resources that share the same name/namespace but are different resources?
	ns := params.Namespace
	obj := GetObjectMeta(k, "", name, ns)
	if obj == nil {
		// For some reason, the resource we are searching for does not exist. Return the searched name.
		return name
	}
	for len(obj.GetObjectMeta().GetOwnerReferences()) > 0 {
		// Only do the lookup on the first owner reference.
		ref := obj.GetObjectMeta().GetOwnerReferences()[0]
		obj = GetObjectMeta(k, ref.Kind, ref.Name, ns)
		name = ref.Name
		if obj == nil {
			// For some reason, the resource referenced does not exist. Return the searched name.
			break
		}
	}

	return name
}

func GetObjectMeta(k k8s.Interface, kind, name, namespace string) metav1.ObjectMetaAccessor {
	// Query each of the valid Kinds until something matches
	switch kind {
	case "DaemonSet":
		obj, err := k.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting daemonset %s/%s", namespace, name)
			return nil
		}
		return obj
	case "Deployment":
		obj, err := k.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting deployment %s/%s", namespace, name)
			return nil
		}
		return obj
	case "ReplicaSet":
		obj, err := k.AppsV1().ReplicaSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting replicaset %s/%s", namespace, name)
			return nil
		}
		return obj
	case "StatefulSet":
		obj, err := k.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting statefulset %s/%s", namespace, name)
			return nil
		}
		return obj
	case "Job":
		obj, err := k.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting job %s/%s", namespace, name)
			return nil
		}
		return obj
	case "Pod":
		obj, err := k.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting pod %s/%s", namespace, name)
			return nil
		}
		return obj
	case "CronJob":
		obj, err := k.BatchV1beta1().CronJobs(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting cronjob %s/%s", namespace, name)
			return nil
		}
		return obj
	default:
		// We do not know the kind and need to search each type separately.
	}

	if obj, err := k.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting daemonset %s/%s", namespace, name)
			return nil
		}
		return obj
	}
	if obj, err := k.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting deployment %s/%s", namespace, name)
			return nil
		}
		return obj
	}
	if obj, err := k.AppsV1().ReplicaSets(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting replicaset %s/%s", namespace, name)
			return nil
		}
		return obj
	}
	if obj, err := k.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting statefulset %s/%s", namespace, name)
			return nil
		}
		return obj
	}
	if obj, err := k.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting job %s/%s", namespace, name)
			return nil
		}
		return obj
	}
	if obj, err := k.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting pod %s/%s", namespace, name)
			return nil
		}
		return obj
	}
	if obj, err := k.BatchV1beta1().CronJobs(namespace).Get(name, metav1.GetOptions{}); obj != nil {
		if err != nil {
			log.WithError(err).Infof("Error getting cronjob %s/%s", namespace, name)
			return nil
		}
		return obj
	}

	return nil
}
