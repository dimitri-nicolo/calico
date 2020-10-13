// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package capture

import (
	"fmt"
	"github.com/projectcalico/calicoctl/calicoctl/commands/common"
	log "github.com/sirupsen/logrus"
	"strings"
)

// CopyCommand is a kubectl command that will be executed to copy capture files from a pod
const CopyCommand = "kubectl cp %s/%s:%s/%s/%s/ %s"
// CleanCommand is a kubectl command that will executed to clean capture files from a pod
const CleanCommand = "kubectl exec -n %s %s -- rm -r %s/%s/%s"
// GetCalicoNodesCommand is a kubectl command that will executed retrieve the tuple (calico-node name, host)
const GetCalicoNodesCommand = "kubectl get pod -o=custom-columns=NAME:.metadata.name,NODE:.spec.nodeName -ncalico-system -l k8s-app=calico-node --no-headers"
// FindCaptureFileCommand is a kubectl command that will be executed ti determine capture files have been generated
const FindCaptureFileCommand = "kubectl exec -n %s %s -- stat %s/%s/%s"
// GetPodByNodeName is a kubectl command that will be executed to retrieve a pod scheduled on a node
const GetPodByNodeName = "kubectl get pods -n %s --no-headers --field-selector spec.nodeName=%s -o=custom-columns=NAME:..metadata.name"
// Namespace used to execute commands inside pods
const entryNamespace = "tigera-fluentd"

// commands is wrapper over the CmdExecutor
type commands struct {
	cmdExecutor CmdExecutor
}

// NewCommands returns new capture commands that use kubectl
func NewCommands(cmd CmdExecutor) commands {
	return commands{cmdExecutor: cmd}
}

// CmdExecutor will execute a command and return its output and its error
type CmdExecutor interface {
	Execute(cmdStr string) (string, error)
}

// kubectlCmd is a kubectl wrapper for any query that will be executed
type kubectlCmd struct {
	kubeConfig string
}

// NewKubectlCmd return a CmdExecutor that uses kubectl
func NewKubectlCmd(kubeConfigPath string) *kubectlCmd {
	return &kubectlCmd{kubeConfig: kubeConfigPath}
}

func (k *kubectlCmd) Execute(cmdStr string) (string, error) {
	var out, err = common.ExecCmd(strings.Replace(cmdStr, "kubectl", fmt.Sprintf("kubectl --kubeconfig %s",k.kubeConfig), 1))
	if out != nil {
		return out.String(), err
	}
	return "", err
}

// ResolveEntryPoints will resolve capture files and match any fluentD pods that have been scheduled on the same node
func (cmd *commands) ResolveEntryPoints(captureDir, captureName, captureNs string) ([]string, string) {
	var locations []string

	var nodeNames, err = cmd.resolveNodeNames(captureDir, captureName, captureNs)
	if err != nil {
		log.WithError(err).Warnf("Could not resolve capture files for %s/%s", captureNs, captureName)
		return locations, ""
	}

	for _, nodeName := range nodeNames {
		var pod, err = cmd.resolveEntryPod(nodeName, entryNamespace)
		if err != nil {
			log.WithError(err).Warnf("Could not resolve capture files for %s/%s", captureNs, captureName)
		} else {
			locations = append(locations, pod)
		}
	}

	return locations, entryNamespace
}

// Copy will copy capture files from the entryPods from entryNamespace under captureDir/captureNamespace/captureName at destination
func (cmd *commands) Copy(entryPods []string, entryNamespace, captureName, captureNamespace, captureDir, destination string) error {
	for _, pod := range entryPods {
		output, err := cmd.copyCaptureFiles(entryNamespace, pod, captureDir, captureNamespace, captureName, destination)
		if err != nil {
			log.WithError(err).Warnf("Could not copy capture files for %s/%s from %s/%s", captureNamespace, captureName, entryNamespace, pod)
			return err
		}
		log.Infof("Copy command output %s", output)
		fmt.Printf("Copy capture files for %s/%s to %s\n", captureNamespace, captureName, destination)
	}

	return nil
}

func (cmd *commands) copyCaptureFiles(entryNamespace string, pod string, captureDir string, captureNamespace string, captureName string, destination string) (string, error) {
	output, err := cmd.cmdExecutor.Execute(fmt.Sprintf(
		CopyCommand,
		entryNamespace,
		pod,
		captureDir,
		captureNamespace,
		captureName,
		destination,
	))
	return output, err
}

// Clean will clean capture files from the entryPods from entryNamespace located at captureDir/captureNamespace/captureName
func (cmd *commands) Clean(entryPods []string, entryNamespace, captureName, captureNamespace, captureDir string) error {
	for _, pod := range entryPods {
		output, err := cmd.cleanCaptureFiles(entryNamespace, pod, captureDir, captureNamespace, captureName)
		if err != nil {
			log.WithError(err).Warnf("Could not clean capture files for %s/%s from %s/%s", captureNamespace, captureName, entryNamespace, pod)
			return err
		}
		log.Infof("Clean command output %s", output)
		fmt.Printf("Clean capture files for %s/%s\n", captureNamespace, captureName)
	}
	return nil
}

func (cmd *commands) cleanCaptureFiles(entryNamespace string, pod string, captureDir string, captureNamespace string, captureName string) (string, error) {
	output, err := cmd.cmdExecutor.Execute(fmt.Sprintf(
		CleanCommand,
		entryNamespace,
		pod,
		captureDir,
		captureNamespace,
		captureName,
	))
	return output, err
}

func (cmd *commands) resolveNodeNames(captureDir, captureName, captureNs string) ([]string, error) {
	output, err := cmd.cmdExecutor.Execute(GetCalicoNodesCommand)
	if err != nil {
		return nil, err
	}

	var nodes []string
	var entries = strings.Split(output, "\n")
	for _, entry := range entries {
		if len(entry) != 0 {
			var calicoNode = strings.Split(entry, "   ")
			_, err := cmd.cmdExecutor.Execute(fmt.Sprintf(FindCaptureFileCommand, common.CalicoNamespace, calicoNode[0], captureDir, captureNs, captureName))
			if err != nil {
				log.Debugf("No capture files are found under %s/%s/%s for %s on node %s", captureDir, captureNs, captureName, calicoNode[0], calicoNode[1])
				continue
			}
			log.Infof("Capture %s/%s has generated capture files on node %s", captureNs, captureName, calicoNode[1])
			nodes = append(nodes, calicoNode[1])
		}
	}

	return nodes, nil
}

func (cmd *commands) resolveEntryPod(nodeName, namespace string) (string, error) {
	output, err := cmd.cmdExecutor.Execute(fmt.Sprintf(
		GetPodByNodeName,
		namespace,
		nodeName,
	))

	if err != nil {
		return "", err
	}

	return strings.Trim(output, " \n"), nil
}
