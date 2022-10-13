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
	"k8s.io/apimachinery/pkg/labels"
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

	collectGlobalClusterInformation(dir)
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

	labelMap, err := v1.LabelSelectorAsMap(selector)
	if err != nil {
		fmt.Printf("ERROR forming pod selector: %v\n", err)
		return
	}
	selectorString := labels.SelectorFromSet(labelMap).String()

	// List pods matching the namespace and selector.
	pl, err := kubeClient.CoreV1().Pods(ns).List(context.TODO(), v1.ListOptions{LabelSelector: selectorString})
	if err != nil {
		fmt.Printf("ERROR listing pods in namespace %v matching '%v': %v\n", ns, selectorString, err)
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

func collectCalicoResource(dir string) {
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
}

func collectTigeraOperator(dir string) {
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
}

func collectKubernetesResource(dir string) {
	fmt.Println("Collecting core kubernetes resources...")

	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect nodes",
			CmdStr:   "kubectl get nodes -o wide",
			FilePath: fmt.Sprintf("%s/nodes.txt", dir),
		},
		{
			Info:     "Collect nodes",
			CmdStr:   "kubectl get nodes -o yaml",
			FilePath: fmt.Sprintf("%s/nodes.yaml", dir),
		},
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

// collectGlobalClusterInformation collects the Kubernetes resource, Calico Resource and Tigera operator details
func collectGlobalClusterInformation(dir string) {
	fmt.Println("Collecting kubernetes version...")
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     "Collect kubernetes Client and Server version",
			CmdStr:   "kubectl version -o yaml",
			FilePath: fmt.Sprintf("%s/version.txt", dir),
		},
	})

	collectCalicoResource(dir)
	collectTigeraOperator(dir)
	collectKubernetesResource(dir)
}

// func collectDiagsForPod(pod, namespace, dir /*node_name*/, sinceFlag string) {
func collectDiagsForPod(dir string, opts *diagOpts, kubeClient *kubernetes.Clientset, nodeName, namespace, podName string) {
	nodeDir := fmt.Sprintf("%s/%s", dir, nodeName)
	if _, err := os.Stat(nodeDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(nodeDir, os.ModePerm)
		if err != nil {
			fmt.Printf("error creating node diagnostics directory: %v", err)
			return
		}
	}
	fmt.Printf("Collecting diags for pod: %s\n", podName)
	common.ExecAllCmdsWriteToFile([]common.Cmd{
		{
			Info:     fmt.Sprintf("Collect logs for pod %s", podName),
			CmdStr:   fmt.Sprintf("kubectl logs --since=%s -n %s %s", opts.Since, namespace, podName),
			FilePath: fmt.Sprintf("%s/%s.log", nodeDir, podName),
		},
		{
			Info:     fmt.Sprintf("Collect describe for pod %s", podName),
			CmdStr:   fmt.Sprintf("kubectl -n %s describe pods %s", namespace, podName),
			FilePath: fmt.Sprintf("%s/%s-describe.txt", nodeDir, podName),
		},
	})

	if strings.Contains(podName, "calico-node") {
		collectCalicoNodeDiags(nodeDir, opts, kubeClient, nodeName, namespace, podName)
	}
}

func collectCalicoNodeDiags(dir string, opts *diagOpts, kubeClient *kubernetes.Clientset, nodeName, namespace, podName string) {
	fmt.Printf("Collecting diags for calico-node: %s\n", podName)

	curNodeDir := fmt.Sprintf("%s/%s", dir, podName)
	err := os.Mkdir(curNodeDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating diagnostics directory for calico-node %s: %v\n", podName, err)
		return
	}

	common.ExecAllCmdsWriteToFile([]common.Cmd{
		// ip diagnostics
		{
			Info:     fmt.Sprintf("Collect iptables for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- iptables-save -c", namespace, podName),
			FilePath: fmt.Sprintf("%s/iptables-save.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ip routes for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip route", namespace, podName),
			FilePath: fmt.Sprintf("%s/iproute.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ipv6 routes for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip -6 route", namespace, podName),
			FilePath: fmt.Sprintf("%s/ipv6route.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ip rule for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip rule", namespace, podName),
			FilePath: fmt.Sprintf("%s/iprule.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ip route show table all for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip route show table all", namespace, podName),
			FilePath: fmt.Sprintf("%s/iproute-all-table.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ip addr for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip addr", namespace, podName),
			FilePath: fmt.Sprintf("%s/ipaddr.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ip link for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip link", namespace, podName),
			FilePath: fmt.Sprintf("%s/iplink.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ip neigh for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ip neigh", namespace, podName),
			FilePath: fmt.Sprintf("%s/ipneigh.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect ipset list for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- ipset list", namespace, podName),
			FilePath: fmt.Sprintf("%s/ipsetlist.txt", curNodeDir),
		},
		// eBPF diagnostics
		{
			Info:     fmt.Sprintf("Collect eBPF conntrack for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf conntrack dump", namespace, podName),
			FilePath: fmt.Sprintf("%s/eBPFconntrack.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect eBPF ipsets for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf ipsets dump", namespace, podName),
			FilePath: fmt.Sprintf("%s/eBPFipsets.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect eBPF nat for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf nat dump", namespace, podName),
			FilePath: fmt.Sprintf("%s/eBPFnat.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect eBPF routes for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- calico-node -bpf routes dump", namespace, podName),
			FilePath: fmt.Sprintf("%s/eBPFroutes.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect eBPF prog for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool prog list", namespace, podName),
			FilePath: fmt.Sprintf("%s/eBPFprog.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect eBPF map for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool map list", namespace, podName),
			FilePath: fmt.Sprintf("%s/eBPFmap.txt", curNodeDir),
		},
		{
			Info:     fmt.Sprintf("Collect tc qdisc for node %s", podName),
			CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- tc qdisc show", namespace, podName),
			FilePath: fmt.Sprintf("%s/tcqdisc.txt", curNodeDir),
		},
	})

	output, err := common.ExecCmd(fmt.Sprintf(
		"kubectl exec -n %s -t %s -- bpftool map list",
		namespace,
		podName,
	))
	if err != nil {
		fmt.Printf("Could not retrieve eBPF maps: %s\n", err)
	} else {
		bpfMaps := strings.Split(strings.TrimSpace(output.String()), "\n")
		log.Debugf("eBPF maps: %s\n", bpfMaps)

		for _, bpfMap := range bpfMaps {
			if strings.Contains(bpfMap, "cali") {
				id := strings.Split(bpfMap, ":")
				common.ExecAllCmdsWriteToFile([]common.Cmd{
					{
						Info:     fmt.Sprintf("Collect eBPF map id %s dumps for node %s", id[0], podName),
						CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- bpftool map dump id %s", namespace, podName, id[0]),
						FilePath: fmt.Sprintf("%s/eBPFmap-%s.txt", curNodeDir, id[0]),
					},
				})
			}
		}
	}

	// Collect all of the CNI logs
	output, err = common.ExecCmd(fmt.Sprintf(
		"kubectl exec -n %s -t %s -- ls /var/log/calico/cni",
		namespace,
		podName,
	))
	if err != nil {
		fmt.Printf("Error listing the Calico CNI logs at /var/log/calico/cni/: %s\n", err)
	} else {
		cniLogFiles := strings.Split(strings.TrimSpace(output.String()), "\n")
		for _, logFile := range cniLogFiles {
			common.ExecCmdWriteToFile(common.Cmd{
				Info:     fmt.Sprintf("Collect CNI log %s for the node %s", logFile, podName),
				CmdStr:   fmt.Sprintf("kubectl exec -n %s -t %s -- cat /var/log/calico/cni/%s", namespace, podName, logFile),
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
