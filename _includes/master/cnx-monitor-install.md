1. If your cluster is connected to the Internet, use the following command to apply the Prometheus 
   operator manifest.
   
   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml
   ```
   
   > **Note**: You can also 
   > [view the manifest in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml){:target="_blank"}.
   {: .alert .alert-info}
   
   > For offline installs, complete the following steps instead.
   >
   > 1. Download the Prometheus operator manifest.
   >
   >    ```
   >    curl --compressed -o\
   >    {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml
   >    ``` 
   > 
   > 1. Replace `quay.io` in the `image` paths with your own private registry.
   > 
   >    **Command**
   >    ```bash
   >    sed -i -e "s?quay.io?<YOUR-REGISTRY>?g" operator.yaml
   >    ```
   >
   >    **Example**
   >    ```bash
   >    sed -i -e "s?quay.io?my-registry.com?g" operator.yaml
   >    ```
   >    
   > 1. Apply the manifest.
   >    
   >    ```
   >    kubectl apply -f operator.yaml
   >    ```

1. Wait for the `alertmanagers.monitoring.coreos.com`, `prometheuses.monitoring.coreos.com` and `servicemonitors.monitoring.coreos.com` custom resource definitions to be created. Check by running:

   ```
   kubectl get customresourcedefinitions
   ```
   
1. If your cluster is connected to the Internet, use the following command to install Prometheus
   and Alertmanager.
   
   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml
   ```
   
   > **Note**: You can also 
   > [view the manifest in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml){:target="_blank"}.
   {: .alert .alert-info}
   
   > For offline installs, complete the following steps instead.
   >
   > 1. Download the Prometheus and Alertmanager manifest.
   >
   >    ```
   >    curl --compressed -o\
   >    {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml
   >    ``` 
   >      
   > 1. Replace `quay.io` in the `image` paths with your own private registry.
   > 
   >    **Command**
   >    ```bash
   >    sed -i -e "s?quay.io?<YOUR-REGISTRY>?g" monitor-calico.yaml
   >    ```
   >
   >    **Example**
   >    ```bash
   >    sed -i -e "s?quay.io?my-registry.com?g" monitor-calico.yaml
   >    ```
   >       
   > 1. Apply the manifest.
   > 
   >    ```
   >    kubectl apply -f monitor-calico.yaml
   >    ```
      
