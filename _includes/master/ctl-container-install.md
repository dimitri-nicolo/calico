If you are on Kubernetes, we provide two manifests that make it easy to deploy `calicoctl`
as a pod.

- **etcd datastore**:

   [calicoctl.yaml]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/calicoctl.yaml){:target="_blank"}

- **Kubernetes API datastore**:

   [calicoctl.yaml]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoctl.yaml){:target="_blank"}

- **Start the pod**:

   ```
   kubectl apply -f calicoctl.yaml
   ```


In other environments, use the following command.

```
docker pull {{site.imageNames["calicoctl"]}}:{{site.data.versions[page.version].first.title}}
```
