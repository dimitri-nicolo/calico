---
title: Installing CNX on OpenShift
---

Installation of {{site.prodname}} in OpenShift is integrated in openshift-ansible v3.6.
The information below explains the variables which must be set during
during the standard [Advanced Installation](https://docs.openshift.org/latest/install_config/install/advanced_install.html#configuring-cluster-variables).

## Preparation

1. Load the `cnx-apiserver`, `cnx-node`, and `cnx-manager` binaries into your
   private Docker registry.

   See [Obtaining {{site.prodname}}][obtaining-cnx] for more information
   on how to acquire these images.

1. Ensure your Docker daemon on all OpenShift nodes and masters is authenticated to pull images from that registry.

   > **Note**: See the [OCP Advanced Installation Instructions][ocp-advanced-install]
   for more information on setting up custom Docker registries using the OpenShift installer.
   {: .alert .alert-info}


## Installation

To install {{site.prodname}} in OpenShift, set the following `OSEv3:vars` in your
inventory file:

  - `os_sdn_network_plugin_name=cni`
  - `openshift_use_calico=true`
  - `openshift_use_openshift_sdn=false`
  - `calico_node_image={{site.nodecontainer}}`

Also ensure that you have an explicitly defined host in the `[etcd]` group.

**Sample Inventory File:**

```
[OSEv3:children]
masters
nodes
etcd

[OSEv3:vars]
os_sdn_network_plugin_name=cni
openshift_use_calico=true
openshift_use_openshift_sdn=false
calico_node_image=calico/node:{{site.data.versions[page.version].first.title}}

[masters]
master1

[nodes]
node1

[etcd]
etcd1
```

You are now ready to execute the ansible provision which will install {{site.prodname}}. Note that by default, 
{{site.prodname}} will connect to the same etcd that OpenShift uses, and in order to do so, will distribute etcd's
certs to each node. If you would prefer Calico not connect to the same etcd as OpenShift, you may modify the install
such that Calico connects to an etcd you have already set up by following the [dedicated etcd install guide](dedicated-etcd).

### Installing the CNX Manager

1. Create a Kubernetes secret from your etcd certificates. Example command:

   ```
   kubectl create -n kube-system secret generic calico-etcd-secrets \
   --from-file=etcd-ca=/etc/origin/calico/calico.etcd-ca.crt \
   --from-file=etcd-cert=/etc/origin/calico/calico.etcd-client.crt \
   --from-file=etcd-key=/etc/origin/calico/calico.etcd-client.key
   ```

   >{{site.prodname}} APIServer and Manager require etcd connection information and
   >certificates to be stored in kubernetes objects.
   >The following preparation steps will upload this data.
   >
   >If you are unsure of what to set these values to, check >`/etc/calico/calicoctl.cfg`
   >on your master node, which will show what `calicoctl` is currently using to connect to etcd.
   {: .alert .alert-info}

1. Download the {{site.prodname}} configmap: [calico-config.yaml](calico-config.yaml)

1. Update the CNX configmap with the path to your private Docker registry.
   Substitute `$ETCD_ENDPOINTS` with the address of your etcd cluster.

       sed -i -e "s?<ETCD_ENDPOINTS>?$ETCD_ENDPOINTS?g" calico-config.yaml

1. Apply it:

       kubectl apply -f ./calico-config.yaml

1. Download the {{site.prodname}} manifest:

   - [cnx-etcd.yaml](/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml)

1. Rename the file `cnx.yaml` - this is what subsequent instructions will refer to:

       mv cnx-etcd.yaml cnx.yaml

1. Update the login method to "Token":

       sed -i -e 's/tigera.cnx-manager.authentication-type:.*$/tigera.cnx-manager.authentication-type: "Token"/g' cnx.yaml

1. Update the manifest with the path to your private Docker registry. Substitute
   `mydockerregistry:5000` with the location of your Docker registry.

       sed -i -e 's?<YOUR_PRIVATE_DOCKER_REGISTRY>?mydockerregistry:5000?g' cnx.yaml

1. Open the file in a text editor, and update the ConfigMap `tigera-cnx-manager-config`
   according to the instructions in the file and your chosen authentication method.

   You might want to reconfigure the service that gets traffic to the CNX Manager
   web server as well.

   {{site.prodname}} for OpenShift requires SSL certs to connect to etcd, so be sure to uncomment
   all secrets connected to `calico-secrets`.

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   CNX Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the CNX Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` service.

1. Store those credentials as `cert` and `key` in a secret named
   `cnx-manager-tls`.  For example:

       kubectl create secret generic cnx-manager-tls -n kube-system --from-file=cert=/path/to/certificate --from-file=key=/path/to/key

1. Apply the manifest to install CNX Manager and the CNX API server.

   ```
   kubectl apply -f cnx.yaml
   ```

1. Allow cnx-manager to run as root:

       oadm policy add-scc-to-user anyuid system:serviceaccount:kube-system:cnx-manager

1. Configure authentication to allow CNX Manager users to edit policies.  Consult the
   [CNX Manager](../../reference/cnx/policy-editor) and
   [Tiered policy RBAC](../../reference/cnx/rbac-tiered-policies)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.

#### Signing into the CNX Manager UI in OpenShift

The CNX Manager requires an [OpenShift access token](https://docs.openshift.com/container-platform/3.7/install_config/configuring_authentication.html#token-options) for login. You will need
to consult the OpenShift docs on how to generate an access token using your configured
identity provider.

If you are unsure, a simple way to generate this token is to log into the OpenShift UI, then visit `<master>/oauth/token/request` for your token.

### Installing Policy Violation Alerting

Policy Violation Alerting is mostly the same in {{site.prodname}} for OpenShift as it is in {{site.prodname}}. Below,
we'll cover how to enable metrics in {{site.prodname}} and how to launch Prometheus using Prometheus-Operator.

#### Enable Metrics

**Prerequisite**: `calicoctl` [installed](../../usage/calicoctl/install) and [configured](../../usage/calicoctl/configure).

Enable metrics in {{site.prodname}} for OpenShift by updating the global `FelixConfiguration` resource (`default`) and opening up the necessary port on the host.

{% include {{page.version}}/enable-felix-prometheus-reporting.md %}

1. Allow Prometheus to scrape the metrics by opening up the port on the host:

   ```
   iptables -A INPUT -p tcp --dport 9081 -j ACCEPT
   iptables -I INPUT 1 -p tcp --dport 9081 -j ACCEPT
   ```

#### Configure Prometheus

With metrics enabled, you are ready to monitor `calico/node` by scraping the endpoint on each node
in the cluster. If you do not have your own Prometheus, the following commands will launch a Prometheus
Operator, Prometheus, and Alertmanager instances for you.

1. Allow Prometheus to run as root:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring anyuid -z default
   ```

1. Allow Prometheus to configure and use a Security Context.

   ```
   oadm policy add-scc-to-user anyuid system:serviceaccount:calico-monitoring:prometheus
   ```

1. Allow sleep pod to run with host networking:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring hostnetwork -z default
   ```

1. Apply the Openshift patches for Prometheus Operator:

   ```
   oc apply -f operator-openshift-patch.yaml
   ```

   >[Click here to view operator-openshift-patch.yaml](operator-openshift-patch.yaml)

1. Apply the Prometheus Operator manifest:

   ```
   oc apply -f operator.yaml
   ```

   >[Click here to view operator.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml)

1. Apply Prometheus and Alertmanager:

   ```
   oc apply -f monitor-calico.yaml
   ```

   >[Click here to view monitor-calico.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml)

Once running, access Prometheus and Alertmanager using the NodePort from the created service.
See [Policy Violation Alerting](../../reference/cnx/policy-violations) for more information.

#### Policy Query with calicoq

Once {{site.prodname}} is installed in OpenShift, each node is automatically configured with
a `calicoctl.cfg` (owned by the root user) which is used by {{site.prodname}} to locate and authenticate
requests to etcd.

To install `calicoq` in OpenShift:

1. Download it to any node.
1. Run it as root user.

See the [calicoq reference](../../reference/calicoq/) for more information on using `calicoq`.

### Next Steps

- [Policy Auditing](../../reference/cnx/policy-auditing).

[obtaining-cnx]: {{site.baseurl}}/{{page.version}}/getting-started/
[ocp-advanced-install]: https://access.redhat.com/documentation/en-us/openshift_container_platform/3.6/html-single/installation_and_configuration/#system-requirements

