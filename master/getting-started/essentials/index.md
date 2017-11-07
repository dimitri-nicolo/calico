---
title: Obtaining Tigera CNX
---

{{site.prodname}} consists of the open source Project Calico, with
complementary monitoring and management tools.  Most of the software is open
source, but the additions are not.  This guide details how to obtain copies of
the relevant binaries.

### Essentials Specific Binaries

Your support representative will provide you with a link to a Google Drive folder
containing the binaries that are modified from or supplementary to open source
Project Calico.  These are:

1. the `calicoq` policy query tool,
2. the `calicoctl` configuration tool, and
3. `calico-node.tar.xz` - a `calico/node` image with additional monitoring capabilities.

If you are unable to use Google Drive, please contact your support representative
for an alternative distribution mechanism.

### Open Source Binaries

{{site.prodname}} uses standard open source Kubernetes and `calicoctl` binaries.  These
can be obtained via the usual channels, although the appropriate version of
`calicoctl ` is included in the Google Drive folder for convenience.

## Setting up a Docker Registry

Rather than directly loading the `calico/node` image onto every host directly,
we recommend you host the image in a Docker Registry which hosts can then pull
from.  The instructions and manifests provided elsewhere in the documentation 
will assume that the image will be pulled from a private Docker Registry.

If you already have a Docker Registry set up, then you can load the image into it.
Please do not upload the image to a publically accessible registry.

### Creating the Registry

Please refer to the Docker documentation on [setting up a registry](https://docs.docker.com/registry/deploying/#running-a-domain-registry)
to create your registry.

### Using the Registry

Once you have a suitable registry, load the `calico/node` image into it (substituting
the domain and port appropriately).
```
unxz calico-node-{{site.data.versions[page.version].first.title}}.tar.xz
docker load -i calico-node-{{site.data.versions[page.version].first.title}}.tar
docker tag calico/node:v2.5.0-e1.1.0 myregistrydomain.com:5000/calico/node:{{site.data.versions[page.version].first.title}}
docker push myregistrydomain.com:5000/calico/node:{{site.data.versions[page.version].first.title}}
```

Then, the images can be pulled easily from other hosts.
```
docker pull myregistrydomain.com:5000/calico/node:{{site.data.versions[page.version].first.title}}
```
