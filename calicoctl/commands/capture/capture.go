// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package capture

import (
	"fmt"
	"github.com/projectcalico/calicoctl/v3/calicoctl/commands/common"
	log "github.com/sirupsen/logrus"
	"path"
	"regexp"
	"strings"
)

// CopyCommand is a kubectl command that will be executed to copy capture files from a pod
const CopyCommand = "kubectl cp %s/%s:%s/%s/%s/ %s -c %s"

// CleanCommand is a kubectl command that will executed to clean capture files from a pod
const CleanCommand = "kubectl exec -n %s %s -c %s -- rm -r %s/%s/%s"

// GetFluentDNodesCommand is a kubectl command that will executed retrieve the fluentD pods
const GetFluentDNodesCommand = "kubectl get pod -o=custom-columns=NAME:.metadata.name -ntigera-fluentd -l k8s-app=fluentd-node --no-headers"

// FindCaptureFileCommand is a kubectl command that will be executed to determine capture files have been generated
const FindCaptureFileCommand = "kubectl exec -n %s %s -c %s -- find %s -type d -maxdepth 2"

// CaptureNamespace used to execute commands inside pods
const TigeraFluentDNS = "tigera-fluentd"
// Container name used to execute commands inside pods
const TigeraFluentD = "fluentd"

// commands is wrapper for available capture commands
type commands struct {
	cmdExecutor common.CmdExecutor
}

// NewCommands returns new capture commands
func NewCommands(cmd common.CmdExecutor) commands {
	return commands{cmdExecutor: cmd}
}

// Location represents the exact location of a capture file in the k8s cluster
type Location struct {
	CaptureNamespace string
	Name             string
	Namespace        string
	Pod              string
	Container        string
	Dir              string
}

func (l Location) RelativePath() string {
	return path.Join(l.CaptureNamespace, l.Name)
}

// Copy will copy capture files capture files from fluentD pods located at captureDir/captureNamespace/captureName to destination
func (cmd *commands) Copy(locations []Location, destination string) (int, []error) {
	var errors []error
	var successfulResults int

	for _, loc := range locations {
		log.Debugf("Retrieving capture files for: %s", loc.RelativePath())

		output, err := cmd.copyCaptureFiles(loc, destination)
		if err != nil {
			log.WithError(err).Warnf("Could not copy capture files %#v", loc)
			errors = append(errors, err)
			continue
		}
		log.Infof("Copy command output %s", output)
		fmt.Printf("Copy capture files for %s to %s\n", loc.RelativePath(), destination)
		successfulResults++
	}

	return successfulResults, errors
}

func (cmd *commands) copyCaptureFiles(loc Location, destination string) (string, error) {
	output, err := cmd.cmdExecutor.Execute(fmt.Sprintf(
		CopyCommand,
		loc.Namespace,
		loc.Pod,
		loc.Dir,
		loc.CaptureNamespace,
		loc.Name,
		destination,
		loc.Container,
	))
	return output, err
}

// Clean will clean capture files from fluentD pods located at captureDir/captureNamespace/captureName
func (cmd *commands) Clean(locations []Location) (int, []error) {
	var errors []error
	var successfulResults int

	for _, loc := range locations {
		output, err := cmd.cleanCaptureFiles(loc)
		if err != nil {
			log.WithError(err).Warnf("Could not clean capture files for %#v", loc)
			errors = append(errors, err)
			continue
		}
		log.Infof("Clean command output %s", output)
		fmt.Printf("Clean capture files for %s\n", loc.RelativePath())
		successfulResults++
	}

	return successfulResults, errors
}

func (cmd *commands) cleanCaptureFiles(loc Location) (string, error) {
	output, err := cmd.cmdExecutor.Execute(fmt.Sprintf(
		CleanCommand,
		loc.Namespace,
		loc.Pod,
		loc.Container,
		loc.Dir,
		loc.CaptureNamespace,
		loc.Name,
	))
	return output, err
}

// List will check nodes to see if capture files have been generated for a captureName in captureNamespace
// The capture will be listed across all the namespaces if captureNamespace is emtpy
func (cmd *commands) List(captureDir, captureName, captureNamespace string) ([]Location, error) {
	output, err := cmd.cmdExecutor.Execute(GetFluentDNodesCommand)
	if err != nil {
		log.WithError(err).Warnf("Fail to resolve capture files for %s", captureName)
		return nil, err
	}

	var locations []Location
	var entries = strings.Split(output, "\n")
	for _, fluentDPod := range entries {
		if len(fluentDPod) != 0 {
			out, err := cmd.cmdExecutor.Execute(fmt.Sprintf(FindCaptureFileCommand, TigeraFluentDNS, fluentDPod, TigeraFluentD, captureDir))
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
						var resolvedNamespace = matches[0][2]
						if len(captureNamespace) == 0 || captureNamespace == resolvedNamespace {
							log.Debugf("Capture %s/%s has generated capture files on %s", resolvedNamespace, captureName, fluentDPod)
							locations = append(locations, Location{
								Name: captureName, Dir: captureDir, Pod: fluentDPod, CaptureNamespace: resolvedNamespace,
								Container: TigeraFluentD, Namespace: TigeraFluentDNS,
							})
						}
					}
				}
			}
		}
	}

	return locations, nil
}
