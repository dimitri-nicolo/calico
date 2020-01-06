---
title: Felix configuration
canonical_url: https://docs.tigera.io/v2.3/reference/calicoctl/resources/felixconfig
---

A [Felix]({{site.url}}/{{page.version}}/reference/architecture/#felix) configuration resource (`FelixConfiguration`) represents Felix configuration options for the cluster.

For `calicoctl` [commands]({{site.url}}/{{page.version}}/reference/calicoctl/), the following case-insensitive aliases
may be used to specify the resource type on the CLI:
`felixconfiguration`, `felixconfig`, `felixconfigurations`, `felixconfigs`.

This resource is not supported in `kubectl`.

See [Configuring Felix]({{site.url}}/{{page.version}}/reference/felix/configuration) for more details.

### Sample YAML

```yaml
apiVersion: projectcalico.org/v3
kind: FelixConfiguration
metadata:
  name: default
spec:
  ipv6Support: false
  ipipMTU: 1400
  chainInsertMode: Append
```

### Felix configuration definition

#### Metadata

| Field       | Description                 | Accepted Values   | Schema |
|-------------|-----------------------------|-------------------|--------|
| name     | Unique name to describe this resource instance. Required. | Alphanumeric string with optional `.`, `_`, or `-`. | string |

- {{site.tseeprodname}} automatically creates a resource named `default` containing the global default configuration settings for Felix. You can use [calicoctl]({{site.url}}/{{page.version}}/reference/calicoctl/) to view and edit these settings
- The resources with the name `node.<nodename>` contain the node-specific overrides, and will be applied to the node `<nodename>`. When deleting a node the FelixConfiguration resource associated with the node will also be deleted.

#### Spec

| Field                              | Description                 | Accepted Values   | Schema | Default    |
|------------------------------------|-----------------------------|-------------------|--------|------------|
| dropActionOverride | Controls what happens to each packet that is denied by the current {{site.tseeprodname}} policy. Normally the `Drop` or `LogAndDrop` value should be used. However when experimenting or debugging a scenario that is not behaving as you expect, the `Accept` and `LogAndAccept` values can be useful: then the packet will be still be allowed through. When one of the `LogAnd...` values is set, each denied packet is logged in syslog.\* | `Drop`, `Accept`, `LogAndDrop`, `LogAndAccept` | string | `Drop` |
| chainInsertMode                    | Controls whether Felix hooks the kernel's top-level iptables chains by inserting a rule at the top of the chain or by appending a rule at the bottom. `Insert` is the safe default since it prevents {{site.tseeprodname}}'s rules from being bypassed.  If you switch to `Append` mode, be sure that the other rules in the chains signal acceptance by falling through to the {{site.tseeprodname}} rules, otherwise the {{site.tseeprodname}} policy will be bypassed. | `Insert`, `Append` | string | `Insert` |
| defaultEndpointToHostAction        | This parameter controls what happens to traffic that goes from a workload endpoint to the host itself (after the traffic hits the endpoint egress policy).  By default {{site.tseeprodname}} blocks traffic from workload endpoints to the host itself with an iptables "DROP" action. If you want to allow some or all traffic from endpoint to host, set this parameter to `Return` or `Accept`.  Use `Return` if you have your own rules in the iptables "INPUT" chain; {{site.tseeprodname}} will insert its rules at the top of that chain, then `Return` packets to the "INPUT" chain once it has completed processing workload endpoint egress policy.  Use `Accept` to unconditionally accept packets from workloads after processing workload endpoint egress policy. | `Drop`, `Return`, `Accept` | string | `Drop` |
| failsafeInboundHostPorts           | UDP/TCP protocol/port pairs that Felix will allow incoming traffic to host endpoints on irrespective of the security policy. This is useful to avoid accidentally cutting off a host with incorrect configuration.  The default value allows SSH access, etcd, BGP and DHCP. |  | List of [ProtoPort](#protoport) | {::nomarkdown}<p><code> - protocol: tcp<br>&nbsp;&nbsp;port: 22<br>- protocol: udp<br>&nbsp;&nbsp;port: 68<br>- protocol: tcp<br>&nbsp;&nbsp;port: 179<br>- protocol: tcp<br>&nbsp;&nbsp;port: 2379<br>- protocol: tcp<br>&nbsp;&nbsp;port: 2380<br>- protocol: tcp<br>&nbsp;&nbsp;port: 6666<br>- protocol: tcp<br>&nbsp;&nbsp;port: 6667</code></p>{:/} |
| failsafeOutboundHostPorts          | UDP/TCP protocol/port pairs that Felix will allow outgoing traffic from host endpoints to irrespective of the security policy. This is useful to avoid accidentally cutting off a host with incorrect configuration.  The default value opens etcd's standard ports to ensure that Felix does not get cut off from etcd as well as allowing DHCP and DNS. | | List of [ProtoPort](#protoport) | {::nomarkdown}<p><code> - protocol: udp<br>&nbsp;&nbsp;port: 53<br>- protocol: udp<br>&nbsp;&nbsp;port: 67<br>- protocol: tcp<br>&nbsp;&nbsp;port: 179<br>- protocol: tcp<br>&nbsp;&nbsp;port: 2379<br>- protocol: tcp<br>&nbsp;&nbsp;port: 2380<br>- protocol: tcp<br>&nbsp;&nbsp;port: 6666<br>- protocol: tcp<br>&nbsp;&nbsp;port: 6667</code></p>{:/} |
| ignoreLooseRPF                     | Set to `true` to allow Felix to run on systems with loose reverse path forwarding (RPF). **Warning**: {{site.tseeprodname}} relies on "strict" RPF checking being enabled to prevent workloads, such as VMs and privileged containers, from spoofing their IP addresses and impersonating other workloads (or hosts).  Only enable this flag if you need to run with "loose" RPF and you either trust your workloads or have another mechanism in place to prevent spoofing.  | `true`, `false` | boolean | `false` |
| interfaceExclude                   | A comma-separated list of interface names that should be excluded when Felix is resolving host endpoints.  The default value ensures that Felix ignores Kubernetes' internal `kube-ipvs0` device. If you want to exclude multiple interface names using a single value, the list supports regular expressions. For regular expressions you must wrap the value with `/`. For example having values `/^kube/,veth1` will exclude all interfaces that begin with `kube` and also the interface `veth1`. | string | string | `kube-ipvs0` |
| interfacePrefix                    | The interface name prefix that identifies workload endpoints and so distinguishes them from host endpoint interfaces.  Note: in environments other than bare metal, the orchestrators configure this appropriately.  For example our Kubernetes and Docker integrations set the 'cali' value, and our OpenStack integration sets the 'tap' value. | string | string | `cali` |
| ipipEnabled                        | Whether Felix should configure an IPinIP interface on the host. Set automatically to `true` by `{{site.nodecontainer}}` or `calicoctl` when you create an IPIP-enabled pool. | `true`, `false` | boolean | `false` |
| ipipMTU                            | The MTU to set on the tunnel device. See [Configuring MTU]({{site.url}}/{{page.version}}/networking/mtu) | int | int | `1440` |
| ipsetsRefreshInterval              | Period, in seconds, at which Felix re-checks the IP sets in the dataplane to ensure that no other process has accidentally broken {{site.tseeprodname}}'s rules. Set to 0 to disable IP sets refresh.  Note: the default for this value is lower than the other refresh intervals as a workaround for a [Linux kernel bug](https://github.com/projectcalico/felix/issues/1347) that was fixed in kernel version 4.11. If you are using v4.11 or greater you may want to set this to a higher value to reduce Felix CPU usage. | int | int | `10` |
| iptablesFilterAllowAction          | This parameter controls what happens to traffic that is accepted by a Felix policy chain in the iptables filter table (i.e. a normal policy chain). The default will immediately `Accept` the traffic. Use `Return` to send the traffic back up to the system chains for further processing.| `Accept`, `Return` |  string | `Accept` |
| iptablesBackend                    | This parameter controls which variant of iptables binary Felix uses.  If using Felix on a system that uses the netfilter-backed iptables binaries, set this to `NFT`. | `Legacy`, `NFT` | string | `Legacy` |
| iptablesLockFilePath               | Location of the iptables lock file.  You may need to change this if the lock file is not in its standard location (for example if you have mapped it into Felix's container at a different path). | string | string | `/run/xtables.lock` |
| iptablesLockProbeIntervalMillis    | Time, in milliseconds, that Felix will wait between attempts to acquire the iptables lock if it is not available.  Lower values make Felix more responsive when the lock is contended, but use more CPU. | int | int | `50` |
| iptablesLockTimeoutSecs            | Time, in seconds, that Felix will wait for the iptables lock, or 0, to disable.  To use this feature, Felix must share the iptables lock file with all other processes that also take the lock.  When running Felix inside a container, this requires the /run directory of the host to be mounted into the {{site.nodecontainer}} or calico/felix container. | int | int | `0` (Disabled) |
| iptablesMangleAllowAction          | This parameter controls what happens to traffic that is accepted by a Felix policy chain in the iptables mangle table (i.e. a pre-DNAT policy chain). The default will immediately `Accept` the traffic. Use `Return` to send the traffic back up to the system chains for further processing. | `Accept`, `Return` |  string | `Accept` |
| iptablesMarkMask                   | Mask that Felix selects its IPTables Mark bits from. Should be a 32 bit hexadecimal number with at least 8 bits set, none of which clash with any other mark bits in use on the system. | netmask | netmask | `0xffff0000` |
| iptablesNATOutgoingInterfaceFilter | This parameter can be used to limit the host interfaces on which Calico will apply SNAT to traffic leaving a Calico IPAM pool with "NAT outgoing" enabled.  This can be useful if you have a main data interface, where traffic should be SNATted and a secondary device (such as the docker bridge) which is local to the host and doesn't require SNAT.  This parameter uses the iptables interface matching syntax, which allows `+` as a wildcard.  Most users will not need to set this.  Example: if your data interfaces are eth0 and eth1 and you want to exclude the docker bridge, you could set this to `eth+` | string | string | `""` |
| iptablesPostWriteCheckIntervalSecs | Period, in seconds, after Felix has done a write to the dataplane that it schedules an extra read back in order to check the write was not clobbered by another process.  This should only occur if another application on the system doesn't respect the iptables lock. | int | int | `1` |
| iptablesRefreshIntervalSecs        | Period, in seconds, at which Felix re-checks all iptables state to ensure that no other process has accidentally broken {{site.tseeprodname}}'s rules. Set to 0 to disable iptables refresh. | int | int | `90` |
| ipv6Support                        | IPv6 support for Felix | `true`, `false` | boolean | `true` |
| logFilePath                        | The full path to the Felix log. Set to `""` to disable file logging. | string | string | `/var/log/calico/felix.log` |
| logPrefix                          | The log prefix that Felix uses when rendering LOG rules. | string | string | `calico-packet` |
| logSeverityFile                    | The log severity above which logs are sent to the log file. | Same as logSeveritySys | string | `Info` |
| logSeverityScreen                  | The log severity above which logs are sent to the stdout. | Same as logSeveritySys | string | `Info` |
| logSeveritySys                     | The log severity above which logs are sent to the syslog. Set to `""` for no logging to syslog. | `Debug`, `Info`, `Warning`, `Error`, `Fatal` | string | `Info` |
| maxIpsetSize                       | Maximum size for the ipsets used by Felix to implement tags. Should be set to a number that is greater than the maximum number of IP addresses that are ever expected in a tag. | int | int | `1048576` |
| metadataAddr                       | The IP address or domain name of the server that can answer VM queries for cloud-init metadata. In OpenStack, this corresponds to the machine running nova-api (or in Ubuntu, nova-api-metadata). A value of `none` (case insensitive) means that Felix should not set up any NAT rule for the metadata path.  | IPv4, hostname, none | string | `127.0.0.1` |
| metadataPort                       | The port of the metadata server. This, combined with global.MetadataAddr (if not 'None'), is used to set up a NAT rule, from 169.254.169.254:80 to MetadataAddr:MetadataPort. In most cases this should not need to be changed. | int | int | `8775` |
| natOutgoingAddress                 | The source address to use for outgoing NAT. By default an iptables MASQUERADE rule determines the source address which will use the address on the host interface the traffic leaves on. | IPV4 | string | `""` |
| policySyncPathPrefix               | File system path where Felix notifies services of policy changes over Unix domain sockets. This is only required if you're configuring [application layer policy]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/app-layer-policy). Set to `""` to disable. | string | string | `""` |
| prometheusGoMetricsEnabled         | Set to `false` to disable Go runtime metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. | `true`, `false` | boolean | `true` |
| prometheusMetricsEnabled           | Set to `true` to enable the experimental Prometheus metrics server in Felix. | `true`, `false` | boolean | `false` |
| prometheusMetricsPort              | Experimental: TCP port that the Prometheus metrics server should bind to. | int | int | `9091` |
| prometheusProcessMetricsEnabled    | Set to `false` to disable process metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. | `true`, `false` | boolean | `true` |
| prometheusReporterEnabled | Set to `true` to enable configure Felix to keep count of recently denied packets and publish these as Prometheus metrics. Refer to the [Metrics]({{site.url}}/{{page.version}}/security/metrics/metrics) section for more details. Note that denied packet metrics are independent of the `dropActionOverride` setting.  Specifically, if packets that would normally be denied are being allowed through by a setting of `Accept` or `LogAndAccept`, those packets still get counted as denied packets. | `true`, `false` | boolean | `false` |
| prometheusReporterPort | The TCP port on which to report denied packet metrics, if `prometheusReporterEnabled` is set to `true`. |  |  | `9092` |
| reportingIntervalSecs              | Interval at which Felix reports its status into the datastore or 0 to disable.  Must be non-zero in OpenStack deployments. | int | int | `30` |
| reportingTTLSecs                   | Time-to-live setting for process-wide status reports. | int | int | `90` |
| routeRefreshIntervalSecs           | Period, in seconds, at which Felix re-checks the routes in the dataplane to ensure that no other process has accidentally broken {{site.tseeprodname}}'s rules. Set to 0 to disable route refresh. | int | int | `90` |
| ipsecMode                  | Controls which mode IPsec is operating on. The only supported value is `PSK`. An empty value means IPsec is not enabled. | PSK | string | `""` |
| ipsecAllowUnsecuredTraffic | When set to `false`, only IPsec-protected traffic will be allowed on the packet paths where IPsec is supported.  When set to `true`, IPsec will be used but non-IPsec traffic will be accepted.  In general, setting this to `true` is less safe since it allows an attacker to inject packets.  However, it is useful when transitioning from non-IPsec to IPsec since it allows traffic to flow while the cluster negotiates the IPsec mesh.  | `true`, `false` | boolean | `false` |
| ipsecIKEAlgorithm          | IPsec IKE algorithm. Default is NIST suite B recommendation.| string  | string | `aes128gcm16-prfsha256-ecp256` |
| ipsecESPAlgorithm          | IPsec ESP algorithm. Default is NIST suite B recommendation.| string  | string | `aes128gcm16-ecp256`
| ipsecLogLevel              | Controls log level for IPsec components. Set to `None` for no logging. | `None`, `Notice`, `Info`, `Debug`, `Verbose` | string | `Info` |
| ipsecPSKFile               | The path to the pre shared key file for IPsec. | string | string | `""` |
| flowLogsFileEnabled   | Set to `true`, enables flow logs. If set to `false` no flow logging will occur. Flow logs are written to a file `flows.log` and sent to Elasticsearch. The location of this file can be configured using the `flowLogsFileDirectory` field. File rotation settings for this `flows.log` file can be configured using the fields `flowLogsFileMaxFiles` and `flowLogsFileMaxFileSizeMB`. Note that flow log exports to Elasticsearch are dependent on flow logs getting written to this file. Setting this parameter to `false` will disable flow logs. | `true`, `false` | boolean | `false` |
| flowLogsFileDirectory | Set the directory where flow logs files are stored. This parameter only takes effect when `flowLogsFileEnabled` is set to `true`. | string | string | `/var/log/calico/flowlogs` |
| flowLogsFileMaxFiles  | Set the number of log files to keep. This parameter only takes effect when `flowLogsFileEnabled` is set to `true`. | int | int | `5` |
| flowLogsFileMaxFileSizeMB | Set the max size in MB of flow logs files before rotation. This parameter only takes effect when `flowLogsFileEnabled` is set to `true`. | int | int | `100` |
| flowLogsFlushInterval | The period, in seconds, at which Felix exports the flow logs. | int | int | `300` |
| flowLogsFileAggregationKindForAllowed | How much to aggregate the flow logs sent to Elasticsearch for allowed traffic.  Bear in mind that changing this value may have a dramatic impact on the volume of flow logs sent to Elasticsearch. | 0-2 | [AggregationKind](#aggregationkind) | 2 |
| flowLogsFileAggregationKindForDenied | How much to aggregate the flow logs sent to Elasticsearch for denied traffic.  Bear in mind that changing this value may have a dramatic impact on the volume of flow logs sent to Elasticsearch. | 0-2 | [AggregationKind](#aggregationkind) | 1 |
| flowLogsFileIncludeLabels | When set to `true`, include source and destination endpoint labels in the aggregated flow log. Note that only Kubernetes endpoints or network sets are included; arbitrary networks do not contain labels. | `true`, `false` | boolean | `false` |
| flowLogsFileIncludePolicies | When set to `true`, include all policies in the aggregated flow logs that acted upon and matches the flow log traffic. | `true`, `false` | boolean | `false` |
| flowLogsEnableNetworkSets | When set to `true`, include an arbitrary network set in the aggregated flow log that matches the IP address of the flow log endpoint. | `true`, `false` | boolean | `false` |
| sidecarAccelerationEnabled         | Enable experimental acceleration between application and proxy sidecar when using [application layer policy]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/app-layer-policy). [Default: `false`] | `true`, `false` | boolean | `false` |
| vxlanEnabled                       | Automatically set when needed, you shouldn't need to change this setting: whether Felix should create the VXLAN tunnel device for VXLAN networking. | `true`, `false` | boolean | `false` |
| vxlanMTU                           | MTU to use for the VXLAN tunnel device. | int | int | `1410` |
| vxlanPort                          | Port to use for VXLAN traffic. A value of `0` means "use the kernel default". | int | int | `4789` |
| vxlanVNI                           | Virtual network ID to use for VXLAN traffic. A value of `0` means "use the kernel default". | int | int | `4096` |
| dnsCacheFile                       | The name of the file that Felix uses to preserve learnt DNS information when restarting. | file name | string | `/var/run/calico/felix-dns-cache.txt` |
| dnsCacheSaveInterval               | The period, in seconds, at which Felix saves learnt DNS information to the cache file. | int | int | `60` |
| dnsTrustedServers                  | The DNS servers that Felix should trust. Each entry here must be an IP, or `k8s-service:<name>`, where `<name>` is the name of a Kubernetes Service in the `kube-system` namespace. | IPs or service names | comma-separated strings | `k8s-service:kube-dns` |
| dnsLogsFileEnabled                 | Set to `true`, enables DNS logs. If set to `false` no DNS logging will occur. DNS logs are written to a file `dns.log` and sent to Elasticsearch. The location of this file can be configured using the `DNSLogsFileDirectory` field. File rotation settings for this `dns.log` file can be configured using the fields `DNSLogsFileMaxFiles` and `DNSLogsFileMaxFileSizeMB`. Note that DNS log exports to Elasticsearch are dependent on DNS logs getting written to this file. Setting this parameter to `false` will disable DNS logs. | `true`, `false` | boolean | `false` |
| dnsLogsFileDirectory               | The directory where DNS logs files are stored. This parameter only takes effect when `DNSLogsFileEnabled` is `true`. | directory | string | `/var/log/calico/dnslogs` |
| dnsLogsFileMaxFiles                | The number of files to keep when rotating DNS log files. This parameter only takes effect when `DNSLogsFileEnabled` is `true`. | int | int | `5` |
| dnsLogsFileMaxFileSizeMB           | The max size in MB of DNS log files before rotation. This parameter only takes effect when `DNSLogsFileEnabled` is `true`. | int | int | `100` |
| dnsLogsFlushInterval               | The period, in seconds, at which Felix exports DNS logs. | int | int | `300` |
| dnsLogsFileAggregationKind         | How much to aggregate DNS logs.  Bear in mind that changing this value may have a dramatic impact on the volume of flow logs sent to Elasticsearch.  `0` means no aggregation, `1` means aggregate similar DNS logs from workloads in the same ReplicaSet. | `0`,`1` | int | `1` |
| dnsLogsFileIncludeLabels           | Whether to include client and server workload labels in DNS logs. | `true`, `false` | boolean | `true` |
| dnsLogsFilePerNodeLimit            | Limit on the number of DNS logs that can be emitted within each flush interval.  When this limit has been reached, Felix counts the number of unloggable DNS responses within the flush interval, and emits a WARNING log with that count at the same time as it flushes the buffered DNS logs. | int | int | `0` (no limit) |


\* When `dropActionOverride` is set to `LogAndDrop` or `LogAndAccept`, the `syslog` entries look something like the following.
   ```
   May 18 18:42:44 ubuntu kernel: [ 1156.246182] calico-drop: IN=tunl0 OUT=cali76be879f658 MAC= SRC=192.168.128.30 DST=192.168.157.26 LEN=60 TOS=0x00 PREC=0x00 TTL=62 ID=56743 DF PROTO=TCP SPT=56248 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0xa000000
   ```
   {: .no-select-button}

\*\* Duration is denoted by the numerical amount followed by the unit of time. Valid units of time include nanoseconds (ns), microseconds (µs), milliseconds (ms), seconds (s), minutes (m), and hours (h). Units of time can also be used together e.g. `3m30s` to represent 3 minutes and 30 seconds. Any amounts of time that can be converted into larger units of time will be converted e.g. `90s` will become `1m30s`.

#### ProtoPort

| Field    | Description          | Accepted Values   | Schema |
|----------|----------------------|-------------------|--------|
| port     | The exact port match | 0-65535           | int    |
| protocol | The protocol match   | tcp, udp          | string |

#### AggregationKind

| Value | Description |
|-------|-------------|
| 0     | No aggregation |
| 1     | Aggregate all flows that share a source port on each node |
| 2     | Aggregate all flows that share source ports or are from the same ReplicaSet on each node |

### Supported operations

| Datastore type        | Create  | Delete | Delete (Global `default`)  |  Update  | Get/List | Notes
|-----------------------|---------|--------|----------------------------|----------|----------|------
| etcdv3                | Yes     | Yes    | No                         | Yes      | Yes      |
| Kubernetes API server | Yes     | Yes    | No                         | Yes      | Yes      |
