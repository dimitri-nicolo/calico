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
