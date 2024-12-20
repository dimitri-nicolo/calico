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

func LoadIPSetsPolicyProgram(ipSetID uint64, bpfLogLevel string, ipFamily int) error {
	logLevel := strings.ToLower(bpfLogLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	suffix := "_v4.o"
	if ipFamily == 6 {
		suffix = "_v6.o"
	}
	fileToLoad := "ipt_match_ipset_" + logLevel + suffix
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
			if err := libbpf.IPTSetGlobals(m, ipSetID); err != nil {
				return fmt.Errorf("error setting globals: %w", err)
			}
			continue
		}
		log.Debugf("Pinning map %s k %d v %d", mapName, m.KeySize(), m.ValueSize())
		pinDir := bpf.MapPinDir()
		if err := m.SetPinPath(path.Join(pinDir, mapName)); err != nil {
			return fmt.Errorf("error pinning map %s k %d v %d: %w", mapName, m.KeySize(), m.ValueSize(), err)
		}
		log.Infof("Pin path %s", path.Join(pinDir, mapName))
	}

	err = obj.Load()
	if err != nil {
		return fmt.Errorf("error loading program %v", err)
	}

	pinPath := path.Join(bpfdefs.DnsObjDir, fmt.Sprintf("%d_v%d", ipSetID, ipFamily))
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

func CreateObjPinDir() error {
	_, err := os.Stat(bpfdefs.DnsObjDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(bpfdefs.DnsObjDir, 0700); err != nil {
		return err
	}
	return nil
}

func CleanupObjPinDir() {
	os.RemoveAll(bpfdefs.DnsObjDir)
}
