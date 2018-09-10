---
title: Configuring Prometheus
---

Note: This document assumes that Prometheus and Alertmanager have been setup
using _Prometheus Operator_ as described
[here](../../getting-started/kubernetes/installation/hosted/cnx/cnx).

#### Updating Denied Packets Rules

This is an example of how to modify the sample rule created by the sample manifest.
The process of updating rules is the same as for user created rules (documented below).

  - Save the current alert rule:

    ```
    $ kubectl -n calico-monitoring get configmap calico-prometheus-dp-rate -o yaml > calico-prometheus-alert-rule-dp.yaml
    ```

  - Make necessary edits to the alerting rules then apply the updated configmap.

    ```
    $ kubectl apply -f calico-prometheus-alert-rule-dp.yaml
    ```

Your changes should be applied in a few seconds by the prometheus-config-reloader
container inside the prometheus pod launched by the prometheus-operator
(usually named `prometheus-<your-prometheus-instance-name>`).

As an example, the range query in this _ConfigMap_ is 10 seconds.

```
{% raw %}
apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-prometheus-dp-rate
  namespace: calico-monitoring
  labels:
    role: calico-prometheus-rules
    prometheus: calico-node-prometheus
data:
  calico.rules: |
    ALERT DeniedPacketsRate
      IF rate(calico_denied_packets[10s]) > 50
      LABELS { severity = "critical" }
      ANNOTATIONS {
         summary = "Instance {{$labels.instance}} - Large rate of packets denied",
         description = "{{$labels.instance}} with calico-node pod {{$labels.pod}} has been denying packets at a fast rate {{$labels.sourceIp}} by policy {{$labels.policy}}."
      }
{% endraw %}
```

To update this alerting rule, to say, execute the query with a range of
20 seconds modify the manifest to this:

```
{% raw %}
apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-prometheus-dp-rate
  namespace: calico-monitoring
  labels:
    role: calico-prometheus-rules
    prometheus: calico-node-prometheus
data:
  calico.rules: |
    ALERT DeniedPacketsRate
      IF rate(calico_denied_packets[20s]) > 50
      LABELS { severity = "critical" }
      ANNOTATIONS {
         summary = "Instance {{$labels.instance}} - Large rate of packets denied",
         description = "{{$labels.instance}} with calico-node pod {{$labels.pod}} has been denying packets at a fast rate {{$labels.sourceIp}} by policy {{$labels.policy}}."
      }
{% endraw %}
```

#### Creating a New Alerting Rule

Creating a new alerting rule is straightforward once you figure out what you
want your rule to look for. Check [alerting
rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
and [Queries](https://prometheus.io/docs/querying/examples/) for more
information.

To add the new alerting rule to our Prometheus instance, define a _ConfigMap_
in the `calico-monitoring` namespace with the labels
`role: calico-prometheus-rules` and `prometheus: calico-node-prometheus`. The
labels should match the labels defined by the `ruleSelector` field of the
_Prometheus_ manifest.

As an example, to fire a alert when a {{site.noderunning}} instance has been down for
more than 5 minutes, save the following to a file, say `calico-node-down-alert.yaml`

```
{% raw %}
apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-prometheus-calico-node-down
  namespace: calico-monitoring
  labels:
    role: calico-prometheus-rules
    prometheus: calico-node-prometheus
data:
  instance.rules: |
    ALERT CalicoNodeInstanceDown
      IF up == 0
      FOR 5m
      LABELS { severity = "warning" }
      ANNOTATIONS {
         summary = "Instance {{$labels.instance}} Pod: {{$labels.pod}} is down",
         description = "{{$labels.instance}} of job {{$labels.job}} has been down for more than 5 minutes"
      }
{% endraw %}
```

Then _create_/_apply_ this config map in kubernetes.

```
$ kubectl apply -f calico-node-down-alert.yaml
```

Your changes should be applied in a few seconds by the prometheus-config-reloader
container inside the prometheus pod launched by the prometheus-operator
(usually named `prometheus-<your-prometheus-instance-name>`).

#### Additional Alerting Rules

The Alerting Rules installed by the _CNX_ install manifest is a simple
one that fires an alert when the rate of denied packets denied by a policy on
a node from a particular Source IP exceeds a certain packets per second
threshold. The _Prometheus_ query used for this (ignoring the threshold value
20) is:

```
rate(calico_denied_packets[10s])
```

and this query will return results something along the lines of:

```
{endpoint="calico-metrics-port",instance="10.240.0.81:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-hn0kl",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.167.129"}	0.6
{endpoint="calico-metrics-port",instance="10.240.0.84:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-97m3g",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.167.175"}	0.2
{endpoint="calico-metrics-port",instance="10.240.0.84:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-97m3g",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.252.157"}	0.4
{endpoint="calico-metrics-port",instance="10.240.0.81:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-hn0kl",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.167.175"}	1
{endpoint="calico-metrics-port",instance="10.240.0.84:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-97m3g",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.167.129"}	0.4
{endpoint="calico-metrics-port",instance="10.240.0.81:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-hn0kl",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.167.159"}	0.4
{endpoint="calico-metrics-port",instance="10.240.0.81:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-hn0kl",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.252.175"}	0.4
{endpoint="calico-metrics-port",instance="10.240.0.84:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-97m3g",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.252.175"}	0.6
{endpoint="calico-metrics-port",instance="10.240.0.81:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-hn0kl",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.252.157"}	0.6
{endpoint="calico-metrics-port",instance="10.240.0.84:9081",job="calico-node-metrics",namespace="kube-system",pod="calico-node-97m3g",policy="profile/k8s_ns.test/0/deny",service="calico-node-metrics",srcIP="192.168.167.159"}	0.6
```

We can modify this query to find out all packets dropped by different policies
on every node.

```
(sum by (instance,policy) (rate(calico_denied_packets[10s])))
```

This query will aggregate the results from all different Source IPs, and
preserve the `policy` and `instance` labels. Note that the `instance` label
represents the calico node's IP Address and `PrometheusReporterPort`. This
query will return results like so:

```
{instance="10.240.0.84:9081",policy="profile/k8s_ns.test/0/deny"}	2
{instance="10.240.0.81:9081",policy="profile/k8s_ns.test/0/deny"}	2.8
```

To include the pod name in these results, add the label `pod` to the labels
listed in the `by` expression like so:

```
(sum by (instance,pod,policy) (rate(calico_denied_packets[10s])))
```

which will the following results:

```
{instance="10.240.0.84:9081",pod="calico-node-97m3g",policy="profile/k8s_ns.test/0/deny"}	2
{instance="10.240.0.81:9081",pod="calico-node-hn0kl",policy="profile/k8s_ns.test/0/deny"}	2.8
```

An interesting use case is when a rogue _Pod_ is using tools such as _nmap_ to
scan a subnet for open ports. To do this, we have to execute a query that will
aggregate across all policies on all instances while preserving the source IP
address. This can be done using this query:

```
(sum by (srcIP) (rate(calico_denied_packets[10s])))
```

which will return results, different source IP address:

```
{srcIP="192.168.167.159"}	1.0000000000000002
{srcIP="192.168.167.129"}	1.2000000000000002
{srcIP="192.168.252.175"}	1.4000000000000001
{srcIP="192.168.167.175"}	0.4
{srcIP="192.168.252.157"}	1.0000000000000002
```

To use these queries as Alerting Rules, follow the instructions defined in the
[Creating a new Alerting Rule](#creating-a-new-alerting-rule) section and create
a _ConfigMap_ with the appropriate query.

#### Storage

Prometheus stores metrics at regular intervals. If Prometheus is restarted and
if ephemeral storage is used, metrics will be lost. Configure the _Storage_ spec
to store metrics persistently.

The sample manifests do not define a _Storage_ spec. This means that if you
use the default manifest, Prometheus will be deployed with a `emptyDir`
volume. Using a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
supported by Kubernetes is strongly recommended.

The _Prometheus_ third party resource defined by the _Prometheus Operator_
supports defining a _StorageClass_

As an example, to use GCE Persistent Disks for Prometheus storage, define a
_StorageClass_ spec, like so:

```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
  zone: us-west1-b
```

Then add the `storage` field to the _Prometheus_ manifest.

```
apiVersion: monitoring.coreos.com/v1alpha1
kind: Prometheus
metadata:
  name: calico-node-prometheus
  namespace: calico-monitoring
spec:
  serviceAccountName: prometheus
  serviceMonitorSelector:
    matchLabels:
      team: network-operators
  version: v1.6.3
  retention: 1h
  resources:
    requests:
      memory: 800Mi
  ruleSelector:
    matchLabels:
      role: calico-prometheus-rules
      prometheus: calico-node-prometheus
  alerting:
    alertmanagers:
      - namespace: calico-monitoring
        name: calico-node-alertmanager
        port: web
        scheme: http
  storage:
    class: ssd
    resources:
      requests:
        storage: 80Gi
```

_Prometheus Operator_ also supports other manual storage provisioning
mechanisms. More information can be found [here](https://github.com/coreos/prometheus-operator/blob/d360a3bae5ab054732837d5a715814cd1c78d745/Documentation/user-guides/storage.md).

Combining storage resource with proper retention time for metrics will ensure
that Prometheus will use the storage effectively. The `retention` field is used
to configure the amount of time that metrics are stored on disk.

#### Updating the scrape interval

You may wish to modify the scrape interval (time between Prometheus polling each node for new denied packet information).
Increasing the interval reduces load on Prometheus and the amount of storage required, but decreases the detail of the collected metrics.

  The scrape interval of endpoints ({{site.noderunning}} in our case) is defined as part of
  the _ServiceMonitor_ manifest. To change the interval:

  - Save the current _ServiceMonitor_ manifest:

    ```
    $ kubectl -n calico-monitoring get servicemonitor calico-node-monitor -o yaml > calico-node-monitor.yaml
    ```

  - Update the `interval` field under `endpoints` to desired settings and
   _apply_ the updated manifest.

    ```
    $ kubectl apply -f calico-node-monitor.yaml
    ```

Your changes should be applied in a few seconds by the prometheus-config-reloader
container inside the prometheus pod launched by the prometheus-operator
(usually named `prometheus-<your-prometheus-instance-name>`).

As an example on what to update, the interval in this _ServiceMonitor_ manifest
is 5 seconds (`5s`).

```
apiVersion: monitoring.coreos.com/v1alpha1
kind: ServiceMonitor
metadata:
  name: calico-node-monitor
  namespace: calico-monitoring
  labels:
    team: network-operators
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  namespaceSelector:
    matchNames:
    - kube-system
  endpoints:
  - port: calico-metrics-port
    interval: 5s
```

To update calico Prometheus' scrape interval to 10 seconds modify the manifest
to this:

```
apiVersion: monitoring.coreos.com/v1alpha1
kind: ServiceMonitor
metadata:
  name: calico-node-monitor
  namespace: calico-monitoring
  labels:
    team: network-operators
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  namespaceSelector:
    matchNames:
    - kube-system
  endpoints:
  - port: calico-metrics-port
    interval: 10s
```

#### Enabling Secure Connections for Prometheus metrics

Because denied packet/bytes metrics may contain sensitive information, you may wish
to encrypt the communications between Prometheus and Calico with TLS. To accomplish
this, complete the following steps.

##### Format your certificates

In order to secure connections between Prometheus and Calico, you will need to first
have the following:

- A Certificate Authority (CA) certificate (Used to sign the Calico/Prometheus certificate and key)
(`ca.pem` in this example)
- A certificate for Calico (`calico.pem` in this example)
- A private key for Calico (`calico-key.pem` in this example)
- A certificate for Prometheus (`prom.pem` in this example)
- A private key for Prometheus (`prom-key.pem` in this example)

For just the Calico certificate, you will need to concatenate your certificate
to the CA certificate.

```
cat calico.pem ca.pem >> concat-cert.pem
```

##### Mount your certificates into Calico

You now need to mount the Calico certificate (the concatenated certificate) and key
into the `calico-node` daemonset.

<div class="alert alert-info" role="alert">
<b>Note</b>: The <samp>calico-node</samp> daemonset is found in the <samp>calico-cnx.yaml</samp> file provided as an example.
</div>

Encode the concatenated certificate, the corresponding private key, and the CA
certificate used to sign the Prometheus certificate and key in base64 format. In the
following commands, we call these files `concat-cert.pem`, `calico-key.pem`, and
`ca.pem`, respectively.

```
cat concat-cert.pem | base64 -w 0
cat calico-key.pem | base64 -w 0
cat ca.pem | base64 -w 0
```

Create a secret for the files and place this in the `calico-cnx.yaml` file.

```
apiVersion: v1
kind: Secret
metadata:
  name: certs
  namespace: kube-system
data:
  concat-cert.pem: <Your base64 encoding of concat-cert.pem goes here>
  calico-key.pem: <Your base64 encoding of your calico-key.pem goes here>
  ca.pem: <Your base64 encoding of your ca.pem goes here>
```

Add the appropriate `volumeMounts` and `volumes` to their corresponding sections in
the `{{site.noderunning}}` daemonset.

```
      ...
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /etc
              name: tls-certs-dir
      ...

      volumes:
        # Used by calico/node.
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        # Used to install CNI.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        - name: tls-certs-dir
          secret:
            secretName: certs
      ...
```
<div class="alert alert-info" role="alert">
<b>Note</b>: Alternatively, you can mount the location of your certificates directly
into the container instead of using secrets.
</div>
```
      ...
      volumes:
        # Used by calico/node.
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        # Used to install CNI.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        - name: tls-certs-dir
          hostPath:
            path: <path to your certs goes here>
      ...
```

Make sure that Calico knows where to read the certificates from by setting the
`FELIX_PROMETHEUSREPORTERCERTFILE` and `FELIX_PROMETHEUSREPORTERKEYFILE` environment
variables. You must also specify where to find the CA certificate used
to sign the client certificate in `FELIX_PROMETHEUSREPORTERCAFILE` (either the CA
certificate from above or the optional CA certificate). You can set
these in the `spec.template.spec.containers.env` section of the `calico-node`
daemonset as shown below.

```
            ...
            - name: FELIX_PROMETHEUSREPORTERPORT
              value: "9081"
            # The TLS certs and keys for testing denied packet metrics
            - name: FELIX_PROMETHEUSREPORTERCERTFILE
              value: "/etc/certs/concat-cert.pem"
            - name: FELIX_PROMETHEUSREPORTERKEYFILE
              value: "/etc/certs/calico-key.pem"
            - name: FELIX_PROMETHEUSREPORTERCAFILE
              value: "/etc/certs/ca.pem"
            ...
```

##### Mount your certificates into Prometheus

<div class="alert alert-info" role="alert">
<b>Note</b>: The following changes need to be made to the <samp>monitor-calico.yaml</samp> file or your equivalent manifest.
</div>

Encode your Prometheus certificate, your Prometheus private key, and your
CA certificate in base64 format. In the following commands, we refer to these
files as `prom.pem`, `prom-key.pem`, and `ca.pem` respectively.

```
cat ca.pem | base64 -w 0
cat prom.pem | base64 -w 0
cat prom-key.pem | base64 -w 0
```

Take the base64 output and add it to a secret in the same manifest file as
the service monitor `calico-node-monitor` (found in the `monitor-calico.yaml`
file provided as an example). Make sure that the secret is in the same
namespace as your `calico-node-monitor` (`calico-monitoring` in the example).

```
apiVersion: v1
kind: Secret
metadata:
  name: certs
  namespace: calico-monitoring
data:
  ca.pem: <Your base64 certificate output goes here>
  prom.pem: <Your base64 certificate output goes here>
  prom-key.pem: <Your base64 private key output goes here>
```

Add your secrets so that they can be mounted in the service monitor. In the
manifest for your Prometheus instance (`calico-node-prometheus` in the example),
add a `secrets` section to the `spec` listing the secrets you defined.

```
...
alerting:
    alertmanagers:
      - namespace: calico-monitoring
        name: calico-node-alertmanager
        port: web
        scheme: http
  secrets:
    - certs
...
```

Prometheus will mount your secrets at `/etc/prometheus/secrets/` in the container.
Specify the location of your secrets in the `spec.endpoints.tlsConfig` section of your
service monitor (`calico-node-monitor` in the example).  Also make sure to change the
endpoint scheme to use TLS by specifying `scheme: https`.

```
...
endpoints:
  - port: calico-metrics-port
    interval: 5s
    scheme: https
    tlsConfig:
      caFile: /etc/prometheus/secrets/certs/ca.pem
      certFile: /etc/prometheus/secrets/certs/prom.pem
      keyFile: /etc/prometheus/secrets/certs/prom-key.pem
      serverName: <the common name used in the calico certificate goes here>
      insecureSkipVerify: false
...
```

<div class="alert alert-info" role="alert">
<b>Note</b>: Make sure that <samp>serverName</samp> is set to the common name field
of your calico certificate. If this is misconfigured, then connections will fail verification.
If you wish to skip certificate verification, then you can ignore the <samp>serverName</samp>
field and instead set <samp>insecureskipVerify</samp> to <samp>true</samp>.
</div>

##### Reapply your changes

Apply your changes.

```
kubectl apply -f calico-cnx.yaml
kubectl apply -f monitor-calico.yaml
```

Congratulations! Your metrics are now secured with TLS.

<div class="alert alert-info" role="alert">
<b>Note</b>: Changes to the daemonset may require you to delete the existing pods
in order to schedule new pods with your changes.
</div>

### Troubleshooting Config Updates

Check config reloader logs to see if they detected any recent activity.

  - For prometheus run:

    ```
    $ kubectl -n calico-monitoring logs prometheus-<your-prometheus-name> prometheus-config-reloader
    ```

  - For alertmanager run:

    ```
    $ kubectl -n calico-monitoring logs alertmanager-<your-prometheus-name> config-reloader
    ```

The config-reloaders watch each pods file-system for updated config from
_ConfigMap_'s or _Secret_'s and will perform steps necessary for reloading
the configuration.
