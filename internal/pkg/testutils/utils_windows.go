// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/Microsoft/hcsshim"
	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	types020 "github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/current"
	version "github.com/mcuadros/go-version"
	"github.com/projectcalico/cni-plugin/internal/pkg/utils"
	"github.com/projectcalico/cni-plugin/pkg/k8s"
	plugintypes "github.com/projectcalico/cni-plugin/pkg/types"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const HnsNoneNs = "none"

// Delete all K8s pods from the "test" namespace
func WipeK8sPods(netconf string) {
	conf := plugintypes.NetConf{}
	if err := json.Unmarshal([]byte(netconf), &conf); err != nil {
		panic(err)
	}
	logger := log.WithFields(log.Fields{
		"Namespace": HnsNoneNs,
	})
	clientset, err := k8s.NewK8sClient(conf, logger)
	if err != nil {
		panic(err)
	}

	log.WithField("clientset:", clientset).Info("DEBUG")
	pods, err := clientset.CoreV1().Pods(K8S_TEST_NS).List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, pod := range pods.Items {
		err = clientset.CoreV1().Pods(K8S_TEST_NS).Delete(pod.Name, &metav1.DeleteOptions{})

		if err != nil {
			if kerrors.IsNotFound(err) {
				continue
			}
			panic(err)
		}
	}
	log.Info("WipeK8sPods Sucess")
}

func CreateContainerUsingDocker() (string, error) {
	var image string
	if os.Getenv("WINDOWS_OS") == "Windows1903container" {
		image = "mcr.microsoft.com/windows/servercore/insider:10.0.18362.113"
	} else if os.Getenv("WINDOWS_OS") == "Windows1809container" {
		image = "mcr.microsoft.com/windows/servercore:1809"
	}
	command := fmt.Sprintf("docker run --net none -d -i %s powershell", image)
	cmd := exec.Command("powershell.exe", command)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	temp := strings.TrimSpace(string(out))
	log.Debugf("container ID: %s", temp)
	return temp, nil
}

func DeleteContainerUsingDocker(containerId string) error {
	command := fmt.Sprintf("docker rm -f %s", containerId)
	cmd := exec.Command("powershell.exe", command)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.WithError(err).WithField("id", containerId).Error("Failed to stop docker container")
		return err
	}
	return nil
}

func CreateContainer(netconf, podName, podNamespace, ip, k8sNs string) (containerID string, result *current.Result, contVeth string, contAddr []string, contRoutes []string, err error) {
	containerID, err = CreateContainerUsingDocker()
	if err != nil {
		return "", nil, "", []string{}, []string{}, err
	}
	result, contVeth, contAddr, contRoutes, err = RunCNIPluginWithId(netconf, podName, podNamespace, ip, containerID, "", k8sNs)
	if err != nil {
		return containerID, nil, "", []string{}, []string{}, err
	}
	return
}

// Create container with the giving containerId when containerId is not empty
//
// Deprecated: Please call CreateContainerNamespace and then RunCNIPluginWithID directly.
func CreateContainerWithId(netconf, podName, podNamespace, ip, overrideContainerID, k8sNs string) (containerID string, result *current.Result, contVeth string, contAddr []string, contRoutes []string, err error) {
	containerID, err = CreateContainerUsingDocker()
	if err != nil {
		return "", nil, "", []string{}, []string{}, err
	}

	result, contVeth, contAddr, contRoutes, err = RunCNIPluginWithId(netconf, podName, podNamespace, ip, containerID, "", k8sNs)
	if err != nil {
		log.Errorf("Error: ", err)
		return containerID, nil, "", []string{}, []string{}, err
	}
	return
}

// RunCNIPluginWithId calls CNI plugin with a containerID and targetNs passed to it.
// This is for when you want to call CNI for an existing container.
func RunCNIPluginWithId(
	netconf,
	podName,
	podNamespace,
	ip,
	containerId,
	ifName,
	k8sNs string,
) (
	result *current.Result,
	contVeth string,
	contAddr []string,
	contRoutes []string,
	err error,
) {
	// Set up the env for running the CNI plugin
	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, k8sNs)

		// Append IP=<ip> to CNI_ARGS only if it's not an empty string.
		if ip != "" {
			k8sEnv = fmt.Sprintf("%s;IP=%s", k8sEnv, ip)
		}
	}

	if ifName == "" {
		ifName = "eth0"
	}

	env := os.Environ()
	env = append(env, []string{
		"CNI_COMMAND=ADD",
		fmt.Sprintf("CNI_IFNAME=%s", ifName),
		fmt.Sprintf("CNI_PATH=%s", os.Getenv("BIN")),
		fmt.Sprintf("CNI_CONTAINERID=%s", containerId),
		fmt.Sprintf("CNI_NETNS=%s", podNamespace),
		k8sEnv,
	}...)
	args := &cniArgs{env}

	// Invoke the CNI plugin, returning any errors to the calling code to handle.
	var r types.Result
	pluginPath := fmt.Sprintf("%s\\%s", os.Getenv("BIN"), os.Getenv("PLUGIN"))
	log.Debugf("pluginPath: %v", pluginPath)
	r, err = invoke.ExecPluginWithResult(pluginPath, []byte(netconf), args, nil)
	if err != nil {
		log.Errorf("error from invoke.ExecPluginWithResult %v", err)
		_ = DeleteContainerUsingDocker(containerId)
		return
	}

	// Extract the target CNI version from the provided network config.
	var nc types.NetConf
	if err = json.Unmarshal([]byte(netconf), &nc); err != nil {
		log.Errorf("unmarshal err: ", err)
		panic(err)
	}
	// Parse the result as the target CNI version.
	if version.Compare(nc.CNIVersion, "0.3.0", "<") {
		// Special case for older CNI verisons.
		var out []byte
		out, err = json.Marshal(r)
		log.Infof("CNI output: %s", out)
		r020 := types020.Result{}
		if err = json.Unmarshal(out, &r020); err != nil {
			log.Errorf("Error unmarshaling output to Result: %v", err)
			return
		}

		result, err = current.NewResultFromResult(&r020)
		if err != nil {
			return
		}

	} else {
		result, err = current.GetResult(r)
		if err != nil {
			return
		}
	}

	return
}

// Executes the Calico CNI plugin and return the error code of the command.
func DeleteContainer(netconf, podName, podNamespace, k8sNs string) (exitCode int, err error) {
	return DeleteContainerWithId(netconf, podName, podNamespace, "", k8sNs)
}

//func DeleteContainerWithId(netconf, netnspath, podName, podNamespace, containerId string) (exitCode int, err error) {
func DeleteContainerWithId(netconf, podName, podNamespace, containerId, k8sNs string) (exitCode int, err error) {
	return DeleteContainerWithIdAndIfaceName(netconf, podName, podNamespace, containerId, "eth0", k8sNs)
}

//func DeleteContainerWithIdAndIfaceName(netconf, netnspath, podName, podNamespace, containerId, ifaceName string) (exitCode int, err error) {
func DeleteContainerWithIdAndIfaceName(netconf, podName, podNamespace, containerId, ifaceName, k8sNs string) (exitCode int, err error) {
	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, k8sNs)
	}

	// Set up the env for running the CNI plugin
	env := os.Environ()
	env = append(env, []string{
		"CNI_COMMAND=DEL",
		fmt.Sprintf("CNI_CONTAINERID=%s", containerId),
		fmt.Sprintf("CNI_NETNS=%s", podNamespace),
		"CNI_IFNAME=" + ifaceName,
		fmt.Sprintf("CNI_PATH=%s", os.Getenv("BIN")),
		k8sEnv,
	}...)

	log.Infof("Deleting container with ID %v CNI plugin with the following env vars: %v", containerId, env)
	//now delete the container
	if containerId != "" {
		log.Debugf(" calling DeleteContainerUsingDocker with ContainerID %v", containerId)
		err = DeleteContainerUsingDocker(containerId)
		if err != nil {
			log.Errorf("Error deleting container %s", containerId)
		}
	}

	// Run the CNI plugin passing in the supplied netconf
	args := &cniArgs{env}
	pluginPath := fmt.Sprintf("%s\\%s", os.Getenv("BIN"), os.Getenv("PLUGIN"))
	log.Debugf("pluginPath: %v", pluginPath)
	err = invoke.ExecPluginWithoutResult(pluginPath, []byte(netconf), args, nil)
	if err != nil {
		log.Errorf("error from invoke.ExecPluginWithoutResult %v", err)
		return
	}

	return
}

func NetworkPod(
	netconf string,
	podName string,
	ip string,
	ctx context.Context,
	calicoClient client.Interface,
	result *current.Result,
	containerID string,
	netns string,
	k8sNs string,
) (err error) {

	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, k8sNs)
		// Append IP=<ip> to CNI_ARGS only if it's not an empty string.
		if ip != "" {
			k8sEnv = fmt.Sprintf("%s;IP=%s", k8sEnv, ip)
		}
	}

	var args *skel.CmdArgs
	args = &skel.CmdArgs{
		ContainerID: containerID,
		Netns:       netns,
		IfName:      "eth0",
		Args:        k8sEnv,
		Path:        os.Getenv("BIN"),
		StdinData:   []byte(netconf),
	}
	conf := plugintypes.NetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	var logger *log.Entry

	logger = log.WithFields(log.Fields{
		"ContainerID": containerID,
		"Pod":         podName,
		"Namespace":   netns,
	})
	_, _, err = utils.DoNetworking(ctx, calicoClient, args, conf, result, logger, "", nil)
	return err
}

func CreateNetwork(netconf string) (*hcsshim.HNSNetwork, error) {
	var conf plugintypes.NetConf
	if err := json.Unmarshal([]byte(netconf), &conf); err != nil {
		log.Errorf("unmarshal err: ", err)
		panic(err)
	}

	result := &current.Result{}

	_, subNet, _ := net.ParseCIDR(conf.IPAM.Subnet)

	var logger *log.Entry
	logger = log.WithFields(log.Fields{
		"Name": conf.Name,
	})

	var networkName string
	if conf.WindowsUseSingleNetwork {
		logger.WithField("name", conf.Name).Info("Overriding network name, only a single IPAM block will be supported on this host")
		networkName = conf.Name
	} else {
		networkName = utils.CreateNetworkName(conf.Name, subNet)
	}

	hnsNetwork, err := utils.EnsureNetworkExists(networkName, subNet, result, logger)
	if err != nil {
		logger.Errorf("Unable to create hns network %s", networkName)
		return nil, err
	}

	return hnsNetwork, nil
}

func CreateEndpoint(hnsNetwork *hcsshim.HNSNetwork, netconf string) (*hcsshim.HNSEndpoint, error) {
	var conf plugintypes.NetConf
	if err := json.Unmarshal([]byte(netconf), &conf); err != nil {
		log.Errorf("unmarshal err: ", err)
		panic(err)
	}

	var result *current.Result

	_, subNet, _ := net.ParseCIDR(conf.IPAM.Subnet)

	var logger *log.Entry
	logger = log.WithFields(log.Fields{
		"Name": conf.Name,
	})

	epName := hnsNetwork.Name + "_ep"
	hnsEndpoint, err := utils.CreateAndAttachHostEP(epName, hnsNetwork, subNet, result, logger)
	if err != nil {
		logger.Errorf("Unable to create host hns endpoint %s", epName)
		return nil, err
	}

	return hnsEndpoint, nil
}
