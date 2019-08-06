---
title: Configuring Felix
redirect_from: latest/reference/felix/configuration
canonical_url: https://docs.tigera.io/v2.3/reference/felix/configuration
---

Configuration for Felix is read from one of four possible locations, in
order, as follows.

1.  Environment variables.
2.  The Felix configuration file.
3.  Host-specific `FelixConfiguration` resources.
4.  The global `FelixConfiguration` resource (`default`).

The value of any configuration parameter is the value read from the
*first* location containing a value. For example, if an environment variable
contains a value, it takes top precedence.

If not set in any of these locations, most configuration parameters have
defaults, and it should be rare to have to explicitly set them.

The full list of parameters which can be set is as follows.

> **Note**: The following tables detail the configuration file and
> environment variable parameters. For `FelixConfiguration` resource settings,
> refer to [Felix Configuration Resource](../resources/felixconfig).
{: .alert .alert-info}

#### General configuration

| Configuration parameter           | Environment variable                    | Description  | Schema |
| --------------------------------- | --------------------------------------- | -------------| ------ |
| `DatastoreType`                   | `FELIX_DATASTORETYPE`                   | The datastore that Felix should read endpoints and policy information from. [Default: `etcdv3`] | `etcdv3`, `kubernetes`|
| `ExternalNodesCIDRList`           | `FELIX_EXTERNALNODESCIDRLIST`           | Comma-delimited list of IPv4 or CIDR of external-non-calico-nodes from which IPIP traffic is accepted by calico-nodes. [Default: ""] | string |
| `FailsafeInboundHostPorts`        | `FELIX_FAILSAFEINBOUNDHOSTPORTS`        | Comma-delimited list of UDP/TCP ports that Felix will allow incoming traffic to host endpoints on irrespective of the security policy. This is useful to avoid accidentally cutting off a host with incorrect configuration. Each port should be specified as `tcp:<port-number>` or `udp:<port-number>`. For backwards compatibility, if the protocol is not specified, it defaults to "tcp". To disable all inbound host ports, use the value `none`. The default value allows ssh access, DHCP, BGP and etcd. [Default: `tcp:22, udp:68, tcp:179, tcp:2379, tcp:2380, tcp:6666, tcp:6667`] | string |
| `FailsafeOutboundHostPorts`       | `FELIX_FAILSAFEOUTBOUNDHOSTPORTS`       | Comma-delimited list of UDP/TCP ports that Felix will allow outgoing traffic from host endpoints to irrespective of the security policy. This is useful to avoid accidently cutting off a host with incorrect configuration. Each port should be specified as `tcp:<port-number>` or `udp:<port-number>`.  For backwards compatibility, if the protocol is not specified, it defaults to "tcp". To disable all outbound host ports, use the value `none`. The default value opens etcd's standard ports to ensure that Felix does not get cut off from etcd as well as allowing DHCP, DNS, BGP. [Default: `udp:53, udp:67, tcp:179, tcp:2379, tcp:2380, tcp:6666, tcp:6667`]  | string |
| `FelixHostname`                   | `FELIX_FELIXHOSTNAME`                   | The hostname Felix reports to the plugin. Should be used if the hostname Felix autodetects is incorrect or does not match what the plugin will expect. [Default: `socket.gethostname()`] | string |
| `HealthEnabled`                   | `FELIX_HEALTHENABLED`                   | When enabled, exposes felix health information via an http endpoint. | boolean |
| `HealthHost`                      | `FELIX_HEALTHHOST`                      | The address on which Felix will respond to health requests. [Default: `localhost`] | string |
| `IpInIpEnabled`                   | `FELIX_IPINIPENABLED`                   | Whether Felix should configure an IPinIP interface on the host. Set automatically to `true` by `{{site.nodecontainer}}` or `calicoctl` when you create an IPIP-enabled pool. [Default: `false`] | boolean |
| `IpInIpMtu`                       | `FELIX_IPINIPMTU`                       | The MTU to set on the IPIP tunnel device. See [Configuring MTU]({{site.baseurl}}/{{page.version}}/networking/mtu) [Default: `1440`] | int |
| `IPv4VXLANTunnelAddr`             |                                         | IPv4 address of the VXLAN tunnel. This is system configured and should not be updated manually. | string |
| `LogFilePath`                     | `FELIX_LOGFILEPATH`                     | The full path to the Felix log. Set to `none` to disable file logging. [Default: `/var/log/calico/felix.log`] | string |
| `LogSeverityFile`                 | `FELIX_LOGSEVERITYFILE`                 | The log severity above which logs are sent to the log file. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `LogSeverityScreen`               | `FELIX_LOGSEVERITYSCREEN`               | The log severity above which logs are sent to the stdout. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `LogSeveritySys`                  | `FELIX_LOGSEVERITYSYS`                  | The log severity above which logs are sent to the syslog. Set to `""` for no logging to syslog. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `PolicySyncPathPrefix`            | `FELIX_POLICYSYNCPATHPREFIX`            | File system path where Felix notifies services of policy changes over Unix domain sockets. This is only required if you're configuring [application layer policy](../../getting-started/kubernetes/installation/app-layer-policy). Set to `""` to disable. [Default: `""`] | string |
| `PrometheusGoMetricsEnabled`      | `FELIX_PROMETHEUSGOMETRICSENABLED`      | Set to `false` to disable Go runtime metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. [Default: `true`]  | boolean |
| `PrometheusMetricsEnabled`        | `FELIX_PROMETHEUSMETRICSENABLED`        | Set to `true` to enable the Prometheus metrics server in Felix. [Default: `false`] | boolean |
| `PrometheusMetricsPort`           | `FELIX_PROMETHEUSMETRICSPORT`           | Experimental: TCP port that the Prometheus metrics server should bind to. [Default: `9091`] | int |
| `PrometheusProcessMetricsEnabled` | `FELIX_PROMETHEUSPROCESSMETRICSENABLED` | Set to `false` to disable process metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. [Default: `true`] | boolean |
| `ReportingIntervalSecs`           | `FELIX_REPORTINGINTERVALSECS`           | Interval at which Felix reports its status into the datastore or `0` to disable. Must be non-zero in OpenStack deployments. [Default: `30`] | int |
| `ReportingTTLSecs`                | `FELIX_REPORTINGTTLSECS`                | Time-to-live setting for process-wide status reports. [Default: `90`] | int |
| `SidecarAccelerationEnabled`      | `FELIX_SIDECARACCELERATIONENABLED`      | Enable experimental acceleration between application and proxy sidecar when using [application layer policy]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/app-layer-policy). [Default: `false`] | boolean |
| `UsageReportingEnabled`           | `FELIX_USAGEREPORTINGENABLED`           | Reports anonymous {{site.prodname}} version number and cluster size to projectcalico.org. Logs warnings returned by the usage server. For example, if a significant security vulnerability has been discovered in the version of {{site.prodname}} being used. [Default: `true`] | boolean |
| `UsageReportingInitialDelaySecs`  | `FELIX_USAGEREPORTINGINITIALDELAYSECS`  | Minimum delay before first usage report, in seconds. [Default: `300`] | int |
| `UsageReportingIntervalSecs`      | `FELIX_USAGEREPORTINGINTERVALSECS`      | Interval at which to make usage reports, in seconds. [Default: `86400`] | int |
| `VXLANEnabled`                    | `FELIX_VXLANENABLED`                    | Automatically set when needed, you shouldn't need to change this setting: whether Felix should create the VXLAN tunnel device for VXLAN networking. [Default: `false`] | boolean |
| `VXLANMTU`                        | `FELIX_VXLANMTU`                        | The MTU to set on the VXLAN tunnel device. See [Configuring MTU]({{site.baseurl}}/{{page.version}}/networking/mtu) [Default: `1410`] | int |
| `VXLANPort`                       | `FELIX_VXLANPORT`                       | The UDP port to use for VXLAN. [Default: `4789`] | int |
| `VXLANTunnelMACAddr`              |                                         | MAC address of the VXLAN tunnel. This is system configured and should not be updated manually. | string |
| `VXLANVNI`                        | `FELIX_VXLANVNI`                        | The virtual network ID to use for VXLAN. [Default: `4096`] | int |
| `TyphaAddr`                       | `FELIX_TYPHAADDR`                       | IPv4 address at which Felix should connect to Typha. [Default: none] | string |
| `TyphaK8sServiceName`             | `FELIX_TYPHAK8SSERVICENAME`             | Name of the Typha Kubernetes service | string |
| `Ipv6Support`                     | `FELIX_IPV6SUPPORT`                     | Enable {{site.prodname}} networking and security for IPv6 traffic as well as for IPv4. | boolean |

#### etcd datastore configuration

| Configuration parameter | Environment variable  | Description | Schema |
| ----------------------- | --------------------- | ----------- | ------ |
| `EtcdCaFile`            | `FELIX_ETCDCAFILE`    | Path to the file containing the root certificate of the certificate authority (CA) that issued the etcd server certificate. Configures Felix to trust the CA that signed the root certificate. The file may contain multiple root certificates, causing Felix to trust each of the CAs included. To disable authentication of the server by Felix, set the value to `none`. [Default: `/etc/ssl/certs/ca-certificates.crt`] | string |
| `EtcdCertFile`          | `FELIX_ETCDCERTFILE`  | Path to the file containing the client certificate issued to Felix. Enables Felix to participate in mutual TLS authentication and identify itself to the etcd server. Example: `/etc/felix/cert.pem` (optional) | string |
| `EtcdEndpoints`         | `FELIX_ETCDENDPOINTS` | Comma-delimited list of etcd endpoints to connect to. Example: `http://127.0.0.1:2379,http://127.0.0.2:2379`. | `<scheme>://<ip-or-fqdn>:<port>` |
| `EtcdKeyFile`           | `FELIX_ETCDKEYFILE`   | Path to the file containing the private key matching Felix's client certificate. Enables Felix to participate in mutual TLS authentication and identify itself to the etcd server. Example: `/etc/felix/key.pem` (optional) | string |

#### Kubernetes API datastore configuration

The Kubernetes API datastore driver reads its configuration from Kubernetes-provided environment variables.

#### iptables dataplane configuration

| Configuration parameter              | Environment variable                       | Description | Schema |
| ------------------------------------ | ------------------------------------------ | ----------- | ------ |
| `ChainInsertMode`                    | `FELIX_CHAININSERTMODE`                    | Controls whether Felix hooks the kernel's top-level iptables chains by inserting a rule at the top of the chain or by appending a rule at the bottom.  `Insert` is the safe default since it prevents {{site.prodname}}'s rules from being bypassed.  If you switch to `Append` mode, be sure that the other rules in the chains signal acceptance by falling through to the {{site.prodname}} rules, otherwise the {{site.prodname}} policy will be bypassed. [Default: `Insert`]  | `Insert`, `Append` |
| `DefaultEndpointToHostAction`        | `FELIX_DEFAULTENDPOINTTOHOSTACTION`        | This parameter controls what happens to traffic that goes from a workload endpoint to the host itself (after the traffic hits the endpoint egress policy). By default {{site.prodname}} blocks traffic from workload endpoints to the host itself with an iptables `Drop` action. If you want to allow some or all traffic from endpoint to host, set this parameter to `Return` or `Accept`.  Use `Return` if you have your own rules in the iptables "INPUT" chain; {{site.prodname}} will insert its rules at the top of that chain, then `Return` packets to the "INPUT" chain once it has completed processing workload endpoint egress policy. Use `Accept` to unconditionally accept packets from workloads after processing workload endpoint egress policy. [Default: `Drop`] | `Drop`, `Return`, `Accept` |
| `IgnoreLooseRPF`                     | `FELIX_IGNORELOOSERPF`                     | Set to `true` to allow Felix to run on systems with loose reverse path forwarding (RPF). **Warning**: {{site.prodname}} relies on "strict" RPF checking being enabled to prevent workloads, such as VMs and privileged containers, from spoofing their IP addresses and impersonating other workloads (or hosts). Only enable this flag if you need to run with "loose" RPF and you either trust your workloads or have another mechanism in place to prevent spoofing. | `true`,`false` |
| `InterfaceExclude`                   | `FELIX_INTERFACEEXCLUDE`                   | A comma-separated list of interface names that should be excluded when Felix is resolving host endpoints. The default value ensures that Felix ignores Kubernetes' internal `kube-ipvs0` device. [Default: `kube-ipvs0`] | string |
| `IpInIpEnabled`                      | `FELIX_IPINIPENABLED`                      | Whether Felix should configure an IPinIP interface on the host. Set automatically to `true` by `{{site.nodecontainer}}` or `calicoctl` when you create an IPIP-enabled pool. [Default: `false`] | boolean |
| `IpsetsRefreshIntervalSecs`          | `FELIX_IPSETSREFRESHINTERVAL`              | Period, in seconds, at which Felix re-checks the IP sets in the dataplane to ensure that no other process has accidentally broken {{site.prodname}}'s rules. Set to 0 to disable IP sets refresh.  Note: the default for this value is lower than the other refresh intervals as a workaround for a [Linux kernel bug](https://github.com/projectcalico/felix/issues/1347) that was fixed in kernel version 4.11. If you are using v4.11 or greater you may want to set this to, a higher value to reduce Felix CPU usage. [Default: `10`] | int |
| `IptablesBackend`                    | `FELIX_IPTABLESBACKEND`                    | This parameter controls which variant of iptables binary Felix uses.  If using Felix on a system that uses the netfilter-backed iptables binaries, set this to `NFT`. [Default: `Legacy`]  | `Legacy`, `NFT` |
| `IptablesFilterAllowAction`          | `FELIX_IPTABLESFILTERALLOWACTION`          | This parameter controls what happens to traffic that is allowed by a Felix policy chain in the iptables filter table (i.e., a normal policy chain). The default will immediately `Accept` the traffic. Use `Return` to send the traffic back up to the system chains for further processing. [Default: `Accept`]  | `Accept`, `Return` |
| `IptablesLockFilePath`               | `FELIX_IPTABLESLOCKFILEPATH`               | *Deprecated:* For iptables versions prior to v1.6.2, location of the iptables lock file (later versions of iptables always use value "/run/xtables.lock").  You may need to change this if the lock file is not in its standard location (for example if you have mapped it into Felix's container at a different path). [Default: `/run/xtables.lock`]  | string |
| `IptablesLockProbeIntervalMillis`    | `FELIX_IPTABLESLOCKPROBEINTERVALMILLIS`    | Time, in milliseconds, that Felix will wait between attempts to acquire the iptables lock if it is not available.  Lower values make Felix more responsive when the lock is contended, but use more CPU. [Default: `50`]  | int |
| `IptablesLockTimeoutSecs`            | `FELIX_IPTABLESLOCKTIMEOUTSECS`            | Time, in seconds, that Felix will wait for the iptables lock.  Versions of iptables prior to v1.6.2 support disabling the iptables lock by setting this value to 0; v1.6.2 and above do not so Felix will default to 10s if a non-positive number is used. To use this feature, Felix must share the iptables lock file with all other processes that also take the lock.  When running Felix inside a container, this typically requires the file /run/xtables.lock on the host to be mounted into the `{{site.nodecontainer}}` or `calico/felix` container. [Default: `0` disabled for iptables <v1.6.2 or 10s for later versions] | int |
| `IptablesMangleAllowAction`          | `FELIX_IPTABLESMANGLEALLOWACTION`          | This parameter controls what happens to traffic that is allowed by a Felix policy chain in the iptables mangle table (i.e., a pre-DNAT policy chain). The default will immediately `Accept` the traffic. Use `Return` to send the traffic back up to the system chains for further processing. [Default: `Accept`]  | `Accept`, `Return` |
| `IptablesMarkMask`                   | `FELIX_IPTABLESMARKMASK`                   | Mask that Felix selects its IPTables Mark bits from. Should be a 32 bit hexadecimal number with at least 8 bits set, none of which clash with any other mark bits in use on the system.  When using {{site.prodname}} with Kubernetes' `kube-proxy` in IPVS mode, [we recommend allowing at least 16 bits](#ipvs-bits). [Default: `0xffff0000`] | netmask |
| `IptablesNATOutgoingInterfaceFilter` | `FELIX_IPTABLESNATOUTGOINGINTERFACEFILTER` | This parameter can be used to limit the host interfaces on which Calico will apply SNAT to traffic leaving a Calico IPAM pool with "NAT outgoing" enabled.  This can be useful if you have a main data interface, where traffic should be SNATted and a secondary device (such as the docker bridge) which is local to the host and doesn't require SNAT.  This parameter uses the iptables interface matching syntax, which allows `+` as a wildcard.  Most users will not need to set this.  Example: if your data interfaces are eth0 and eth1 and you want to exclude the docker bridge, you could set this to `eth+` | string |
| `IptablesPostWriteCheckIntervalSecs` | `FELIX_IPTABLESPOSTWRITECHECKINTERVALSECS` | Period, in seconds, after Felix has done a write to the dataplane that it schedules an extra read back in order to check the write was not clobbered by another process.  This should only occur if another application on the system doesn't respect the iptables lock. [Default: `1`] | int |
| `IptablesRefreshInterval`            | `FELIX_IPTABLESREFRESHINTERVAL`            | Period, in seconds, at which Felix re-checks all iptables state to ensure that no other process has accidentally broken {{site.prodname}}'s rules. Set to 0 to disable iptables refresh. [Default: `90`] | int |
| `LogPrefix`                          | `FELIX_LOGPREFIX`                          | The log prefix that Felix uses when rendering LOG rules. [Default: `calico-packet`] | string |
| `MaxIpsetSize`                       | `FELIX_MAXIPSETSIZE`                       | Maximum size for the ipsets used by Felix to implement tags. Should be set to a number that is greater than the maximum number of IP addresses that are ever expected in a tag. [Default: `1048576`] | int |
| `NATPortRange`                       | `FELIX_NATPORTRANGE`                       | Port range used by iptables for port mapping when doing outgoing NAT. (Example: `32768:65000`).  [Default: iptables maps source ports below 512 to other ports below 512: those between 512 and 1023 inclusive will be mapped to ports below 1024, and other ports will be mapped to 1024 or above. Where possible, no port alteration will occur.]  | string |
| `NATOutgoingAddress`                       | `FELIX_NATOUTGOINGADDRESS`                       | Source address used by iptables for an SNAT rule when doing outgoing NAT. [Default: an iptables `MASQUERADE` rule is used for outgoing NAT which will use the address on the interface traffic is leaving on.]  | `<IPv4-address>` |
| `NetlinkTimeoutSecs`                 | `FELIX_NETLINKTIMEOUTSECS`                 | Time, in seconds, that Felix will wait for netlink (i.e. routing table list/update) operations to complete before giving up and retrying. [Default: `10`] | float |
| `RouteRefreshIntervalSecs`           | `FELIX_ROUTEREFRESHINTERVAL`               | Period, in seconds, at which Felix re-checks the routes in the dataplane to ensure that no other process has accidentally broken {{site.prodname}}'s rules. Set to 0 to disable route refresh. [Default: `90`] | int |

#### Kubernetes-specific configuration

| Configuration parameter | Environment variable       | Description  | Schema |
| ------------------------|----------------------------| ------------ | ------ |
| `KubeNodePortRanges`    | `FELIX_KUBENODEPORTRANGES` | A list of port ranges that Felix should treat as Kubernetes node ports.  Only when `kube-proxy` is configured to use IPVS mode:  Felix assumes that traffic arriving at the host one one of these ports will ultimately be forwarded instead of being terminated by a host process.  [Default: `30000:32767`] <a id="ipvs-portranges"></a>  | Comma-delimited list of `<min>:<max>` port ranges or single ports. |


> **Note**: <a id="ipvs-bits"></a> When using {{site.prodname}} with Kubernetes' `kube-proxy` in IPVS mode, {{site.prodname}} uses additional iptables mark bits to store an ID for each local {{site.prodname}} endpoint.
> For example, the default `IptablesMarkMask` value, `0xffff0000` gives {{site.prodname}} 16 bits, up to 6 of which are used for internal purposes, leaving 10 bits for endpoint IDs.
> 10 bits is enough for 1024 different values and {{site.prodname}} uses 2 of those for internal purposes, leaving enough for 1022 endpoints on the host.
{: .alert .alert-info}

#### Bare metal specific configuration

| Configuration parameter | Environment variable    | Description | Schema |
| ----------------------- | ----------------------- | ----------- | ------ |
| `InterfacePrefix`       | `FELIX_INTERFACEPREFIX` | The interface name prefix that identifies workload endpoints and so distinguishes them from host endpoint interfaces. Accepts more than one interface name prefix in comma-delimited format, e.g., `tap,cali`. Note: in environments other than bare metal, the orchestrators configure this appropriately.  For example our Kubernetes and Docker integrations set the `cali` value. [Default: `cali`] | string |

#### {{site.prodname}} specific configuration

| Setting                      | Environment variable               | Default | Meaning                                 |
|------------------------------|------------------------------------|---------|-----------------------------------------|
| `DropActionOverride`         | `FELIX_DROPACTIONOVERRIDE`         | `Drop`  | How to treat packets that are disallowed by the current {{site.prodnamew}} policy.  For more detail please see below. |
| `LogDropActionOverride`      | `FELIX_LOGDROPACTIONOVERRIDE`      | `false` | Set to `true` to add the `DropActionOverride` to the syslog entries. For more detail please see below. |
| `PrometheusReporterEnabled`  | `FELIX_PROMETHEUSREPORTERENABLED`  | `false` | Set to `true` to enable Prometheus reporting of denied packet metrics.  For more detail please see below. |
| `PrometheusReporterPort`     | `FELIX_PROMETHEUSREPORTERPORT`     | `9092`  | The TCP port on which to report denied packet metrics.  |
| `PrometheusReporterCertFile` | `FELIX_PROMETHEUSREPORTERCERTFILE` | None    | Certificate for encrypting Prometheus denied packet metrics.  |
| `PrometheusReporterKeyFile`  | `FELIX_PROMETHEUSREPORTERKEYFILE`  | None    | Private key for encrypting Prometheus denied packet metrics.  |
| `PrometheusReporterCAFile`   | `FELIX_PROMETHEUSREPORTERCAFILE`   | None    | Trusted CA file for clients attempting to read Prometheus denied packet metrics.  |
| `PrometheusMetricsCertFile`  | `FELIX_PROMETHEUSMETRICSCERTFILE`  | None    | Certificate for encrypting general Felix Prometheus metrics.  |
| `PrometheusMetricsKeyFile`   | `FELIX_PROMETHEUSMETRICSKEYFILE`   | None    | Private key for encrypting general Felix Prometheus metrics.  |
| `PrometheusMetricsCAFile`    | `FELIX_PROMETHEUSMETRICSCAFILE`    | None    | Trusted CA file for clients attempting to read general Felix Prometheus metrics.  |
| `IPSecMode`                  | `FELIX_IPSECMODE`                  | None    | Controls which mode IPsec is operating on. The only supported value is `PSK`. An empty value means IPsec is not enabled. |
| `IPSecAllowUnsecuredTraffic` | `FELIX_IPSECALLOWUNSECUREDTRAFFIC` | `false` | When set to false, only IPsec-protected traffic will be allowed on the packet paths where IPsec is supported.  When set to true, IPsec will be used but non-IPsec traffic will be accepted.  In general, setting this to `true` is less safe since it allows an attacker to inject packets.  However, it is useful when transitioning from non-IPsec to IPsec since it allows traffic to flow while the cluster negotiates the IPsec mesh.  |
| `IPSecIKEAlgorithm`          | `FELIX_IPSECIKEALGORITHM`          | `aes128gcm16-prfsha256-ecp256`   | IPsec IKE algorithm. Default is NIST suite B recommendation.|
| `IPSecESPAlgorithm`          | `FELIX_IPSECESPALGORITHM`          | `aes128gcm16-ecp256`             | IPsec ESP algorithm. Default is NIST suite B recommendation.|
| `IPSecLogLevel`              | `FELIX_IPSECLOGLEVEL`              | `Info`  | Controls log level for IPsec components. Set to `None` for no logging. Other valid values are `Notice`, `Info`, `Debug` and `Verbose`. |
| `IPSecPSKFile`               | `FELIX_IPSECPSKFILE`               | None    | The path to the pre shared key file for IPsec. |
| `FlowLogsFileEnabled`        | `FELIX_FLOWLOGSFILEENABLED`        | `false` | Set to `true`, enables flow logs. If set to `false` no flow logging will occur. Flow logs are written to a file `flows.log` and sent to Elasticsearch. The location of this file can be configured using the `FlowLogsFileDirectory` field. File rotation settings for this `flows.log` file can be configured using the fields `FlowLogsFileMaxFiles` and `FlowLogsFileMaxFileSizeMB`. Note that flow log exports to Elasticsearch are dependent on flow logs getting written to this file. Setting this parameter to `false` will disable flow logs. |
| `FlowLogsFileDirectory`      | `FELIX_FLOWLOGSFILEDIRECTORY`      | `/var/log/calico/flowlogs` | The directory where flow logs files are stored. This parameter only takes effect when `FlowLogsFileEnabled` is set to `true`. |
| `FlowLogsFileMaxFiles`       | `FELIX_FLOWLOGSFILEMAXFILES`       | `5`     | The number of files to keep when rotating flow log files. This parameter only takes effect when `FlowLogsFileEnabled` is set to `true`. |
| `FlowLogsFileMaxFileSizeMB`  | `FELIX_FLOWLOGSFILEMAXFILESIZEMB`  | `100`   | The max size in MB of flow logs files before rotation. This parameter only takes effect when `FlowLogsFileEnabled` is set to `true`.|
| `FlowLogsFlushInterval`      | `FELIX_FLOWLOGSFLUSHINTERVAL`      | `300`   | The period, in seconds, at which Felix exports the flow logs. |
| `FlowLogsEnableNetworkSets`  | `FELIX_FLOWLOGSENABLENETWORKSETS`  | `false` | Whether to specify the network set a flow log originates from. |
| `FlowLogsFileAggregationKindForAllowed` | `FELIX_FLOWLOGSFILEAGGREGATIONKINDFORALLOWED` | `2` | How much to aggregate the flow logs sent to Elasticsearch for allowed traffic.  Bear in mind that changing this value may have a dramatic impact on the volume of flow logs sent to Elasticsearch.  `0` means no aggregation, `1` means aggregate all flows that share a source port on each node, and `2` means aggregate all flows that share source ports or are from the same ReplicaSet. |
| `FlowLogsFileAggregationKindForDenied` | `FELIX_FLOWLOGSFILEAGGREGATIONKINDFORDENIED` | `1` | How much to aggregate the flow logs sent to Elasticsearch for denied traffic.  Bear in mind that changing this value may have a dramatic impact on the volume of flow logs sent to Elasticsearch.  `0` means no aggregation, `1` means aggregate all flows that share a source port on each node, and `2` means aggregate all flows that share source ports or are from the same ReplicaSet. |
| `DNSCacheFile`         | `FELIX_DNSCACHEFILE`         | `/var/run/calico/felix-dns-cache.txt` | The name of the file that Felix uses to preserve learnt DNS information when restarting. |
| `DNSCacheSaveInterval` | `FELIX_DNSCACHESAVEINTERVAL` | `60s` | The periodic interval at which Felix saves learnt DNS information to the cache file. |
| `DNSTrustedServers`    | `FELIX_DNSTRUSTEDSERVERS`    | `k8s-service:kube-dns` | The DNS servers that Felix should trust. Each entry here must be an IP, or `k8s-service:<name>`, where `<name>` is the name of a Kubernetes Service in the `kube-system` namespace. |
| `DNSLogsFileEnabled`         | `FELIX_DNSLOGSFILEENABLED`        	| `false` | Set to `true`, enables DNS logs. If set to `false` no DNS logging will occur. DNS logs are written to a file `dns.log` and sent to Elasticsearch. The location of this file can be configured using the `DNSLogsFileDirectory` field. File rotation settings for this `dns.log` file can be configured using the fields `DNSLogsFileMaxFiles` and `DNSLogsFileMaxFileSizeMB`. Note that DNS log exports to Elasticsearch are dependent on DNS logs getting written to this file. Setting this parameter to `false` will disable DNS logs. |
| `DNSLogsFileDirectory`       | `FELIX_DNSLOGSFILEDIRECTORY`      	| `/var/log/calico/dnslogs` | The directory where DNS logs files are stored. This parameter only takes effect when `DNSLogsFileEnabled` is `true`. |
| `DNSLogsFileMaxFiles`        | `FELIX_DNSLOGSFILEMAXFILES`       	| `5`     | The number of files to keep when rotating DNS log files. This parameter only takes effect when `DNSLogsFileEnabled` is `true`. |
| `DNSLogsFileMaxFileSizeMB`   | `FELIX_DNSLOGSFILEMAXFILESIZEMB`  	| `100`   | The max size in MB of DNS log files before rotation. This parameter only takes effect when `DNSLogsFileEnabled` is `true`.|
| `DNSLogsFlushInterval`       | `FELIX_DNSLOGSFLUSHINTERVAL`      	| `300`   | The period, in seconds, at which Felix exports DNS logs. |
| `DNSLogsFileAggregationKind` | `FELIX_DNSLOGSFILEAGGREGATIONKIND` | `1` | How much to aggregate DNS logs.  Bear in mind that changing this value may have a dramatic impact on the volume of flow logs sent to Elasticsearch.  `0` means no aggregation, `1` means aggregate similar DNS logs from workloads in the same ReplicaSet. |
| `DNSLogsFileIncludeLabels`   | `FELIX_DNSLOGSFILEINCLUDELABELS`   | `true` | Whether to include client and server workload labels in DNS logs. |
| `DNSLogsFilePerNodeLimit`    | `FELIX_DNSLOGSFILEPERNODELIMIT`    | `0` (no limit) | Limit on the number of DNS logs that can be emitted within each flush interval.  When this limit has been reached, Felix counts the number of unloggable DNS responses within the flush interval, and emits a WARNING log with that count at the same time as it flushes the buffered DNS logs. |

DropActionOverride controls what happens to each packet that is denied by
the current {{site.prodname}} policy - i.e. by the ordered combination of all the
configured policies and profiles that apply to that packet.  It may be
set to one of the following values:

- `Drop`
- `Accept`
- `LogAndDrop`
- `LogAndAccept`

Normally the `Drop` or `LogAndDrop` value should be used, as dropping a
packet is the obvious implication of that packet being denied.  However when
experimenting, or debugging a scenario that is not behaving as you expect, the
`Accept` and `LogAndAccept` values can be useful: then the packet will be
still be allowed through.

When set to `LogAndDrop` or `LogAndAccept`, each denied packet is logged in
syslog, with an entry like this:

```
May 18 18:42:44 ubuntu kernel: [ 1156.246182] calico-drop: IN=tunl0 OUT=cali76be879f658 MAC= SRC=192.168.128.30 DST=192.168.157.26 LEN=60 TOS=0x00 PREC=0x00 TTL=62 ID=56743 DF PROTO=TCP SPT=56248 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0xa000000
```
{: .no-select-button}

If the `LogDropActionOverride` flag is set, then the `DropActionOverride` will also appear in the syslog entry:

```
May 18 18:42:44 ubuntu kernel: [ 1156.246182] calico-drop LOGandDROP: IN=tunl0 OUT=cali76be879f658 MAC= SRC=192.168.128.30 DST=192.168.157.26 LEN=60 TOS=0x00 PREC=0x00 TTL=62 ID=56743 DF PROTO=TCP SPT=56248 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0xa000000
```
{: .no-select-button}

When the reporting of denied packet metrics is enabled, Felix keeps counts of
recently denied packets and publishes these as Prometheus metrics on the port
configured by the `PrometheusReporterPort` setting.  Please
see the
[Metrics]({{site.url}}/{{page.version}}/security/metrics/metrics) section for
more details.

Note that denied packet metrics are independent of the DropActionOverride
setting.  Specifically, if packets that would normally be denied are being
allowed through by a setting of `Accept` or `LogAndAccept`, those packets
still contribute to the denied packet metrics as just described.

When the `PrometheusReporter...File` and `PrometheusMetrics...File`
parameters are set, Felix's Prometheus ports are TLS-secured such that
only a validated client can read Prometheus metrics, and the data is
encrypted in transit.  A valid client must then connect over HTTPS and
present a certificate that is signed by one of the trusted CAs in the
relevant `Prometheus...CAFile` setting.

#### Felix-Typha TLS configuration

| Configuration parameter | Environment variable   | Description | Schema |
| ----------------------- | ---------------------- | ----------- | ------ |
| `TyphaCAFile`           | `FELIX_TYPHACAFILE`    | Path to the file containing the root certificate of the CA that issued the Typha server certificate. Configures Felix to trust the CA that signed the root certificate. The file may contain multiple root certificates, causing Felix to trust each of the CAs included. Example: `/etc/felix/ca.pem` | string |
| `TyphaCertFile`         | `FELIX_TYPHACERTFILE`  | Path to the file containing the client certificate issued to Felix. Enables Felix to participate in mutual TLS authentication and identify itself to the Typha server. Example: `/etc/felix/cert.pem` | string |
| `TyphaCN`               | `FELIX_TYPHACN`        | If set, the `Common Name` that Typha's certificate must have. If you have enabled TLS on the communications from Felix to Typha, you must set a value here or in `TyphaURISAN`. You can set values in both, as well, such as to facilitate a migration from using one to the other. If either matches, the communication succeeds. [Default: none] | string |
| `TyphaKeyFile`          | `FELIX_TYPHAKEYFILE`   | Path to the file containing the private key matching the Felix client certificate. Enables Felix to participate in mutual TLS authentication and identify itself to the Typha server. Example: `/etc/felix/key.pem` (optional) | string |
| `TyphaURISAN`           | `FELIX_TYPHAURISAN`    | If set, a URI SAN that Typha's certificate must have. We recommend populating this with a [SPIFFE](https://github.com/spiffe/spiffe/blob/master/standards/SPIFFE-ID.md#2-spiffe-identity) string that identifies Typha. All Typha instances should use the same SPIFFE ID. If you have enabled TLS on the communications from Felix to Typha, you must set a value here or in `TyphaCN`. You can set values in both, as well, such as to facilitate a migration from using one to the other. If either matches, the communication succeeds. [Default: none] | string |

For more information on how to use and set these variables, refer to
[Connections from Felix to Typha (Kubernetes)](../../security/comms/crypto-auth#connections-from-felix-to-typha-kubernetes).

### Environment variables

The highest priority of configuration is that read from environment
variables. To set a configuration parameter via an environment variable,
set the environment variable formed by taking `FELIX_` and appending the
uppercase form of the variable name. For example, to set the etcd
address, set the environment variable `FELIX_ETCDADDR`. Other examples
include `FELIX_ETCDSCHEME`, `FELIX_ETCDKEYFILE`, `FELIX_ETCDCERTFILE`,
`FELIX_ETCDCAFILE`, `FELIX_FELIXHOSTNAME`, `FELIX_LOGFILEPATH` and
`FELIX_METADATAADDR`.

### Configuration file

On startup, Felix reads an ini-style configuration file. The path to
this file defaults to `/etc/calico/felix.cfg` but can be overridden
using the `-c` or `--config-file` options on the command line. If the
file exists, then it is read (ignoring section names) and all parameters
are set from it.

In OpenStack, we recommend putting all configuration into configuration
files, since the etcd database is transient (and may be recreated by the
OpenStack plugin in certain error cases). However, in a Docker
environment the use of environment variables or etcd is often more
convenient.

### Datastore

Felix also reads configuration parameters from the datastore.  It supports
a global setting and a per-host override.

1. Get the current felixconfig settings.

   ```bash
   calicoctl get felixconfig default -o yaml --export > felix.yaml
   ```

1. Modify logFilePath to your intended path, e.g. "/tmp/felix.log"

   ```bash
   vim felix.yaml
   ```
   > **Tip**: For a global change set name to "default".
   > For a node-specific change: set name to the node name, e.g. "{{site.prodname}}-Node-1"
   {: .alert .alert-success}

1. Replace the current felixconfig settings

   ```bash
   calicoctl replace -f felix.yaml
   ```

For more information, see [Felix Configuration Resource](../resources/felixconfig).
