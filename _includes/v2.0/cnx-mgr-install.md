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

1. Open the file in a text editor, and update the ConfigMap `tigera-cnx-manager-config`
   according to the instructions in the file and your chosen authentication method.

   You might want to reconfigure the service that gets traffic to the {{site.tseeprodname}} Manager
   web server as well.

   > If {{site.nodecontainer}} was not deployed with a Kubernetes manifest and
   > no `calico-config` ConfigMap was created you must also update the file
   > anywhere `calico-config` is referenced with the appropriate information.
   {: .alert .alert-info}

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   {{site.tseeprodname}} Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the {{site.tseeprodname}} Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` service.

1. Store those credentials as `cert` and `key` in a secret named
   `cnx-manager-tls`.  For example:

   ```
   kubectl create secret generic cnx-manager-tls --from-file=cert=/path/to/certificate --from-file=key=/path/to/key -n kube-system
   ```

1. Apply the manifest to install {{site.tseeprodname}} Manager and the {{site.tseeprodname}} API server.

   ```
   kubectl apply -f cnx.yaml
   ```

1. Configure the kube-apiserver to allow
   cross-origin resource sharing (CORS). This will allow the {{site.tseeprodname}} Manager to communicate with {{site.tseeprodname}} API server. CORS can be enabled by setting the flag [--cors-allowed-origins](https://kubernetes.io/docs/reference/generated/kube-apiserver/) on kube-apiserver. kube-apiserver should be restarted for the --cors-allowed-origin flag to take effect.

   Example command:
   ```
   sudo sed -i \
   "/- kube-apiserver/a\    - --cors-allowed-origins=\"https://your-cnx-manager-url.example.com:cnx-manager-port\"" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

1. Configure authorization to allow {{site.tseeprodname}} Manager users to edit policies.  Consult the
   [{{site.tseeprodname}} Manager]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.

1. Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml) manifest.

   ```
   kubectl apply -f operator.yaml
   ```

1. Wait for the `alertmanagers.monitoring.coreos.com`, `prometheuses.monitoring.coreos.com` and `servicemonitors.monitoring.coreos.com` custom resource definitions to be created. Check by running:

   ```
   kubectl get customresourcedefinitions
   ```

1. Apply the [monitor-calico.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

   ```
   kubectl apply -f monitor-calico.yaml
   ```
