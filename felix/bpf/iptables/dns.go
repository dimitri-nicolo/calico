package iptables

import (
	"fmt"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/bpf"
	"github.com/projectcalico/calico/felix/bpf/bpfdefs"
	"github.com/projectcalico/calico/felix/bpf/libbpf"
)

func progFileName(logLevel string, ipver int) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	return fmt.Sprintf("ipt_match_ipset_%s_co-re_v%d.o", logLevel, ipver)
}

func LoadIPSetsPolicyProgram(ipSetID uint64, bpfLogLevel string, ipver int) error {
	preCompiledBinary := path.Join(bpfdefs.ObjectDir, progFileName(bpfLogLevel, ipver))
	obj, err := bpf.LoadObject(preCompiledBinary, &libbpf.IPTDnsGlobalData{IPSetID: ipSetID})
	if err != nil {
		return fmt.Errorf("error loading ipsets policy program for dns %w", err)
	}
	defer obj.Close()
	pinPath := path.Join(bpfdefs.DnsObjDir, fmt.Sprintf("%d_v%d", ipSetID, ipver))
	err = obj.PinPrograms(pinPath)
	if err != nil {
		return fmt.Errorf("error pinning program %v", err)
	}
	return nil
}

func LoadDNSParserBPFProgram(bpfLogLevel string) error {
	logLevel := strings.ToLower(bpfLogLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	fileToLoad := "ipt_parse_dns_" + logLevel + "_co-re.o"
	preCompiledBinary := path.Join(bpfdefs.ObjectDir, fileToLoad)
	obj, err := libbpf.OpenObject(preCompiledBinary)
	if err != nil {
		return fmt.Errorf("error opening BPF object %v", err)
	}
	defer obj.Close()
	for m, err := obj.FirstMap(); m != nil && err == nil; m, err = m.NextMap() {
		mapName := m.Name()
		if m.IsMapInternal() {
			if strings.HasPrefix(mapName, ".rodata") {
				continue
			}
		}

		log.Debugf("Pinning map %s k %d v %d", mapName, m.KeySize(), m.ValueSize())
		pinDir := bpf.MapPinDir()
		if err := m.SetPinPath(path.Join(pinDir, mapName)); err != nil {
			return fmt.Errorf("error pinning map %s k %d v %d: %w", mapName, m.KeySize(), m.ValueSize(), err)
		}
	}

	err = obj.Load()
	if err != nil {
		return fmt.Errorf("error loading program %v", err)
	}
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
		if err := os.MkdirAll(bpfdefs.DnsObjDir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func CleanupDNSObjPinDir() {
	os.RemoveAll(bpfdefs.DnsObjDir)
}
