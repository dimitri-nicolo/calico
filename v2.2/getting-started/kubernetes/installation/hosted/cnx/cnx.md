---
title: Customizing the Tigera Secure EE manifests (advanced)
redirect_from: latest/getting-started/kubernetes/installation/hosted/cnx/cnx
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/hosted/cnx/cnx
---

## About the manifests

The **[cnx-etcd.yaml](1.7/cnx-etcd.yaml)** and **[cnx-kdd.yaml](1.7/cnx-kdd.yaml)** manifests do the following:
  - Installs the {{site.prodname}} API server, and configures the APIService to tell
    the Kubernetes API server to delegate to it.
  - Installs the {{site.prodname}} Manager web server, and configures it with the location
    of the Kubernetes API, login methods and SSL certificates.

The **[operator.yaml](1.7/operator.yaml)** manifest does the following:
  - Create a namespace called calico-monitoring
  - Create RBAC artifacts
      - ServiceAccounts: prometheus-operator and prometheus
      - Corresponding ClusterRole and ClusterRoleBindings.
  - Deploys prometheus-operator (in namespace calico-monitoring)
    - Creates a kubernetes deployment, which in turn creates 3 _Custom Resource
      Definitions_(CRD): `prometheus`, `alertmanager` and "servicemonitor".

The **[monitor-calico.yaml](1.7/monitor-calico.yaml)** manifest does the following:
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
  - ConfigMaps that define some [alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
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
(port 30909) and `calico-alertmanager-dash` (port 30903).

## Ports of Services and NetworkPolicy

There are a number of ports defined via _Services_. Ensure that they are updated
to match your environment and also update any _NetworkPolicy_ that
have the same ports.

For example, if you want the {{site.prodname}} Manager to listen on a port other than
30003 or you plan to set up a load balancer in front of it, edit the
`Service` object named `cnx-manager` as required. In this paticular case, there are no 
network policies to update for this service.

## Node Selectors

There are 3 places where `nodeSelectors` can be customized. Ensure to update
these.

- [operator.yaml](1.7/operator.yaml) - The _Deployment_
  manifest for the `calico-prometheus-operator`
- [monitor-calico.yaml](1.7/monitor-calico.yaml) - The _Prometheus_ and
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
        image: quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
        resources:
          requests:
            cpu: 100m
            memory: 50Mi
          limits:
            cpu: 200m
            memory: 100Mi
```

## Configure the {{site.prodname}} Manager

The **[cnx-etcd.yaml](1.7/cnx-etcd.yaml)** and **[cnx-kdd.yaml](1.7/cnx-kdd.yaml)** manifests must be updated with
the following changes.  Some of the parameters depend on the chosen
authentication method.  Authentication methods, and the relevant parameters
are described [here]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication).

1. If using Google login, update the `tigera.cnx-manager.oidc-client-id` field
   in the `tigera-cnx-manager-config` ConfigMap.

1. Consider updating the NodePort that exposes the web application.  Note
   that for Google login, the URL of the application must be well known,
   and configured in the Google console (with /login/oidc/callback appended).

## Configure Alertmanager Notifications

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

## Advanced Alertmanager Notifications

Included in the manifests is a sample Alertmanager webhook. _Apply_ this to
deploy a web server that will pretty print to its stdout a JSON message (if it
received a valid message).

Ensure that your Alertmanager configuration is as follows (this is similar to
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
