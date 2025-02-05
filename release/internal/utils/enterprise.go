package utils

import (
	"path/filepath"

	"github.com/projectcalico/calico/release/internal/command"
)

const (
	CalicoEnterprise = "calico enterprise"

	EnterpriseProductName = "Calico Enterprise"

	TigeraManager = "manager"

	// EnterpriseProductCode is the code for calico enterprise.
	EnterpriseProductCode = "cnx"
)

func DetermineCalicoVersion(repoRoot string) (string, error) {
	args := []string{"-Po", `CALICO_VERSION=\K(.*)`, "Makefile"}
	out, err := command.RunInDir(filepath.Join(repoRoot, "node"), "grep", args)
	if err != nil {
		return "", err
	}
	return out, nil
}
