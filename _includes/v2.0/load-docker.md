## Private image and registry set up

CNX is delivered as a set of `.tar.xz` compressed Docker image files.  To use
those, you need to load each image into your local Docker engine, then push
from there to a private Docker registry, then deploy into Kubernetes by
applying manifests that will pull from that registry.

### Prerequisite

You must have a private registry that each node can access.

> **Important**: Do not push the private {{site.prodname}} images to a public registry.
{: .alert .alert-danger}

If you do not already have a private registry, consider one of the following options.
- [Docker Hub](https://hub.docker.com/)
- [Google Container Registry](https://cloud.google.com/container-registry/)
- [Amazon Web Services (AWS) Elastic Compute Cloud (EC2) Container Registry](https://aws.amazon.com/ecr/pricing/)
- [Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/)
- [Quay](https://quay.io/repository/)
- [OpenShift Container Platform](https://access.redhat.com/documentation/en-us/openshift_container_platform/3.6/html-single/installation_and_configuration/#system-requirements)


### Loading and pushing the private images


1. Import the files into the local Docker engine.

   ```
   docker load -i tigera_cnx-apiserver_{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}.tar.xz
   docker load -i tigera_cnx-node_{{site.data.versions[page.version].first.components["cnx-node"].version}}.tar.xz
   docker load -i tigera_cnx-manager_{{site.data.versions[page.version].first.components["cnx-manager"].version}}.tar.xz
   ```

1. Confirm that the images have loaded by typing `docker images`.

   ```
   REPOSITORY            TAG               IMAGE ID       CREATED         SIZE
   tigera/cnx-manager    {{site.data.versions[page.version].first.components["cnx-manager"].version}}  e07d59b0eb8a   2 minutes ago   30.8MB
   tigera/cnx-node       {{site.data.versions[page.version].first.components["cnx-node"].version}}  2bf19d491aac   3 minutes ago   263MB
   tigera/cnx-apiserver  {{site.data.versions[page.version].first.components["cnx-apiserver"].version}}  acd3faa772d0   5 minutes ago   277MB
   ```

1. Retag the images as desired and necessary to load them to your private registry.

1. If you have not configured your local Docker instance with the credentials that will
   allow you to access your private registry, do so now.

   ```
   docker login [registry-domain]
   ```

1. Use the following commands to push the `cnx-manager`, `cnx-node`, and `cnx-apiserver`
   images to the private registry, replacing `<YOUR_PRIVATE_DOCKER_REGISTRY>` with the
   location of your registry first.

   ```
   docker push {{page.registry}}{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker push {{page.registry}}{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker push {{page.registry}}{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   ```

1. Next, you must determine how to configure Kubernetes to pull from your private registry. The method varies according to your private registry vendor and Kubernetes hosting. For specific instructions, refer to:
   - The documentation of your private registry vendor
   - Kubernetes [Using a Private Registry](https://kubernetes.io/docs/concepts/containers/images/#using-a-private-registry)
   - Heptio [How to: Pull from private registries with Kubernetes](http://docs.heptio.com/content/private-registries.html)
