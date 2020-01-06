---
title: Configuring Felix
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
> refer to [Felix Configuration Resource](../calicoctl/resources/felixconfig).
{: .alert .alert-info}

#### General configuration

| Configuration parameter           | Environment variable                    | Description  | Schema |
| --------------------------------- | --------------------------------------- | -------------| ------ |
| `DatastoreType`                   | `FELIX_DATASTORETYPE`                   | The datastore that Felix should read endpoints and policy information from. [Default: `etcdv3`] | `etcdv3`, `kubernetes`|
| `FelixHostname`                   | `FELIX_FELIXHOSTNAME`                   | The hostname Felix reports to the plugin. Should be used if the hostname Felix autodetects is incorrect or does not match what the plugin will expect. [Default: `socket.gethostname()`] | string |
| `LogFilePath`                     | `FELIX_LOGFILEPATH`                     | The full path to the Felix log. Set to `none` to disable file logging. [Default: `/var/log/calico/felix.log`] | string |
| `LogSeveritySys`                  | `FELIX_LOGSEVERITYSYS`                  | The log severity above which logs are sent to the syslog. Set to `""` for no logging to syslog. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `LogSeverityFile`                 | `FELIX_LOGSEVERITYFILE`                 | The log severity above which logs are sent to the log file. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `LogSeverityScreen`               | `FELIX_LOGSEVERITYSCREEN`               | The log severity above which logs are sent to the stdout. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `PrometheusMetricsEnabled`        | `FELIX_PROMETHEUSMETRICSENABLED`        | Set to `true` to enable the experimental Prometheus metrics server in Felix. [Default: `false`] | boolean |
| `PrometheusMetricsPort`           | `FELIX_PROMETHEUSMETRICSPORT`           | Experimental: TCP port that the Prometheus metrics server should bind to. [Default: `9091`] | int |
| `PrometheusGoMetricsEnabled`      | `FELIX_PROMETHEUSGOMETRICSENABLED`      | Set to `false` to disable Go runtime metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. [Default: `true`]  | boolean |
| `PrometheusProcessMetricsEnabled` | `FELIX_PROMETHEUSPROCESSMETRICSENABLED` | Set to `false` to disable process metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. [Default: `true`] | boolean |
| `UsageReportingEnabled`           | `FELIX_USAGEREPORTINGENABLED`           | Reports anonymous {{site.tseeprodname}} version number and cluster size to projectcalico.org. Logs warnings returned by the usage server. For example, if a significant security vulnerability has been discovered in the version of {{site.tseeprodname}} being used. [Default: `true`] | boolean |
| `UsageReportingInitialDelaySecs`  | `FELIX_USAGEREPORTINGINITIALDELAYSECS`  | Minimum delay before first usage report, in seconds. [Default: `300`] | int |
| `UsageReportingIntervalSecs`      | `FELIX_USAGEREPORTINGINTERVALSECS`      | Interval at which to make usage reports, in seconds. [Default: `86400`] | int |
| `FailsafeInboundHostPorts`        | `FELIX_FAILSAFEINBOUNDHOSTPORTS`        | Comma-delimited list of UDP/TCP ports that Felix will allow incoming traffic to host endpoints on irrespective of the security policy. This is useful to avoid accidentally cutting off a host with incorrect configuration. Each port should be specified as `tcp:<port-number>` or `udp:<port-number>`. For backwards compatibility, if the protocol is not specified, it defaults to "tcp". To disable all inbound host ports, use the value `none`. The default value allows ssh access, DHCP, BGP and etcd. [Default: `tcp:22, udp:68, tcp:179, tcp:2379, tcp:2380, tcp:6666, tcp:6667`] | string |
| `FailsafeOutboundHostPorts`       | `FELIX_FAILSAFEOUTBOUNDHOSTPORTS`       | Comma-delimited list of UDP/TCP ports that Felix will allow outgoing traffic from host endpoints to irrespective of the security policy. This is useful to avoid accidentally cutting off a host with incorrect configuration. Each port should be specified as `tcp:<port-number>` or `udp:<port-number>`.  For backwards compatibility, if the protocol is not specified, it defaults to "tcp". To disable all outbound host ports, use the value `none`. The default value opens etcd's standard ports to ensure that Felix does not get cut off from etcd as well as allowing DHCP, DNS, BGP. [Default: `udp:53, udp:67, tcp:179, tcp:2379, tcp:2380, tcp:6666, tcp:6667`]  | string |
| `ReportingIntervalSecs`           | `FELIX_REPORTINGINTERVALSECS`           | Interval at which Felix reports its status into the datastore or `0` to disable. Must be non-zero in OpenStack deployments. [Default: `30`] | int |
| `ReportingTTLSecs`                | `FELIX_REPORTINGTTLSECS`                | Time-to-live setting for process-wide status reports. [Default: `90`] | int |
| `IpInIpMtu`                       | `FELIX_IPINIPMTU`                       | The MTU to set on the tunnel device. See [Configuring MTU]({{site.baseurl}}/{{page.version}}/usage/configuration/mtu) [Default: `1440`] | int |

#### etcdv3 datastore configuration

| Configuration parameter | Environment variable  | Description | Schema |
| ----------------------- | --------------------- | ----------- | ------ |
| `EtcdEndpoints`         | `FELIX_ETCDENDPOINTS` | Comma-delimited list of etcd endpoints to connect to. Example: `http://etcd1:2379,http://etcd2:2379`. | `<scheme>://<ip-or-fqdn>:<port>` |
| `EtcdKeyFile`           | `FELIX_ETCDKEYFILE`   | The full path to the etcd private key file. | string |
| `EtcdCertFile`          | `FELIX_ETCDCERTFILE`  | The full path to the etcd certificate file. | string |
| `EtcdCaFile`            | `FELIX_ETCDCAFILE`    | The full path to the etcd Certificate Authority certificate file. To disable authentication of the server by Felix, set the value to `none`. [Default: `/etc/ssl/certs/ca-certificates.crt`] | string |


#### Kubernetes datastore configuration

The Kubernetes datastore driver reads its configuration from Kubernetes-provided environment variables.

#### iptables dataplane configuration

| Configuration parameter              | Environment variable                       | Description | Schema |
| ------------------------------------ | ------------------------------------------ | ----------- | ------ |
| `DefaultEndpointToHostAction`        | `FELIX_DEFAULTENDPOINTTOHOSTACTION`        | This parameter controls what happens to traffic that goes from a workload endpoint to the host itself (after the traffic hits the endpoint egress policy). By default {{site.tseeprodname}} blocks traffic from workload endpoints to the host itself with an iptables `Drop` action. If you want to allow some or all traffic from endpoint to host, set this parameter to `Return` or `Accept`.  Use `Return` if you have your own rules in the iptables "INPUT" chain; {{site.tseeprodname}} will insert its rules at the top of that chain, then `Return` packets to the "INPUT" chain once it has completed processing workload endpoint egress policy. Use `Accept` to unconditionally accept packets from workloads after processing workload endpoint egress policy. [Default: `Drop`] | `Drop`, `Return`, `Accept` |
| `IgnoreLooseRPF`                     | `FELIX_IGNORELOOSERPF`                     | Set to `true` to allow Felix to run on systems with loose reverse path forwarding (RPF). **Warning**: {{site.tseeprodname}} relies on "strict" RPF checking being enabled to prevent workloads, such as VMs and privileged containers, from spoofing their IP addresses and impersonating other workloads (or hosts). Only enable this flag if you need to run with "loose" RPF and you either trust your workloads or have another mechanism in place to prevent spoofing. | `true`,`false` |
| `InterfaceExclude`                   | `FELIX_INTERFACEEXCLUDE`                   | A comma-separated list of interface names that should be excluded when Felix is resolving host endpoints. The default value ensures that Felix ignores Kubernetes' internal `kube-ipvs0` device. [Default: `kube-ipvs0`] | string |
| `IpInIpEnabled`                      | `FELIX_IPINIPENABLED`                      | Whether Felix should configure an IPinIP interface on the host. Set automatically to `true` by `{{site.nodecontainer}}` or `calicoctl` when you create an IPIP-enabled pool. [Default: `false`] | boolean |
| `IptablesFilterAllowAction`          | `FELIX_IPTABLESALLOWACTION`                | This parameter controls what happens to traffic that is allowed by a Felix policy chain in the iptables filter table (i.e., a normal policy chain). The default will immediately `Accept` the traffic. Use `Return` to send the traffic back up to the system chains for further processing. [Default: `Accept`]  | `Accept`, `Return` |
| `IptablesMangleAllowAction`          | `FELIX_IPTABLESALLOWACTION`                | This parameter controls what happens to traffic that is allowed by a Felix policy chain in the iptables mangle table (i.e., a pre-DNAT policy chain). The default will immediately `Accept` the traffic. Use `Return` to send the traffic back up to the system chains for further processing. [Default: `Accept`]  | `Accept`, `Return` |
| `IptablesMarkMask`                   | `FELIX_IPTABLESMARKMASK`                                     | Mask that Felix selects its IPTables Mark bits from. Should be a 32 bit hexadecimal number with at least 8 bits set, none of which clash with any other mark bits in use on the system. [Default: `0xff000000`] | netmask |
| `IptablesRefreshInterval`            | `FELIX_IPTABLESREFRESHINTERVAL`            | Period, in seconds, at which Felix re-checks all iptables state to ensure that no other process has accidentally broken {{site.tseeprodname}}'s rules. Set to 0 to disable iptables refresh. [Default: `90`] | int |
| `IptablesPostWriteCheckIntervalSecs` | `FELIX_IPTABLESPOSTWRITECHECKINTERVALSECS` | Period, in seconds, after Felix has done a write to the dataplane that it schedules an extra read back in order to check the write was not clobbered by another process.  This should only occur if another application on the system doesn't respect the iptables lock. [Default: `1`] | int |
| `RouteRefreshIntervalSecs`           | `FELIX_ROUTEREFRESHINTERVAL`               | Period, in seconds, at which Felix re-checks the routes in the dataplane to ensure that no other process has accidentally broken {{site.tseeprodname}}'s rules. Set to 0 to disable route refresh. [Default: `90`] | int |
| `IpsetsRefreshIntervalSecs`          | `FELIX_IPSETSREFRESHINTERVAL`              | Period, in seconds, at which Felix re-checks the IP sets in the dataplane to ensure that no other process has accidentally broken {{site.tseeprodname}}'s rules. Set to 0 to disable IP sets refresh.  Note: the default for this value is lower than the other refresh intervals as a workaround for a [Linux kernel bug](https://github.com/projectcalico/felix/issues/1347) that was fixed in kernel version 4.11. If you are using v4.11 or greater you may want to set this to, a higher value to reduce Felix CPU usage. [Default: `10`] | int |
| `MaxIpsetSize`                       | `FELIX_MAXIPSETSIZE`                       | Maximum size for the ipsets used by Felix to implement tags. Should be set to a number that is greater than the maximum number of IP addresses that are ever expected in a tag. [Default: `1048576`] | int |
| `ChainInsertMode`                    | `FELIX_CHAININSERTMODE`                    | Controls whether Felix hooks the kernel's top-level iptables chains by inserting a rule at the top of the chain or by appending a rule at the bottom.  `Insert` is the safe default since it prevents {{site.tseeprodname}}'s rules from being bypassed.  If you switch to `Append` mode, be sure that the other rules in the chains signal acceptance by falling through to the {{site.tseeprodname}} rules, otherwise the {{site.tseeprodname}} policy will be bypassed. [Default: `Insert`]  | `Insert`, `Append` |
| `LogPrefix`                          | `FELIX_LOGPREFIX`                          | The log prefix that Felix uses when rendering LOG rules. [Default: `calico-packet`] | string |
| `IptablesLockTimeoutSecs`            | `FELIX_IPTABLESLOCKTIMEOUTSECS`            | Time, in seconds, that Felix will wait for the iptables lock, or 0, to disable. To use this feature, Felix must share the iptables lock file with all other processes that also take the lock.  When running Felix inside a container, this requires the /run directory of the host to be mounted into the `{{site.nodecontainer}}` or `calico/felix` container. [Default: `0` disabled] | int |
| `IptablesLockFilePath`               | `FELIX_IPTABLESLOCKFILEPATH`               | Location of the iptables lock file.  You may need to change this if the lock file is not in its standard location (for example if you have mapped it into Felix's container at a different path). [Default: `/run/xtables.lock`]  | string |
| `IptablesLockProbeIntervalMillis`    | `FELIX_IPTABLESLOCKPROBEINTERVALMILLIS`    | Time, in milliseconds, that Felix will wait between attempts to acquire the iptables lock if it is not available.  Lower values make Felix more responsive when the lock is contended, but use more CPU. [Default: `50`]  | int |
| `NetlinkTimeoutSecs`                 | `FELIX_NETLINKTIMEOUTSECS`                 | Time, in seconds, that Felix will wait for netlink (i.e. routing table list/update) operations to complete before giving up and retrying. [Default: `10`] | float |


#### Bare metal specific configuration

| Configuration parameter | Environment variable    | Description | Schema |
| ----------------------- | ----------------------- | ----------- | ------ |
| `InterfacePrefix`       | `FELIX_INTERFACEPREFIX` | The interface name prefix that identifies workload endpoints and so distinguishes them from host endpoint interfaces.  Note: in environments other than bare metal, the orchestrators configure this appropriately.  For example our Kubernetes integration sets the `cali` value. [Default: `cali`] | string |

#### Tigera CNX specific configuration

| Setting                     | Environment variable               | Default  | Meaning                                 |
|-----------------------------|------------------------------------|----------|-----------------------------------------|
| `DropActionOverride`        | `FELIX_DROPACTIONOVERRIDE`         | `Drop`   | How to treat packets that are disallowed by the current Calico policy.  For more detail please see below. |
| `PrometheusReporterEnabled` | `FELIX_PROMETHEUSREPORTERENABLED`  | `false`  | Set to `true` to enable Prometheus reporting of denied packet metrics.  For more detail please see below. |
| `PrometheusReporterPort`    | `FELIX_PROMETHEUSREPORTERPORT`     | `9092`   | The TCP port on which to report denied packet metrics.  |

DropActionOverride controls what happens to each packet that is denied by
the current Calico policy - i.e. by the ordered combination of all the
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

When the reporting of denied packet metrics is enabled, Felix keeps counts of
recently denied packets and publishes these as Prometheus metrics on the port
configured by the `PrometheusReporterPort` setting.  Please
see
[Policy Violation Monitoring & Reporting]({{site.baseurl}}/{{page.version}}/reference/cnx/policy-violations) for
more details.

Note that denied packet metrics are independent of the DropActionOverride
setting.  Specifically, if packets that would normally be denied are being
allowed through by a setting of `Accept` or `LogAndAccept`, those packets
still contribute to the denied packet metrics as just described.

Environment variables
---------------------

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

### Datastore

Felix also reads configuration parameters from the datastore.  It supports
a global setting and a per-host override.

```
# Get the current felixconfig settings
$ calicoctl get felixconfig -o yaml > felix.yaml

# Modify logFilePath to your intended path, e.g. "/tmp/felix.log"
#   Global change: set name to "default"
#   Node-specific change: set name to the node name, e.g. "{{site.tseeprodname}}-Node-1"
$ vim felix.yaml

# Replace the current felixconfig settings
$ calicoctl replace -f felix.yaml
```

For more information, see [Felix Configuration Resource](../calicoctl/resources/felixconfig).

