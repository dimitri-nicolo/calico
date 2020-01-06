---
title: Installing CNX for OpenShift
---

1. Load the `cnx-apiserver`, `cnx-node`, and `cnx-manager` binaries into your
   private Docker registry.

   See [Obtaining {{site.tseeprodname}}][obtaining-cnx] for more information
   on how to acquire these images.

1. Ensure your Docker daemon on all OpenShift nodes and masters is authenticated to pull images from that registry.

   > **Note**: See the [OCP Advanced Installation Instructions][ocp-advanced-install]
   for more information on setting up custom Docker registries using the OpenShift installer.
   {: .alert .alert-info}

1. Set `calico_node_image` to {{site.nodecontainer}} in your OpenShift inventory file.

   Example Inventory File:

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

1. Create a Kubernetes secret from your etcd certificates. Example command:

   ```
   kubectl create -n kube-system secret generic calico-etcd-secrets \
   --from-file=etcd-ca=/etc/origin/calico/calico.etcd-ca.crt \
   --from-file=etcd-cert=/etc/origin/calico/calico.etcd-client.crt \
   --from-file=etcd-key=/etc/origin/calico/calico.etcd-client.key
   ```

   >{{site.tseeprodname}} APIServer and Manager require etcd connection information and
   >certificates to be stored in kubernetes objects.
   >The following preparation steps will upload this data.
   >
   >If you are unsure of what to set these values to, check >`/etc/calico/calicoctl.cfg`
   >on your master node, which will show what `calicoctl` is currently using to connect to etcd.
   {: .alert .alert-info}

1. Download the {{site.tseeprodname}} configmap: [calico-config.yaml](calico-config.yaml)

1. Update the CNX configmap with the path to your private Docker registry.
   Substitute `$ETCD_ENDPOINTS` with the address of your etcd cluster.

       sed -i -e "s?<ETCD_ENDPOINTS>?$ETCD_ENDPOINTS?g" calico-config.yaml

1. Apply it:

       kubectl apply -f ./calico-config.yaml

1. Download the {{site.tseeprodname}} manifest:

   - [cnx-etcd.yaml](/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml)

1. Rename the file `cnx.yaml` - this is what subsequent instructions will refer to:

       mv cnx-etcd.yaml cnx.yaml

1. Update the login method to "Token":

       sed -i -e 's/tigera.cnx-manager.authentication-type:.*$/tigera.cnx-manager.authentication-type: "Token"/g' cnx.yaml

1. Update the manifest with the path to your private Docker registry. Substitute
   `mydockerregistry:5000` with the location of your Docker registry.

       sed -i -e 's?<YOUR_PRIVATE_DOCKER_REGISTRY>?mydockerregistry:5000?g' cnx.yaml

1. Open the file in a text editor, and update the ConfigMap `tigera-cnx-manager-config`
   according to the instructions in the file and your chosen authentication method.

   You might want to reconfigure the service that gets traffic to the CNX Manager
   web server as well.

   {{site.tseeprodname}} for OpenShift requires SSL certs to connect to etcd, so be sure to uncomment
   all secrets connected to `calico-secrets`.

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   CNX Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the CNX Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` service.

1. Store those credentials as `cert` and `key` in a secret named
   `cnx-manager-tls`.  For example:

       kubectl create secret generic cnx-manager-tls -n kube-system --from-file=cert=/path/to/certificate --from-file=key=/path/to/key

1. Apply the manifest to install CNX Manager and the CNX API server.

   ```
   kubectl apply -f cnx.yaml
   ```

1. Allow cnx-manager to run as root:

       oadm policy add-scc-to-user anyuid system:serviceaccount:kube-system:cnx-manager

1. Configure authentication to allow CNX Manager users to edit policies.  Consult the
   [CNX Manager](../../../reference/cnx/policy-editor) and
   [Tiered policy RBAC](../../../reference/cnx/rbac-tiered-policies)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.

## Signing into the CNX Manager UI in OpenShift

The CNX Manager requires an [OpenShift access token](https://docs.openshift.com/container-platform/3.7/install_config/configuring_authentication.html#token-options) for login. You will need
to consult the OpenShift docs on how to generate an access token using your configured
identity provider.

If you are unsure, a simple way to generate this token is to log into the OpenShift UI, then visit `<master>/oauth/token/request` for your token.

### Next Steps

- For information on customizing the CNX install manifest, see [Customizing the CNX Manager Manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/cnx)

- [Using {{site.tseeprodname}} for OpenShift](usage)

### Next Steps

See [Using {{site.tseeprodname}} for OpenShift](usage).

[obtaining-cnx]: {{site.baseurl}}/{{page.version}}/getting-started/
[ocp-advanced-install]: https://access.redhat.com/documentation/en-us/openshift_container_platform/3.6/html-single/installation_and_configuration/#system-requirements
