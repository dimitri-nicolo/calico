---
title: Installing Tigera Secure EE on OpenShift
---

Installation of {{site.tseeprodname}} in OpenShift is integrated in openshift-ansible.
The information below explains the variables which must be set during the standard
[Advanced Installation](https://docs.openshift.org/latest/install_config/install/advanced_install.html#configuring-cluster-variables).

## Before you begin

- Ensure that you meet the {{site.tseeprodname}} [system requirements](/{{page.version}}/getting-started/openshift/requirements).

- Ensure that you have the [private registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

{% include {{page.version}}/load-docker.md orchestrator="openshift" yaml="calico" %}

## <a name="install-cnx"></a>Installing {{site.tseeprodname}} and OpenShift

### Edit inventory file

To install {{site.tseeprodname}} in OpenShift, set the following `OSEv3:vars` in your
inventory file:

  - `os_sdn_network_plugin_name=cni`
  - `openshift_use_calico=true`
  - `openshift_use_openshift_sdn=false`
  - `calico_node_image=<YOUR-REGISTRY>/{{site.imageNames["cnx-node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}`
  - `calico_url_policy_controller=<YOUR-REGISTRY>/{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}}`
  - `calico_cni_image={{site.imageNames["cni"]}}:{{site.data.versions[page.version].first.components["calico/cni"].version}}`

For OpenShift Container Platform 3.11 also specify the following variables: 
  - `oreg_auth_user`
  - `oreg_auth_password`

If your private registry requires a pull secret, specify the path on each host to your
private registry credentials (typically a `~/.docker/config.json` file) in the following variable:
  - `calico_image_credentials`
  
Also ensure that you have an explicitly defined host in the `[etcd]` group.
 
A sample inventory file follows.

```
[OSEv3:children]
masters
nodes
etcd

[OSEv3:vars]
ansible_become=true
deployment_type=openshift-enterprise
os_sdn_network_plugin_name=cni
openshift_use_openshift_sdn=false
openshift_use_calico=true
calico_node_image=<YOUR-REGISTRY>/{{site.imageNames["cnx-node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
calico_url_policy_controller=<YOUR-REGISTRY>/{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}}
calico_cni_image={{site.imageNames["cni"]}}:{{site.data.versions[page.version].first.components["calico/cni"].version}}

[masters]
master1 ansible_host=127.0.0.1

[nodes]
node1 ansible_host=127.0.0.1 openshift_schedulable=true openshift_node_group_name='node-config-master-infra'

[etcd]
etcd1
```

### Execute Ansible provisioning script

You are now ready to execute the Ansible provision which will install {{site.tseeprodname}}. Note that by default,
{{site.tseeprodname}} will connect to the same etcd that OpenShift uses and distribute etcd's
certs to each node. If you would prefer {{site.tseeprodname}} not connect to the same etcd as OpenShift, you may modify the install
such that {{site.tseeprodname}} connects to an etcd you have already set up by following the [dedicated etcd install guide](dedicated-etcd).

{% include {{page.version}}/apply-license.md init="openshift" %}

{% include {{page.version}}/cnx-mgr-install.md init="openshift" %}

1. Download [oauth-client.yaml](/{{page.version}}/getting-started/openshift/installation/oauth-client.yaml).

   ```bash
   curl {{site.url}}/{{page.version}}/getting-started/openshift/installation/oauth-client.yaml -O
   ```

1. To make the following commands easier to copy and paste, set an environment variable called
   `CNX_MANAGER_ADDR` containing the address of your {{site.tseeprodname}} Manager web interface.
   An example follows.

   ```bash
   CNX_MANAGER_ADDR=127.0.0.1:30003
   ```

1. Use the following command to replace the value of `<CNX_MANAGER_ADDR>` in `oauth-client.yaml`
   with the address of your cnx-manager service.

   ```bash
   sed -i -e "s?<CNX_MANAGER_ADDR>?$CNX_MANAGER_ADDR?g" oauth-client.yaml
   ```

1. Apply it:

   ```bash
   oc apply -f oauth-client.yaml
   ```

## Installing metrics and logs

### Configure metrics and logs

With metrics enabled, you are ready to monitor `{{site.nodecontainer}}` by scraping the endpoint on each node
in the cluster. If you do not have your own Prometheus, the following commands will launch a Prometheus
Operator, Prometheus, and Alertmanager instances for you. They will also deploy Fluentd, and
optionally Elasticsearch and Kibana in order to enable logs.


1. Allow Prometheus to scrape the metrics by opening up the port on each host:

   ```
   iptables -I INPUT -p tcp --dport 9081 -j ACCEPT
   ```

1. For production installs, we recommend using your own Elasticsearch cluster. If you are performing a
   production install, do not complete any more steps on this page. Instead, refer to
   [Using your own Elasticsearch for logs](byo-elasticsearch) for the final steps.

   For demonstration or proof of concept installs, you can use the bundled
   [Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator). Continue to the
   next step to complete a demonstration or proof of concept install.

   > **Important**: The bundled Elasticsearch operator does not provide reliable persistent storage
   of logs or authenticate access to Kibana.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-monitor-install.md orch="openshift" elasticsearch="operator" %}

Once running, access Prometheus and Alertmanager using the NodePort from the created service.
See the [Metrics](/{{page.version}}/usage/metrics/) section for more information.

{% include {{page.version}}/gs-openshift-next-steps.md %}
