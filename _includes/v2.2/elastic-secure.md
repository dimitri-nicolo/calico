
1. Set up secret with username and password for `Fluentd` to authenticate with Elasticsearch.
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

1. If your cluster is connected to the internet, use the following command to install Prometheus,
   Alertmanager, and Elasticsearch.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/secure-es/monitor-calico.yaml
   ```

   > **Note**: You can also
   > [view the manifest in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/secure-es/monitor-calico.yaml){:target="_blank"}.
   {: .alert .alert-info}

   > For offline installs, complete the following steps instead.
   >
   > 1. Download the Prometheus and Alertmanager manifest.
   >
   >    ```
   >    curl --compressed -o \
   >    {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/secure-es/monitor-calico.yaml
   >    ```
   >
   > 1. Use the following commands to set an environment variable called `REGISTRY` containing the
   >    location of the private registry and replace `quay.io` in the manifest with the location
   >    of your private registry.
   >
   >    ```bash
   >    REGISTRY=my-registry.com \
   >    sed -i -e "s?quay.io?$REGISTRY?g" monitor-calico.yaml
   >    ```
   >
   >    **Tip**: If you're hosting your own private registry, you may need to include
   >    a port number. For example, `my-registry.com:5000`.
   >    {: .alert .alert-success}
   >
   > 1. Apply the manifest.
   >
   >    ```
   >    kubectl apply -f monitor-calico.yaml
   >    ```
