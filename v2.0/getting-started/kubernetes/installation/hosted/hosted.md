---
title: Standard Hosted Install
redirect_from: latest/getting-started/kubernetes/installation/hosted/hosted
---

The following steps install Calico as a Kubernetes add-on using your own etcd cluster.

{% include {{page.version}}/load-docker.md %}

## RBAC

If deploying Calico on an RBAC enabled cluster, you should first apply the `ClusterRole` and `ClusterRoleBinding` specs:

```
kubectl apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/rbac.yaml
```

## Install Calico

To install Calico:

1. Download [calico.yaml](calico.yaml)

1. Configure `etcd_endpoints` in the provided ConfigMap to match your etcd cluster.

1. Update the manifest with the path to your private docker registry.  Substitute
   `mydockerregistry:5000` with the location of your docker registry.
  
   ```
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/mydockerregistry:5000/g' calico.yaml
   ```
  
1. Then simply apply the manifest:

   ```shell
   kubectl apply -f calico.yaml
   ```

> **Note**: Make sure you configure the provided ConfigMap with the 
> location of your etcd cluster before running the above command.
{: .alert .alert-info}


## Configuration Options

The above manifest supports a number of configuration options documented [here](index#configuration-options)

## Adding Tigera CNX

Now you've installed Calico with the enhanced CNX node agent, you're ready to
[add CNX Manager](essentials/cnx).
