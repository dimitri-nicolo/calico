// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config_test

import (
	"regexp"

	. "github.com/projectcalico/felix/config"
	"github.com/projectcalico/libcalico-go/lib/set"

	"io/ioutil"
	"net"
	"reflect"
	"time"

	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

var _ = Describe("FelixConfig vs ConfigParams parity", func() {
	var fcFields map[string]reflect.StructField
	var cpFields map[string]reflect.StructField
	cpFieldsToIgnore := []string{
		"sourceToRawConfig",
		"rawValues",
		"Err",
		"numIptablesBitsAllocated",
		"LicenseValid",
		"LicensePollingIntervalSecs",

		// Moved to ClusterInformation
		"ClusterGUID",
		"ClusterType",
		"CalicoVersion",
		"CNXVersion",

		// Moved to Node.
		"IpInIpTunnelAddr",
		"IPv4VXLANTunnelAddr",
		"VXLANTunnelMACAddr",
		"NodeIP",

		// The rekey time is used by the IPsec tests but it isn't exposed in FelixConfiguration.
		"IPSecRekeyTime",

		"EnableNflogSize",
	}
	cpFieldNameToFC := map[string]string{
		"IpInIpEnabled":                      "IPIPEnabled",
		"IpInIpMtu":                          "IPIPMTU",
		"Ipv6Support":                        "IPv6Support",
		"IptablesLockTimeoutSecs":            "IptablesLockTimeout",
		"IptablesLockProbeIntervalMillis":    "IptablesLockProbeInterval",
		"IptablesPostWriteCheckIntervalSecs": "IptablesPostWriteCheckInterval",
		"NetlinkTimeoutSecs":                 "NetlinkTimeout",
		"ReportingIntervalSecs":              "ReportingInterval",
		"ReportingTTLSecs":                   "ReportingTTL",
		"UsageReportingInitialDelaySecs":     "UsageReportingInitialDelay",
		"UsageReportingIntervalSecs":         "UsageReportingInterval",
		"EndpointReportingDelaySecs":         "EndpointReportingDelay",
		"CloudWatchMetricsPushIntervalSecs":  "CloudWatchMetricsPushInterval",
	}
	fcFieldNameToCP := map[string]string{}
	for k, v := range cpFieldNameToFC {
		fcFieldNameToCP[v] = k
	}

	BeforeEach(func() {
		fcFields = fieldsByName(v3.FelixConfigurationSpec{})
		cpFields = fieldsByName(Config{})
		for _, name := range cpFieldsToIgnore {
			delete(cpFields, name)
		}
	})

	It("FelixConfigurationSpec should contain all Config fields", func() {
		missingFields := set.New()
		for n, f := range cpFields {
			mappedName := cpFieldNameToFC[n]
			if mappedName != "" {
				n = mappedName
			}
			if strings.HasPrefix(n, "Debug") {
				continue
			}
			if strings.Contains(string(f.Tag), "local") {
				continue
			}
			if _, ok := fcFields[n]; !ok {
				missingFields.Add(n)
			}
		}
		Expect(missingFields).To(BeEmpty())
	})
	It("Config should contain all FelixConfigurationSpec fields", func() {
		missingFields := set.New()
		for n := range fcFields {
			mappedName := fcFieldNameToCP[n]
			if mappedName != "" {
				n = mappedName
			}
			if _, ok := cpFields[n]; !ok {
				missingFields.Add(n)
			}
		}
		Expect(missingFields).To(BeEmpty())
	})
})

func fieldsByName(example interface{}) map[string]reflect.StructField {
	fields := map[string]reflect.StructField{}
	t := reflect.TypeOf(example)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fields[f.Name] = f
	}
	return fields
}

var _ = DescribeTable("Config parsing",
	func(key, value string, expected interface{}, errorExpected ...bool) {
		config := New()
		config.UpdateFrom(map[string]string{key: value},
			EnvironmentVariable)
		configPtr := reflect.ValueOf(config)
		configElem := configPtr.Elem()
		fieldRef := configElem.FieldByName(key)
		newVal := fieldRef.Interface()
		Expect(newVal).To(Equal(expected))
		if len(errorExpected) > 0 && errorExpected[0] {
			Expect(config.Err).To(HaveOccurred())
		} else {
			Expect(config.Err).NotTo(HaveOccurred())
		}
	},

	Entry("CloudWatchLogsRetentionDays - good", "CloudWatchLogsRetentionDays", "30", 30),
	Entry("CloudWatchLogsRetentionDays - bad", "CloudWatchLogsRetentionDays", "31", 7, true),

	Entry("CloudWatch Metrics update interval - in range", "CloudWatchMetricsPushIntervalSecs", "90", time.Duration(90*time.Second), false),
	Entry("CloudWatch Metrics update interval - out of range should be converted to default", "CloudWatchMetricsPushIntervalSecs", "5", time.Duration(60*time.Second), false),
	Entry("CloudWatch Metrics update interval - default value", "CloudWatchMetricsPushIntervalSecs", "", time.Duration(60*time.Second), false),
	Entry("Netlink Timeout - default value", "NetlinkTimeoutSecs", "", time.Duration(10*time.Second), false),

	Entry("FelixHostname", "FelixHostname", "hostname", "hostname"),
	Entry("FelixHostname FQDN", "FelixHostname", "hostname.foo.bar.com", "hostname.foo.bar.com"),
	Entry("FelixHostname as IP", "FelixHostname", "1.2.3.4", "1.2.3.4"),

	Entry("EtcdAddr IP", "EtcdAddr", "10.0.0.1:1234", "10.0.0.1:1234"),
	Entry("EtcdAddr Empty", "EtcdAddr", "", "127.0.0.1:2379"),
	Entry("EtcdAddr host", "EtcdAddr", "host:1234", "host:1234"),
	Entry("EtcdScheme", "EtcdScheme", "https", "https"),

	// Etcd key files will be tested for existence, skipping for now.

	Entry("EtcdEndpoints HTTP", "EtcdEndpoints",
		"http://127.0.0.1:1234, http://host:2345",
		[]string{"http://127.0.0.1:1234/", "http://host:2345/"}),
	Entry("EtcdEndpoints HTTPS", "EtcdEndpoints",
		"https://127.0.0.1:1234/, https://host:2345",
		[]string{"https://127.0.0.1:1234/", "https://host:2345/"}),

	Entry("TyphaAddr empty", "TyphaAddr", "", ""),
	Entry("TyphaAddr set", "TyphaAddr", "foo:1234", "foo:1234"),
	Entry("TyphaK8sServiceName empty", "TyphaK8sServiceName", "", ""),
	Entry("TyphaK8sServiceName set", "TyphaK8sServiceName", "calico-typha", "calico-typha"),
	Entry("TyphaK8sNamespace empty", "TyphaK8sNamespace", "", "kube-system"),
	Entry("TyphaK8sNamespace set", "TyphaK8sNamespace", "default", "default"),
	Entry("TyphaK8sNamespace none", "TyphaK8sNamespace", "none", "kube-system", true),

	Entry("InterfacePrefix", "InterfacePrefix", "tap", "tap"),
	Entry("InterfacePrefix list", "InterfacePrefix", "tap,cali", "tap,cali"),

	Entry("InterfaceExclude one value no regexp", "InterfaceExclude", "kube-ipvs0", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
	Entry("InterfaceExclude list no regexp", "InterfaceExclude", "kube-ipvs0,dummy", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
		regexp.MustCompile("^dummy$"),
	}),
	Entry("InterfaceExclude one value regexp", "InterfaceExclude", "/kube-ipvs/", []*regexp.Regexp{
		regexp.MustCompile("kube-ipvs"),
	}),
	Entry("InterfaceExclude list regexp", "InterfaceExclude", "kube-ipvs0,dummy,/^veth.*$/", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
		regexp.MustCompile("^dummy$"),
		regexp.MustCompile("^veth.*$"),
	}),
	Entry("InterfaceExclude no regexp", "InterfaceExclude", "/^kube.*/,/veth/", []*regexp.Regexp{
		regexp.MustCompile("^kube.*"),
		regexp.MustCompile("veth"),
	}),
	Entry("InterfaceExclude list empty regexp", "InterfaceExclude", "kube,//", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
	Entry("InterfaceExclude list bad comma use", "InterfaceExclude", "/kube,/,dummy", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
	Entry("InterfaceExclude list invalid regexp symbol", "InterfaceExclude", `/^kube\K/`, []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),

	Entry("ChainInsertMode append", "ChainInsertMode", "append", "append"),
	Entry("ChainInsertMode append", "ChainInsertMode", "Append", "append"),

	Entry("IptablesPostWriteCheckIntervalSecs", "IptablesPostWriteCheckIntervalSecs",
		"1.5", 1500*time.Millisecond),
	Entry("IptablesLockFilePath", "IptablesLockFilePath",
		"/host/run/xtables.lock", "/host/run/xtables.lock"),
	Entry("IptablesLockTimeoutSecs", "IptablesLockTimeoutSecs",
		"123", 123*time.Second),
	Entry("IptablesLockProbeIntervalMillis", "IptablesLockProbeIntervalMillis",
		"123", 123*time.Millisecond),
	Entry("IptablesLockProbeIntervalMillis garbage", "IptablesLockProbeIntervalMillis",
		"garbage", 50*time.Millisecond),

	Entry("DefaultEndpointToHostAction", "DefaultEndpointToHostAction",
		"RETURN", "RETURN"),
	Entry("DefaultEndpointToHostAction", "DefaultEndpointToHostAction",
		"ACCEPT", "ACCEPT"),

	Entry("DropActionOverride", "DropActionOverride",
		"Accept", "ACCEPT"),
	Entry("DropActionOverride norm", "DropActionOverride",
		"accept", "ACCEPT"),
	Entry("DropActionOverride LogAndAccept", "DropActionOverride",
		"LogAndAccept", "LOGandACCEPT"),
	Entry("DropActionOverride logandaccept", "DropActionOverride",
		"logandaccept", "LOGandACCEPT"),
	Entry("DropActionOverride LogAndDrop", "DropActionOverride",
		"LogAndDrop", "LOGandDROP"),

	Entry("IptablesFilterAllowAction", "IptablesFilterAllowAction",
		"RETURN", "RETURN"),
	Entry("IptablesMangleAllowAction", "IptablesMangleAllowAction",
		"RETURN", "RETURN"),

	Entry("LogFilePath", "LogFilePath", "/tmp/felix.log", "/tmp/felix.log"),

	Entry("LogSeverityFile", "LogSeverityFile", "debug", "DEBUG"),
	Entry("LogSeverityFile", "LogSeverityFile", "warning", "WARNING"),
	Entry("LogSeverityFile", "LogSeverityFile", "error", "ERROR"),
	Entry("LogSeverityFile", "LogSeverityFile", "fatal", "FATAL"),

	Entry("LogSeverityScreen", "LogSeverityScreen", "debug", "DEBUG"),
	Entry("LogSeverityScreen", "LogSeverityScreen", "warning", "WARNING"),
	Entry("LogSeverityScreen", "LogSeverityScreen", "error", "ERROR"),
	Entry("LogSeverityScreen", "LogSeverityScreen", "fatal", "FATAL"),

	Entry("LogSeveritySys", "LogSeveritySys", "debug", "DEBUG"),
	Entry("LogSeveritySys", "LogSeveritySys", "warning", "WARNING"),
	Entry("LogSeveritySys", "LogSeveritySys", "error", "ERROR"),
	Entry("LogSeveritySys", "LogSeveritySys", "fatal", "FATAL"),

	Entry("IpInIpEnabled", "IpInIpEnabled", "true", true),
	Entry("IpInIpEnabled", "IpInIpEnabled", "y", true),
	Entry("IpInIpEnabled", "IpInIpEnabled", "True", true),

	Entry("IpInIpMtu", "IpInIpMtu", "1234", int(1234)),
	Entry("IpInIpTunnelAddr", "IpInIpTunnelAddr",
		"10.0.0.1", net.ParseIP("10.0.0.1")),

	Entry("ReportingIntervalSecs", "ReportingIntervalSecs", "31", 31*time.Second),
	Entry("ReportingTTLSecs", "ReportingTTLSecs", "91", 91*time.Second),

	Entry("EndpointReportingEnabled", "EndpointReportingEnabled",
		"true", true),
	Entry("EndpointReportingEnabled", "EndpointReportingEnabled",
		"yes", true),
	Entry("EndpointReportingDelaySecs", "EndpointReportingDelaySecs",
		"10", 10*time.Second),

	Entry("MaxIpsetSize", "MaxIpsetSize", "12345", int(12345)),
	Entry("IptablesMarkMask", "IptablesMarkMask", "0xf0f0", uint32(0xf0f0)),

	Entry("HealthEnabled", "HealthEnabled", "true", true),
	Entry("HealthHost", "HealthHost", "127.0.0.1", "127.0.0.1"),
	Entry("HealthPort", "HealthPort", "1234", int(1234)),

	Entry("PrometheusMetricsEnabled", "PrometheusMetricsEnabled", "true", true),
	Entry("PrometheusMetricsPort", "PrometheusMetricsPort", "1234", int(1234)),
	Entry("PrometheusGoMetricsEnabled", "PrometheusGoMetricsEnabled", "false", false),
	Entry("PrometheusProcessMetricsEnabled", "PrometheusProcessMetricsEnabled", "false", false),

	Entry("FailsafeInboundHostPorts old syntax", "FailsafeInboundHostPorts", "1,2,3,4",
		[]ProtoPort{
			{Protocol: "tcp", Port: 1},
			{Protocol: "tcp", Port: 2},
			{Protocol: "tcp", Port: 3},
			{Protocol: "tcp", Port: 4},
		}),
	Entry("FailsafeOutboundHostPorts old syntax", "FailsafeOutboundHostPorts", "1,2,3,4",
		[]ProtoPort{
			{Protocol: "tcp", Port: 1},
			{Protocol: "tcp", Port: 2},
			{Protocol: "tcp", Port: 3},
			{Protocol: "tcp", Port: 4},
		}),
	Entry("FailsafeInboundHostPorts new syntax", "FailsafeInboundHostPorts", "tcp:1,udp:2",
		[]ProtoPort{
			{Protocol: "tcp", Port: 1},
			{Protocol: "udp", Port: 2},
		}),
	Entry("FailsafeOutboundHostPorts new syntax", "FailsafeOutboundHostPorts", "tcp:1,udp:2",
		[]ProtoPort{
			{Protocol: "tcp", Port: 1},
			{Protocol: "udp", Port: 2},
		}),
	Entry("FailsafeInboundHostPorts mixed syntax", "FailsafeInboundHostPorts", "1,udp:2",
		[]ProtoPort{
			{Protocol: "tcp", Port: 1},
			{Protocol: "udp", Port: 2},
		}),
	Entry("FailsafeOutboundHostPorts mixed syntax", "FailsafeOutboundHostPorts", "1,udp:2",
		[]ProtoPort{
			{Protocol: "tcp", Port: 1},
			{Protocol: "udp", Port: 2},
		}),
	Entry("FailsafeInboundHostPorts bad syntax -> defaulted", "FailsafeInboundHostPorts", "foo:1",
		[]ProtoPort{
			{Protocol: "tcp", Port: 22},
			{Protocol: "udp", Port: 68},
			{Protocol: "tcp", Port: 179},
			{Protocol: "tcp", Port: 2379},
			{Protocol: "tcp", Port: 2380},
			{Protocol: "tcp", Port: 6666},
			{Protocol: "tcp", Port: 6667},
		},
		true,
	),
	Entry("FailsafeInboundHostPorts too many parts -> defaulted", "FailsafeInboundHostPorts", "tcp:1:bar",
		[]ProtoPort{
			{Protocol: "tcp", Port: 22},
			{Protocol: "udp", Port: 68},
			{Protocol: "tcp", Port: 179},
			{Protocol: "tcp", Port: 2379},
			{Protocol: "tcp", Port: 2380},
			{Protocol: "tcp", Port: 6666},
			{Protocol: "tcp", Port: 6667},
		},
		true,
	),

	Entry("FailsafeInboundHostPorts none", "FailsafeInboundHostPorts", "none", []ProtoPort(nil)),
	Entry("FailsafeOutboundHostPorts none", "FailsafeOutboundHostPorts", "none", []ProtoPort(nil)),

	Entry("FailsafeInboundHostPorts empty", "FailsafeInboundHostPorts", "",
		[]ProtoPort{
			{Protocol: "tcp", Port: 22},
			{Protocol: "udp", Port: 68},
			{Protocol: "tcp", Port: 179},
			{Protocol: "tcp", Port: 2379},
			{Protocol: "tcp", Port: 2380},
			{Protocol: "tcp", Port: 6666},
			{Protocol: "tcp", Port: 6667},
		},
	),
	Entry("FailsafeOutboundHostPorts empty", "FailsafeOutboundHostPorts", "",
		[]ProtoPort{
			{Protocol: "udp", Port: 53},
			{Protocol: "udp", Port: 67},
			{Protocol: "tcp", Port: 179},
			{Protocol: "tcp", Port: 2379},
			{Protocol: "tcp", Port: 2380},
			{Protocol: "tcp", Port: 6666},
			{Protocol: "tcp", Port: 6667},
		},
	),
	Entry("KubeNodePortRanges empty", "KubeNodePortRanges", "",
		[]numorstring.Port{
			{30000, 32767, ""},
		},
	),
	Entry("KubeNodePortRanges range", "KubeNodePortRanges", "30001:30002,30030:30040,30500:30600",
		[]numorstring.Port{
			{30001, 30002, ""},
			{30030, 30040, ""},
			{30500, 30600, ""},
		},
	),

	Entry("IptablesNATOutgoingInterfaceFilter", "IptablesNATOutgoingInterfaceFilter", "cali-123", "cali-123"),
	Entry("IptablesNATOutgoingInterfaceFilter", "IptablesNATOutgoingInterfaceFilter", "cali@123", "", false),

	Entry("IPSecMode", "IPSecMode", "PSK", "PSK"),
	Entry("IPSecPSKFile", "IPSecPSKFile", "/proc/1/cmdline", "/proc/1/cmdline"),
	Entry("IPSecIKEAlgorithm", "IPSecIKEAlgorithm", "aes256gcm16-prfsha384-ecp384", "aes256gcm16-prfsha384-ecp384"),
	Entry("IPSecESPAlgorithm", "IPSecESPAlgorithm", "aes256gcm16-ecp384", "aes256gcm16-ecp384"),
	Entry("IPSecPolicyRefreshInterval", "IPSecPolicyRefreshInterval", "1.5", 1500*time.Millisecond),

	Entry("IPSecLogLevel", "IPSecLogLevel", "none", ""),
	Entry("IPSecLogLevel", "IPSecLogLevel", "notice", "NOTICE"),
	Entry("IPSecLogLevel", "IPSecLogLevel", "info", "INFO"),
	Entry("IPSecLogLevel", "IPSecLogLevel", "debug", "DEBUG"),
	Entry("IPSecLogLevel", "IPSecLogLevel", "verbose", "VERBOSE"),

	Entry("IPSecRekeyTime", "IPSecRekeyTime", "123", 123*time.Second),
)

var _ = DescribeTable("OpenStack heuristic tests",
	func(clusterType, metadataAddr, metadataPort, ifacePrefixes interface{}, expected bool) {
		c := New()
		values := make(map[string]string)
		if clusterType != nil {
			values["ClusterType"] = clusterType.(string)
		}
		if metadataAddr != nil {
			values["MetadataAddr"] = metadataAddr.(string)
		}
		if metadataPort != nil {
			values["MetadataPort"] = metadataPort.(string)
		}
		if ifacePrefixes != nil {
			values["InterfacePrefix"] = ifacePrefixes.(string)
		}
		_, err := c.UpdateFrom(values, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.OpenstackActive()).To(Equal(expected))
	},
	Entry("no config", nil, nil, nil, nil, false),

	Entry("explicit openstack as cluster type", "openstack", nil, nil, nil, true),
	Entry("explicit openstack at start of cluster type", "openstack,k8s", nil, nil, nil, true),
	Entry("explicit openstack at end of cluster type", "k8s,openstack", nil, nil, nil, true),
	Entry("explicit openstack in middle of cluster type", "k8s,openstack,k8s", nil, nil, nil, true),

	Entry("metadataAddr set", nil, "10.0.0.1", nil, nil, true),
	Entry("metadataAddr = none", nil, "none", nil, nil, false),
	Entry("metadataAddr = ''", nil, "", nil, nil, false),

	Entry("metadataPort set", nil, nil, "1234", nil, true),
	Entry("metadataPort = none", nil, nil, "none", nil, false),

	Entry("ifacePrefixes = tap", nil, nil, nil, "tap", true),
	Entry("ifacePrefixes = cali,tap", nil, nil, nil, "cali,tap", true),
	Entry("ifacePrefixes = tap,cali ", nil, nil, nil, "tap,cali", true),
	Entry("ifacePrefixes = cali ", nil, nil, nil, "cali", false),
)

var _ = Describe("DatastoreConfig tests", func() {
	var c *Config
	Describe("with IPIP enabled", func() {
		BeforeEach(func() {
			c = New()
			c.DatastoreType = "k8s"
			c.IpInIpEnabled = true
		})
		It("should leave node polling enabled", func() {
			Expect(c.DatastoreConfig().Spec.K8sDisableNodePoll).To(BeFalse())
		})
	})
	Describe("with IPIP disabled", func() {
		BeforeEach(func() {
			c = New()
			c.DatastoreType = "k8s"
			c.IpInIpEnabled = false
		})
		It("should leave node polling enabled", func() {
			Expect(c.DatastoreConfig().Spec.K8sDisableNodePoll).To(BeTrue())
		})
	})
})

var _ = DescribeTable("Config validation",
	func(settings map[string]string, ok bool) {
		cfg := New()
		_, err := cfg.UpdateFrom(settings, ConfigFile)
		log.WithError(err).Info("UpdateFrom result")
		if err == nil {
			err = cfg.Validate()
			log.WithError(err).Info("Validation result")
		}
		if !ok {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},

	Entry("no settings", map[string]string{}, true),
	Entry("just one TLS setting", map[string]string{
		"TyphaKeyFile": "/usr",
	}, false),
	Entry("TLS certs and key but no CN or URI SAN", map[string]string{
		"TyphaKeyFile":  "/usr",
		"TyphaCertFile": "/usr",
		"TyphaCAFile":   "/usr",
	}, false),
	Entry("TLS certs and key and CN but no URI SAN", map[string]string{
		"TyphaKeyFile":  "/usr",
		"TyphaCertFile": "/usr",
		"TyphaCAFile":   "/usr",
		"TyphaCN":       "typha-peer",
	}, true),
	Entry("TLS certs and key and URI SAN but no CN", map[string]string{
		"TyphaKeyFile":  "/usr",
		"TyphaCertFile": "/usr",
		"TyphaCAFile":   "/usr",
		"TyphaURISAN":   "spiffe://k8s.example.com/typha-peer",
	}, true),
	Entry("all Felix-Typha TLS params", map[string]string{
		"TyphaKeyFile":  "/usr",
		"TyphaCertFile": "/usr",
		"TyphaCAFile":   "/usr",
		"TyphaCN":       "typha-peer",
		"TyphaURISAN":   "spiffe://k8s.example.com/typha-peer",
	}, true),
	Entry("valid OpenstackRegion", map[string]string{
		"OpenstackRegion": "region1",
	}, true),
	Entry("OpenstackRegion with uppercase", map[string]string{
		"OpenstackRegion": "RegionOne",
	}, false),
	Entry("OpenstackRegion with slash", map[string]string{
		"OpenstackRegion": "us/east",
	}, false),
	Entry("OpenstackRegion with underscore", map[string]string{
		"OpenstackRegion": "my_region",
	}, false),
	Entry("OpenstackRegion too long", map[string]string{
		"OpenstackRegion": "my-region-has-a-very-long-and-extremely-interesting-name",
	}, false),
)

var _ = DescribeTable("Config InterfaceExclude",
	func(excludeList string, expected []*regexp.Regexp) {
		cfg := New()
		cfg.UpdateFrom(map[string]string{"InterfaceExclude": excludeList}, EnvironmentVariable)
		regexps := cfg.InterfaceExclude
		Expect(regexps).To(Equal(expected))
	},

	Entry("empty exclude list", "", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
	Entry("non-regexp single value", "kube-ipvs0", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
	Entry("non-regexp multiple values", "kube-ipvs0,veth1", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
		regexp.MustCompile("^veth1$"),
	}),
	Entry("regexp single value", "/^veth.*/", []*regexp.Regexp{
		regexp.MustCompile("^veth.*"),
	}),
	Entry("regexp multiple values", "/veth/,/^kube.*/", []*regexp.Regexp{
		regexp.MustCompile("veth"),
		regexp.MustCompile("^kube.*"),
	}),
	Entry("both non-regexp and regexp values", "kube-ipvs0,/veth/,/^kube.*/", []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
		regexp.MustCompile("veth"),
		regexp.MustCompile("^kube.*"),
	}),
	Entry("invalid non-regexp value", `not.a.valid.interf@e!!`, []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
	Entry("invalid regexp value", `/^kube\K/`, []*regexp.Regexp{
		regexp.MustCompile("^kube-ipvs0$"),
	}),
)

var _ = Describe("IPSec PSK parameters test", func() {
	var c *Config
	psk := "pre-shared-key"
	pskFile := "./tmp-psk-file-ut"

	Describe("with IPSec PSK File", func() {
		BeforeEach(func() {
			c = New()
			err := ioutil.WriteFile(pskFile, []byte(psk), 0600)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			err := os.Remove(pskFile)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should read PSK correctly", func() {
			c.IPSecMode = "PSK"
			c.IPSecPSKFile = pskFile
			Expect(c.GetPSKFromFile()).To(Equal(psk))
		})
		It("should read empty PSK if IPSec is not enabled", func() {
			c.IPSecMode = ""
			c.IPSecPSKFile = pskFile
			Expect(c.GetPSKFromFile()).Should(BeEmpty())
		})
		It("should panic on empty PSK file", func() {
			c.IPSecMode = "PSK"
			c.IPSecPSKFile = pskFile
			err := ioutil.WriteFile(pskFile, []byte{}, 0600)
			Expect(err).NotTo(HaveOccurred())

			panicWrapper := func() { c.GetPSKFromFile() }
			Expect(panicWrapper).To(Panic())
		})
	})

	It("should ignore IPIP params if IPsec is turned on", func() {
		cfg := New()
		_, err := cfg.UpdateFrom(map[string]string{
			"IpInIpEnabled":    "true",
			"IpInIpTunnelAddr": "10.0.0.1",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.IpInIpTunnelAddr.String()).To(Equal("10.0.0.1"))
		Expect(cfg.IpInIpEnabled).To(BeTrue())
		Expect(cfg.IPSecEnabled()).To(BeFalse())
		Expect(cfg.IPSecMode).To(Equal(""))

		_, err = cfg.UpdateFrom(map[string]string{
			"IPSecMode": "PSK",
		}, DatastoreGlobal)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.IpInIpTunnelAddr).To(BeNil())
		Expect(cfg.IpInIpEnabled).To(BeFalse())
		Expect(cfg.IPSecEnabled()).To(BeTrue())
		Expect(cfg.IPSecMode).To(Equal("PSK"))
	})
})

var _ = DescribeTable("CloudWatchLogs config validation",
	func(settings map[string]string, ok bool) {
		cfg := New()
		cfg.UpdateFrom(settings, ConfigFile)
		err := cfg.Validate()
		log.WithError(err).Info("Validation result")
		if !ok {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},

	Entry("reporter enabled", map[string]string{
		"CloudWatchLogsReporterEnabled": "true",
	}, true),
	Entry("reporter enabled, allowed and denied disabled", map[string]string{
		"CloudWatchLogsReporterEnabled":   "true",
		"CloudWatchLogsEnabledForAllowed": "false",
		"CloudWatchLogsEnabledForDenied":  "false",
	}, false),
	Entry("reporter enabled, allowed enabled and denied disabled", map[string]string{
		"CloudWatchLogsReporterEnabled":   "true",
		"CloudWatchLogsEnabledForAllowed": "true",
		"CloudWatchLogsEnabledForDenied":  "false",
	}, true),
	Entry("reporter enabled, allowed disabled and denied enabled", map[string]string{
		"CloudWatchLogsReporterEnabled":   "true",
		"CloudWatchLogsEnabledForAllowed": "false",
		"CloudWatchLogsEnabledForDenied":  "true",
	}, true),
)

var _ = Describe("CloudWatch deprecated config fields", func() {
	var c *Config

	BeforeEach(func() {
		c = New()
	})

	It("should preferentially take the value of FlowLogsFlushInterval over CloudWatchLogsFlushInterval", func() {
		By("setting no values and default value of FlowLogsFlushInterval is used")
		_, err := c.UpdateFrom(map[string]string{}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsFlushInterval).To(Equal(300 * time.Second))
		Expect(c.FlowLogsFlushInterval).To(Equal(300 * time.Second))

		By("setting CloudWatchLogsFlushInterval and checking that value is used")
		changed, err := c.UpdateFrom(map[string]string{
			"CloudWatchLogsFlushInterval": "800",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsFlushInterval).To(Equal(800 * time.Second))
		Expect(c.FlowLogsFlushInterval).To(Equal(800 * time.Second))
		Expect(changed).To(BeTrue())

		By("setting both FlowLogsFlushInterval and checking for FlowLogsFlushInterval value")
		changed, err = c.UpdateFrom(map[string]string{
			"FlowLogsFlushInterval": "600",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsFlushInterval).To(Equal(600 * time.Second))
		Expect(c.FlowLogsFlushInterval).To(Equal(600 * time.Second))
		Expect(changed).To(BeTrue())

		By("setting CloudWatchLogsFlushInterval to a lower value and checking unchanged")
		changed, err = c.UpdateFrom(map[string]string{
			"CloudWatchLogsFlushInterval": "500",
			"FlowLogsFlushInterval":       "600",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsFlushInterval).To(Equal(600 * time.Second))
		Expect(c.FlowLogsFlushInterval).To(Equal(600 * time.Second))
		Expect(changed).To(BeFalse())

		By("setting swapping the values around and checking changed")
		changed, err = c.UpdateFrom(map[string]string{
			"CloudWatchLogsFlushInterval": "600",
			"FlowLogsFlushInterval":       "500",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsFlushInterval).To(Equal(500 * time.Second))
		Expect(c.FlowLogsFlushInterval).To(Equal(500 * time.Second))
		Expect(changed).To(BeTrue())
	})

	It("should combine the value of CloudWatchLogsEnableHostEndpoint and FlowLogsEnableHostEndpoint", func() {
		By("setting no values and default value of FlowLogsEnableHostEndpoint is used")
		_, err := c.UpdateFrom(map[string]string{}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsEnableHostEndpoint).To(BeFalse())
		Expect(c.FlowLogsEnableHostEndpoint).To(BeFalse())

		By("setting CloudWatchLogsEnableHostEndpoint to true and checking value is now true")
		changed, err := c.UpdateFrom(map[string]string{
			"CloudWatchLogsEnableHostEndpoint": "true",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsEnableHostEndpoint).To(BeTrue())
		Expect(c.FlowLogsEnableHostEndpoint).To(BeTrue())
		Expect(changed).To(BeTrue())

		By("setting CloudWatchLogsEnableHostEndpoint to false and checking value is now false")
		changed, err = c.UpdateFrom(map[string]string{
			"CloudWatchLogsEnableHostEndpoint": "false",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsEnableHostEndpoint).To(BeFalse())
		Expect(c.FlowLogsEnableHostEndpoint).To(BeFalse())
		Expect(changed).To(BeTrue())

		By("setting FlowLogsEnableHostEndpoint to true and checking value is now true")
		changed, err = c.UpdateFrom(map[string]string{
			"FlowLogsEnableHostEndpoint": "true",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsEnableHostEndpoint).To(BeTrue())
		Expect(c.FlowLogsEnableHostEndpoint).To(BeTrue())
		Expect(changed).To(BeTrue())

		By("setting CloudWatchLogsEnableHostEndpoint to true and checking value is still true")
		changed, err = c.UpdateFrom(map[string]string{
			"CloudWatchLogsEnableHostEndpoint": "true",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsEnableHostEndpoint).To(BeTrue())
		Expect(c.FlowLogsEnableHostEndpoint).To(BeTrue())
		Expect(changed).To(BeFalse())

		By("setting FlowLogsEnableHostEndpoint to false and checking value is now false")
		changed, err = c.UpdateFrom(map[string]string{
			"FlowLogsEnableHostEndpoint": "false",
		}, EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.CloudWatchLogsEnableHostEndpoint).To(BeFalse())
		Expect(c.FlowLogsEnableHostEndpoint).To(BeFalse())
		Expect(changed).To(BeTrue())
	})
})
