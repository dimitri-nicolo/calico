---
title: Customizing the CNX manifests (advanced)
---

## About the manifests

The **[cnx-etcd.yaml](1.7/cnx-etcd.yaml)** and **[cnx-kdd.yaml](1.7/cnx-kdd.yaml)** manifests do the following:
  - Installs the CNX API server, and configures the APIService to tell
    the Kubernetes API server to delegate to it.
  - Installs the CNX Manager web server, and configures it with the location
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
        image: quay.io/coreos/prometheus-operator:v0.15.0
        resources:
          requests:
            cpu: 100m
            memory: 50Mi
          limits:
            cpu: 200m
            memory: 100Mi
```

## Enabling TLS Verification for a Kubernetes extension API Server

The Kubernetes extension API Server deployed by the provided
**[cnx-etcd.yaml](1.7/cnx-etcd.yaml)** and **[cnx-kdd.yaml](1.7/cnx-kdd.yaml)**
manifests will communicate with the Kubernetes
API Server.  The manifest, by default, requires no updates to work but does not
enable TLS verification on the connection between the two API servers. We
recommend that this is enabled and you can follow the directions below to
enable TLS verification.

To enable TLS verification you will need to obtain or generate the following
in PEM format:
- Certificate Authority (CA) certificate
- certificate signed by the CA
- private key for the generated certificate

#### Generating certificate files

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
   `api.kube-system.svc`


1. Generate the signed certificate
   ```
   openssl x509 -req -in calico.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out calico.crt -days 500 -sha256
   ```

When including the contents of the CA certificate, generated signed
certificate, and generated private key files the contents must be base64
encoded before being added to the manifest file.
Here is an example command to do the base64 encoding:
`cat rootCA.pem | base64 -w 0`.

#### Add Certificate Files to the Manifest

The **[cnx-etcd.yaml](1.7/cnx-etcd.yaml)** and **[cnx-kdd.yaml](1.7/cnx-kdd.yaml)** manifests must be updated
with the following changes

1. Remove the line `insecureSkipTLSVerify: true` from the `APIService` section.
1. Uncomment the line `caBundle:` in the `APIService` and append the base64
   encoded CA file contents.
1. Uncomment the line `apiserver.key:` in the `cnx-apiserver-certs` `Secret`
   and append the base64 encoded key file contents.
1. Uncomment the line `apiserver.crt:` in the `cnx-apiserver-certs` `Secret`
   and append the base64 encoded certificate file contents.
1. Uncomment the lines associated with `volumeMounts` and `volumes` named `apiserver-certs`.

## Configure the {{site.tseeprodname}} Manager

The **[cnx-etcd.yaml](1.7/cnx-etcd.yaml)** and **[cnx-kdd.yaml](1.7/cnx-kdd.yaml)** manifests must be updated with
the following changes.  Some of the parameters depend on the chosen
authentication method.  Authentication methods, and the relevant parameters
are described [here]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication).

1. If using Google login, update the `tigera.cnx-manager.oidc-client-id` field
   in the `tigera-cnx-manager-config` ConfigMap.

1. Update the `tigera.cnx-manager.kubernetes-api` field
   in the `tigera-cnx-manager-config` ConfigMap.  It must be a URL which
   the web client can use to reach the Kubernetes API server.  Note that it
   must be reachable from any system which is going to access the {{site.tseeprodname}}
   Manager web application (not just inside the cluster).

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

## Enabling Typha

{{site.tseeprodname}}'s Typha component helps {{site.tseeprodname}} deployments that use the Kubernetes API
datastore scale to high numbers of nodes without over-taxing the Kubernetes API server. 
It sits between Felix ({{site.tseeprodname}}'s per-host agent) and the API server, as fan-out proxy. 

> **Important**: Typha runs as a host-networked pod and it opens a port on the host for Felix 
> to connect to.  If your cluster runs in an untrusted environment, you **must** take steps to secure that
> port so that only your Kubernetes nodes can access it.  You may wish to add a `nodeSelector` to the 
> manifest to control where Typha runs (for example on the master) and then use {{site.tseeprodname}} host protection
> to secure those hosts.
{: .alert .alert-danger}

We recommend enabling Typha if you have more than 50 Kubernetes nodes in your cluster.  Without Typha, the 
load on the API server and Felix's CPU usage increases substantially as the number of nodes is increased.
In our testing, beyond 100 nodes, both Felix and the API server use an unacceptable amount of CPU.

To enable Typha in either the {{site.tseeprodname}} networking manifest or the policy only manifest:

1. [Download the private, CNX-specific `typha` image](/{{page.version}}/getting-started/#images).

1. Import the file into the local Docker engine.

   ```
   docker load -i tigera_typha_{{site.data.versions[page.version].first.components["typha"].version}}.tar.xz
   ```

1. Confirm that the image has loaded by typing `docker images`.

   ```
   REPOSITORY            TAG               IMAGE ID       CREATED         SIZE
   tigera/typha          {{site.data.versions[page.version].first.components["typha"].version}}  e07d59b0eb8a   2 minutes ago   30.8MB
   ```

1. Retag the image as desired and necessary to load it to your private registry.

1. If you have not configured your local Docker instance with the credentials that will
   allow you to access your private registry, do so now.

   ```
   docker login [registry-domain]
   ```

1. Use the following command to push the `typha` image to the private registry, replacing 
   `<YOUR_PRIVATE_DOCKER_REGISTRY>` with the location of your registry first.

   ```
   docker push {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   ```
   
1. Open the manifest that corresponds to your desired configuration. 
     - [Option 1: CNX policy with CNX networking](../kubernetes-datastore/calico-networking/1.7/calico.yaml){:target="_blank"}
     - [Option 2: CNX policy-only with user-supplied networking](../kubernetes-datastore/policy-only/1.7/calico.yaml){:target="_blank"}
   
   You should have a modified copy stored locally.

1. Change the `typha_service_name` variable in the ConfigMap from `"none"` to `"calico-typha"`.

1. Modify the replica count in the `calico-typha` Deployment section to the desired number of replicas:
    
   ```
   apiVersion: apps/v1beta1
   kind: Deployment
   metadata:
     name: calico-typha
     ...
   spec:
     ...
     replicas: <number of replicas>
   ```
   
   We recommend starting at least one replica for every 200 nodes and, at most, 20 replicas (since each 
   replica places some load on the API server).
   
   In production, we recommend starting at least 3 replicas to reduce the impact of rolling upgrades
   and failures.

   > **Note**: If you set `typha_service_name` without increasing the replica count from its default 
   > of `0` Felix will fail to start because it will try to connect to Typha but there 
   > will be no Typha instances to connect to.
   {: .alert .alert-info}
