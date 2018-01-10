1. Use the following `sed` command to update the manifest to point to the private
   Docker registry. Before issuing this command, replace `<REPLACE_ME>` 
   with the location of your private Docker registry.

   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/<REPLACE_ME>/g' cnx.yaml
   ```

1. Open the file in a text editor, and update the ConfigMap `tigera-cnx-manager-config`
   according to the instructions in the file and your chosen authentication method.

   You might want to reconfigure the service that gets traffic to the {{site.prodname}} Manager
   web server as well.

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

1. Configure the kube-apiserver to allow
   cross-origin resource sharing (CORS). This will allow the {{site.prodname}} Manager to communicate with {{site.prodname}} API server. CORS can be enabled by setting the flag [--cors-allowed-origins](https://kubernetes.io/docs/reference/generated/kube-apiserver/) on kube-apiserver. kube-apiserver should be restarted for the --cors-allowed-origin flag to take effect.

   Example command:
   ```
   sudo sed -i \
   "/- kube-apiserver/a\    - --cors-allowed-origins=\"https://your-cnx-manager-url.example.com:cnx-manager-port\"" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

1. Configure authorization to allow {{site.prodname}} Manager users to edit policies.  Consult the
   [{{site.prodname}} Manager](../../../../../reference/cnx/non-admin-workflows)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.

1. Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml](1.7/operator.yaml) manifest.

   ```
   kubectl apply -f operator.yaml
   ```

1. Wait for custom resource definitions to be created. Check by running:

   ```
   kubectl get customresourcedefinitions
   ```

1. Apply the [monitor-calico.yaml](1.7/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

   ```
   kubectl apply -f monitor-calico.yaml
   ```