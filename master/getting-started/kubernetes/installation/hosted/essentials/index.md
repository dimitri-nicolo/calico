---
title: Tigera Essentials Toolkit Hosted Install
---

> **Note**: These instructions do not apply to OpenShift users. Instead, see 
> [Installing Essentials for OpenShift]({{site.baseurl}}/{{page.version}}/getting-started/openshift/essentials/installation).
{: .alert .alert-info}

## Requirements

Ensure that the kube-apiserver has been started with the appropriate flags.
Refer to the Kubernetes documentation to
[Configure the aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/)
with the proper flags.

> **Note**: If the Kubernetes cluster was installed with kubeadm the necessary
> flags are configured by default.
{: .alert .alert-info}

## Installation

To install Tigera Essentials Toolkit, run the following commands.

- Setup etcd: [calico-etcd.yaml](1.6/calico-etcd.yaml)

```
kubectl apply -f calico-etcd.yaml
```

- Edit [calico-essentials.yaml](1.6/calico-essentials.yaml) file by following
  [these instructions](#customizing-the-manifests) and then run the command below
  to install/configure calico.

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

- Edit [calico-k8sapiserver.yaml](1.6/calico-k8sapiserver.yaml) by following
  [these instructions](#enabling-tls-verification-for-a-kubernetes-extension-api-server)
  and then run the command below to install a Kubernetes extension API server.

```
$ kubectl apply -f calico-k8sapiserver.yaml
```

### Customizing the manifests

#### Configure calico/node image repository

In the [calico-essentials.yaml](1.6/calico-essentials.yaml) file, make sure
that the `image` field of the `calico-node` container contains the correct
repository.

For example if you set up the docker registry to be accessible at
`myregistrydomain.com:5000` then make sure that the `image:` field points
to `myregistrydomain.com:5000/calico/node:{{site.data.versions[page.version].first.title}}`

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
  `etcd_endpoints` key in the `calico-config` _ConfigMap_.

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

#### Enabling TLS Verification for a Kubernetes extension API Server

The Kubernetes extension API Server deployed by the provided
[manifest](1.6/calico-k8sapiserver.yaml) will communicate with the Kubernetes
API Server.  The manifest, by default, requires no updates to work but does not
enable TLS verification on the connection between the two API servers. We
recommend that this is enabled and you can follow the directions below to
enable TLS.

To enable TLS verification you will need to obtain or generate the following
in PEM format:
- Certificate Authority (CA) certificate
- certificate signed by the CA
- private key for the generated certificate

##### Generating certificate files

1. Create a root key (This is only needed if you are generating your CA)
   ```
   openssl genrsa -out rootCA.key 2048
   ```

1. Create a Certificate Authority (CA) certificate (This is only needed if you are generating your CA)
   ```
   openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.pem
   ```
   At each of the prompts press enter.

1. Generate a private key
   ```
   openssl genrsa -out calico.key 2048
   ```

1. Generate a signing request
   ```
   openssl req -new -key calico.key -out calico.csr
   ```
   At each of the prompts press enter except at the Common Name prompt enter
   `calico-k8sapiserver.kube-system.svc`


1. Generate the signed certificate
   ```
   openssl x509 -req -in calico.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out calico.crt -days 500 -sha256
   ```

When including the contents of the CA certificate, generated signed
certificate, and generated private key files the contents must be base64
encoded before being added to the manifest file.
Here is an example command to do the base64 encoding:
`cat rootCA.pem | base64 -w 0`.

##### Add Certificate Files to the Manifest

The [calico-k8sapiserver.yaml](1.6/calico-k8sapiserver.yaml) must be updated
with the following changes:

1. The line `insecureSkipTLSVerify: true` must be removed from the
   `APIService` section.
1. Uncomment the line `caBundle:` in the `APIService` and append the base64
   encoded CA file contents.
1. Uncomment the line `apiserver.key:` in the `Secret` and append the base64
   encoded key file contents.
1. Uncomment the line `apiserver.crt:` in the `Secret` and append the base64
   encoded certificate file contents.


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
in the [modifying your manifest to install essentials](#modifying-an-existing-manifest-to-install-essentials)

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
