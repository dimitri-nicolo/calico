package cis

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	defaultOpenshiftVersion = "ocp-3.10"
	masterNodeLabelKey      = "node-role.kubernetes.io/master"
	openshiftLabel          = "openshift.io/control-plane=true"
)

// Determines the follow:
// - if the cluster is running openshift
// - the version of openshift running
// - if the node is master or worker
func determineOpenshiftArgs(nodename string) ([]string, error) {
	// Get k8s client config.
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Error("Failed to fetch cluster config")
		return nil, err
	}

	// Init k8s client.
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.WithError(err).Error("Failed to init k8s client")
		return nil, err
	}

	// Determine if cluster is running openshift.
	ocpVersion, err := isRunningOpenshift(k8sClient.CoreV1().Pods("kube-system"))
	if err != nil {
		return nil, err
	}

	// An empty ocpVersion without an error means cluster is not Openshift.
	if ocpVersion == "" {
		return []string{}, nil
	}

	// Get node.
	node, err := k8sClient.CoreV1().Nodes().Get(nodename, metav1.GetOptions{})
	if err != nil {
		log.WithField("node", nodename).WithError(err).Error("Failed to get node")
		return nil, err
	}

	// Determine if node is master or worker.
	versionFlag := fmt.Sprintf("--version=ocp-%s", ocpVersion)
	if isMasterNode(node) {
		return []string{"master", versionFlag}, nil
	}

	return []string{"node", versionFlag}, nil
}

func isRunningOpenshift(podClient corev1Client.PodInterface) (string, error) {
	// List pods that contain an Openshift specific label.
	listOpts := metav1.ListOptions{LabelSelector: openshiftLabel}
	pods, err := podClient.List(listOpts)
	if err != nil {
		log.WithError(err).Error("Failed to list for Openshift pods")
		return "", err
	}

	// If no pods are found, then the cluster is not running Openshift.
	if len(pods.Items) == 0 {
		log.Debug("Cluster is not running Openshift, returning nothing successfully")
		return "", nil
	}

	// Determine version tag on image of the first pod.
	image := strings.Split(pods.Items[0].Spec.Containers[0].Image, ":")
	tag := image[len(image)-1]
	versions := strings.Split(strings.Trim(tag, "v"), ".")
	if len(versions) <= 1 {
		log.Debug("Failed to determine Openshift version, returning default version")
		return defaultOpenshiftVersion, nil
	}

	version := strings.Join(versions[0:2], ".")
	log.WithField("version", version).Debug("Successfully determined Openshift version")
	return version, nil
}

func isMasterNode(node *corev1.Node) bool {
	val, exists := node.ObjectMeta.Labels[masterNodeLabelKey]
	if !exists {
		return false
	}
	return val == "true"
}
