// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package cluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calicoctl/v3/calicoctl/commands/argutils"
	"github.com/projectcalico/calicoctl/v3/calicoctl/commands/common"
	"github.com/projectcalico/calicoctl/v3/calicoctl/commands/constants"
)

const (
	archiveName   = "calico-diagnostics.tar.gz"
	directoryName = "calico-diagnostics"
)

// Diags executes a series of kubectl exec commands to retrieve logs and resource information
// for the configured cluster.
func Diags(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl cluster diags [--since=<SINCE>] [--config=<CONFIG>]

Options:
  -h --help                Show this screen.
     --since=<SINCE>       Only collect logs newer than provided relative duration, in seconds (s), minutes (m) or hours (h)
  -c --config=<CONFIG>     Path to the file containing connection configuration in
                           YAML or JSON format.
                           [default: ` + constants.DefaultConfigPath + `]

Description:
  The cluster diags command collects a snapshot of diagnostic info and logs related to Calico for the given cluster.
`
	parsedArgs, err := docopt.ParseArgs(doc, args, "")
	if err != nil {
		return fmt.Errorf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.", strings.Join(args, " "))
	}
	if len(parsedArgs) == 0 {
		return nil
	}

	since := parsedArgs["--since"]
	// Set a default if since flag was not specified
	if since == nil {
		since = "0s"
	}
	return collectDiags(since.(string))
}

func collectDiags(since string) error {
	// Ensure since value is valid with proper time unit
	sinceFlag := argutils.ValidateSinceDuration(since)

	// Ensure kubectl command is available (since we need it to access BGP information)
	if err := common.KubectlExists(); err != nil {
		return fmt.Errorf("missing dependency: %s", err)
	}

	fmt.Println("==== Begin collecting diagnostics. ====")

	// Create a temp folder to house all diagnostic files. Use empty string for dir parameter.
	// TempDir will use the default directory for temporary files (see os.TempDir).
	rootDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	// Clean up everything that is temporary afterwards
	defer os.RemoveAll(rootDir)

	// Within temp dir create a folder that will be used to zip everything up in the end
	dir := fmt.Sprintf("%s/%s", rootDir, directoryName)
	err = os.Mkdir(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating root diagnostics directory: %v", err)
	}

	collectBasicState(dir, sinceFlag)
	collectOperatorDiags(dir, sinceFlag)
	collectIPAMDiags(dir)
	collectTyphaLogs(dir, sinceFlag)
	collectNodeDiags(dir, sinceFlag)
	createArchive(rootDir)

	return nil
}

// collectBasicState collects namespace data and all resources from the Calico and Operator namespaces,
// as well as state on each of the main user-facing operator related resources.
func collectBasicState(dir, sinceFlag string) {
	fmt.Println("Collecting basic cluster state ...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect all namespaces",
			CmdStr:   "kubectl get ns",
			FilePath: fmt.Sprintf("%s/namespaces.txt", dir),
		},
		{
			Info:     fmt.Sprintf("Collect all in %s", common.CalicoNamespace),
			CmdStr:   fmt.Sprintf("kubectl get all -n %s -o wide", common.CalicoNamespace),
			FilePath: fmt.Sprintf("%s/calico-system.txt", dir),
		},
		{
			Info:     fmt.Sprintf("Collect all in %s", common.TigeraOperatorNamespace),
			CmdStr:   fmt.Sprintf("kubectl get all -n %s -o wide", common.TigeraOperatorNamespace),
			FilePath: fmt.Sprintf("%s/tigera-operator.txt", dir),
		},
	})

	fmt.Println("Collecting host node details ...")

	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect nodes",
			CmdStr:   "kubectl get nodes -o wide",
			FilePath: fmt.Sprintf("%s/nodes.txt", dir),
		},
		{
			Info:     "Collect nodes yaml",
			CmdStr:   "kubectl get nodes -o yaml",
			FilePath: fmt.Sprintf("%s/nodes.yaml", dir),
		},
	})
}

// collectOperatorDiags retrieves diagnostics related to Tigera operator.
func collectOperatorDiags(dir, sinceFlag string) {
	operatorDir := fmt.Sprintf("%s/%s", dir, "operator.tigera.io")
	err := os.Mkdir(operatorDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating operator diagnostics directory: %v\n", err)
		return
	}

	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect installations",
			CmdStr:   "kubectl get installations -o yaml",
			FilePath: fmt.Sprintf("%s/installations.yaml", operatorDir),
		},
		{
			Info:     "Collect apiservers",
			CmdStr:   "kubectl get apiservers -o yaml",
			FilePath: fmt.Sprintf("%s/apiservers.yaml", operatorDir),
		},
		{
			Info:     "Collect compliances",
			CmdStr:   "kubectl get compliances -o yaml",
			FilePath: fmt.Sprintf("%s/compliances.yaml", operatorDir),
		},
		{
			Info:     "Collect intrusiondetections",
			CmdStr:   "kubectl get intrusiondetections -o yaml",
			FilePath: fmt.Sprintf("%s/intrusiondetections.yaml", operatorDir),
		},
		{
			Info:     "Collect managers",
			CmdStr:   "kubectl get managers -o yaml",
			FilePath: fmt.Sprintf("%s/managers.yaml", operatorDir),
		},
		{
			Info:     "Collect logcollectors",
			CmdStr:   "kubectl get logcollectors -o yaml",
			FilePath: fmt.Sprintf("%s/logcollectors.yaml", operatorDir),
		},
		{
			Info:     "Collect logstorages",
			CmdStr:   "kubectl get logstorages -o yaml",
			FilePath: fmt.Sprintf("%s/logstorages.yaml", operatorDir),
		},
		{
			Info:     "Collect managementclusterconnections",
			CmdStr:   "kubectl get managementclusterconnections -o yaml",
			FilePath: fmt.Sprintf("%s/managementclusterconnections.yaml", operatorDir),
		},
	})

	fmt.Println("Collecting tigera-operator logs ...")
	common.ExecCmdWriteToFile(common.Cmd{
		Info:     "Collect ipamblocks yaml",
		CmdStr:   fmt.Sprintf("kubectl logs --since=%s -n %s deployment/tigera-operator", sinceFlag, common.TigeraOperatorNamespace),
		FilePath: fmt.Sprintf("%s/tigera-operator.logs", dir),
	})

	fmt.Println("Collecting TigeraStatus details ...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect tigerastatus",
			CmdStr:   "kubectl get tigerastatus",
			FilePath: fmt.Sprintf("%s/tigerastatus.txt", dir),
		},
		{
			Info:     "Collect tigerastatus yaml",
			CmdStr:   "kubectl get tigerastatus -o yaml",
			FilePath: fmt.Sprintf("%s/tigerastatus.yaml", dir),
		},
	})
}

// collectIPAMDiags collects metadata related to Calico IPAM.
func collectIPAMDiags(dir string) {
	fmt.Println("Collecting IPAM diagnostics ...")
	ipamDir := fmt.Sprintf("%s/%s", dir, "ipam")
	err := os.Mkdir(ipamDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating ipam diagnostics directory: %v\n", err)
		return
	}
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect ipamblocks yaml",
			CmdStr:   "kubectl get ipamblocks -o yaml",
			FilePath: fmt.Sprintf("%s/ipamblocks.yaml", ipamDir),
		},
		{
			Info:     "Collect blockaffinities yaml",
			CmdStr:   "kubectl get blockaffinities -o yaml",
			FilePath: fmt.Sprintf("%s/blockaffinities.yaml", ipamDir),
		},
		{
			Info:     "Collect ipamhandles yaml",
			CmdStr:   "kubectl get ipamhandles -o yaml",
			FilePath: fmt.Sprintf("%s/ipamhandles.yaml", ipamDir),
		},
	})
}

// collectTyphaLogs iterates over each Typha pod and collects the logs from the pod using the provided
// sinceFlag value to filter down logs as needed.
func collectTyphaLogs(dir, sinceFlag string) {
	fmt.Println("Collecting calico/typha logs ...")

	typhaDir := fmt.Sprintf("%s/%s", dir, "typhas")
	err := os.Mkdir(typhaDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating typha diagnostics directory: %v\n", err)
		return
	}

	output, err := common.ExecCmd(fmt.Sprintf(
		"kubectl get pods -n %s -l k8s-app=calico-typha -o go-template --template {{range.items}}{{.metadata.name}},{{end}}",
		common.CalicoNamespace,
	))
	if err != nil {
		fmt.Printf("Could not retrieve typha pods: %s\n", err)
		return
	}
	typhaPods := strings.TrimSuffix(output.String(), ",")
	log.Debugf("typha pods: %s\n", typhaPods)

	pods := strings.Split(strings.TrimSpace(typhaPods), ",")
	for _, p := range pods {
		common.ExecCmdWriteToFile(common.Cmd{
			Info:     fmt.Sprintf("Collect logs for typha pod %s", p),
			CmdStr:   fmt.Sprintf("kubectl logs --since=%s -n %s %s", sinceFlag, common.CalicoNamespace, p),
			FilePath: fmt.Sprintf("%s/%s.log", typhaDir, p),
		})
	}
}

// collectNodeDiags iterates over each Calico node pod and collects logs and network info from the pod
// using the provided sinceFlag value to filter down logs as needed.
func collectNodeDiags(dir, sinceFlag string) {
	fmt.Println("Collecting per-node logs and network information ...")

	nodeDir := fmt.Sprintf("%s/%s", dir, "nodes")
	err := os.Mkdir(nodeDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating node diagnostics directory: %v\n", err)
		return
	}

	output, err := common.ExecCmd(fmt.Sprintf(
		"kubectl get pods -n %s -l k8s-app=calico-node -o go-template --template {{range.items}}{{.metadata.name}},{{end}}",
		common.CalicoNamespace,
	))
	if err != nil {
		fmt.Printf("Could not retrieve node pods: %s\n", err)
		return
	}
	nodePods := strings.TrimSuffix(output.String(), ",")
	log.Debugf("calico node pods: %s\n", nodePods)

	pods := strings.Split(strings.TrimSpace(nodePods), ",")
	for _, p := range pods {
		fmt.Printf("Collecting logs for node: %s\n", p)

		curNodeDir := fmt.Sprintf("%s/%s", nodeDir, p)
		err := os.Mkdir(curNodeDir, os.ModePerm)
		if err != nil {
			fmt.Printf("Error creating node diagnostics directory for node %s: %v\n", p, err)
			// Skip to the next node
			continue
		}

		common.ExecAllCmdsWriteToFile([]common.Cmd{
			{
				Info:     fmt.Sprintf("Collect logs for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl logs --since=%s -n %s %s", sinceFlag, common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/%s.log", curNodeDir, p),
			},
			{
				Info:     fmt.Sprintf("Collect iptables for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- iptables-save -c", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/iptables-save.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ip routes for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip route", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/iproute.txt", curNodeDir),
			},
		})

		// Collect all of the CNI logs
		output, err := common.ExecCmd(fmt.Sprintf(
			"kubectl exec -n %s -t %s -- ls /var/log/calico/cni",
			common.CalicoNamespace,
			p,
		))
		if err != nil {
			fmt.Printf("Error listing the Calico CNI logs at /var/log/calico/cni/: %s\n", err)
			// Skip to the next node
			continue
		}

		cniLogFiles := strings.Split(strings.TrimSpace(output.String()), "\n")
		for _, logFile := range cniLogFiles {
			common.ExecCmdWriteToFile(common.Cmd{
				Info:     fmt.Sprintf("Collect CNI log %s for the node %s", logFile, p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- cat /var/log/calico/cni/%s", common.CalicoNamespace, p, logFile),
				FilePath: fmt.Sprintf("%s/%s.log", curNodeDir, logFile),
			})
		}
	}
}

// createArchive attempts to bundle all the diagnostics files into a single compressed archive.
func createArchive(dir string) {
	fmt.Println("\n==== Producing a diagnostics bundle. ====")

	// Attempt to remove archive file (if it previously existed)
	err := os.Remove(fmt.Sprintf("rm -f %s", archiveName))
	if err != nil {
		// Not an error case we need to show the user
		log.Debugf("Could not remove previous version of %s: %s\n", archiveName, err)
	}

	// Attempt to create new archive
	output, err := common.ExecCmd(fmt.Sprintf("tar cfz ./%s -C %s %s", archiveName, dir, directoryName))
	log.Debugf("creating archive %s: output %s", archiveName, output.String())
	if err != nil {
		fmt.Printf("Could not create new archive %s: %s\n", archiveName, err)
		return
	}

	fmt.Printf("Diagnostic bundle available at ./%s\n", archiveName)
}
