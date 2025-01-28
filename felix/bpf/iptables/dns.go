// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package iptables

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/projectcalico/calico/felix/bpf"
	"github.com/projectcalico/calico/felix/bpf/bpfdefs"
	"github.com/projectcalico/calico/felix/bpf/libbpf"
)

func progFileName(logLevel string, ipver uint8) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	return fmt.Sprintf("ipt_match_ipset_%s_co-re_v%d.o", logLevel, ipver)
}

func LoadIPSetsPolicyProgram(ipSetID uint64, bpfLogLevel string, ipver uint8) error {
	if _, err := os.Stat(bpfdefs.IPSetMatchProg(ipSetID, ipver)); err == nil {
		return nil
	}

	preCompiledBinary := path.Join(bpfdefs.ObjectDir, progFileName(bpfLogLevel, ipver))
	obj, err := bpf.LoadObject(preCompiledBinary, &libbpf.IPTDnsGlobalData{IPSetID: ipSetID})
	if err != nil {
		return fmt.Errorf("error loading ipsets policy program for dns %w", err)
	}
	defer obj.Close()
	err = obj.PinPrograms(bpfdefs.IPSetMatchPinPath(ipSetID, ipver))
	if err != nil {
		return fmt.Errorf("error pinning program %v", err)
	}
	return nil
}

func RemoveIPSetMatchProgram(ipSetID uint64, ipver uint8) error {
	progPath := bpfdefs.IPSetMatchProg(ipSetID, ipver)
	err := os.RemoveAll(progPath)
	if err != nil {
		return fmt.Errorf("error deleting ipset match program at %s : %w", progPath, err)
	}
	return nil
}

func LoadDNSParserBPFProgram(bpfLogLevel string, dnsMapsToPin []string) error {
	if _, err := os.Stat(bpfdefs.DnsParserPinPath); err == nil {
		return nil
	}
	logLevel := strings.ToLower(bpfLogLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	fileToLoad := "ipt_parse_dns_" + logLevel + "_co-re.o"
	preCompiledBinary := path.Join(bpfdefs.ObjectDir, fileToLoad)

	err := CreateDNSObjPinDir()
	if err != nil {
		return fmt.Errorf("error creating dns obj directory %w", err)
	}

	obj, err := bpf.LoadObject(preCompiledBinary, nil, dnsMapsToPin...)
	if err != nil {
		return fmt.Errorf("error loading bpf dns parser program for iptables %w", err)
	}
	defer obj.Close()
	err = obj.PinPrograms(bpfdefs.DnsObjDir)
	if err != nil {
		return fmt.Errorf("error pinning program %v", err)
	}
	return nil
}

func CreateDNSObjPinDir() error {
	_, err := os.Stat(bpfdefs.DnsObjDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(bpfdefs.DnsObjDir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func Cleanup() {
	os.RemoveAll(bpfdefs.DnsObjDir)
}
