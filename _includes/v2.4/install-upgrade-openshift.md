{% unless include.upgrade %}
Installation of {{site.prodname}} in OpenShift is integrated in openshift-ansible.
The information below explains the variables which must be set during the standard
[Advanced Installation](https://docs.openshift.org/latest/install_config/install/advanced_install.html#configuring-cluster-variables).

## Before you begin

- Ensure that you meet the {{site.prodname}} [system requirements](/{{page.version}}/getting-started/openshift/requirements).

- Ensure that you have the [private registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).
{% endunless %}
{% include {{page.version}}/load-docker.md orchestrator="openshift" yaml="calico" %}

## <a name="install-cnx"></a>Installing {{site.prodname}} and OpenShift

### Edit inventory file

To install {{site.prodname}} in OpenShift, set the following `OSEv3:vars` in your
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
private registry credentials (typically a `~/.docker/config.json` file)in the following variable:
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
{% if include.upgrade %}
### Run the custom upgrade playbook

Once your inventory file has been properly configured, download the
[upgrade playbook](/{{page.version}}/getting-started/openshift/upgrade-calico.yaml){:target="_blank"}
and copy it to the `playbooks` directory inside your OpenShift Ansible directory.
This is typically found at `/usr/share/ansible/openshift-ansible/playbooks`.

```
curl {{site.url}}/{{page.version}}/getting-started/openshift/upgrade-calico.yaml -o /usr/share/ansible/openshift-ansible/playbooks/upgrade-calico.yaml
```

For users running OpenShift 3.10 and earlier, replace the above command with the following:
```
curl {{site.url}}/{{page.version}}/getting-started/openshift/upgrade-calico-3.10.yaml -o /usr/share/ansible/openshift-ansible/playbooks/upgrade-calico.yaml
```

Run the playbook.

```
ansible-playbook -i <YOUR-INVENTORY-FILE> /usr/share/ansible/openshift-ansible/playbooks/upgrade-calico.yaml
```
{% else %}
### Execute Ansible provisioning script

You are now ready to execute the Ansible provision which will install {{site.prodname}}. Note that by default,
{{site.prodname}} will connect to the same etcd that OpenShift uses and distribute etcd's
certs to each node. If you would prefer {{site.prodname}} not connect to the same etcd as OpenShift, you may modify the install
such that {{site.prodname}} connects to an etcd you have already set up by following the [dedicated etcd install guide](dedicated-etcd).
{% endif %}

{% include {{page.version}}/cnx-api-install.md init="openshift" %}

1. Continue to [Applying your license key](#applying-your-license-key).

{% include {{page.version}}/apply-license.md init="openshift" cli="oc" %}

{% if include.upgrade %}
## Installing metrics and logs
{% include {{page.version}}/byo-intro.md upgrade=include.upgrade orch="openshift" %}

### Set up access to your cluster from OpenShift

{% include {{page.version}}/elastic-secure.md orch="openshift" %}

{% include {{page.version}}/cnx-monitor-install.md orch="openshift" elasticsearch="external" upgrade=include.upgrade %}
{% else %}
{% include {{page.version}}/cnx-monitor-install.md orch="openshift" elasticsearch="operator" %}
{% endif %}

Once running, access Prometheus and Alertmanager using the NodePort from the created service.
See the [Metrics](/{{page.version}}/security/metrics/) section for more information.

{% if include.upgrade %}
{% include {{page.version}}/cnx-mgr-install.md init="openshift" elasticsearch="external" upgrade=include.upgrade %}
{% else %}
{% include {{page.version}}/cnx-mgr-install.md init="openshift" %}
{% endif %}

1. Download [oauth-client.yaml](/{{page.version}}/getting-started/openshift/installation/oauth-client.yaml).

   ```bash
   curl {{site.url}}/{{page.version}}/getting-started/openshift/installation/oauth-client.yaml -O
   ```

1. To make the following commands easier to copy and paste, set an environment variable called
   `CNX_MANAGER_ADDR` containing the address of your {{site.prodname}} Manager web interface.
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
{% unless include.upgrade %}
{% include {{page.version}}/gs-openshift-next-steps.md %}
{% endunless %}
