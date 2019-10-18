{% unless include.upgrade %}
Installation of {{site.prodname}} in OpenShift v3 is integrated in openshift-ansible.
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
  - `calico_node_image=<YOUR-REGISTRY>/{% include component_image component="cnx-node" %}`
  - `calico_url_policy_controller=<YOUR-REGISTRY>/{% include component_image component="cnx-kube-controllers" %}`
  - `calico_cni_image={% include component_image component="calico/cni" %}`
For OpenShift Container Platform 3.11 also specify the following variables:
  - `oreg_auth_user`
  - `oreg_auth_password`

> **Note**: See the OpenShift
> [documentation](https://docs.openshift.com/container-platform/3.11/install/configuring_inventory_file.html#advanced-install-configuring-registry-location)
> for more details.
{: .alert .alert-info}

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
# Authentication variables for OpenShift Container Platform 3.11
oreg_auth_user=<Username for accessing your RedHat image registry>
oreg_auth_password=<Password for accessing your RedHat image registry>

# {{site.prodname}} OpenShift installation configuration
ansible_become=true
deployment_type=openshift-enterprise
os_sdn_network_plugin_name=cni
openshift_use_openshift_sdn=false
openshift_use_calico=true
calico_node_image=<YOUR-REGISTRY>/{% include component_image component="cnx-node" %}
calico_url_policy_controller=<YOUR-REGISTRY>/{% include component_image component="cnx-kube-controllers" %}
calico_cni_image={% include component_image component="calico/cni" %}

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

### Enable access to the required images in your OpenShift cluster
{% include {{page.version}}/pull-secret.md orch="openshift" %}
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
