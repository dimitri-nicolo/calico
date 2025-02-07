package main

import (
	"fmt"

	cli "github.com/urfave/cli/v2"

	"github.com/projectcalico/calico/release/internal/registry"
	"github.com/projectcalico/calico/release/internal/utils"
)

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
		Value:   utils.TigeraManager,
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
		Name:    "chart-version",
		Usage:   "The version suffix for the helm charts",
		EnvVars: []string{"HELM_RELEASE", "CHART_VERSION"},
	}

	helmRegistryFlag = &cli.StringFlag{
		Name:    "helm-registry",
		Usage:   "The registry to publish the helm charts (hashrelease ONLY)",
		EnvVars: []string{"HELM_REGISTRY"},
		Value:   registry.HelmDevRegistry,
	}
)

var (
	publishWindowsArchiveFlag = &cli.BoolFlag{
		Name:    "publish-windows-archive",
		Usage:   "Publish the Windows archive to GCS",
		EnvVars: []string{"PUBLISH_WINDOWS_ARCHIVE"},
		Value:   true,
	}

	publishChartsFlag = &cli.BoolFlag{
		Name:    "publish-charts",
		Usage:   "Publish the helm charts",
		EnvVars: []string{"PUBLISH_CHARTS"},
		Value:   true,
	}

	publishToS3Flag = &cli.BoolFlag{
		Name:    "publish-to-s3",
		Usage:   "Publish the release to S3",
		EnvVars: []string{"PUBLISH_TO_S3"},
		Value:   true,
	}
)

var skipRPMsFlag = &cli.BoolFlag{
	Name:    "skip-rpms",
	Usage:   "Skip building or publishing RPMs",
	EnvVars: []string{"SKIP_RPMS"},
	Value:   false,
}

var hashreleaseNameFlag = &cli.StringFlag{
	Name:     "hashrelease",
	Usage:    "The name of the hashrelease the release is based on",
	EnvVars:  []string{"HASHRELEASE"},
	Required: true,
}

var operatorVersionFlag = &cli.StringFlag{
	Name:     "operator-version",
	Usage:    "The version of operator used in the release",
	EnvVars:  []string{"OPERATOR_VERSION"},
	Required: true,
}

var releaseVersionFlag = &cli.StringFlag{
	Name:     "version",
	Usage:    "The version of Enterprise to release",
	EnvVars:  []string{"RELEASE_VERSION"},
	Required: true,
}

var confirmFlag = &cli.BoolFlag{
	Name:    "confirm",
	Usage:   "Run the release. If not set, the release will be a dry-run",
	EnvVars: []string{"CONFIRM"},
	Value:   false,
}

var awsProfileFlag = &cli.StringFlag{
	Name:     "aws-profile",
	Usage:    "The AWS profile to use",
	EnvVars:  []string{"AWS_PROFILE"},
	Value:    "default",
	Required: true,
}

var skipReleaseVersionCheckFlag = &cli.BoolFlag{
	Name:    "skip-version-check",
	Usage:   "Skip checking the release version matches the determined version",
	EnvVars: []string{"SKIP_VERSION_CHECK"},
	Value:   false,
	Action: func(ctx *cli.Context, b bool) error {
		if ctx.Bool(skipValidationFlag.Name) && !b {
			return fmt.Errorf("must skip branch check if %s is set", skipValidationFlag.Name)
		}
		return nil
	},
}
