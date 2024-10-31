// Copyright (c) 2019, 2022 Tigera, Inc. All rights reserved.
package policyrec

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"

	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
)

const wildcardSuffix = "-*"

// GeneratePolicyName generates a policy namespace for policy recommendation, using the endpoint
// name and namespace.
func GeneratePolicyName(k lmak8s.ClientSet, name, namespace string) string {
	// Checks the owner reference and returns the name of highest owner in the chain.
	// Remove the trailing -* wildcard suffix from the name if it exists.
	policyName := strings.TrimSuffix(name, wildcardSuffix)

	// TODO: What to do about no namespace for global policies?
	// TODO: What do we do about resources that share the same name/namespace but are different resources?
	obj := GetObjectMeta(k, "", policyName, namespace)
	if obj == nil {
		// For some reason, the resource we are searching for does not exist. Return the searched name.
		return policyName
	}
	for len(obj.GetObjectMeta().GetOwnerReferences()) > 0 {
		// Only do the lookup on the first owner reference.
		ref := obj.GetObjectMeta().GetOwnerReferences()[0]
		obj = GetObjectMeta(k, ref.Kind, ref.Name, namespace)
		policyName = ref.Name
		if obj == nil {
			// For some reason, the resource referenced does not exist. Return the searched name.
			break
		}
	}

	return policyName
}

// GetObjectMeta returns the object metadata of a resource, if that resource exists.
func GetObjectMeta(k k8s.Interface, kind, name, namespace string) metav1.ObjectMetaAccessor {
	ctx := context.Background()
	// Query each of the valid Kinds until something matches.
	switch kind {
	case "DaemonSet":
		obj, err := k.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting daemonset %s/%s", namespace, name)
			return nil
		}
		return obj
	case "Deployment":
		obj, err := k.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting deployment %s/%s", namespace, name)
			return nil
		}
		return obj
	case "ReplicaSet":
		obj, err := k.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting replicaset %s/%s", namespace, name)
			return nil
		}
		return obj
	case "StatefulSet":
		obj, err := k.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting statefulset %s/%s", namespace, name)
			return nil
		}
		return obj
	case "Job":
		obj, err := k.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting job %s/%s", namespace, name)
			return nil
		}
		return obj
	case "Pod":
		obj, err := k.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting pod %s/%s", namespace, name)
			return nil
		}
		return obj
	case "CronJob":
		obj, err := k.BatchV1beta1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Infof("Error getting cronjob %s/%s", namespace, name)
			return nil
		}
		return obj
	default:
		// We do not know the kind and need to search each type separately.
	}

	if obj, err := k.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get daemonset %s/%s", namespace, name)
	}
	if obj, err := k.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get deployment %s/%s", namespace, name)
	}
	if obj, err := k.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get replicaset %s/%s", namespace, name)
	}
	if obj, err := k.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get statefulset %s/%s", namespace, name)
	}
	if obj, err := k.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get job %s/%s", namespace, name)
	}
	if obj, err := k.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get pod %s/%s", namespace, name)
	}
	if obj, err := k.BatchV1beta1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{}); obj != nil {
		if err == nil {
			return obj
		}
		log.WithError(err).Debugf("Could not get cronjob %s/%s", namespace, name)
	}

	// Couldn't find any object that matches the given name.
	log.Warnf("Could not find any valid object that matches %s/%s", namespace, name)
	return nil
}
