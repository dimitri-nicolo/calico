---
title: Component architecture
description: Understand the Calico Enterprise components and the basics of BGP networking.
canonical_url: '/reference/architecture/overview'
---

### {{site.prodname}} components

The following diagram shows the required and optional {{site.prodname}} components for a Kubernetes, on-premises deployment with networking and network policy.

![calico-components]({{site.baseurl}}/images/architecture-ee.svg)

{{site.prodname}} provide additional value-added components on top of the basic open-source Calico components.

**{{site.prodname}} components**

 - [API server](#api-server)
 - [cnx-node](#cnx-node)
 - [fluentd](#fluentd)
 - [Prometheus](#prometheus)
 - [Elasticsearch and Kibana](#elasticsearch-and-kibana)
 - [Manager UI](#manager-ui)
 - [kubectl](#kubectl)

**Calico components**

 - [Felix](#felix)
 - [BIRD](#bird)
 - [confd](#confd)
 - [Dikastes](#dikastes)
 - [CNI plugin](#cni-plugin)
 - [Datastore plugin](#datastore-plugin)
 - [IPAM plugin](#ipam-plugin)
 - [kube-controllers](#kube-controllers)
 - [Typha](#typha)
 - [calicoctl and calicoq](#calicoctl-and-calicoq)

**Cloud orchestrator plugins**

 - [Plugins for cloud orchestrators](#plugins-for-cloud-orchestrators)

### API server

**Main task**: Handles requests for all {{site.prodname}} API resources.

The APIServer installs the Tigera API server and related resources.  The Kubernetes API server proxies requests for {{site.prodname}} API resources to the {{site.prodname}} API server through an aggregation layer. [API server]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.APIServer).

### cnx-node

**Main task**: Bundles together the various components required for networking containers with {{site.prodname}}. The key components are:

- Felix
- BIRD
- confd

The calico repository contains the Dockerfile for cnx-node, along with various configuration files to configure and “glue” these components together. In addition, we use runit for logging (svlogd) and init (runsv) services. [cnx-node]({{site.baseurl}}/reference/node/configuration).

### fluentd

**Main task**: Stores Elasticsearch flow and DNS logs. Opensource data collector for unified logging. {% include open-new-window.html text='fluentd open source' url='https://www.fluentd.org/' %}.


### Prometheus

**Main task**: Provides metrics on calico/nodes from Felix. Optional open-source toolkit for systems monitoring and alerting. [Prometheus metrics]({{site.baseurl}}/reference/felix/prometheus), and [Configure Prometheus]({{site.baseurl}}/maintenance/monitor/).


### Elasticsearch and Kibana 

**Main task**: Built-in search-engine and visualization dashboard for {{site.prodname}}. Installed and configured by default for easy onboarding. [Elasticsearch]({{site.baseurl}}/visibility/).


### Manager UI

**Main task**: Provides network traffic visibility, centralized multi-cluster management, threat-defense troubleshooting, and compliance to multiple roles/stakeholders. Optional user interface. [Manager UI]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Manager).


### kubectl

**Main task**: Command line interface to create, read, update, and delete Calico objects. `calicoctl` and `calicoq` command lines are also available on any host with network access to the {{site.prodname}} datastore as either a binary or a container. {% include open-new-window.html text='kubectl' url='https://kubernetes.io/docs/reference/kubectl/overview/' %}. 

### Felix

**Main task**: Programs routes and ACLs, and anything else required on the host to provide desired connectivity for the endpoints on that host. Runs on each machine that hosts endpoints. Runs as an agent daemon. [Felix resource]({{site.baseurl}}/reference/resources/felixconfig).

Depending on the specific orchestrator environment, Felix is responsible for:

- **Interface management**
    
    Programs information about interfaces into the kernel so the kernel can correctly handle the traffic from that endpoint. In particular, it ensures that the host responds to ARP requests from each workload with the MAC of the host, and enables IP forwarding for interfaces that it manages. It also monitors interfaces to ensure that the programming is applied at the appropriate time.

- **Route programming**
   
    Programs routes to the endpoints on its host into the Linux kernel FIB (Forwarding Information Base). This ensures that packets destined for those endpoints that arrive on at the host are forwarded accordingly.

- **ACL programming**
    
    Programs ACLs into the Linux kernel to ensure that only valid traffic can be sent between endpoints, and that endpoints cannot circumvent Calico Enterprise security measures.

- **State reporting**

    Provides network health data. In particular, it reports errors and problems when configuring its host. This data is written to the datastore so it visible to other components and operators of the network.

> **Note**: `{{site.nodecontainer}}` can be run in *policy only mode* where Felix runs without BIRD and confd. This provides policy management without route distribution between hosts, and is used for deployments like managed cloud providers. You enable this mode by setting the environment variable, `CALICO_NETWORKING=false` before starting the node.
{: .alert .alert-info}

### BIRD

**Main task**: Gets routes from Felix and distributes to BGP peers on the network for inter-host routing. Runs on each node that hosts a Felix agent. Open source, internet routing daemon. [BIRD]({{site.baseurl}}/reference/node/configuration#content-main).

The BGP client is responsible for:

- **Route distribution**

    When Felix inserts routes into the Linux kernel FIB, the BGP client distributes them to other nodes in the deployment. This ensures efficient traffic routing for the deployment.

- **BGP route reflector configuration**

    BGP route reflectors are often configured for large deployments rather than a standard BGP client. (Standard BGP requires that every BGP client be connected to every other BGP client in a mesh topology, which is difficult to maintain.) 
    For redundancy, you can seamlessly deploy multiple BGP route reflectors. Note that BGP route reflectors are involved only in control of the network: endpoint data does not passes through them. When the {{site.prodname}} BGP client advertises 
    routes from its FIB to the route reflector, the route reflector advertises those routes to the other nodes in the deployment.


### confd

**Main task**: Monitors {{site.prodname}} datastore for changes to BGP configuration and global defaults such as AS number, logging levels, and IPAM information. Open source, lightweight configuration management tool. 

Confd dynamically generates BIRD configuration files based on the updates to data in the datastore. When the configuration file changes, confd triggers BIRD to load the new files. [Configure confd]({{site.baseurl}}/reference/node/configuration#content-main), and {% include open-new-window.html text='confd project' url='https://github.com/kelseyhightower/confd' %}.


### Dikastes

**Main task**: Enforces network policy for Istio service mesh. Runs on a cluster as a sidecar proxy to Istio Envoy. 

(Optional) {{site.prodname}} enforces network policy for workloads at both the Linux kernel (using iptables, L3-L4), and at L3-L7 using a Envoy sidecar proxy called Dikastes, with cryptographic authentication of requests. Using multiple enforcement points establishes the identity of the remote endpoint based on multiple criteria. The host Linux kernel enforcement protects your workloads even if the workload pod is compromised, and the Envoy proxy is bypassed. [Dikastes]({{site.baseurl}}/reference/dikastes/configuration), and {% include open-new-window.html text='Istio docs' url='https://istio.io/latest/docs/setup/install/' %}.


### CNI plugin

**Main task**: Provides {{site.prodname}} networking for Kubernetes clusters. 

The Calico binary that presents this API to Kubernetes is called the CNI plugin, and must be installed on every node in the Kubernetes cluster. The Calico CNI plugin allows you to use Calico networking for any orchestrator that makes use of the CNI networking specification. Configured through the standard {% include open-new-window.html text='CNI configuration mechanism' 
url='https://github.com/containernetworking/cni/blob/master/SPEC.md#network-configuration' %}, and [Calico CNI plugin]({{site.baseurl}}/reference/cni-plugin/configuration).


### Datastore plugin

**Main task**: Increases scale by reducing each node’s impact on the datastore. It is one of the {{site.prodname}} [CNI plugins]({{site.baseurl}}/reference/cni-plugin/configuration).

- **Kubernetes API datastore (kdd)**

   The advantages of using the Kubernetes API datastore with Calico Enterprise are:

   - Simpler to manage because it does not require an extra datastore
   - Use Kubernetes RBAC to control access to Calico resources
   - Use Kubernetes audit logging to generate audit logs of changes to Calico resources

### IPAM plugin

**Main task**: Uses {{site.prodname}}’s IP pool resource to control how IP addresses are allocated to pods within the cluster. It is the default plugin used by most {{site.prodname}} installations. It is one of the {{site.prodname}} [CNI plugins]({{site.baseurl}}/reference/cni-plugin/configuration).


### kube-controllers


### Typha

**Main task**: Increases scale by reducing each node’s impact on the datastore. Runs as a daemon between the datastore and instances of Felix. Installed by default, but not configured. {% include open-new-window.html text='Typha description' url='https://github.com/projectcalico/typha' %}, and [Typha component]({{site.baseurl}}/reference/typha/).

Typha maintains a single datastore connection on behalf of all of its clients like Felix and confd. It caches the datastore state and deduplicates events so that they can be fanned out to many listeners. Because one Typha instance can support hundreds of Felix instances, it reduces the load on the datastore by a large factor. And because Typha can filter out updates that are not relevant to Felix, it also reduces Felix’s CPU usage. In a high-scale (100+ node) Kubernetes cluster, this is essential because the number of updates generated by the API server scales with the number of nodes.

### calicoctl and calicoq

**Main task**: Command line interface to create, read, update, and delete {{site.prodname}} objects. `calicoctl` command line is available on any host with network access to the {{site.prodname}} datastore as either a binary or a container. Requires separate installation. [calicoctl]({{site.baseurl}}/reference/calicoctl/), and [calicoq]({{site.baseurl}}/reference/calicoq/).

### Plugins for cloud orchestrators

**Main task**: Translates the orchestrator APIs for managing networks to the {{site.prodname}} data-model and datastore.

For cloud providers, {{site.prodname}} has a separate plugin for each major cloud orchestration platform. This allows {{site.prodname}} to tightly bind to the orchestrator, so users can manage the {{site.prodname}} network using their orchestrator tools. When required, the orchestrator plugin provides feedback from the {{site.prodname}} network to the orchestrator. For example, providing information about Felix liveness, and marking specific endpoints as failed if network setup fails.
