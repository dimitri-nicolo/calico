

1. [Download the private, CNX-specific `calicoctl` image](/{{page.version}}/getting-started/#images).

1. Import the file into the local Docker engine.

   ```
   docker load -i tigera_calicoctl_{{site.data.versions[page.version].first.components["calicoctl"].version}}.tar.xz
   ```
1. Confirm that the image has loaded by typing `docker images`.

   ```
   REPOSITORY            TAG               IMAGE ID       CREATED         SIZE
   tigera/calicoctl      {{site.data.versions[page.version].first.components["calicoctl"].version}}  e07d59b0eb8a   2 minutes ago   30.8MB
   ```

1. If you want to run `calicoctl` from the current host, skip to **Next step**! 
   
   Otherwise, you must upload the image to a private repository accessible to
   each node, ensure that each node has the credentials to access the repository,
   deploy the `calicoctl` container image to each node, and then **Next step**. 
   We provide the following Kubernetes manifests to make the _deployment_ part easier.
   
      - **etcd datastore**: [calicoctl.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/calicoctl.yaml){:target="_blank"}
      
      - **Kubernetes API datastore**: [calicoctl.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoctl.yaml){:target="_blank"}


