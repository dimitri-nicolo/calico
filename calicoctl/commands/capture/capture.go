// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package capture

import (
	"fmt"
	"github.com/projectcalico/calicoctl/calicoctl/commands/common"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// CopyCommand is a kubectl command that will be executed to copy capture files from a pod
const CopyCommand = "kubectl cp %s/%s:%s/%s/%s/ %s"

// CleanCommand is a kubectl command that will executed to clean capture files from a pod
const CleanCommand = "kubectl exec -n %s %s -- rm -r %s/%s/%s"

// GetFluentDNodesCommand is a kubectl command that will executed retrieve the fluentD pods
const GetFluentDNodesCommand = "kubectl get pod -o=custom-columns=NAME:.metadata.name -ntigera-fluentd -l k8s-app=fluentd-node --no-headers"

// FindCaptureFileCommand is a kubectl command that will be executed to determine capture files have been generated
const FindCaptureFileCommand = "kubectl exec -n %s %s -- find %s -type d -maxdepth 2"

// Namespace used to execute commands inside pods
const TigeraFluentDNS = "tigera-fluentd"

// commands is wrapper for available capture commands
type commands struct {
	cmdExecutor CmdExecutor
}

// NewCommands returns new capture commands
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
	var out, err = common.ExecCmd(strings.Replace(cmdStr, "kubectl", fmt.Sprintf("kubectl --kubeconfig %s", k.kubeConfig), 1))
	if out != nil {
		return out.String(), err
	}
	return "", err
}

// Location maps out a capture location
type Location struct {
	Namespace string
	Name      string
	Pod       string
	Dir       string
}

// Copy will copy capture files capture files from fluentD pods located at captureDir/captureNamespace/captureName to destination
func (cmd *commands) Copy(locations []Location, destination string) (int, []error) {
	var errors []error
	var successfulResults int

	for _, loc := range locations {
		log.Debugf("Retrieving capture files for: %s/%s", loc.Namespace, loc.Name)

		output, err := cmd.copyCaptureFiles(TigeraFluentDNS, loc.Pod, loc.Dir, loc.Namespace, loc.Name, destination)
		if err != nil {
			log.WithError(err).Warnf("Could not copy capture files for %s/%s from %s/%s", loc.Namespace, loc.Name, TigeraFluentDNS, loc.Pod)
			errors = append(errors, err)
			continue
		}
		log.Infof("Copy command output %s", output)
		fmt.Printf("Copy capture files for %s/%s to %s\n", loc.Namespace, loc.Name, destination)
		successfulResults++
	}

	return successfulResults, errors
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

// Clean will clean capture files from fluentD pods located at captureDir/captureNamespace/captureName
func (cmd *commands) Clean(locations []Location) (int, []error) {
	var errors []error
	var successfulResults int

	for _, loc := range locations {
		output, err := cmd.cleanCaptureFiles(TigeraFluentDNS, loc.Pod, loc.Dir, loc.Namespace, loc.Name)
		if err != nil {
			log.WithError(err).Warnf("Could not clean capture files for %s/%s from %s/%s", loc.Namespace, loc.Name, TigeraFluentDNS, loc.Pod)
			errors = append(errors, err)
			continue
		}
		log.Infof("Clean command output %s", output)
		fmt.Printf("Clean capture files for %s/%s\n", loc.Namespace, loc.Name)
		successfulResults++
	}

	return successfulResults, errors
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

// Resolve will check nodes to see if capture have been generated in any namespace
func (cmd *commands) Resolve(captureDir, captureName string) ([]Location, error) {
	output, err := cmd.cmdExecutor.Execute(GetFluentDNodesCommand)
	if err != nil {
		log.WithError(err).Warnf("Fail to resolve capture files for %s", captureName)
		return nil, err
	}

	var locations []Location
	var entries = strings.Split(output, "\n")
	for _, fluentDPod := range entries {
		if len(fluentDPod) != 0 {
			out, err := cmd.cmdExecutor.Execute(fmt.Sprintf(FindCaptureFileCommand, TigeraFluentDNS, fluentDPod, captureDir))
			if err != nil {
				log.Debugf("No capture files are found under %s for %s on %s", captureDir, captureName, fluentDPod)
				continue
			}
			var dirs = strings.Split(out, "\n")
			var reg = regexp.MustCompile(fmt.Sprintf("(%s)/(.*)/(%s)", captureDir, captureName))
			for _, d := range dirs {
				if len(d) != 0 {
					var matches = reg.FindAllStringSubmatch(d, -1)
					if matches != nil {
						log.Infof("Capture %s/%s has generated capture files on %s", matches[0][2], captureName, fluentDPod)
						locations = append(locations, Location{
							Name: captureName, Dir: captureDir, Pod: fluentDPod, Namespace: matches[0][2],
						})
					}
				}
			}
		}
	}

	return locations, nil
}
