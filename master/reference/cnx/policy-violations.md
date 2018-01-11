---
title: Policy Violation Monitoring & Reporting
---

{{site.prodname}} adds the ability to monitor violations of policy
configured in your cluster. By defining a set of simple rules and thresholds,
you can monitor denied traffic and receive alerts when it exceeds configured
thresholds.

### Architecture

```

 +-----------------+
 | Host            |
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

Policy Violation & Reporting is accomplished using 3 key pieces:

1. A {{site.prodname}} specific Felix binary running inside `{{site.noderunning}}` container
   monitors the host for Denied packets and collects metrics.
2. Prometheus Server(s) deployed as part of the {{site.prodname}} manifest scrapes
   every configured `{{site.noderunning}}` target. Alerting rules querying denied packet
   metrics are configured in Prometheus and when triggered, fire alerts to
   the Prometheus Alertmanager.
3. Prometheus Alertmanager (or simply Alertmanager), also deployed as part of
   the _CNX_ manifest, receives alerts from Prometheus and forwards
   alerts to various alerting mechanisms such as _Pager Duty_, or _OpsGenie_.

### Metrics

The metrics generated are:

- `calico_denied_packets` - Total number of packets denied by {{site.prodname}} policies.
- `calico_denied_bytes` - Total number of bytes denied by {{site.prodname}} policies.

Using these metrics, one can identify the policy that denied packets as well as
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

See
the
[Felix configuration reference]({{site.baseurl}}/{{page.version}}/reference/felix/configuration#tigera-cnx-specific-configuration) for
the settings that control the reporting of these metrics.  CNX manifests
normally set `PrometheusReporterEnabled=true` and
`PrometheusReporterPort=9081`, so these metrics are available on each compute
node at `http://<node-IP>:9081/metrics`.

### Lifetime of a Metric

#### When are Metrics Generated?

Metrics with a `{policy, srcIP}` label pair will only be generated at a node
when there are packets directed at an _Endpoint_ that are being actively denied
by a policy. Once generated they stay alive for 60 seconds after the last
packet was denied by a policy.

#### Prometheus

Once Prometheus scrapes a node and collects denied packet metrics, it will be
available at Prometheus until the metric is considered _stale_, i.e.,
Prometheus has not seen any updates to this metric for some time. This time is
configurable and details on how to do this are available in the
[Prometheus Configuration]({{site.baseurl}}/{{page.version}}/usage/configuration/prometheus) document.

#### Metric Resets and Empty Responses

Because of metrics being expired, as just described, it is entirely possible
for a GET on the denied metrics URL to return no information.  This is expected
if there have not been any packets being denied by a policy on that node, in
the last 60 seconds.
