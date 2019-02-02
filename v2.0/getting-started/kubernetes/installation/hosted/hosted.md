---
title: Standard Hosted Install
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/
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

{% include {{page.version}}/cnx-cred-sed.md %}

   > **Note**: Refer to [Configuration options](index#configuration-options) for additional
   > settings that can be modified in the manifest.
   {: .alert .alert-info}

1. Then simply apply the manifest:

   ```shell
   kubectl apply -f calico.yaml
   ```

   > **Note**: Make sure you configure the provided ConfigMap with the
   > location of your etcd cluster before running the above command.
   {: .alert .alert-info}

## Installing the CNX Manager

1. [Open cnx-etcd.yaml in a new tab](cnx/1.7/cnx-etcd.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.

{% include {{page.version}}/cnx-mgr-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
