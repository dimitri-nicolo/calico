---
title: Installing CNX on OpenShift
---

Installation of {{site.prodname}} in OpenShift is integrated in openshift-ansible v3.6.
The information below explains the variables which must be set during the standard
[Advanced Installation](https://docs.openshift.org/latest/install_config/install/advanced_install.html#configuring-cluster-variables).

{% include {{page.version}}/load-docker.md orchestrator="openshift" %}

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

### Installing {{site.prodname}} Manager

1. Create a Kubernetes secret from your etcd certificates. Example command:

   ```
   kubectl create -n kube-system secret generic calico-etcd-secrets \
   --from-file=etcd-ca=/etc/origin/calico/calico.etcd-ca.crt \
   --from-file=etcd-cert=/etc/origin/calico/calico.etcd-client.crt \
   --from-file=etcd-key=/etc/origin/calico/calico.etcd-client.key
   ```

   >{{site.prodname}} APIServer and Manager require etcd connection information and
   >certificates to be stored in Kubernetes objects.
   >The following preparation steps will upload this data.
   >
   >If you are unsure of what to set these values to, check `/etc/calico/calicoctl.cfg`
   >on your master node, which will show what `calicoctl` is currently using to connect to etcd.
   {: .alert .alert-info}

1. Download [calico-config.yaml](calico-config.yaml).

1. Use the following command to replace the value `<ETCD_ENDPOINTS>` in `calico-config.yaml` 
   with the address of your etcd cluster:

   **Command**
   ```shell
   sed -i -e "s?<ETCD_ENDPOINTS>?<REPLACE_ME>?g" calico-config.yaml
   ```
   
   **Example**
   ```shell
   sed -i -e 's?<ETCD_ENDPOINTS>?https://etcd:2379?g' calico-config.yaml
   ```

1. Use the following command to replace the value of `<CNX_MANAGER_ADDR>` in `calico-config.yaml` with the address of your cnx-manager service:

   **Command**
   ```shell
   sed -i -e "s?<CNX_MANAGER_ADDR>?<REPLACE_ME>?g" calico-config.yaml
   ```
   
   **Example**
   ```shell
   sed -i -e 's?<CNX_MANAGER_ADDR>?127.0.0.1:30003?g' calico-config.yaml
   ```

1. Apply it:

       oc apply -f ./calico-config.yaml

1. [Open cnx-etcd.yaml in a new tab](../kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.

{% include {{page.version}}/cnx-mgr-install.md orchestrator="openshift" %}

#### Signing into the CNX Manager UI in OpenShift

The CNX Manager requires an [OpenShift access token](https://docs.openshift.com/container-platform/3.7/install_config/configuring_authentication.html#token-options) for login. You will need
to consult the OpenShift docs on how to generate an access token using your configured
identity provider.

If you are unsure, a simple way to generate this token is to log into the OpenShift UI, then visit `<master>/oauth/token/request` for your token.

### Installing Policy Violation Alerting

Below, we'll cover how to enable metrics in {{site.prodname}} and how to launch Prometheus using Prometheus-Operator.

#### Enable Metrics

**Prerequisite**: `calicoctl` [installed](../../usage/calicoctl/install) and [configured](../../usage/calicoctl/configure/).

Enable metrics in {{site.prodname}} for OpenShift by updating the global `FelixConfiguration` resource (`default`) and opening up the necessary port on the host.

{% include {{page.version}}/enable-felix-prometheus-reporting.md %}

1. Allow Prometheus to scrape the metrics by opening up the port on the host:

   ```
   iptables -A INPUT -p tcp --dport 9081 -j ACCEPT
   iptables -I INPUT 1 -p tcp --dport 9081 -j ACCEPT
   ```

#### Configure Prometheus

With metrics enabled, you are ready to monitor `{{site.nodecontainer}}` by scraping the endpoint on each node
in the cluster. If you do not have your own Prometheus, the following commands will launch a Prometheus
Operator, Prometheus, and Alertmanager instances for you.

1. Allow Prometheus to run as root:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring anyuid -z default
   ```

1. Allow Prometheus to configure and use a security context.

   ```
   oadm policy add-scc-to-user anyuid system:serviceaccount:calico-monitoring:prometheus
   ```

1. Allow sleep pod to run with host networking:

   ```
   oadm policy add-scc-to-user --namespace=calico-monitoring hostnetwork -z default
   ```

1. Apply the OpenShift patches for Prometheus Operator:

   ```
   oc apply -f operator-openshift-patch.yaml
   ```

   >[Click here to view operator-openshift-patch.yaml](operator-openshift-patch.yaml)

1. Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml) manifest.

   ```
   oc apply -f operator.yaml
   ```

1. Wait for the `alertmanagers.monitoring.coreos.com`, `prometheuses.monitoring.coreos.com` and `servicemonitors.monitoring.coreos.com` custom resource definitions to be created. Check by running:

   ```
   oc get customresourcedefinitions
   ```

1. Apply the [monitor-calico.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml) manifest which will
  install Prometheus and Alertmanager.

   ```
   oc apply -f monitor-calico.yaml
   ```

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
