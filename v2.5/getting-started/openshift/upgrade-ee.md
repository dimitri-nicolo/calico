---
title: Upgrading an OpenShift cluster with Calico to Tigera Secure EE
canonical_url: https://docs.tigera.io/v2.3/getting-started/openshift/upgrade-ee
---

## Prerequisites

- Ensure that the open source Calico cluster is running the latest version of Calico {{site.data.versions[page.version].first.components["calico"].minor_version | append: '.x' }}

  If not, follow the [Calico OpenShift installation documentation](https://docs.projectcalico.org/{{site.data.versions[page.version].first.components["calico"].minor_version}}/getting-started/openshift/installation)
  before continuing.

- Ensure that you have satisfied all of the {{site.prodname}} prerequisites listed
  by the [OpenShift installation documentation]({{site.url}}/{{page.version}}/getting-started/openshift/installation#before-you-begin).

## Upgrading Calico to {{site.prodname}}

### Upgrading by rerunning OpenShift Ansible

The simplest way to upgrade from Calico to {{site.prodname}} is by following the
[OpenShift installation documentation]({{site.url}}/{{page.version}}/getting-started/openshift/installation)
to reuse the Ansible playbooks. This will overwrite your Calico installation with
{{site.prodname}}. It will also reinstance your OpenShift installation, which will
make your cluster unavailable during the installation.

### Upgrading through a custom playbook

If you do not wish to rerun the OpenShift installation, you can also use
a custom playbook in order to rerun the {{site.prodname}}-specific sections of 
the OpenShift install. First, make sure that you have properly edited your
[inventory file]({{site.url}}/{{page.version}}/getting-started/openshift/installation#edit-inventory-file).

Once your inventory file has been properly configured, download the
[upgrade playbook](upgrade-calico.yaml){:target="_blank"}
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
