---
title: Obtaining Tigera CNX
---

{{site.prodname}} consists of the open source Project Calico, with
complementary monitoring and management tools.  Most of the software is open
source, but the additions are not.  This guide details how to obtain copies of
the relevant binaries.

### {{site.prodname}} Specific Binaries

Your support representative will provide you with a link to a Google Drive folder
containing the binaries that are modified from or supplementary to open source
Project Calico.  These are:

1. the `calicoq` policy query tool,
2. the `calicoctl` configuration tool, and
3. `calico-node.tar.xz` - a `calico/node` image with additional monitoring capabilities.
4. `tigera-cnx-manager-web.tar.xz` - the web server for {{site.prodname}} Manager.
5. `calico-k8sapiserver.tar.xz` - the API server component of {{site.prodname}} Manager.

If you are unable to use Google Drive, please contact your support representative
for an alternative distribution mechanism.

### Open Source Binaries

{{site.prodname}} also uses standard open source Kubernetes and Calico binaries.  These
can be obtained via the usual channels, as described in these or the open source
documentation.

## Setting up a Docker Registry

Rather than directly loading the Docker images onto every host directly,
we recommend you host the images in a Docker registry which hosts can then pull
from.  The instructions and manifests provided elsewhere in the documentation 
assume that the images will be pulled from a private Docker registry.

If you already have a Docker registry set up, then you can load the images into it.
Please do not upload the images to a publicly accessible registry.

### Creating the registry

Please refer to the Docker documentation on [setting up a registry](https://docs.docker.com/registry/deploying/#running-a-domain-registry)
to create your registry.

### Using the registry

Once you have a suitable registry, load the images into it (substituting
the domain and port appropriately).
```
REGISTRY_DOMAIN=<URL for your Docker registry>
REGISTRY_PORT=<port for your Docker registry>

load_image()
{
    IMAGE_FILE=$1
    IMAGE_NAME=$2
    unxz $IMAGE_FILE-{{site.data.versions[page.version].first.title}}.tar.xz
    docker load -i $IMAGE_FILE-{{site.data.versions[page.version].first.title}}.tar
    docker tag $IMAGE_NAME:{{site.data.versions[page.version].first.title}} $REGISTRY_DOMAIN:$REGISTRY_PORT/$IMAGE_NAME:{{site.data.versions[page.version].first.title}}
    docker push $REGISTRY_DOMAIN:$REGISTRY_PORT/$IMAGE_NAME:{{site.data.versions[page.version].first.title}}
}

load_image calico-node calico/node
load_image calico-k8sapiserver calico/k8sapiserver
load_image tigera-cnx-manager-web tigera/cnx-manager-web
```

Then, the images can be pulled easily from other hosts.
```
docker pull $REGISTRY_DOMAIN:$REGISTRY_PORT/$IMAGE_NAME:{{site.data.versions[page.version].first.title}}
```
