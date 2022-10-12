// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package cluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/common"
	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/constants"
)

const (
	archiveName   = "calico-diagnostics.tar.gz"
	directoryName = "calico-diagnostics"
)

// Diags executes a series of kubectl exec commands to retrieve logs and resource information
// for the configured cluster.
func Diags(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl cluster diags [--since=<SINCE>] [--config=<CONFIG>] [--allow-version-mismatch]

Options:
  -h --help                    Show this screen.
     --since=<SINCE>           Only collect logs newer than provided relative duration, in seconds (s), minutes (m) or hours (h)
  -c --config=<CONFIG>         Path to the file containing connection configuration in
                               YAML or JSON format.
                               [default: ` + constants.DefaultConfigPath + `]
     --allow-version-mismatch  Allow client and cluster versions mismatch.

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

	err = common.CheckVersionMismatch(parsedArgs["--config"], parsedArgs["--allow-version-mismatch"])
	if err != nil {
		return err
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

	collectGlobalClusterInformation(dir)
	collectNodeDiags(dir, sinceFlag)
	collectCalicoTigeraPodsServicesAndEndpointsDetails(dir, sinceFlag)
	createArchive(rootDir)

	return nil
}

// collectBasicState collects namespace data and all resources from the Calico and Operator namespaces,
// as well as state on each of the main user-facing operator related resources.
func collectGlobalClusterInformation(dir string) {
	fmt.Println("Collecting kubernetes version...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect kubernetes Client and Server version",
			CmdStr:   "kubectl version -o yaml",
			FilePath: fmt.Sprintf("%s/version.txt", dir),
		},
	})

	fmt.Println("Collecting Calico resources...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect Calico clusterinformations",
			CmdStr:   "kubectl get clusterinformations -o wide",
			FilePath: fmt.Sprintf("%s/clusterinformations.txt", dir),
		},
		{
			Info:     "Collect Calico clusterinformations",
			CmdStr:   "kubectl get clusterinformations -o yaml",
			FilePath: fmt.Sprintf("%s/clusterinformations.yaml", dir),
		},
		{
			Info:     "Collect Calico felixconfigurations",
			CmdStr:   "kubectl get felixconfigurations -o wide",
			FilePath: fmt.Sprintf("%s/felixconfigurations.txt", dir),
		},
		{
			Info:     "Collect Calico felixconfigurations",
			CmdStr:   "kubectl get felixconfigurations -o yaml",
			FilePath: fmt.Sprintf("%s/felixconfigurations.yaml", dir),
		},
		{
			Info:     "Collect Calico bgppeers",
			CmdStr:   "kubectl get bgppeers -o wide",
			FilePath: fmt.Sprintf("%s/bgppeers.txt", dir),
		},
		{
			Info:     "Collect Calico bgppeers",
			CmdStr:   "kubectl get bgppeers -o yaml",
			FilePath: fmt.Sprintf("%s/bgppeers.yaml", dir),
		},
		{
			Info:     "Collect Calico bgpconfigurations",
			CmdStr:   "kubectl get bgpconfigurations -o wide",
			FilePath: fmt.Sprintf("%s/bgpconfigurations.txt", dir),
		},
		{
			Info:     "Collect Calico bgpconfigurations",
			CmdStr:   "kubectl get bgpconfigurations -o yaml",
			FilePath: fmt.Sprintf("%s/bgpconfigurations.yaml", dir),
		},
		{
			Info:     "Collect Calico ipamblocks",
			CmdStr:   "kubectl get ipamblocks -o wide",
			FilePath: fmt.Sprintf("%s/ipamblocks.txt", dir),
		},
		{
			Info:     "Collect Calico ipamblocks",
			CmdStr:   "kubectl get ipamblocks -o yaml",
			FilePath: fmt.Sprintf("%s/ipamblocks.yaml", dir),
		},
		{
			Info:     "Collect Calico blockaffinities",
			CmdStr:   "kubectl get blockaffinities -o wide",
			FilePath: fmt.Sprintf("%s/blockaffinities.txt", dir),
		},
		{
			Info:     "Collect Calico blockaffinities",
			CmdStr:   "kubectl get blockaffinities -o yaml",
			FilePath: fmt.Sprintf("%s/blockaffinities.yaml", dir),
		},
		{
			Info:     "Collect Calico ipamhandles",
			CmdStr:   "kubectl get ipamhandles -o wide",
			FilePath: fmt.Sprintf("%s/ipamhandles.txt", dir),
		},
		{
			Info:     "Collect Calico ipamhandles",
			CmdStr:   "kubectl get ipamhandles -o yaml",
			FilePath: fmt.Sprintf("%s/ipamhandles.yaml", dir),
		},
		{
			Info:     "Collect Calico tiers",
			CmdStr:   "kubectl get tiers -o wide",
			FilePath: fmt.Sprintf("%s/tiers.txt", dir),
		},
		{
			Info:     "Collect Calico tiers",
			CmdStr:   "kubectl get tiers -o yaml",
			FilePath: fmt.Sprintf("%s/tiers.yaml", dir),
		},
		{
			Info:     "Collect Calico networkpolicies",
			CmdStr:   "kubectl get networkpolicies -o wide",
			FilePath: fmt.Sprintf("%s/networkpolicies.txt", dir),
		},
		{
			Info:     "Collect Calico networkpolicies",
			CmdStr:   "kubectl get networkpolicies -o yaml",
			FilePath: fmt.Sprintf("%s/networkpolicies.yaml", dir),
		},
		{
			Info:     "Collect Calico clusterinformations",
			CmdStr:   "kubectl get clusterinformations -o wide",
			FilePath: fmt.Sprintf("%s/clusterinformations.txt", dir),
		},
		{
			Info:     "Collect Calico clusterinformations",
			CmdStr:   "kubectl get clusterinformations -o yaml",
			FilePath: fmt.Sprintf("%s/clusterinformations.yaml", dir),
		},
		{
			Info:     "Collect Calico hostendpoints",
			CmdStr:   "kubectl get hostendpoints -o wide",
			FilePath: fmt.Sprintf("%s/hostendpoints.txt", dir),
		},
		{
			Info:     "Collect Calico hostendpoints",
			CmdStr:   "kubectl get hostendpoints -o yaml",
			FilePath: fmt.Sprintf("%s/hostendpoints.yaml", dir),
		},
		{
			Info:     "Collect Calico ippools",
			CmdStr:   "kubectl get ippools -o wide",
			FilePath: fmt.Sprintf("%s/ippools.txt", dir),
		},
		{
			Info:     "Collect Calico ippools",
			CmdStr:   "kubectl get ippools -o yaml",
			FilePath: fmt.Sprintf("%s/ippools.yaml", dir),
		},
		{
			Info:     "Collect Calico licensekeys",
			CmdStr:   "kubectl get licensekeys -o wide",
			FilePath: fmt.Sprintf("%s/licensekeys.txt", dir),
		},
		{
			Info:     "Collect Calico licensekeys",
			CmdStr:   "kubectl get licensekeys -o yaml",
			FilePath: fmt.Sprintf("%s/licensekeys.yaml", dir),
		},
		{
			Info:     "Collect Calico networksets",
			CmdStr:   "kubectl get networksets -o wide",
			FilePath: fmt.Sprintf("%s/networksets.txt", dir),
		},
		{
			Info:     "Collect Calico networksets",
			CmdStr:   "kubectl get networksets -o yaml",
			FilePath: fmt.Sprintf("%s/networksets.yaml", dir),
		},
		{
			Info:     "Collect Calico globalnetworksets",
			CmdStr:   "kubectl get globalnetworksets -o wide",
			FilePath: fmt.Sprintf("%s/globalnetworksets.txt", dir),
		},
		{
			Info:     "Collect Calico globalnetworksets",
			CmdStr:   "kubectl get globalnetworksets -o yaml",
			FilePath: fmt.Sprintf("%s/globalnetworksets.yaml", dir),
		},
		{
			Info:     "Collect Calico globalnetworkpolicies",
			CmdStr:   "kubectl get globalnetworkpolicies -o wide",
			FilePath: fmt.Sprintf("%s/globalnetworkpolicies.txt", dir),
		},
		{
			Info:     "Collect Calico globalnetworkpolicies",
			CmdStr:   "kubectl get globalnetworkpolicies -o yaml",
			FilePath: fmt.Sprintf("%s/globalnetworkpolicies.yaml", dir),
		},
	})

	fmt.Println("Collecting Tigera operator details ...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect tigerastatuses",
			CmdStr:   "kubectl get tigerastatuses -o wide",
			FilePath: fmt.Sprintf("%s/tigerastatuses.txt", dir),
		},
		{
			Info:     "Collect tigerastatuses",
			CmdStr:   "kubectl get tigerastatuses -o yaml",
			FilePath: fmt.Sprintf("%s/tigerastatuses.yaml", dir),
		},
		{
			Info:     "Collect installations",
			CmdStr:   "kubectl get installations -o wide",
			FilePath: fmt.Sprintf("%s/installations.txt", dir),
		},
		{
			Info:     "Collect installations",
			CmdStr:   "kubectl get installations -o yaml",
			FilePath: fmt.Sprintf("%s/installations.yaml", dir),
		},
		{
			Info:     "Collect apiservers",
			CmdStr:   "kubectl get apiservers -o wide",
			FilePath: fmt.Sprintf("%s/apiservers.txt", dir),
		},
		{
			Info:     "Collect apiservers",
			CmdStr:   "kubectl get apiservers -o yaml",
			FilePath: fmt.Sprintf("%s/apiservers.yaml", dir),
		},
		{
			Info:     "Collect compliances",
			CmdStr:   "kubectl get compliances -o wide",
			FilePath: fmt.Sprintf("%s/compliances.txt", dir),
		},
		{
			Info:     "Collect compliances",
			CmdStr:   "kubectl get compliances -o yaml",
			FilePath: fmt.Sprintf("%s/compliances.yaml", dir),
		},
		{
			Info:     "Collect intrusiondetections",
			CmdStr:   "kubectl get intrusiondetections -o wide",
			FilePath: fmt.Sprintf("%s/intrusiondetections.txt", dir),
		},
		{
			Info:     "Collect intrusiondetections",
			CmdStr:   "kubectl get intrusiondetections -o yaml",
			FilePath: fmt.Sprintf("%s/intrusiondetections.yaml", dir),
		},
		{
			Info:     "Collect managers",
			CmdStr:   "kubectl get managers -o wide",
			FilePath: fmt.Sprintf("%s/managers.txt", dir),
		},
		{
			Info:     "Collect managers",
			CmdStr:   "kubectl get managers -o yaml",
			FilePath: fmt.Sprintf("%s/managers.yaml", dir),
		},
		{
			Info:     "Collect logcollectors",
			CmdStr:   "kubectl get logcollectors -o wide",
			FilePath: fmt.Sprintf("%s/logcollectors.txt", dir),
		},
		{
			Info:     "Collect logcollectors",
			CmdStr:   "kubectl get logcollectors -o yaml",
			FilePath: fmt.Sprintf("%s/logcollectors.yaml", dir),
		},
		{
			Info:     "Collect logstorages",
			CmdStr:   "kubectl get logstorages -o wide",
			FilePath: fmt.Sprintf("%s/logstorages.txt", dir),
		},
		{
			Info:     "Collect logstorages",
			CmdStr:   "kubectl get logstorages -o yaml",
			FilePath: fmt.Sprintf("%s/logstorages.yaml", dir),
		},
		{
			Info:     "Collect managementclusterconnections",
			CmdStr:   "kubectl get managementclusterconnections -o wide",
			FilePath: fmt.Sprintf("%s/managementclusterconnections.txt", dir),
		},
		{
			Info:     "Collect managementclusterconnections",
			CmdStr:   "kubectl get managementclusterconnections -o yaml",
			FilePath: fmt.Sprintf("%s/managementclusterconnections.yaml", dir),
		},
	})

	fmt.Println("Collecting core kubernetes resources...")

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
		// Need to exclude secrets
		{
			Info:     "Collect pods",
			CmdStr:   "kubectl get pods --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/pods.yaml", dir),
		},
		{
			Info:     "Collect pods",
			CmdStr:   "kubectl get pods --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/pods.txt", dir),
		},
		{
			Info:     "Collect deployments",
			CmdStr:   "kubectl get deployments --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/deployments.txt", dir),
		},
		{
			Info:     "Collect deployments",
			CmdStr:   "kubectl get deployments --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/deployments.yaml", dir),
		},
		{
			Info:     "Collect daemonsets",
			CmdStr:   "kubectl get daemonsets --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/daemonsets.txt", dir),
		},
		{
			Info:     "Collect daemonsets",
			CmdStr:   "kubectl get daemonsets --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/daemonsets.yaml", dir),
		},
		{
			Info:     "Collect services",
			CmdStr:   "kubectl get services --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/services.txt", dir),
		},
		{
			Info:     "Collect services",
			CmdStr:   "kubectl get services --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/services.yaml", dir),
		},
		{
			Info:     "Collect endpoints",
			CmdStr:   "kubectl get endpoints --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/endpoints.txt", dir),
		},
		{
			Info:     "Collect endpoints",
			CmdStr:   "kubectl get endpoints --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/endpoints.yaml", dir),
		},
		{
			Info:     "Collect configmaps",
			CmdStr:   "kubectl get configmaps --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/configmaps.txt", dir),
		},
		{
			Info:     "Collect configmaps",
			CmdStr:   "kubectl get configmaps --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/configmaps.yaml", dir),
		},
		{
			Info:     "Collect persistent volume claim",
			CmdStr:   "kubectl get pvc --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/pvc.txt", dir),
		},
		{
			Info:     "Collect persistent volume claim",
			CmdStr:   "kubectl get pvc --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/pvc.yaml", dir),
		},
		{
			Info:     "Collect persistent volume",
			CmdStr:   "kubectl get pv --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/pv.txt", dir),
		},
		{
			Info:     "Collect persistent volume",
			CmdStr:   "kubectl get pv --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/pv.yaml", dir),
		},
		{
			Info:     "Collect storage class",
			CmdStr:   "kubectl get sc --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/sc.txt", dir),
		},
		{
			Info:     "Collect storage class",
			CmdStr:   "kubectl get sc --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/sc.yaml", dir),
		},
		{
			Info:     "Collect all namespaces",
			CmdStr:   "kubectl get namespaces -o wide",
			FilePath: fmt.Sprintf("%s/namespaces.txt", dir),
		},
		{
			Info:     "Collect all namespaces",
			CmdStr:   "kubectl get namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/namespaces.yaml", dir),
		},
	})
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
				Info:     fmt.Sprintf("Collect describe for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl -n %s describe pods %s", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/%s-describe.txt", curNodeDir, p),
			},
			// ip diagnostics
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
			{
				Info:     fmt.Sprintf("Collect ipv6 routes for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip -6 route", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/ipv6route.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ip rule for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip rule", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/iprule.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ip route show table all for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip route show table all", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/iproute-all-table.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ip addr for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip addr", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/ipaddr.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ip link for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip link", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/iplink.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ip neigh for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip neigh", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/ipneigh.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect ipset list for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ipset list", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/ipsetlist.txt", curNodeDir),
			},
			// eBPF diagnostics
			{
				Info:     fmt.Sprintf("Collect eBPF conntrack for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf conntrack dump", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/eBPFconntrack.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect eBPF ipsets for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf ipsets dump", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/eBPFipsets.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect eBPF nat for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf nat dump", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/eBPFnat.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect eBPF routes for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf routes dump", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/eBPFroutes.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect eBPF prog for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool prog list", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/eBPFprog.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect eBPF map for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool map list", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/eBPFmap.txt", curNodeDir),
			},
			{
				Info:     fmt.Sprintf("Collect tc qdisc for node %s", p),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- tc qdisc show", common.CalicoNamespace, p),
				FilePath: fmt.Sprintf("%s/tcqdisc.txt", curNodeDir),
			},
		})

		/*
			output, err := common.ExecCmd(fmt.Sprintf(
				"kubectl exec -n %s -t %s -- bpftool map list | grep cali | awk '{print $1}'",
				common.CalicoNamespace,
				p,
			))
		*/
		output, err := common.ExecCmd(fmt.Sprintf(
			"kubectl exec -n %s -t %s -- bpftool map list | awk '{print $1}'",
			common.CalicoNamespace,
			p,
		))
		if err != nil {
			fmt.Printf("Could not retrieve eBPF maps: %s\n", err)
		} else {
			bpfMaps := strings.Split(strings.TrimSpace(output.String()), "\n")
			log.Debugf("eBPF maps: %s\n", bpfMaps)

			for _, bpfMap := range bpfMaps {
				id := strings.TrimSuffix(bpfMap, ":")
				common.ExecAllCmdsWriteToFile([]common.Cmd{
					{
						Info:     fmt.Sprintf("Collect eBPF map id %s dumps for node %s", id, p),
						CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool map dump id %s", common.CalicoNamespace, p, id),
						FilePath: fmt.Sprintf("%s/eBPFmap-%s.txt", curNodeDir, id),
					},
				})
			}
		}

		// Collect all of the CNI logs
		output, err = common.ExecCmd(fmt.Sprintf(
			"kubectl exec -n %s -t %s -- ls /var/log/calico/cni",
			common.CalicoNamespace,
			p,
		))
		if err != nil {
			fmt.Printf("Error listing the Calico CNI logs at /var/log/calico/cni/: %s\n", err)
		} else {
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
}

func collectTierNetworkPolicy(dir string) {
	fmt.Println("Collecting network policy data for each tier...")

	networkPolicyDir := fmt.Sprintf("%s/%s", dir, "network-policies")
	err := os.Mkdir(networkPolicyDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating network-policies diagnostics directory: %v\n", err)
		return
	}

	output, err := common.ExecCmd("kubectl get tiers -o=custom-columns=NAME:.metadata.name --no-headers")
	if err != nil {
		fmt.Printf("Could not retrieve tiers: %s\n", err)
		return
	}

	tiers := strings.Split(strings.TrimSpace(output.String()), "\n")
	for _, tier := range tiers {
		common.ExecCmdWriteToFile(common.Cmd{
			Info:     fmt.Sprintf("Collect network policy for %s tier", tier),
			CmdStr:   fmt.Sprintf("kubectl get networkpolicies.p -A -l projectcalico.org/tier==%s -o yaml", tier),
			FilePath: fmt.Sprintf("%s/%s-np.yaml", networkPolicyDir, tier),
		})
	}
}

func collectCalicoTigeraPodsServicesAndEndpointsDetails(dir, sinceFlag string) {
	namespaceNames := []string{"calico", "tigera"}

	for _, namespaceName := range namespaceNames {
		/*
			output, err := common.ExecCmd(fmt.Sprintf(
				`kubectl get namespace -o go-template --template='{{range .items}}{{printf "%%s\n" .metadata.name}}{{end}}' | grep '%s'`,
				namespaceName,
			))
		*/
		output, err := common.ExecCmd(fmt.Sprintf(
			`kubectl get namespace -o go-template --template='{{range .items}}{{printf "%s\n" .metadata.name}}{{end}}'`,
		))
		if err != nil {
			fmt.Printf("Could not retrieve the '%s' namespaces: %s\n", namespaceName, err)
			continue
		}

		namespaces := strings.Split(strings.TrimSpace(output.String()), "\n")
		for _, namespace := range namespaces {
			namespaceDir := fmt.Sprintf("%s/%s", dir, namespace)
			err := os.Mkdir(namespaceDir, os.ModePerm)
			if err != nil {
				fmt.Printf("Error creating '%s' namespace directory: %v\n", namespace, err)
				continue
			}
			output, err := common.ExecCmd(fmt.Sprintf(
				"kubectl get pods -n %s -l k8s-app!=calico-node -o go-template --template {{range.items}}{{.metadata.name}},{{end}}",
				namespace,
			))
			if err != nil {
				fmt.Printf("Could not retrieve '%s' namespace's pods: %s\n", namespace, err)
				continue
			}
			calicoPods := strings.TrimSuffix(output.String(), ",")
			log.Debugf("'%s' namespace's pods: %s\n", namespace, calicoPods)

			pods := strings.Split(strings.TrimSpace(calicoPods), ",")
			for _, pod := range pods {
				common.ExecAllCmdsWriteToFile([]common.Cmd{
					{
						Info:     fmt.Sprintf("Collect logs for '%s' pod", pod),
						CmdStr:   fmt.Sprintf("kubectl -n %s logs %s --since=%s", namespace, pod, sinceFlag),
						FilePath: fmt.Sprintf("%s/%s-pod.log", namespaceDir, pod),
					},
					{
						Info:     fmt.Sprintf("Collect describe for '%s' pod", pod),
						CmdStr:   fmt.Sprintf("kubectl -n %s describe pods %s", namespace, pod),
						FilePath: fmt.Sprintf("%s/%s-pod-describe.txt", namespaceDir, pod),
					},
				})
			}

			output, err = common.ExecCmd(fmt.Sprintf(
				"kubectl get services -n %s -o go-template --template {{range.items}}{{.metadata.name}},{{end}}",
				namespace,
			))
			if err != nil {
				fmt.Printf("Could not retrieve '%s' namespace's services: %s\n", namespace, err)
				continue
			}
			calicoServices := strings.TrimSuffix(output.String(), ",")
			log.Debugf("'%s' namespace's services: %s\n", namespace, calicoServices)

			services := strings.Split(strings.TrimSpace(calicoServices), ",")
			for _, service := range services {
				common.ExecAllCmdsWriteToFile([]common.Cmd{
					{
						Info:     fmt.Sprintf("Collect logs for '%s' service", service),
						CmdStr:   fmt.Sprintf("kubectl -n %s logs services/%s --since=%s", namespace, service, sinceFlag),
						FilePath: fmt.Sprintf("%s/%s-service.log", namespaceDir, service),
					},
					{
						Info:     fmt.Sprintf("Collect describe for '%s' service", service),
						CmdStr:   fmt.Sprintf("kubectl -n %s describe services/%s", namespace, service),
						FilePath: fmt.Sprintf("%s/%s-service-describe.txt", namespaceDir, service),
					},
					{
						Info:     fmt.Sprintf("Collect describe for '%s' endpoint", service),
						CmdStr:   fmt.Sprintf("kubectl -n %s describe endpoints/%s", namespace, service),
						FilePath: fmt.Sprintf("%s/%s-endpoint-describe.txt", namespaceDir, service),
					},
				})
			}
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
