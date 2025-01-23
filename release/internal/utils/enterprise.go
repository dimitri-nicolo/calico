package utils

import (
	"path/filepath"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/projectcalico/calico/release/internal/command"
)

const (
	CalicoEnterprise = "calico enterprise"
)

func DetermineCalicoVersion(repoRoot string) (string, error) {
	args := []string{"-Po", `CALICO_VERSION=\K(.*)`, "Makefile"}
	out, err := command.RunInDir(filepath.Join(repoRoot, "node"), "grep", args)
	if err != nil {
		return "", err
	}
	return out, nil
}

func EnterpriseProductName() string {
	return cases.Title(language.English).String(CalicoEnterprise)
}
