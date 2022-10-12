// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package cluster

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/clientmgr"
	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/common"
	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/constants"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

const (
	archiveName   = "calico-diagnostics.tar.gz"
	directoryName = "calico-diagnostics"
)

type diagOpts struct {
	Cluster              bool // Only needed for Bind to work.
	Diags                bool // Only needed for Bind to work.
	Config               string
	Since                string
	MaxLogs              int
	FocusNodes           string
	AllowVersionMismatch bool
}

// Diags executes a series of kubectl exec commands to retrieve logs and resource information
// for the configured cluster.
func Diags(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl cluster diags [options]

Options:
  -h --help                    Show this screen.
     --since=<SINCE>           Only collect logs newer than provided relative duration, in seconds (s), minutes (m) or hours (h).
     --max-logs=<MAXLOGS>      Only collect up to this number of logs, for each kind of Calico component. [default: 5]
     --focus-nodes=<NODES>     Comma-separated list of nodes from which we should try first to collect logs.
  -c --config=<CONFIG>         Path to connection configuration file. [default: ` + constants.DefaultConfigPath + `]
     --allow-version-mismatch  Allow client and cluster versions mismatch.

Description:
  The cluster diags command collects a snapshot of diagnostic info and logs related to Calico for the given cluster.
`
	parsedArgs, err := docopt.ParseArgs(doc, args, "")
	if err != nil {
		return fmt.Errorf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.", strings.Join(args, " "))
	}
	fmt.Printf("DEBUG: parsedArgs=%v\n", parsedArgs)
	if len(parsedArgs) == 0 {
		return nil
	}

	var opts diagOpts
	err = parsedArgs.Bind(&opts)
	if err != nil {
		return fmt.Errorf("error understanding options: %w", err)
	}
	fmt.Printf("DEBUG: opts=%#v\n", opts)

	// Default --since to "0s", which kubectl understands as meaning all logs.
	if opts.Since == "" {
		opts.Since = "0s"
	}
	fmt.Printf("DEBUG: opts=%#v\n", opts)

	err = common.CheckVersionMismatch(parsedArgs["--config"], parsedArgs["--allow-version-mismatch"])
	if err != nil {
		return err
	}

	return collectDiags(&opts)
}

func collectDiags(opts *diagOpts) error {
	// Ensure since value is valid with proper time unit
	argutils.ValidateSinceDuration(opts.Since)

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

	collectBasicState(dir, opts.Since)
	collectOperatorDiags(dir, opts.Since)
	collectNodeDiags(dir, opts.Since)
	collectCalicoTelemetry(dir, opts.Since)
	collectBGPStatus(dir)
	err = collectSelectedNodeLogs(dir, opts)
	if err != nil {
		fmt.Printf("ERROR collecting logs from selected nodes: %v\n", err)
	}
	createArchive(rootDir)

	return nil
}

func collectSelectedNodeLogs(dir string, opts *diagOpts) error {
	// Create Kubernetes client from config or env vars.
	kubeClient, _, _, err := clientmgr.GetClients(opts.Config)
	if err != nil {
		return fmt.Errorf("error creating clients: %w", err)
	}
	if kubeClient == nil {
		return errors.New("can't create Kubernetes client on etcd datastore")
	}

	// If --focus-nodes is specified, put those node names at the start of the node list.
	nodeList := strings.Split(opts.FocusNodes, ",")

	// Keep track of nodes already in the list.
	nodesAlreadyListed := set.New[string]()
	for _, nodeName := range nodeList {
		nodesAlreadyListed.Add(nodeName)
	}

	// Add all other nodes into the list.
	nl, err := kubeClient.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		fmt.Printf("ERROR listing all nodes in cluster: %v\n", err)
		// Continue because we can still use the --focus-nodes, if specified.
	} else {
		for _, node := range nl.Items {
			if !nodesAlreadyListed.Contains(node.Name) {
				nodeList = append(nodeList, node.Name)
			}
		}
	}

	// Iterate through all Calico/Tigera namespaces.
	nsl, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		// Fatal, can't identify our namespaces.
		return fmt.Errorf("error listing namespaces: %w", err)
	}
	for _, ns := range nsl.Items {
		if !(strings.Contains(ns.Name, "calico") || strings.Contains(ns.Name, "tigera")) {
			continue
		}

		fmt.Printf("Collecting detailed diags for namespace %v...\n", ns.Name)

		// Iterate through DaemonSets in this namespace.
		dsl, err := kubeClient.AppsV1().DaemonSets(ns.Name).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			fmt.Printf("ERROR listing DaemonSets in namespace %v: %v\n", ns.Name, err)
			// Continue because deployments or other namespaces might work.
		} else {
			for _, ds := range dsl.Items {
				collectDiagsForSelectedPods(dir, opts, kubeClient, nodeList, ns.Name, ds.Spec.Selector)
			}
		}

		// Iterate through Deployments in this namespace.
		dl, err := kubeClient.AppsV1().Deployments(ns.Name).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			fmt.Printf("ERROR listing Deployments in namespace %v: %v\n", ns.Name, err)
			// Continue because other namespaces might work.
		} else {
			for _, d := range dl.Items {
				collectDiagsForSelectedPods(dir, opts, kubeClient, nodeList, ns.Name, d.Spec.Selector)
			}
		}
	}
	return nil
}

func collectDiagsForSelectedPods(dir string, opts *diagOpts, kubeClient *kubernetes.Clientset, nodeList []string, ns string, selector *v1.LabelSelector) {
	// List pods matching the namespace and selector.
	pl, err := kubeClient.CoreV1().Pods(ns).List(context.TODO(), v1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		fmt.Printf("ERROR listing pods in namespace %v matching '%v': %v\n", ns, selector.String(), err)
		return
	}

	// Map the pod names against their node names.
	podNamesByNode := map[string][]string{}
	for _, p := range pl.Items {
		podNamesByNode[p.Spec.NodeName] = append(podNamesByNode[p.Spec.NodeName], p.Name)
	}

	nextNodeIndex := 0
	for logsWanted := opts.MaxLogs; logsWanted > 0; {
		// Get the next node name to look at.
		if nextNodeIndex >= len(nodeList) {
			// There are no more nodes we can look at.
			break
		}
		nodeName := nodeList[nextNodeIndex]
		nextNodeIndex++

		for _, podName := range podNamesByNode[nodeName] {
			fmt.Printf("Collecting detailed diags for pod %v in namespace %v on node %v...\n", podName, ns, nodeName)
			collectDiagsForPod(dir, opts, kubeClient, nodeName, ns, podName)
			logsWanted--
			if logsWanted <= 0 {
				break
			}
		}
	}
}

func collectDiagsForPod(dir string, opts *diagOpts, kubeClient *kubernetes.Clientset, nodeName, ns, podName string) {
	// Do kubectl logs, with --since and opts.Since.

	// Do kubectl describe.

	// If the pod is a calico-node pod, get dataplane and eBPF diags.
}

// collectBasicState collects namespace data and all resources from the Calico and Operator namespaces,
// as well as state on each of the main user-facing operator related resources.
func collectBasicState(dir, sinceFlag string) {
	fmt.Println("Collecting basic cluster state ...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect kubernetes Client and Server version",
			CmdStr:   "kubectl version --output=yaml",
			FilePath: fmt.Sprintf("%s/version.txt", dir),
		},
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
		{
			Info:     "Collect Tigera installations",
			CmdStr:   "kubectl get Installation.operator.tigera.io -o yaml",
			FilePath: fmt.Sprintf("%s/tigera-installation.txt", dir),
		},
		{
			Info:     "Collect cluster information",
			CmdStr:   "kubectl cluster-info dump",
			FilePath: fmt.Sprintf("%s/cluster-info.txt", dir),
		},
		{
			Info:     "Collect tigera version",
			CmdStr:   "kubectl get clusterinformations.projectcalico.org default -o yaml",
			FilePath: fmt.Sprintf("%s/tigera-version.txt", dir),
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

		output, err := common.ExecCmd(fmt.Sprintf(
			"kubectl exec -n %s -t %s -- bpftool map list | grep 'cali' | awk '{print $1}'",
			common.CalicoNamespace,
			p,
		))
		if err != nil {
			fmt.Printf("Could not retrieve eBPF maps: %s\n", err)
			// Skip to the next node
			continue
		}
		bpfMaps := strings.TrimSuffix(output.String(), ":")
		log.Debugf("eBPF maps: %s\n", bpfMaps)

		ids := strings.Split(strings.TrimSpace(bpfMaps), "\n")
		for _, id := range ids {
			common.ExecAllCmdsWriteToFile([]common.Cmd{
				{
					Info:     fmt.Sprintf("Collect eBPF map id %s dumps for node %s", id, p),
					CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool map dump id %s", common.CalicoNamespace, p, id),
					FilePath: fmt.Sprintf("%s/eBPFmap-%s.txt", curNodeDir, id),
				},
			})
		}

		// Collect all of the CNI logs
		output, err = common.ExecCmd(fmt.Sprintf(
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

func collectCalicoTigeraDescribePodsAndLogs(dir, sinceFlag string) {
	namespaceNames := [...]string{"calico", "tigera"}

	for _, namespaceName := range namespaceNames {
		output, err := common.ExecCmd(fmt.Sprintf(
			"kubectl get namespace | grep '%s' | awk '{print $1}'",
			namespaceName,
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

func collectCalicoTelemetry(dir, sinceFlag string) {
	fmt.Println("Collecting calico telemetry data...")

	telemetryDir := fmt.Sprintf("%s/%s", dir, "telemetry")
	err := os.Mkdir(telemetryDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating calico telemetry diagnostics directory: %v\n", err)
		return
	}

	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect pods statistics",
			CmdStr:   "kubectl get pods --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/pods.yaml", telemetryDir),
		},
		{
			Info:     "Collect pods statistics",
			CmdStr:   "kubectl get pods --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/pods.txt", telemetryDir),
		},
		{
			Info:     "Collect deployments statistics",
			CmdStr:   "kubectl get deployments --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/deployments.yaml", telemetryDir),
		},
		{
			Info:     "Collect deployments statistics",
			CmdStr:   "kubectl get deployments --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/deployments.txt", telemetryDir),
		},
		{
			Info:     "Collect daemonsets statistics",
			CmdStr:   "kubectl get daemonsets --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/daemonsets.yaml", telemetryDir),
		},
		{
			Info:     "Collect daemonsets statistics",
			CmdStr:   "kubectl get daemonsets --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/daemonsets.txt", telemetryDir),
		},
		{
			Info:     "Collect services statistics",
			CmdStr:   "kubectl get services --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/services.yaml", telemetryDir),
		},
		{
			Info:     "Collect services statistics",
			CmdStr:   "kubectl get services --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/services.txt", telemetryDir),
		},
		{
			Info:     "Collect endpoints statistics",
			CmdStr:   "kubectl get endpoints --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/endpoints.yaml", telemetryDir),
		},
		{
			Info:     "Collect endpoints statistics",
			CmdStr:   "kubectl get endpoints --all-namespaces -o wide",
			FilePath: fmt.Sprintf("%s/endpoints.txt", telemetryDir),
		},
		// CMs may contain confidential info. Let us capture only Tigera specific CM if needed.
		{
			Info:     "Collect configmaps statistics",
			CmdStr:   "kubectl get configmaps --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/configmaps.yaml", telemetryDir),
		},
		{
			Info:     "Collect configmaps statistics",
			CmdStr:   "kubectl get configmaps --all-namespaces",
			FilePath: fmt.Sprintf("%s/configmaps.txt", telemetryDir),
		},
		{
			Info:     "Collect ipamblocks yaml",
			CmdStr:   "kubectl get ipamblocks -o yaml",
			FilePath: fmt.Sprintf("%s/ipamblocks.yaml", telemetryDir),
		},
		{
			Info:     "Collect blockaffinities yaml",
			CmdStr:   "kubectl get blockaffinities -o yaml",
			FilePath: fmt.Sprintf("%s/blockaffinities.yaml", telemetryDir),
		},
		{
			Info:     "Collect ipamhandles yaml",
			CmdStr:   "kubectl get ipamhandles -o yaml",
			FilePath: fmt.Sprintf("%s/ipamhandles.yaml", telemetryDir),
		},
		{
			Info:     "Collect tier information",
			CmdStr:   "kubectl get tier.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/tiers.yaml", telemetryDir),
		},
		{
			Info:     "Collect global network policies",
			CmdStr:   "kubectl get globalnetworkpolicies.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/global-network-policies.yaml", telemetryDir),
		},
		{
			Info:     "Collect hostendpoints information",
			CmdStr:   "kubectl get hostendpoints.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/hostendpoints.yaml", telemetryDir),
		},
		{
			Info:     "Collect ippool information",
			CmdStr:   "kubectl get ippools.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/ippools.yaml", telemetryDir),
		},
		{
			Info:     "Collect licensekey data",
			CmdStr:   "kubectl get licensekeys.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/licensekeys.yaml", telemetryDir),
		},
		{
			Info:     "Collect networksets data",
			CmdStr:   "kubectl get networksets.projectcalico.org --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/networksets.yaml", telemetryDir),
		},
		{
			Info:     "Collect global networksets data",
			CmdStr:   "kubectl get globalnetworksets.crd.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/global-networksets.yaml", telemetryDir),
		},
		{
			Info:     "Collect persistent volume claim status",
			CmdStr:   "kubectl get pvc --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/pvc-status.txt", telemetryDir),
		},
		{
			Info:     "Collect persistent volume status",
			CmdStr:   "kubectl get pv --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/pv-status.txt", telemetryDir),
		},
		{
			Info:     "Collect storage class status",
			CmdStr:   "kubectl get sc --all-namespaces -o yaml",
			FilePath: fmt.Sprintf("%s/sc-status.txt", telemetryDir),
		},
	})
	collectTierNetworkPolicy(telemetryDir)
	collectCalicoTigeraDescribePodsAndLogs(telemetryDir, sinceFlag)
}

func collectBGPStatus(dir string) {
	fmt.Println("Collecting BGP status...")

	bgpDir := fmt.Sprintf("%s/%s", dir, "bgpstatus")
	err := os.Mkdir(bgpDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating bgp status diagnostics directory: %v\n", err)
		return
	}

	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect BGP configuration status",
			CmdStr:   "kubectl get bgpconfigurations.crd.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/bgpconfigurations-yaml.yaml", bgpDir),
		},
		{
			Info:     "Collect BGP peers status",
			CmdStr:   "kubectl get bgppeers.crd.projectcalico.org -o yaml",
			FilePath: fmt.Sprintf("%s/bgppeers-yaml.yaml", bgpDir),
		},
	})
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
