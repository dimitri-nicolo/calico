## Private image and repository set up


### Prerequisite

You must have a private registry that each node can access. 

> **Important**: Do not push the private {{site.prodname}} images to a public repository.
{: .alert .alert-danger}

If you do not already have a private registry, consider one of the following options.
- [Docker Hub](https://hub.docker.com/)
- [Google Container Registry](https://cloud.google.com/container-registry/)
- [Amazon Web Services (AWS) Elastic Compute Cloud (EC2) Container Registry](https://aws.amazon.com/ecr/pricing/)
- [Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/)
- [Quay](https://quay.io/repository/)


### Loading and pushing the private images


1. Import the files into the local Docker engine. 

   ```
   docker load -i tigera_cnx-apiserver_v2.0.0-cnx-beta1.tar.xz
   docker load -i tigera_cnx-node_v2.0.0-cnx-beta1.tar.xz
   docker load -i tigera_cnx-manager_v2.0.0-cnx-beta1.xz
   ```

1. Confirm that the images have loaded by typing `docker images`. 

   ```
   REPOSITORY            TAG               IMAGE ID       CREATED         SIZE
   tigera/cnx-manager    v2.0.0-cnx-beta1  e07d59b0eb8a   2 minutes ago   30.8MB
   tigera/cnx-node       v2.0.0-cnx-beta1  2bf19d491aac   3 minutes ago   263MB
   tigera/cnx-apiserver  v2.0.0-cnx-beta1  acd3faa772d0   5 minutes ago   277MB
   ```
   
1. Retag the images as desired and necessary to load them to your private repository.

1. If you have not configured your local Docker instance with the credentials that will 
   allow you to access your private registry, do so now. 

   ```
   docker login [registry-domain]
   ```
   
1. Push the `cnx-manager`, `cnx-node`, and `cnx-apiserver` images to the 
   private repository.

   ```
   docker push {{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker push {{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker push {{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   ```

1. Next, you must determine how to configure Kubernetes to pull from your private repository. The method varies according to your private repository vendor and Kubernetes hosting. For specific instructions, refer to:
   - The documentation of your private repository vendor
   - Kubernetes [Using a private repository](https://kubernetes.io/docs/concepts/containers/images/#using-a-private-registry)
   - Heptio [How to: Pull from private registries with Kubernetes](http://docs.heptio.com/content/private-registries.html)