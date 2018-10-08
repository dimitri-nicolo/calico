
1. Set up secret with username and password for `Fluentd` to authenticate with Elasticsearch.
   ```
   kubectl create secret generic elastic-fluentd-user \
   --from-literal=username=<elastic-search-user> \
   --from-literal=password=<elastic-search-user-password> \
   -n calico-monitoring
   ```

1. Set up configmap with the certificate authority certificate to authenticate Elasticsearch.

   ```bash
   cp <ElasticSearchCA.pem> ca.pem
   kubectl create configmap -n calico-monitoring elastic-ca-config --from-file=ca.pem
   ```

1. Create a Secret containing
   * TLS certificate and the private key used to sign it enable TLS connection from the kube-apiserver to the es-proxy
   * certificate authority certificate to authenticate Elasticsearch backend
   * Base64 encoded <username>:<password> for the es-proxy to authenticate with Elasticsearch

   ```
   kubectl create secret generic tigera-es-proxy \
   --from-file=frontend.crt=frontend-server.crt \
   --from-file=frontend.key=frontend-server.key \
   --from-file=backend-ca.crt=ElasticSearchCA.pem \
   --from-literal=backend.authHeader=authHeader=$(echo -n <username>:<password> | base64) \
   -n calico-monitoring
   ```

1. Create a configmap with information on how to reach the Elasticsearch cluster
   ```
   kubectl create configmap tigera-es-proxy \
   --from-literal=elasticsearch.backend.host="elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local" \
   --from-literal=elasticsearch.backend.port="9200" \
   -n calico-monitoring
   ```

1. Download the configmap patch for {{site.prodname}} Manager.
    ```
    curl --compressed -O {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/secure-es/patch-cnx-manager-configmap.yaml
    ```
1. Apply the configmap patch.
   ```
   kubectl patch configmap tigera-cnx-manager-config -n kube-system -p "$(cat patch-cnx-manager-configmap.yaml)"
   ```
1. Restart {{site.prodname}} Manager pod
   ```
   kubectl delete po -n kube-system -l k8s-app=cnx-manager
   ```