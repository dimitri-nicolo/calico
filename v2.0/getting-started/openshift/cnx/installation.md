---
title: Installing CNX for OpenShift
---

The installation process for {{site.prodname}} for OpenShift is identical to the
standard [Calico-OpenShift installation process](../installation), but with a custom `calico/node` image.

This guide describes how to modify the base install to launch the `calico/node` CNX image in OpenShift.

### Installation

1. Push your `calico/node` CNX image to a private Docker registry.
2. Ensure your Docker daemon on all OpenShift nodes and masters is authenticated to pull images from that registry.
3. Set `calico_node_image` to the {{site.prodname}} `calico/node` image.

See [Obtaining {{site.prodname}}][obtaining-cnx] for more information
on how to acquire the `calico/node` {{site.prodname}} image.

See the [OCP Advanced Installation Instructions][ocp-advanced-install] for more
information on setting up custom Docker registries using the OpenShift installer.

#### Example Inventory File

```
[OSEv3:children]
masters
nodes
etcd

[OSEv3:vars]
os_sdn_network_plugin_name=cni
openshift_use_calico=true
openshift_use_openshift_sdn=false
calico_node_image=calico/node:{{site.data.versions[page.version].first.title}}
calico_ipv4pool_cidr=10.128.0.0/14
calico_etcd_endpoints=http://calico-etcd:2379
calico_etcd_ca_cert_file=/etc/calico/etcd-ca.crt
calico_etcd_cert_file=/etc/calico/etcd-client.crt
calico_etcd_key_file=/etc/calico/etcd-client.key
calico_etcd_cert_dir=/etc/calico/

[masters]
master1

[nodes]
node1

[etcd]
etcd1
```

### Next Steps

See [Using {{site.prodname}} for OpenShift](usage).

[obtaining-cnx]: {{site.baseurl}}/{{page.version}}/getting-started/cnx/
[ocp-advanced-install]: https://access.redhat.com/documentation/en-us/openshift_container_platform/3.6/html-single/installation_and_configuration/#system-requirements
