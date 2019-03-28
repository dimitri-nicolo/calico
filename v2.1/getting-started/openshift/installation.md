---
title: Installing Tigera Secure EE on OpenShift
canonical_url: https://docs.tigera.io/v2.3/getting-started/openshift/installation
---

Installation of {{site.prodname}} in OpenShift is integrated in openshift-ansible.
The information below explains the variables which must be set during the standard
[Advanced Installation](https://docs.openshift.org/latest/install_config/install/advanced_install.html#configuring-cluster-variables).

## Before you begin

- Ensure that you meet the {{site.prodname}} [system requirements](requirements).

- Ensure that you have the [private registry credentials](../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../getting-started/#obtain-a-license-key).

## Pulling the private {{site.prodname}} images

{% include {{page.version}}/load-docker.md orchestrator="openshift" yaml="calico" %}

## Installation

Before we begin, apply the following two patches to your OpenShift install:

1. Add `"nodename_file_optional": true` to {{site.prodname}}'s CNI config:

   ```
   sed -i 's/"name": "calico",$/"name": "calico",\n  "nodename_file_optional": true,/g' /usr/share/ansible/openshift-ansible/roles/calico/templates/10-calico.conf.j2
   ```

1. Configure NetworkManager to not manage calico interfaces. On each host in your cluster, run:
   ```
   cat << EOF | sudo tee /etc/NetworkManager/conf.d/calico.conf
   [keyfile]
   unmanaged-devices=interface-name:cali*;interface-name:tunl0
   EOF
   ```

   Then restart NetworkManager to uptake the changes:

   ```
   sudo systemctl daemon-reload
   sudo systemctl restart NetworkManager
   ```

1. **For users running OpenShift v3.9.0 only**: apply the container_runtime hotfix for OpenShift:

   ```
   echo "- role: container_runtime" >> /usr/share/ansible/openshift-ansible/roles/calico/meta/main.yml
   ```

To install {{site.prodname}} in OpenShift, set the following `OSEv3:vars` in your
inventory file:

  - `os_sdn_network_plugin_name=cni`
  - `openshift_use_calico=true`
  - `openshift_use_openshift_sdn=false`
  - `calico_node_image=<YOUR-REGISTRY>/tigera/cnx-node:{{site.data.versions[page.version].first.components["cnx-node"].version}}`

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
calico_node_image=<YOUR-REGISTRY>/tigera/cnx-node:{{site.data.versions[page.version].first.components["cnx-node"].version}}
calico_url_policy_controller={{page.registry}}{{site.imageNames["calicoKubeControllers"]}}:{{site.data.versions[page.version].first.components["calico/kube-controllers"].version}}
calico_url_ipam={{site.data.versions[page.version].first.components["calico/cni"].download_calico_ipam_url}}
calico_url_cni={{site.data.versions[page.version].first.components["calico/cni"].download_calico_url}}

[masters]
master1

[nodes]
node1

[etcd]
etcd1
```

You are now ready to execute the ansible provision which will install {{site.prodname}}. Note that by default,
{{site.prodname}} will connect to the same etcd that OpenShift uses, and in order to do so, will distribute etcd's
certs to each node. If you would prefer {{site.prodname}} not connect to the same etcd as OpenShift, you may modify the install
such that {{site.prodname}} connects to an etcd you have already set up by following the [dedicated etcd install guide](dedicated-etcd).

Once execution is complete, apply the OpenShift patches for {{site.prodname}}'s kube-controllers:

```
oc apply -f kube-controllers-patch.yaml
```

>[Click here to view kube-controllers-patch.yaml](kube-controllers-patch.yaml)

Now, configure kube-proxy to forward traffic between hosts. On each node in your cluster, open `/etc/origin/node/node-config.yaml` and add a `cluster-cidr` under `proxyArguments`.
You want to populate `cluster-cidr` with the value of `osm_cluster_network_cidr` that you may have set in your ansible inventory file.

```
cluster-cidr:
- <osm_cluster_network_cidr value>
```

If you have not explicitly set `osm_cluster_network_cidr`, the default value is `10.128.0.0/14`. Your `node-config.yaml` should look similar to:

```
...
proxyArguments:
  cluster-cidr:
  - 10.128.0.0/14
  proxy-mode:
     - iptables
...
```

Once you have set the correct kube-proxy arguments, restart the OpenShift node service. This command will differ depending on if your cluster is running OpenShift Origin or OpenShift Container Platform.

OpenShift Container Platform (OCP):

```
sudo systemctl restart atomic-openshift-node
```

OpenShift Origin:

```
sudo systemctl restart origin-node
```

{% include {{page.version}}/apply-license.md init="openshift" %}

## Installing {{site.prodname}} Manager

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
   {: .alert .alert-info}

1. Download [calico-config.yaml](calico-config.yaml).

   ```bash
   curl {{site.url}}/{{page.version}}/getting-started/openshift/calico-config.yaml -O
   ```

1. To make the following commands easier to copy and paste, set two environment variables
   called `ETCD_ENDPOINTS` and `CNX_MANAGER_ADDR` containing the addresses of your etcd
   cluster and {{site.prodname}} Manager web interface, respectively. An example follows.

   ```bash
   ETCD_ENDPOINTS=10.90.89.100:2379,10.90.89.101:2379 \
   CNX_MANAGER_ADDR=127.0.0.1:30003
   ```

1. Use the following command to replace the value `<ETCD_ENDPOINTS>` in `calico-config.yaml`
   with the address of your etcd cluster.

   ```shell
   sed -i -e "s?<ETCD_ENDPOINTS>?$ETCD_ENDPOINTS?g" calico-config.yaml
   ```

1. Use the following command to replace the value of `<CNX_MANAGER_ADDR>` in `calico-config.yaml`
   with the address of your cnx-manager service.

   ```shell
   sed -i -e "s?<CNX_MANAGER_ADDR>?$CNX_MANAGER_ADDR?g" calico-config.yaml
   ```

1. Apply it:

       oc apply -f ./calico-config.yaml

{% include {{page.version}}/cnx-mgr-install.md init="openshift" %}

## Installing Policy Violation Alerting

Below, we'll cover how to enable metrics in {{site.prodname}} and how to launch Prometheus using Prometheus-Operator.

### Enable Metrics

**Prerequisite**: `calicoctl` [installed](../../usage/calicoctl/install) and [configured](../../usage/calicoctl/configure/). We recommend [installing](../../usage/calicoctl/install#installing-calicoctl-as-a-container-on-a-single-host) calicoctl as a container in OpenShift.

Enable metrics in {{site.prodname}} for OpenShift by updating the global `FelixConfiguration` resource (`default`) and opening up the necessary port on the host.

{% include {{page.version}}/enable-felix-prometheus-reporting.md %}

1. Allow Prometheus to scrape the metrics by opening up the port on each host:

   ```
   iptables -I INPUT -p tcp --dport 9081 -j ACCEPT
   ```

#### Configure Prometheus

With metrics enabled, you are ready to monitor `{{site.nodecontainer}}` by scraping the endpoint on each node
in the cluster. If you do not have your own Prometheus, the following commands will launch a Prometheus
Operator, Prometheus, and Alertmanager instances for you.

1. Allow Prometheus to run as root:

   ```
   oc adm policy add-scc-to-user --namespace=calico-monitoring anyuid -z default
   ```

1. Allow Prometheus to configure and use a security context.

   ```
   oc adm policy add-scc-to-user anyuid system:serviceaccount:calico-monitoring:prometheus
   ```

1. Allow sleep pod to run with host networking:

   ```
   oc adm policy add-scc-to-user --namespace=calico-monitoring hostnetwork -z default
   ```

1. Apply the OpenShift patches for Prometheus Operator:

   ```
   oc apply -f operator-openshift-patch.yaml
   ```

   >[Click here to view operator-openshift-patch.yaml](operator-openshift-patch.yaml)

   > **Note**: If you are installing on OpenShift v3.9.0, you will need to allow all pods in the `kube-system` namespace on each node.
   This can be done by adding the `openshift.io/node-selector` annotation to the `kube-system` namespace. Add this annotation by running the following.
   ```
   oc annotate ns kube-system openshift.io/node-selector="" --overwrite
   ```
   {: .alert .alert-info}

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

### Policy query with calicoq

Once {{site.prodname}} is installed in OpenShift, each node is automatically configured with
a `calicoctl.cfg` (owned by the root user) which is used by {{site.prodname}} to locate and authenticate requests to etcd.

We recommend installing `calicoq` as a container in OpenShift. Refer to [Installing calicoq as a container on a single host](../../usage/calicoq/#installing-calicoq-as-a-container-on-a-single-host) for instructions.

> **Note**: Ensure that your configuration takes into account TLS-enabled etcd in OpenShift.
{: .alert .alert-info}

See the [calicoq reference section](../../reference/calicoq/) for more information on using `calicoq`.
