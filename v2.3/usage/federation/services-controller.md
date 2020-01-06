---
title: Configuring federated services
canonical_url: https://docs.tigera.io/v2.3/usage/federation/services-controller
---

Federated Services is a feature that provides cross-cluster service discovery for your local cluster. It uses the
separately installable Federated Services Controller
to provide cross-cluster service discovery. It is expected that this will be used in conjunction with federated
endpoint identity, although this controller is optional if you have an alternative service discovery mechanism.

The Federated Services Controller populates the endpoints of a local service from the endpoints of the set of selected
services across all clusters (local and remote). Unlike Kubernetes federation, the endpoint addresses for the remote pods
are the pod IPs rather than the service IPs. The conservation of the pod IP in the federated service allows fine-grained
policy to be applied between the clusters.

The Federated Services Controller accesses service and endpoints data in the remote clusters directly through the
Kubernetes API. This means that if the remote cluster is using etcd for the {{site.tseeprodname}} datastore, it is necessary to configure
both etcd access details and Kubernetes API datastore access details in the same Remote Cluster Configuration resource. See
[Configuring access to remote clusters](./configure-rcc) for more details.

## Configuring a federated service

A federated service is created as a standard Kubernetes service exposing a ClusterIP and with no pod selector. Rather
than specifying a pod selector, annotations are used to specify the set of services (referred to as the backing
services) whose endpoints will be consolidated into the endpoints of the federated service.

The following configuration options are valid through the annotations:

| Annotation | Description |
| --- | --- |
| `federation.tigera.io/serviceSelector` | {::nomarkdown}<p>This option is used to specify which services are used in the federated service. This field must be specified for the service to be federated. If the value is incorrectly specified, the service will not be federated and endpoint data will be removed from the service. Warning logs will be output in the controller indicating any issues processing this value.</p><p>The format is a standard {{site.tseeprodname}} selector (i.e. the same as {{site.tseeprodname}} policy resources) and selects services based on their labels.</p><p>Only services in the same namespace as the federated service will be included. This implies namespace names across clusters are linked (this is a basic premise of Federated Endpoint Identity).</p>{:/} |

The Federated Services Controller uses the serviceSelector annotation to select the backing services in the same
namespace whose labels match the specified selector. Services are selected from the local and remote clusters.
The controller consolidates the service endpoints from the selected services whose port name and protocol match those
configured on the federated service. The consolidated set of endpoints is then configured in the endpoints resource for
the federated service.

The following labels may also be matched on. These are implicitly added to each service, but the label will not appear
when viewing the service through `kubectl.`

| Label | Description |
| --- | --- |
| `federation.tigera.io/remoteClusterName` | {::nomarkdown}<p>The label is added to all of the remote services, and the value corresponds to the name of the Remote Cluster Configuration that specifies the remote cluster. For services in the local cluster, this label is not added.</p><p>You may wish to use this label if you need to restrict which clusters the services are selected from.</p>{:/} |

The controller monitors any changes to the endpoints for the backings services and maintains the set of endpoints for
each locally federated service. The controller makes no configuration changes to any remote cluster.

The endpoints data configured in the federated service is slightly modified from the original data of the backing service.
For backing services on remote clusters, the `targetRef.name` field in the federated service will be updated to the
form `<Remote Cluster Configuration name>/<original name>`.

> **Please note**:
> -  If a spec.Selector is also specified, the Federated Services Controller will not federate the service.
> -  A service is not included as a backing service of a federated service if that service has any federated services
>    annotations. This means it is not possible to federate another federated service.
> -  The corresponding endpoints resource should not be created by the user. The endpoints resource will be created and
>    managed by the controller. The Federated Services Controller will not update an endpoints resource that does appear to have
>    been created by the controller.
> -  Endpoints will only be selected when the service port name and protocol in the federated service matches the port name
>    and protocol in the backing service.
> -  The target port number in the federated service ports is not used.
> -  The selector annotation used for federation selects services, not pods.
{: .alert .alert-info}

## Accessing a federated service

To access a federated service, the simplest approach is through its corresponding DNS name.

By default, Kubernetes adds DNS entries to access a service locally. For a service called `my-svc` in the namespace
`my-namespace`, the following DNS entry would be added to access the service within the local cluster:

```
    my-svc.my-namespace.svc.cluster.local
```

DNS lookup for this name returns the fixed ClusterIP address assigned for the federated service. The ClusterIP is translated
in iptables to one of the federated service endpoint IPs (it will be load balanced across all of the endpoints).

## Operational flow

As an operator, the expected flow of configuration would be as follows:
1. On each cluster that is providing a particular service, e.g. a set of pods running an application called `my-app`,
   create your service resources as per normal. Configure each service with a common label key and value that can be
   used to identify the common set of services across your clusters (e.g. `run=my-app`). These services should all be in
   the same namespace.
   -  Kubernetes will manage that service, populating the service endpoints from the Pods that match the selector
      configured in the service spec.
1. On a cluster that needs to access the federated set of Pods that are running the application `my-app`, create a
   service on that cluster leaving the spec selector blank and setting the `federation.tigera.io/serviceSelector`
   annotation to be a {{site.tseeprodname}} selector which selects the previously configured services using the chosen label match
   (e.g. `run == "my-app"`).
    - {{site.tseeprodname}} Federated Services Controller will manage this service, populating the service endpoints from
      all of the services that match the service selector configured in the annotation.
1. Any application can access the federated service using the local DNS name for that service.

> **Reminder**:
> -  The spec selector field in the service is used by Kubernetes to select *pods* in the local cluster. This is a
>    Kubernetes style selector.
> -  The spec selector field in the annotation is used by {{site.tseeprodname}} Federated Services Controller to select
>    *services* across the federated set of clusters. This is a {{site.tseeprodname}} style selector.
{: .alert .alert-success}

## Example

For example, suppose both your local cluster and a remote cluster have the following service defined:

```$yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    run: my-app
  name: my-app
  namespace: default
spec:
  selector:
    run: my-app
  ports:
  - name: my-app-ui
    port: 80
    protocol: TCP
    targetPort: 9000
  - name: my-app-console
    port: 81
    protocol: TCP
    targetPort: 9001
  type: ClusterIP
```

This service definition exposes two ports for the application `my-app`. One port for accessing a UI and the other for
accesing a management console. The service has a kubernetes selector specified in the spec, which implies the endpoints
for this service will be automatically populated by kubernetes from matching pods within the services own cluster.

To define a federated service on your local cluster that federates the web access port for both the local and remote
service, you would create a service resource on your local cluster as follows:

```$yaml
apiVersion: v1
kind: Service
metadata:
  name: my-app-federated
  namespace: default
  annotations:
    federation.tigera.io/serviceSelector: run == "my-app"
spec:
  ports:
  - name: my-app-ui
    port: 8080
    protocol: TCP
  type: ClusterIP
```

There is no `spec.selector` specified so Kubernetes will not manage this service. Instead, there is a `federation.tigera.io/selector`
annotation which instructs the Federated Services Controller to manage this service. The controller will match the my-app
services (matching the run label) on both the local and remote clusters, consolidate the endpoints from the my-app-ui TCP
port for both of those services. Since the federated service does not specify the my-app-console port, the controller will not
include those endpoints in the federated service.

The endpoints data for the federated service would be similar to the following (noting the name of the remote cluster is
included in `targetRef.name`).

```$yaml
apiVersion: v1
kind: Endpoints
metadata:
  creationTimestamp: 2018-07-03T19:41:38Z
  annotations:
    federation.tigera.io/serviceSelector: run == "my-app"
  name: my-app-federated
  namespace: default
  resourceVersion: "701812"
  selfLink: /api/v1/namespaces/default/endpoints/my-app-federated
  uid: 1a0427e8-7ef9-11e8-a24c-0259d75c6290
subsets:
- addresses:
  - ip: 192.168.93.12
    nodeName: node1.localcluster.tigera.io
    targetRef:
      kind: Pod
      name: my-app-59cf48cdc7-frf2t
      namespace: default
      resourceVersion: "701655"
      uid: 19f5e914-7ef9-11e8-a24c-0259d75c6290
  ports:
  - name: my-app-ui
    port: 80
    protocol: TCP
- addresses:
  - ip: 192.168.0.28
    nodeName: node1.remotecluster.tigera.io
    targetRef:
      kind: Pod
      name: remotecluster/my-app-7b6f758bd5-ctgbh
      namespace: default
      resourceVersion: "701648"
      uid: 19e2c841-7ef9-11e8-a24c-0259d75c6290
  ports:
  - name: my-app-ui
    port: 80
    protocol: TCP
```
