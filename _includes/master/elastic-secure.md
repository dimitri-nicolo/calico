
1. Set up secret with username and password for `Fluentd` to authenticate with ElasticSearch.
   ```bash
     cat <<EOF | {{cli}} create secret -f -
apiVersion: v1
kind: Secret
metadata:
  name: elastic-fluentd-user
  namespace: calico-monitoring
type: Opaque
data:
  username: $(echo -n <"elastic-search-user"> | base64
  password: $(echo -n <"elastic-search-user-password"> | base64)
EOF
```

1. Set up `ConfigMap` with the certificate authority certificate to authenticate Elasticsearch.

   ```bash
   cp <ElasticSearchCA.pem> ca.pem
   kubectl create configmap -n calico-monitoring elastic-ca-config --from-file=ca.pem
   ```

1. Create a Secret containing
   * TLS certificate and the private key used to sign it enable TLS connection from the kube-apiserver to the es-proxy
   * certificate authority certificate to authenticate Elasticsearch backend
   * Base64 encoded <username>:<password> for the es-proxy to authenticate with Elasticsearch

   ```
   kubectl create configmap -n calico-monitoring elastic-ca-config --from-file=ca.pem
   kubectl create secret generic tigera-es-proxy \
   --from-file=frontend.crt=frontend-server.crt \
   --from-file=frontend.key=frontend-server.key \
   --from-file=backend-ca.crt=ElasticSearchCA.pem \
   --from-literal=backend.authHeader=authHeader=$(echo -n <username>:<password> | base64) \
   -n calico-monitoring
   ```

1. Create a ConfigMap with information on how to reach the Elasticsearch cluster
   ```
   kubectl create configmap tigera-es-proxy \
   --from-literal=elasticsearch.backend.host="elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local" \
   --from-literal=elasticsearch.backend.port="9200" \
   -n calico-monitoring
   ```
   