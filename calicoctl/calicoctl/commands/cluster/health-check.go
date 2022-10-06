// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package cluster

import (
	"fmt"
	"strings"

	"github.com/docopt/docopt-go"

	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/common"
	"github.com/projectcalico/calico/calicoctl/calicoctl/commands/constants"
)

func HealthCheck(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl cluster diags [--config=<CONFIG>] [--allow-version-mismatch]

Options:
  -h --help                    Show this screen.
  -c --config=<CONFIG>         Path to the file containing connection configuration in
                               YAML or JSON format.
                               [default: ` + constants.DefaultConfigPath + `]
     --allow-version-mismatch  Allow client and cluster versions mismatch.

Description:
  The cluster health-check command verify Calico Enterprise installation and configuration for the given cluster.
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

	return verifyHealthCheck()
}

func verifyHealthCheck() error {
	// Ensure kubectl command is available
	if err := common.KubectlExists(); err != nil {
		return fmt.Errorf("missing dependency: %s", err)
	}

	fmt.Println("==== Begin health check. ====")

	checkOperatorBased()
	checkKubernetesVersion()
	checkClusterPodCIDR()
	checkTigeraVersion()
	checkTigeraLicense()
	checkTigeraStatus()
	checkTigeraStatus()
	checkElasticSearchPVCStatus()
	checkTigeraNamespaces()
	checkAPIServerStatus()
	checkKubeAPIServerStatus()
	checkCalicoPods()
	checkTigeraPods()
	checkTier()

	return nil
}

// check_operator_based
func checkOperatorBased() {
}

// update_calico_config_check
// NA

// check_kube_config
// NA

// check_kubeVersion
func checkKubernetesVersion() {
}

// check_cluster_pod_cidr
func checkClusterPodCIDR() {
}

// check_tigera_version
func checkTigeraVersion() {
}

// check_tigera_license
func checkTigeraLicense() {
}

// check_tigerastatus
func checkTigeraStatus() {
}

// check_es_pvc_status
func checkElasticSearchPVCStatus() {
}

// check_tigera_namespaces
func checkTigeraNamespaces() {
}

// check_apiserver_status
func checkAPIServerStatus() {
}

// check_kubeapiserver_status
func checkKubeAPIServerStatus() {
}

// check_calico_pods
func checkCalicoPods() {
}

// check_tigera_pods
func checkTigeraPods() {
}

// check_tier
func checkTier() {
}
