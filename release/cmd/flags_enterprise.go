package main

import (
	"fmt"

	cli "github.com/urfave/cli/v2"

	"github.com/projectcalico/calico/release/internal/registry"
	"github.com/projectcalico/calico/release/internal/utils"
)

const manager = "manager"

var (
	managerFlags = []cli.Flag{managerOrgFlag, managerRepoFlag, managerBranchFlag}

	managerOrgFlag = &cli.StringFlag{
		Name:    "manager-org",
		Usage:   "The GitHub organization of the manager repository",
		EnvVars: []string{"MANAGER_ORG"},
		Value:   utils.TigeraOrg,
	}

	managerRepoFlag = &cli.StringFlag{
		Name:    "manager-repo",
		Usage:   "The GitHub repository of the manager",
		EnvVars: []string{"MANAGER_REPO"},
		Value:   manager,
	}

	managerBranchFlag = &cli.StringFlag{
		Name:    "manager-branch",
		Usage:   "The branch of the manager repository",
		EnvVars: []string{"MANAGER_BRANCH"},
		Value:   utils.DefaultBranch,
	}
)

var (
	chartVersionFlag = &cli.StringFlag{
		Name:  "chart-version",
		Usage: "The version suffix for the helm charts",
	}

	helmRegistryFlag = &cli.StringFlag{
		Name:  "helm-registry",
		Usage: "The registry to publish the helm charts (hashrelease ONLY)",
		Value: registry.HelmDevRegistry,
	}
)

var (
	publishWindowsArchiveFlag = &cli.BoolFlag{
		Name:  "publish-windows-archive",
		Usage: "Publish the Windows archive to GCS",
		Value: true,
		Action: func(ctx *cli.Context, b bool) error {
			if b && ctx.String("gcp-credentials") == "" {
				return fmt.Errorf("gcp-credentials is required when publishing windows archive")
			}
			return nil
		},
	}

	publishChartsFlag = &cli.BoolFlag{
		Name:  "publish-charts",
		Usage: "Publish the helm charts",
		Value: true,
		Action: func(ctx *cli.Context, b bool) error {
			if b && ctx.String(gcpCredentialsFileFlag.Name) == "" {
				return fmt.Errorf("gcp-credentials is required when publishing charts")
			}
			return nil
		},
	}
)

var (
	gcpCredentialsFileFlag = &cli.StringFlag{
		Name:    "gcp-credentials",
		Usage:   "The path to the GCP credentials file",
		EnvVars: []string{"GOOGLE_APPLICATION_CREDENTIALS"},
	}

	authenticateGCPFlag = &cli.BoolFlag{
		Name:  "authenticate-gcp",
		Usage: "Authenticate to GCP. Set to false if no GCP authentication is required",
		Value: true,
		Action: func(ctx *cli.Context, b bool) error {
			if b && ctx.String(gcpCredentialsFileFlag.Name) == "" {
				return fmt.Errorf("gcp-credentials is required when authenticating to GCP")
			}
			return nil
		},
	}
)
