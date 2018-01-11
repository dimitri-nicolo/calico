---
title: Standard Hosted Install
---

The following steps install {{site.prodname}} as a Kubernetes add-on using your own etcd cluster.

## Requirements

- Kubernetes 1.8 or later

{% include {{page.version}}/cnx-k8s-apiserver-requirements.md %}

{% include {{page.version}}/load-docker.md %}

## RBAC

If deploying {{site.prodname}} on an RBAC-enabled cluster, you should first apply the `ClusterRole` and `ClusterRoleBinding` specs:

```
kubectl apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/rbac.yaml
```

## Installing {{site.prodname}}

To install {{site.prodname}}:

1. [Open calico.yaml in a new tab](calico.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as calico.yaml.

1. Open the calico.yaml file in your favorite editor, navigate to the `ConfigMap`
   section, and set `etcd_endpoints` to the correct value. Refer to [etcd configuration](index#etcd-configuration)
   for more details.

1. Use the following `sed` command to update the manifest to point to the private
   Docker registry. Before issuing this command, replace `<REPLACE_ME>` 
   with the name of your private Docker registry.

   **Command**
   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/<REPLACE_ME>/g' calico.yaml
   ```
   
   **Example**

   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/bob/g' calico.yaml
   ```
   > **Tip**: If you're hosting your own private repository, you may need to include
   > a port number. For example, `bob:5000`.
   {: .alert .alert-success}
   
   > **Note**: Refer to [Configuration options](index#configuration-options) for additional
   > settings that can be modified in the manifest.
   {: .alert .alert-info}

1. Then simply apply the manifest:

   ```shell
   kubectl apply -f calico.yaml
   ```

## Installing the CNX Manager

1. [Open cnx-etcd.yaml in a new tab](cnx/1.7/cnx-etcd.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.
   
{% include {{page.version}}/cnx-mgr-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
