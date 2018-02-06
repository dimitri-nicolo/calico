1. Use the following `sed` command to quickly replace `<YOUR_PRIVATE_DOCKER_REGISTRY>`
   in the manifest with the name of your private registry. Since the manifest
   already contains the names of the images and their version numbers, you
   just need to replace `<REPLACE_ME>` with the name of the private
   repository before issuing the command.

   **Command**
   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/<REPLACE_ME>/g' cnx.yaml
   ```

   **Example**

   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/my-repo/g' cnx.yaml
   ```

   > **Tip**: If you're hosting your own private repository, you may need to include
   > a port number. For example, `my-repo:5000`.
   {: .alert .alert-success}

{% if include.orchestrator == "openshift" %}

1. Update the login method to `Token`:

       sed -i -e 's/tigera.cnx-manager.authentication-type:.*$/tigera.cnx-manager.authentication-type: "Token"/g' cnx.yaml

1. Open the file in a text editor, and make the following modifications:

   In the `Deployment` named `cnx-apiserver`:

   - Uncomment the `ETCD_CA_CERT_FILE`, `ETCD_KEY_FILE`, and `ETCD_CERT_FILE` environment variables.
   - Uncomment the `volumeMount` named `etcd-certs`.
   - Uncomment the `volume` named `etcd-certs`.

   You might want to reconfigure the service that gets traffic to the {{site.prodname}} Manager
   web server as well.

{% else %}

1. Open the file in a text editor, and update the ConfigMap `tigera-cnx-manager-config`
   according to the instructions in the file and your chosen authentication method.

   You might want to reconfigure the service that gets traffic to the {{site.prodname}} Manager
   web server as well.

   > If {{site.nodecontainer}} was not deployed with a Kubernetes manifest and
   > no `calico-config` ConfigMap was created you must also update the file
   > anywhere `calico-config` is referenced with the appropriate information.
   {: .alert .alert-info}

{% endif %}

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   {{site.prodname}} Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the {{site.prodname}} Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` service.

1. Store those credentials as `cert` and `key` in a secret named
   `cnx-manager-tls`.  For example:

   ```
   kubectl create secret generic cnx-manager-tls --from-file=cert=/path/to/certificate --from-file=key=/path/to/key -n kube-system
   ```

1. Apply the manifest to install {{site.prodname}} Manager and the {{site.prodname}} API server.

   ```
   kubectl apply -f cnx.yaml
   ```

{% if include.orchestrator == "openshift" %}

1. Allow cnx-manager to run as root:

       oadm policy add-scc-to-user anyuid system:serviceaccount:kube-system:cnx-manager

{% endif %}

1. Configure authorization to allow {{site.prodname}} Manager users to edit policies.  Consult the
   [{{site.prodname}} Manager]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.
