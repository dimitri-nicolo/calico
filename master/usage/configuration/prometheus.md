---
title: Configuring Prometheus
---

Note: This document assumes that Prometheus and Alertmanager have been setup
using _Prometheus Operator_ as described
[here](../../getting-started/kubernetes/installation/hosted/essentials/).

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

Creating a new alerting rule is straighforward once you figure out what you
want your rule to look for. Check [alerting rules](https://prometheus.io/docs/alerting/rules/)
and [Queries](https://prometheus.io/docs/querying/examples/) for more
information.

To add the new alerting rule to our Prometheus instance, define a _ConfigMap_
in the `calico-monitoring` namespace with the labels
`role: calico-prometheus-rules` and `prometheus: calico-node-prometheus`. The
labels should match the labels defined by the `ruleSelector` field of the
_Prometheus_ manifest.

As an example, to fire a alert when a calico/node instance has been down for
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

The Alerting Rules installed by the _Essentials_ install manifest is a simple
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
use the default manifest, Prometheus will be deploymed with a `emptyDir`
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

  The scrape interval of endpoints (calico/node in our case) is defined as part of
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

You may want to secure your denied packet/bytes metrics. To allow for this, you can
secure Prometheus's connection to Calico using TLS.

First you need to properly format your certificates. You should have a certificate
and key for both Prometheus and Calico. For just the Calico certificate, you will
need to concatenate your certificate to the certificate authority certificate.

```
cat calico.pem ca.pem >> concat-cert.pem
```

Once your certificates are formatted correctly, you should then mount the Calico
certificate (the concatenated certificate) and key into the `calico-node` daemonset.

Note: the `calico-node` daemonset is found in the `calico-essentials.yaml` file
provided as an example.

Create a secret for your `concat-cert.pem` file and the corresponding key for the
original certificate (`calico-key.pem` in this example) by base64 encoding the files.

```
cat concat-cert.pem | base64 -w 0
```

Create a secret for the files and place this in the `calico-essentials.yaml` file
```
apiVersion: v1
kind: Secret
metadata:
  name: certs
  namespace: kube-system
data:
  concat-cert.pem: <your base64 encoding of concat-cert.pem goes here>
  calico-key.pem: <your base64 encoding of your calico-key.pem goes here>
```

Add the appropriate `volumeMounts` and `volumes` to their corresponding sections in
the `calico-node` daemonset.

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

Alternatively, you can directly mount the location of your certificates directly
into the container instead of using secrets.

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

Once the certificates are accessible by the `calico-node` daemonset. You need to
make sure that Calico knows where to read them from. This is specified by the
environment variables `FELIX_PROMETHEUSREPORTERCERTFILE` and `FELIX_PROMETHEUSREPORTERKEYFILE`.
You should set these environment variables in the `calico-node` daemonset
`spec.template.spec.containers.env` section.

```
            ...
            - name: FELIX_PROMETHEUSREPORTERPORT
              value: "9081"
            # The TLS certs and keys for testing denied packet metrics
            - name: FELIX_PROMETHEUSREPORTERCERTFILE
              value: "/etc/certs/concat-cert.pem"
            - name: FELIX_PROMETHEUSREPORTERKEYFILE
              value: "/etc/certs/calico-key.pem"
            ...
```

Also make sure DNS is working properly by making sure that the `calico-node`
daemonset `spec.template.spec` has `dnsPolicy: ClusterFirstWithHostNet` set.

```
...
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  template:
    metadata:
      labels:
        k8s-app: calico-node
      annotations:
        # Mark this pod as a critical add-on; when enabled, the critical add-on scheduler
        # reserves resources for critical add-on pods so that they can be rescheduled after
        # a failure.  This annotation works in tandem with the toleration below.
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      tolerations:
...
```

Now that Calico has access to the correct certificates, you need to make sure
that Prometheus can also access the certificates and keys it needs. First
you need create secrets for all of the certificates Prometheus needs:
your certificate authority certificate (`ca.pem`), your Prometheus certificate
(`prom.pem`), and your Prometheus certificate key (`prom-key.pem`).

For each of these files, you need to create a secret. Create a base64 encoding
of your files:

```
cat ca.pem | base64 -w 0
```

Take the base64 output and add it to a secret in the same manifest file as
the service monitor `calico-node-monitor` (found in the `monitor-calico.yaml`
file provided as an example). Make sure that the secret is in the same
namespace as your `calico-node-monitor` (`calico-monitoring` in the example).

```
apiVersion: v1
kind: Secret
metadata:
  name: rootca
  namespace: calico-monitoring
data:
  ca.pem: <Your base64 certificate output goes here>
```

Once a secret has been created for all your files, you need to add your secrets
so that they can be mounted in the service monitor. In the manifest for your
Prometheus instance (`calico-node-prometheus` in the example), add a `secrets`
section to the `spec` listing the secrets you defined.

```
...
alerting:
    alertmanagers:
      - namespace: calico-monitoring
        name: calico-node-alertmanager
        port: web
        scheme: http
  secrets:
    - rootca
    - prometheus
    - prometheus-key
...
```

Now that Prometheus has access to your secrets, it will mount them in
`/etc/prometheus/secrets/` in the container. You need to let the service
monitor know where to access your configuration by specifying the files
from the secrets in the service monitor (`calico-node-monitor` in the example)
under `spec.endpoints.tlsConfig`. Also make sure to change the endpoint
scheme to reflect that we are using TLS by using `scheme: https`.  Make
sure that the `serverName` field is populated with the common name from
your Calico certificate if `insecureSkipVerify` is set to `false`. 
If you do not want to verify your certificate's common name for any
traffic, then change `insecureSkipVerify` to `true`.

```
...
endpoints:
  - port: calico-metrics-port
    interval: 5s
    scheme: https
    tlsConfig:
      caFile: /etc/prometheus/secrets/rootca/ca.pem
      certFile: /etc/prometheus/secrets/client/prom.pem
      keyFile: /etc/prometheus/secrets/client-key/prom-key.pem
      serverName: <the common name from the certificate goes here>
      insecureSkipVerify: false
...
```

Apply your changes and now metrics on the port should be inaccessible
without the proper certificates.

```
kubectl apply -f calico-essentials.yaml
kubectl apply -f monitor-calico.yaml
```

Note: daemonset changes may require you to delete the existing
pods in order to schedule new pods with your changes.

### Troubeshooting Config Updates

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
