---
title: Configuring Prometheus
---

#### Storage

The sample manifests does not define a _Storage_ spec. This means that if you
use the default manifest, Prometheus will be deploymed with a `emptyDir`
volume. Using a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
supported by Kubernetes is strongly recommended.

The _Prometheus_ third party resource defined by the _Prometheus Operator_
supports defining a _StorageClass_

As an example, to run a _Prometheus_ instance in GCE, first define a
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

#### Crashes, Restarts and Metrics Availability

Prometheus stores metrics at regular intervals. If Prometheus is restarted and
if ephemeral storage is used, metrics will be lost. Use Kubernetes Persistent
Volumes to ensure that stored data will be accessible across restart/crashes.

#### Updating the scrape interval

  Scrape interval of endpoints (calico/node in our case) is defined as part of
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

#### Updating the DeniedPackets rule

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
```

To update this alerting rule, to say, execute the query with a range of
20 seconds modify the manifest to this:

```
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
```

#### Creating a new alerting rule.

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
```

Then _create_/_apply_ this config map in kubernetes.

```
$ kubectl apply -f calico-node-down-alert.yaml
```

Your changes should be applied in a few seconds by the prometheus-config-reloader
container inside the prometheus pod launched by the prometheus-operator
(usually named `prometheus-<your-prometheus-instance-name>`).

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
