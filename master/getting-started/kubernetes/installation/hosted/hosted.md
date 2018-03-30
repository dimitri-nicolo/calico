---
title: Standard Hosted Install
canonical_url: 'https://docs.projectcalico.org/v3.0/getting-started/kubernetes/installation/hosted/hosted'
---

The following steps install {{site.prodname}} as a Kubernetes add-on using 
your own etcd cluster.

## Before you begin

- Ensure that your cluster meets the {{site.prodname}} [system requirements](../../requirements). 

- If deploying {{site.prodname}} on an RBAC-enabled cluster, you should 
  first apply the `ClusterRole` and `ClusterRoleBinding` specs:

   ```
   kubectl apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/rbac.yaml
   ```

{% include {{page.version}}/load-docker-intro.md %}

{% include {{page.version}}/load-docker-our-reg.md yaml="calico" %}

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

## Install {{site.prodname}}

To install {{site.prodname}}:

1. Download [calico.yaml](calico.yaml)

1. Configure `etcd_endpoints` in the provided ConfigMap to match your etcd cluster.

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest:

   ```shell
   kubectl apply -f calico.yaml
   ```

> **Note**: Make sure you configure the provided ConfigMap with the 
> location of your etcd cluster before running the above command.
{: .alert .alert-info}


## Configuration Options

The above manifest supports a number of configuration options documented [here](index#configuration-options)

## Installing the CNX Manager

1. [Open cnx-etcd.yaml in a new tab](cnx/1.7/cnx-etcd.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.
   
{% include {{page.version}}/cnx-mgr-install.md %}

{% include {{page.version}}/cnx-monitor-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
