---
title: Adding Tigera CNX to a Calico install
---

## Requirements

Ensure that the kube-apiserver has been started with the appropriate flags.
- Refer to the Kubernetes documentation to
  [Configure the aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/)
  with the proper flags.
- Refer to the [authentication guide]({{site.baseurl}}/{{page.version}}/reference/essentials/authentication) to choose a supported authentication
  mechanism and configure the Kubernetes API server accordingly.

Ensure that Calico has been installed using the enhanced CNX node agent.

```
kubectl get pods -n kube-system | grep cnx-node
```

## Installation

1. Download and modify the Tigera CNX resources.

   - [Kubernetes datastore](1.7/cnx-kdd.yaml)

   - [etcd datastore](1.7/cnx-etcd.yaml)

   Rename the file `cnx.yaml` - this is what subsequent instructions will refer to.

1. Update the manifest with the path to your private docker registry.  Substitute
   `mydockerregistry:5000` with the location of your docker registry.

   ```
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/mydockerregistry:5000/g' cnx.yaml
   ```

1. Open the file in a text editor, and update the ConfigMap `tigera-cnx-manager-config`
   according to the instructions in the file and your chosen authentication method.

   You might want to reconfigure the Service that gets traffic to the CNX Manager
   web server as well.

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   CNX Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the CNX Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` Service.

1. Store those credentials as `cert` and `key` in a Secret named
   `cnx-manager-tls`.  For example:

   ```
   kubectl create secret generic cnx-manager-tls --from-file=cert=/path/to/certificate --from-file=key=/path/to/key -n kube-system
   ```

1. Apply the manifest to install CNX Manager and the CNX API server.

   ```
   kubectl apply -f cnx.yaml
   ```

1. Configure authentication to allow CNX Manager users to edit policies.  Consult the
   [CNX Manager](../../../../../reference/essentials/policy-editor) and
   [Tiered policy RBAC](../../../../../reference/essentials/rbac-tiered-policy)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.

1. Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml](1.7/operator.yaml) manifest.

   ```
   kubectl apply -f operator.yaml
   ```

1. Wait for custom resource definitions to be created. Check by running:

   ```
   kubectl get customresourcedefinitions
   ```

1. Apply the [monitor-calico.yaml](1.7/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

   ```
   kubectl apply -f monitor-calico.yaml
   ```

### Customizing the manifests

#### Ports of Services and NetworkPolicy

There are a number of ports defined via _Services_. Ensure that they are updated
to match your environment and also update any _NetworkPolicy_ that
have the same ports.

#### Node Selectors

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
        image: quay.io/coreos/prometheus-operator:v0.12.0
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
[manifest](1.7/cnx.yaml) will communicate with the Kubernetes
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
   `cnx-apiserver.kube-system.svc`


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

The [cnx.yaml](1.7/cnx.yaml) manifest must be updated
with the following changes

1. Remove the line `insecureSkipTLSVerify: true` from the `APIService` section.
1. Uncomment the line `caBundle:` in the `APIService` and append the base64
   encoded CA file contents.
1. Uncomment the line `apiserver.key:` in the `cnx-apiserver-certs` `Secret`
   and append the base64 encoded key file contents.
1. Uncomment the line `apiserver.crt:` in the `cnx-apiserver-certs` `Secret`
   and append the base64 encoded certificate file contents.

### Configure the {{site.prodname}} Manager Web Application

The [cnx.yaml](1.7/cnx.yaml) manifest must be updated with
the following changes.  Some of the parameters depend on the chosen
authentication method.  Authentication methods, and the relevant parameters
are described [here]({{site.baseurl}}/{{page.version}}/reference/essentials/authentication).

1. If using Google login, update the `tigera.cnx-manager.oidc-client-id` field
   in the `tigera-cnx-manager-config` ConfigMap.

1. Update the `tigera.cnx-manager.kubernetes-api` field
   in the `tigera-cnx-manager-config` ConfigMap.  It must be a URL which
   the web client can use to reach the Kubernetes API server.  Note that it
   must be reachable from any system which is going to access the {{site.prodname}}
   Manager web application (not just inside the cluster).

1. Consider updating the NodePort that exposes the web application.  Note
   that for Google login, the URL of the application must be well known,
   and configured in the Google console (with /login/oidc/callback appended).

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

Included in the manifests is a sample alertmanager webhook. _Apply_ this to
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

### Modifying an existing manifest to install {{site.prodname}}

Edit the manifest that deploys calico/node and update the following:
  - Update the _DaemonSet_ template `image` to point to the new {{site.prodname}} image.
  - Add the environment variable `FELIX_PROMETHEUSREPORTERENABLED` and make
    sure it is set to `true` and
  - Add the environment variable `FELIX_PROMETHEUSREPORTERPORT` and set it to
    desired port.

Create a new _Service_ to expose {{site.prodname}} Prometheus Denied Packet Metrics and
apply this manifest using `kubectl apply`

```
# This manifest installs the service which gets traffic to the calico-node
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

1. Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml](1.7/operator.yaml) manifest.

   ```
   kubectl apply -f operator.yaml
   ```

1. Wait for custom resource definitions to be created. Check by running:

   ```
   kubectl get customresourcedefinitions
   ```

1. Apply the [monitor-calico.yaml](1.7/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

   ```
   kubectl apply -f monitor-calico.yaml
   ```

### Manifest Details

The {{site.prodname}} install manifests are based on [kubeadm hosted install](../kubeadm).

The [cnx.yaml](1.7/calico-essentials.yaml) manifest does the following:
  - Installs the CNX API server, and configures the APIService to tell
    the Kubernetes API server to delegate to it.
  - Installs the CNX Manager web server, and configures it with the location
    of the Kubernetes API, login methods and SSL certificates.

The manifest [operator.yaml](1.7/operator.yaml) does the following:
  - Create a namespace called calico-monitoring
  - Create RBAC artifacts
      - ServiceAccounts: prometheus-operator and prometheus
      - Corresponding ClusterRole and ClusterRoleBindings.
  - Deploys prometheus-operator (in namespace calico-monitoring)
    - Creates a kubernetes deployment, which in turn creates 3 _Custom Resource
      Definitions_(CRD): `prometheus`, `alertmanager` and "servicemonitor".

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

{% include {{page.version}}/gs-next-steps.md %}
