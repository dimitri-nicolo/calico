package iptables

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/projectcalico/calico/felix/bpf"
	"github.com/projectcalico/calico/felix/bpf/bpfdefs"
	"github.com/projectcalico/calico/felix/bpf/dnsresolver"
	"github.com/projectcalico/calico/felix/bpf/ipsets"
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
	mapsToPin := []string{ipsets.MapParameters.VersionedName(),
		dnsresolver.DNSPfxMapParams.VersionedName(),
		dnsresolver.DNSSetMapParams.VersionedName(),
		ipsets.MapV6Parameters.VersionedName(),
		dnsresolver.DNSPfxMapParamsV6.VersionedName(),
		dnsresolver.DNSSetMapParamsV6.VersionedName()}
	logLevel := strings.ToLower(bpfLogLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}

	fileToLoad := "ipt_parse_dns_" + logLevel + "_co-re.o"
	preCompiledBinary := path.Join(bpfdefs.ObjectDir, fileToLoad)
	obj, err := bpf.LoadObject(preCompiledBinary, nil, mapsToPin...)
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
		if err := os.MkdirAll(bpfdefs.DnsObjDir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func CleanupDNSObjPinDir() {
	os.RemoveAll(bpfdefs.DnsObjDir)
}
