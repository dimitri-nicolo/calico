---
title: Tigera Essentials Toolkit Hosted Install
redirect_from: latest/getting-started/kubernetes/installation/hosted/essentials/index
---

To install Tigera Essentials Toolkit, run the following commands.

<div class="alert alert-info" role="alert">
  <b>Note</b>: These instructions do not apply to OpenShift users. Instead, see <a href="{{site.baseurl}}/{{page.version}}/getting-started/openshift/essentials/installation">Installing Essentials for OpenShift</a>.
</div>

- Setup etcd:

```
kubectl apply -f calico-etcd.yaml
```

- Make any changes to configurations to the [calico-essentials.yaml](1.6/calico-essentials.yaml)
  file and then install/configure calico.

```
kubectl apply -f calico-essentials.yaml
```

- Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml](1.6/operator.yaml) manifest.

```
kubectl apply -f operator.yaml
```

- Wait for third party resources to be created. Check by running:

```
$ kubectl get thirdpartyresources
```

- Apply the [monitor-calico.yaml](1.6/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

```
$ kubectl apply -f monitor-calico.yaml
```

### Customizing the manifests

#### Configure calico/node settings

In the [calico-essentials.yaml](1.6/calico-essentials.yaml) file, make sure
that:
  - `FELIX_PROMETHEUSREPORTERENABLED` is set to `true` and
  - `FELIX_PROMETHEUSREPORTERPORT` is set to desired port. This **must** match
     the `targetPort` field of the `calico-node-metrics` _Service_.

Note that the environment variables stated above are different from
`FELIX_PROMETHEUSMETRICSENABLED` and `FELIX_PROMETHEUSMETRICSPORT`. The former
set toggles and configures Calico Denied Packet reporting, while the latter set
toggles and configures Felix white box metrics reporting.

#### Ports of Services and NetworkPolicy

There are a number of ports defined via _Services_. Ensure that they are updated
to match your environment and also update any _NetworkPolicy_ that
have the same ports.

#### Etcd Endpoints

The [calico-essentials.yaml](1.6/calico-essentials.yaml) file contains a manifest
for a _ConfigMap_ which specifies etcd endpoints. The current default is set to
`http://10.96.232.136:6666`. Ensure that this value is updated to not clash
with any service running in your environment. This needs to be changed in 2
places:

- In [calico-etcd.yaml](1.6/calico-etcd.yaml) update the `ClusterIP` and `port`
  fields of the _Service_ manifest.
- In [calico-essentials.yaml](1.6/calico-essentials.yaml) update the
  `etcd_endpoints` key in the `calico-config` _ConfigMap_

#### Node Selectors

There are 3 places where `nodeSelectors` can be customized. Ensure to update
these.

- [operator.yaml](1.6/operator.yaml) - The _Deployment_
  manifest for the `calico-prometheus-operator`
- [monitor-calico.yaml](1.6/monitor-calico.yaml) - The _Prometheus_ and
  _AlertManager_ manifests.

For example, to deploy Prometheus Operator in GKE infrastructure nodes,
customize the manifest like so:

```
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: calico-prometheus-operator
  namespace: calico-monitoring
  labels:
    operator: prometheus
spec:
  replicas: 1
  template:
    metadata:
      labels:
	operator: prometheus
    spec:
      nodeSelector:
	cloud.google.com/gke-nodepool: infrastructure
      serviceAccountName: calico-prometheus-operator
      containers:
      - name: calico-prometheus-operator
	image: quay.io/coreos/prometheus-operator:v0.8.1
	resources:
	  requests:
	    cpu: 100m
	    memory: 50Mi
	  limits:
	    cpu: 200m
	    memory: 100Mi
```

### Configure Alertmanager Notifications

Configuring Alertmanager to send alerts to _PagerDuty_ can be done as follows.
The following Alertmanager configuration file, notifies _PagerDuty_ of all
alerts that are at a severity level of "critical".

```
global:
  resolve_timeout: 5m
route:
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 5m
  routes:
  - match:
      severity: critical
    receiver: 'pageme'
receivers:
- name: 'pageme'
  pagerduty_config:
  - service_key: PUT-YOUR-INTEGRATION-KEY-HERE
```

Information on how to create a _Integration Key_ is available
[here](https://support.pagerduty.com/hc/en-us/articles/202830340-Create-a-Generic-Events-API-Integration).

### Advanced Alertmanager Notifications

Included in the manifests is a sample alertmanager webhook. _apply_ this t
deploy webserver that will pretty print to its stdout a JSON message (if it
received a valid message).

Ensure that your alertmanager configuration is as follows (this is similar to
the one provided in the `monitor-calico.yaml` file. Instructions on how to
apply this is provided in later sections.

```
global:
  resolve_timeout: 5m
route:
  group_by: ['job']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 5m
  receiver: 'webhook'
receivers:
- name: 'webhook'
  webhook_configs:
  - url: 'http://calico-alertmanager-webhook:30501/'
```

To view the JSON output printed, examine the logs of the webhook pod.

### Modifying an existing manifest to install Essentials

Edit the manifest that deploys calico/node and update the following:
  - Update the _DaemonSet_ template `image` to point to the new Calico image.
  - Add the environment variable `FELIX_PROMETHEUSREPORTERENABLED` and make
    sure it is set to `true` and
  - Add the environment variable `FELIX_PROMETHEUSREPORTERPORT` and set it to
    desired port.

Create a new _Service_ to expose Calico Prometheus Denied Packet Metrics and
apply this manifest using `kubectl apply`

```
# This manifest installs the Service which gets traffic to the calico-node
# metrics reporting endpoint.
apiVersion: v1
kind: Service
metadata:
  namespace: kube-system
  name: calico-node-metrics
  labels:
    k8s-app: calico-node
spec:
  selector:
    k8s-app: calico-node
  type: ClusterIP
  clusterIP: None
  ports:
  - name: calico-metrics-port
    port: 9081
    targetPort: 9081
    protocol: TCP
```

- Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml](1.6/operator.yaml) manifest.

```
kubectl apply -f operator.yaml
```

- Wait for third party resources to be created. Check by running:

```
kubectl get thirdpartyresources
```

- Apply the [monitor-calico.yaml](1.6/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

```
kubectl apply -f monitor-calico.yaml
```

### Manifest Details

The Calico install manifests are based on [Kubeadm hosted install](../kubeadm),
however, you can adapt any hosted install manifest by making changes described
in the [modifying your manifest to install essentials](adapt)

The additional things the [calico-essentials.yaml](1.6/calico-essentials.yaml) does
are:
  - Enables prometheus reporting (this is different from felix's prometheus
    settings)

The manifest [operator.yaml](1.6/operator.yaml) does the following:
  - Create a namespace called calico-monitoring
  - Create RBAC artifacts
      - ServiceAccounts: prometheus-operator and prometheus
      - Corresponding ClusterRole and ClusterRoleBindings.
  - Deploys prometheus-operator (in namespace calico-monitoring)
    - Creates a kubernetes deplyoment, which in turn creates 3 _Third Party
      Resources_(TPR): `prometheus`, `alertmanager` and "servicemonitor".

The `monitor-calico.yaml` manifest does the following:
  - Creates a new service: calico-node-metrics exposing prometheus reporting
    port.
  - A secret for storing alertmanager config - Should be customized for your
    environment.
    - Refer to [standard alerting configuration documentation](https://prometheus.io/docs/alerting/configuration/)
      and alertmanager.yaml in the current directory for what is included as a
      default.
  - Create a alertmanager instance and corresponding dash service
  - Create a ServiceMonitor that selects on the calico-node-metrics service's
    ports.
  - ConfigMaps that define some [alerting rules](https://prometheus.io/docs/alerting/rules/)
    for prometheus.
    - We predefine denied packet alerts and instance down alerts. This can be
      customized by modifying appropriate configmaps.
  - Create a Prometheus instance and corresponding dash service. Prometheus
    instance selects on:
    - Alertmanager dash (for configuring alertmanagers)
    - Servicemonitor selector (for populating calico-nodes to actually monitor).
  - Create network policies as required for accessing all the services defined
    above.

The services (type _NodePort_) for prometheus and alertmanager are created in
the `calico-monitoring` namespace and are named `calico-prometheus-dash`
(port 30900) and `calico-alertmanager-dash` (port 30903).
