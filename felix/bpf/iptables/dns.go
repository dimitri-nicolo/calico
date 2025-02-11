// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package iptables

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/bpf"
	"github.com/projectcalico/calico/felix/bpf/bpfdefs"
	"github.com/projectcalico/calico/felix/bpf/libbpf"
)

func LoadIPSetsPolicyProgram(ipSetID uint64, bpfLogLevel string, ipver uint8) error {
	if _, err := os.Stat(bpfdefs.IPSetMatchProg(ipSetID, ipver, bpfLogLevel)); err == nil {
		return nil
	}
	logLevel := strings.ToLower(bpfLogLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	progFileName := fmt.Sprintf("ipt_match_ipset_%s_co-re_v%d.o", logLevel, ipver)
	preCompiledBinary := path.Join(bpfdefs.ObjectDir, progFileName)

	obj, err := bpf.LoadObject(preCompiledBinary, &libbpf.IPTDnsGlobalData{IPSetID: ipSetID})
	if err != nil {
		return fmt.Errorf("error loading ipsets policy program for dns %w", err)
	}
	defer obj.Close()
	err = obj.PinPrograms(bpfdefs.IPSetMatchPinPath(ipSetID, ipver, logLevel))
	if err != nil {
		return fmt.Errorf("error pinning program %v", err)
	}
	return nil
}

func RemoveIPSetMatchProgram(ipSetID uint64, ipver uint8, logLevel string) error {
	progPath := bpfdefs.IPSetMatchProg(ipSetID, ipver, logLevel)
	err := os.RemoveAll(progPath)
	if err != nil {
		return fmt.Errorf("error deleting ipset match program at %s : %w", progPath, err)
	}
	return nil
}

func LoadDNSParserBPFProgram(bpfLogLevel string, dnsMapsToPin []string) error {
	if _, err := os.Stat(bpfdefs.IPTDNSParserProg(bpfLogLevel)); err == nil {
		return nil
	}

	logLevel := strings.ToLower(bpfLogLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	pinPath := path.Join(bpfdefs.DnsObjDir, logLevel)
	fileToLoad := "ipt_parse_dns_" + logLevel + "_co-re.o"
	preCompiledBinary := path.Join(bpfdefs.ObjectDir, fileToLoad)

	err := CreateDNSObjPinDir(logLevel)
	if err != nil {
		return fmt.Errorf("error creating dns obj directory %w", err)
	}

	var obj *libbpf.Obj

	if log.GetLevel() < log.DebugLevel {
		// We allocate and pass the buffer but we ignore it to suppress libbpf output
		// as on some older system it is perfectly ok for the oad to fail. In that
		// case DNS policy mode falls back from Inline to DelayDenied.
		obj, err = bpf.LoadObjectWithLogBuffer(preCompiledBinary, nil, make([]byte, 1<<20), dnsMapsToPin...)
		if err != nil {
			return fmt.Errorf("error loading bpf dns parser program for iptables %w", err)
		}
	} else {
		obj, err = bpf.LoadObject(preCompiledBinary, nil, dnsMapsToPin...)
		if err != nil {
			return fmt.Errorf("error loading bpf dns parser program for iptables %w", err)
		}
	}
	defer obj.Close()
	err = obj.PinPrograms(pinPath)
	if err != nil {
		return fmt.Errorf("error pinning program %v", err)
	}
	return nil
}

func CreateDNSObjPinDir(logLevel string) error {
	pinPath := path.Join(bpfdefs.DnsObjDir, logLevel)
	_, err := os.Stat(pinPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(pinPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

func Cleanup() {
	os.RemoveAll(bpfdefs.DnsObjDir)
}

func CleanupOld(logLevel string) error {
	pinDir := "debug"
	if strings.ToLower(logLevel) == "debug" {
		pinDir = "no_log"
	}
	os.RemoveAll(path.Join(bpfdefs.DnsObjDir, pinDir))
	progsToRemove := []string{}
	err := filepath.Walk(bpfdefs.IPTDnsPinPath(logLevel), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), "ipset_matcher") {
			progsToRemove = append(progsToRemove, path)
		}
		return nil
	})
	for _, prog := range progsToRemove {
		os.RemoveAll(prog)
	}
	return err
}
