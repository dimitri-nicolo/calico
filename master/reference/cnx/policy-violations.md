---
title: Policy Monitoring & Reporting
---

{{site.prodname}} adds the ability to monitor effects of policies configured in your cluster.
By defining a set of simple rules and thresholds, you can monitor traffic metrics and receive
alerts when it exceeds configured thresholds.

### Architecture

```
                                      +------------+
                                      |            |
                                      |    CNX     |
                                      |   Manager  |
                                      |            |
                                      |            |
                                      |            |
                                      +------------+
                                            ^
                                            |
                                            |
                                            |
 +-----------------+                        |
 | Host            |                        |
 | +-----------------+     denied     +------------+     +------------+
 | | Host            |------------->--|            |     |            |--->--
 | | +-----------------+   packet     | Prometheus |     | Prometheus |        alert
 +-| | Host            |----------->--|   Server   |-->--|   Alert    |--->--
   | |   +----------+  |   metrics    |            |     |  Manager   |      mechanisms
   +-|   |  Felix   |-------------->--|            |     |            |--->--
     |   +----------+  |              +------------+     +------------+
     +-----------------+                    ^                   ^
                                            |                   |
                             Collect and store metrics.     Web UI for accessing alert
                             WebUI for accessing and        states.
                             querying metrics.              Configure fan out
                             Configure alerting rules.      notifications to different
                                                            alert receivers.
```

Policy Inspection & Reporting is accomplished using 4 key pieces:

1. A {{site.prodname}} specific Felix binary running inside `{{site.noderunning}}` container 
   monitors the host for denied/allowed packets and collects metrics.
2. Prometheus Server(s) deployed as part of the {{site.prodname}} manifest scrapes
   every configured `{{site.noderunning}}` target. Alerting rules querying denied packet
   metrics are configured in Prometheus and when triggered, fire alerts to
   the Prometheus Alertmanager.
3. Prometheus Alertmanager (or simply Alertmanager), deployed as part of
   the {{site.prodname}} manifest, receives alerts from Prometheus and forwards
   alerts to various alerting mechanisms such as _Pager Duty_, or _OpsGenie_.
4. {{site.prodname}} Manager (UI), also deployed as part of the {{site.prodname}} manifest, 
   processes the metrics using pre-defined prometheus queries and provides with intuitive 
   dashboards and associated workflows.

### {{site.prodname}} Manager Dashboard 

In the Dashboard, you will find graphs associated with allowed and denied packets, bytes, and connections. 
The graphs represent the rates at which the packets, bytes, and connections are being allowed or denied.

In the Dashboard, you will also find a **Packets by Policy** bar graph. Each individual bar represents
a policy that has either denied or allowed a packet.


### The Prometheus Metrics

Internally, the metrics generated are:

- `calico_denied_packets` - Total number of packets denied by {{site.prodname}} policies.
- `calico_denied_bytes` - Total number of bytes denied by {{site.prodname}} policies.
- `cnx_policy_rule_packets` - Sum of allowed/denied packets over rules processed by
  {{site.prodname}} policies.
- `cnx_policy_rule_bytes` - Sum of allowed/denied bytes over rules processed by 
  {{site.prodname}} policies.
- `cnx_policy_rule_connections` - Sum of connections over rules processed by {{site.prodname}} 
  policies.

The metrics `calico_denied_packets` and `calico_denied_bytes` have the labels `policy` and `srcIP`.
Using these two metrics, one can identify the policy that denied packets as well as
the source IP Address of the packets that were denied by this policy. Using
Prometheus terminology, `calico_denied_packets` is the metric Name and `policy`
and `srcIP` are labels. Each one of these metrics will be available as a
combination of `{policy, srcIP}`. An HTTP GET request to retrieve metrics from a
`{{site.noderunning}}` container will provide output like this:

```
# HELP calico_denied_bytes Total number of bytes denied by calico policies.
# TYPE calico_denied_bytes gauge
calico_denied_bytes{policy="profile/k8s_ns.ns-0/0/deny",srcIP="10.245.13.133"} 300
calico_denied_bytes{policy="profile/k8s_ns.ns-0/0/deny",srcIP="10.245.13.149"} 840
# HELP calico_denied_packets Total number of packets denied by calico policies.
# TYPE calico_denied_packets gauge
calico_denied_packets{policy="profile/k8s_ns.ns-0/0/deny",srcIP="10.245.13.133"} 5
calico_denied_packets{policy="profile/k8s_ns.ns-0/0/deny",srcIP="10.245.13.149"} 14
```

This means that the profile `k8s_ns.ns-0` denied 5 packets (totaling 300 bytes)
originating from the IP Address "10.245.13.133" and the same profile denied 14
packets originating from the IP Address "10.245.13.149".

The metrics `cnx_policy_rule_packets`, `cnx_policy_rule_bytes` and `cnx_policy_rule_connections` have the
labels: `tier`, `policy`, `namespace`, `rule_index`, `action`, `traffic_direction`, `rule_direction`.

Using these metrics, one can identify allow, denied, passed byte rate and packet rate both inbound and outbound, indexed by both policy and rule. {{site.prodname}} Manager Dashboard makes heavy usage of these metrics by issuing queries such as: 
```
- Query counts for rules: Packet rates for specific rule by traffic_direction
sum(irate(cnx_policy_rule_packets{namespace="namespace-2",policy="policy-0",rule_direction="ingress",rule_index="rule-5",tier="tier-0"}[30s])) without (instance)

- Query counts for rules: Packet rates for each rule in a policy by traffic_direction
sum(irate(cnx_policy_rule_packets{namespace="namespace-2",policy="policy-0",tier="tier-0"}[30s])) without (instance)

- Query counts for a single policy by traffic_direction and action
sum(irate(cnx_policy_rule_packets{namespace="namespace-2",policy="policy-0",tier="tier-0"}[30s])) without (instance,rule_index,rule_direction)

- Query counts for all policies across all tiers by traffic_direction and action
sum(irate(cnx_policy_rule_packets[30s])) without (instance,rule_index,rule_direction)
```

See the 
[Felix configuration reference]({{site.baseurl}}/{{page.version}}/reference/felix/configuration#tigera-cnx-specific-configuration) for
the settings that control the reporting of these metrics.  CNX manifests
normally set `PrometheusReporterEnabled=true` and
`PrometheusReporterPort=9081`, so these metrics are available on each compute
node at `http://<node-IP>:9081/metrics`.

### Lifetime of a Metric

#### When are Metrics Generated?

Metrics will only be generated at a node when there are packets directed at an _Endpoint_ 
that are being actively profiled by a policy. Once generated they stay alive for 60 seconds 
after the last packet was denied by a policy.

#### Prometheus

Once Prometheus scrapes a node and collects packet metrics, it will be
available at Prometheus until the metric is considered _stale_, i.e.,
Prometheus has not seen any updates to this metric for some time. This time is
configurable and details on how to do this are available in the
[Prometheus Configuration]({{site.baseurl}}/{{page.version}}/usage/configuration/prometheus) document.

#### Metric Resets and Empty Responses

Because of metrics being expired, as just described, it is entirely possible
for a GET on the metrics URL to return no information.  This is expected
if there have not been any packets being denied by a policy on that node, in
the last 60 seconds.

