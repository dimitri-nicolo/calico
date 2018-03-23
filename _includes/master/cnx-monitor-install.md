1. If you are not pulling the private CNX images from a quay.io registry, skip to the next step.

   Download and edit [operator.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml).
   Look for the lines that specify arguments for `calico-prometheus-operator`.
   Uncomment those three lines and replace `<YOUR-REGISTRY>` with your
   registry's URL. For example:

   ```yaml
   containers:
   - name: calico-prometheus-operator
     # When using a private registry to serve container images, delete the "image:" line
     # above, uncomment the following lines, and replace "<YOUR-REGISTRY>" with your
     # registry's URL.
     image: <YOUR-REGISTRY>/prometheus-operator:v0.15.0
     args:
       - --prometheus-config-reloader=<YOUR-REGISTRY>/prometheus-config-reloader:v0.0.2
       - --config-reloader-image=<YOUR-REGISTRY>/configmap-reload:v0.0.1
   ```

   Follow the instructions in the next step to apply your edited `operator.yaml` manifest..

1. Configure calico-monitoring namespace and deploy Prometheus Operator by
  applying the [operator.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml) manifest.

   ```
   kubectl apply -f operator.yaml
   ```

1. Wait for the `alertmanagers.monitoring.coreos.com`, `prometheuses.monitoring.coreos.com` and `servicemonitors.monitoring.coreos.com` custom resource definitions to be created. Check by running:

   ```
   kubectl get customresourcedefinitions
   ```

1. If you are not pulling the private CNX images from a quay.io registry, skip to the next step.

   Download and edit [monitor-calico.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml).
   Look for the the `baseImage:` lines, uncomment the lines, and replace
   `<YOUR-REGISTRY>` with the URL of your private registry. For example:

   ```yaml
   baseImage: <YOUR-REGISTRY>/prometheus
   ```

   and:

   ```yaml
   baseImage: <YOUR-REGISTRY>/alertmanager
   ```

   Follow the instructions in the next step to apply your edited `monitor-calico.yaml` manifest..

1. Apply the [monitor-calico.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml) manifest which will
  install Prometheus and alertmanager.

   ```
   kubectl apply -f monitor-calico.yaml
   ```
