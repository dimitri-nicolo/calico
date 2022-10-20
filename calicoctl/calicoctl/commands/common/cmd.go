// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package common

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CmdExecutor will execute a command and return its output and its error
type CmdExecutor interface {
	Execute(cmdStr string) (string, error)
}

// kubectlCmd is a kubectl wrapper for any query that will be executed
type kubectlCmd struct {
	kubeConfig string
}

// NewKubectlCmd return a CmdExecutor that uses kubectl
func NewKubectlCmd(kubeConfigPath string) *kubectlCmd {
	return &kubectlCmd{kubeConfig: kubeConfigPath}
}

func (k *kubectlCmd) Execute(cmdStr string) (string, error) {
	var out, err = ExecCmd(strings.Replace(cmdStr, "kubectl", fmt.Sprintf("kubectl --kubeconfig %s", k.kubeConfig), 1))
	if out != nil {
		return out.String(), err
	}
	return "", err
}

// ExecCmd is a convenience function that wraps exec.Command.
func ExecCmd(cmdStr string) (*bytes.Buffer, error) {
	var result bytes.Buffer

	parts := strings.Fields(cmdStr)
	log.Debugf("cmd tokens: [%+v]", parts)

	if len(parts) == 0 {
		return nil, fmt.Errorf("no command to execute")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = &result

	log.Debugf("Executing command: %+v ... ", cmd)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command execution failed: %s", err)
	}
	log.Debugln("Completed successfully.")
	return &result, nil
}

// KubectlExists determines whether tar binary exists on the path.
func KubectlExists() error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("unable to locate kubectl in PATH")
	}
	return nil
}

// Cmd is a struct to hold a command to execute, info description to print and a
// filepath location for where output should be written to.
type Cmd struct {
	Info     string
	CmdStr   string
	FilePath string
}

// ExecCmdWriteToFile executes the provided command c and outputs the result to a
// file with the given filepath.
func ExecCmdWriteToFile(c Cmd) {

	if c.Info != "" {
		fmt.Println(c.Info)
	}

	// Create the containing directory, if needed.
	dir := filepath.Dir(c.FilePath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating directory for %v: %v\n", c.FilePath, err)
		return
	}

	parts := strings.Fields(c.CmdStr)
	log.Debugf("cmd tokens: [%+v]", parts)

	log.Debugf("Executing command: %+v ... ", c.CmdStr)
	content, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to run command: %s\nError: %s\n", c.CmdStr, string(content))
	}

	// This is for the commands we want to run but don't want to save the output
	// or for commands that don't produce any output to stdout
	if c.FilePath == "" {
		log.Debugln("Command executed successfully, skipping writing output (no filepath specified)")
		return
	}

	if err := ioutil.WriteFile(c.FilePath, content, 0644); err != nil {
		log.Errorf("Error writing diags to file: %s\n", err)
	}
	log.Debugf("Command executed successfully and outputted to %s", c.FilePath)
}

// ExecAllCmdsWriteToFile iterates through the provided list of Cmd objects and attempts
// to execute each one.
func ExecAllCmdsWriteToFile(cmds []Cmd) {
	for _, c := range cmds {
		ExecCmdWriteToFile(c)
	}
}
