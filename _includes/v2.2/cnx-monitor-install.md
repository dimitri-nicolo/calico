{% if include.orch != "openshift" %}
  {% capture docpath %}{{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7{% endcapture %}
  {% assign cli = "kubectl" %}
{% else %}
  {% capture docpath %}{{site.url}}/{{page.version}}/getting-started/openshift{% endcapture %}
  {% assign cli = "oc" %}
{% endif %}
{% if include.elasticsearch == "operator" %}
  {% assign operators = "Prometheus and Elasticsearch" %}
  {% assign secure = "" %}
{% else %}
  {% assign operators = "Prometheus" %}
  {% assign secure = "/secure-es" %}
{% endif %}

{% if include.orch == "openshift" %}

1. Download the flow logs patch for {{site.prodname}} node.

   ```
   curl --compressed -O {{docpath}}/patch-flow-logs.yaml
   ```

1. Apply the flow logs patch.

   ```
   oc patch daemonset {{site.noderunning}} -n kube-system --patch "$(cat patch-flow-logs.yaml)"
   ```

1. Allow Prometheus to run as root.

   ```
   oc adm policy add-scc-to-user --namespace=calico-monitoring anyuid -z default
   ```

1. Allow Prometheus to configure and use a security context.

   ```
   oc adm policy add-scc-to-user anyuid system:serviceaccount:calico-monitoring:prometheus
   ```

{% if include.elasticsearch == "external" %}

1. Allow Elasticsearch proxy to configure and use a security context.

   ```
   oc adm policy add-scc-to-user anyuid system:serviceaccount:calico-monitoring:tigera-es-proxy
   ```

{% endif %}

1. Allow sleep pod to run with host networking.

   ```
   oc adm policy add-scc-to-user --namespace=calico-monitoring hostnetwork -z default
   ```

1. Allow Prometheus to have pods in `kube-system` namespace on each node.

   ```
   oc annotate ns kube-system openshift.io/node-selector="" --overwrite
   ```

{% endif %}


1. Download the `operator.yaml` manifest.

   ```bash
   curl --compressed -O \
   {{docpath}}{{secure}}/operator.yaml
   ```

1. Use the following commands to set an environment variable called `REGISTRY` containing the
   location of the private registry and replace the paths in the manifest to refer to
   the private registry.

    ```bash
    REGISTRY=my-registry.com \
    sed -i -e "s?quay.io?$REGISTRY?g" operator.yaml {% if include.elasticsearch == "operator" %}\
    sed -i -e "s?upmcenterprises?$REGISTRY/upmcenterprises?g" operator.yaml{% endif %}
    ```

    > **Tip**: If you're hosting your own private registry, you may need to include
    > a port number. For example, `my-registry.com:5000`.
    {: .alert .alert-success}

1. Apply the manifest.

   ```bash
   {{cli}} apply -f operator.yaml
   ```

{% if include.elasticsearch == "operator" %}
1. Wait for the `alertmanagers.monitoring.coreos.com`, `prometheuses.monitoring.coreos.com`, `servicemonitors.monitoring.coreos.com`,
   and `elasticsearchclusters.enterprises.upmc.com` custom resource definitions to be created. Check by running:
{% else %}
1. Wait for the `alertmanagers.monitoring.coreos.com`, `prometheuses.monitoring.coreos.com`, and `servicemonitors.monitoring.coreos.com`
   custom resource definitions to be created. Check by running:
{% endif %}

   ```
   {{cli}} get customresourcedefinitions
   ```

{% if include.orch == "openshift" %}
1. Allow the monitoring pods to be scheduled on the master node.

   ```
   {{cli}} annotate ns calico-monitoring openshift.io/node-selector="" --overwrite
   ```

{% endif %}

{% if include.elasticsearch == "operator" %}
{% include {{page.version}}/elastic-storage.md orch=include.orch %}
{% endif %}

1.  Download the `monitor-calico.yaml` manifest.

    ```bash
    curl --compressed -O \
    {{docpath}}{{secure}}/monitor-calico.yaml
    ```

1. Edit the `GlobalNetworkSet` named `k8sapi-endpoints` to specify the IP addresses of the hosts that are running the Kubernetes API server.

   ```yaml
   apiVersion: projectcalico.org/v3
   kind: GlobalNetworkSet
   metadata:
     name: k8sapi-endpoints
     labels:
       role: k8s-apiserver-endpoints
   spec:
     nets:
     - <Kubernetes API server IP address CIDR>
   ```

   > **Note**: You may need to list all the IP addresses on that host including the IP address of `tunl0`
   > if running in IPIP mode.
   {: .alert .alert-info}

{% include {{page.version}}/cnx-cred-sed.md yaml="monitor-calico" %}

1. Apply the manifest.

   ```bash
   kubectl apply -f monitor-calico.yaml
   ```

1. Edit the `tigera-cnx-manager-config` ConfigMap to update the URL Kibana is accessed at.  By default a NodePort is
   installed that serves Kibana on port 30601, so use the address of a node (for example a master).

   Either edit the `tigera.cnx-manager.kibana-url` field in the `cnx.yaml` manifest and reapply, or use the following patch:

   ```bash
   kubectl patch configmap -n kube-system tigera-cnx-manager-config -p $'data:\n  tigera.cnx-manager.kibana-url: http://<insert-node-address-here>:30601'
   ```

{% if include.orch == "openshift" %}
{% if include.elasticsearch == "operator" %}

1. Reconfigure the Elasticsearch deployment. The following command will save the current configuration
   to `tigera-elasticsearch.yaml`.

   ```
   oc get deployment es-client-tigera-elasticsearch -n calico-monitoring -o yaml --export > tigera-elasticsearch.yaml
   ```

   Run the following command which will fix the configuration for pods to start properly in OpenShift.

   ```
   sed -i '/capabilities/,+2 d' tigera-elasticsearch.yaml
   ```

   Replace the running deployment.
   ```
   oc replace -n calico-monitoring -f tigera-elasticsearch.yaml
   ```

1. Remove the ReplicaSet from the deployment we replaced. You can find this ReplicaSet with the following command.

   ```
   oc get rs -n calico-monitoring
   ```

   The ReplicaSet we will want to replace will have 0 `DESIRED`, `CURRENT`, and `READY` pods. In the following example
   output, `es-client-tigera-elasticsearch-5ddd8dfdfd` is the ReplicaSet we will want to remove.

   ```
   NAME                                        DESIRED   CURRENT   READY     AGE
   calico-prometheus-operator-74dd985b6f       1         1         1         3h
   elasticsearch-operator-5c84946f57           1         1         1         3h
   es-client-tigera-elasticsearch-5ddd8dfdfd   0         0         0         3h
   es-client-tigera-elasticsearch-759997fcbb   1         1         1         19m
   kibana-tigera-elasticsearch-6cb8879697      1         1         1         3h
   ```

   Remove the chosen ReplicaSet with the following command.

   ```
   oc delete rs <YOUR-REPLICASET> -n calico-monitoring
   ```

1. Reconfigure the Elasticsearch data storage. The following will save the current configuration to
   `data-tigera-elasticsearch-storage.yaml`.

   ```
   oc get statefulset es-data-tigera-elasticsearch-elasticsearch-storage -n calico-monitoring -o yaml --export > data-tigera-elasticsearch-storage.yaml
   ```

   Run the following command which will fix the configuration for pods to start properly in OpenShift.

   ```
   sed -i '/capabilities/,+2 d' data-tigera-elasticsearch-storage.yaml
   ```

   Replace the running StatefulSet.

   ```
   oc replace -n calico-monitoring -f data-tigera-elasticsearch-storage.yaml
   ```

1. Reconfigure the Elasticsearch master storage. The following will save the current configuration to
   `master-tigera-elasticsearch-storage.yaml`.

   ```
   oc get statefulset es-master-tigera-elasticsearch-elasticsearch-storage -n calico-monitoring -o yaml --export > master-tigera-elasticsearch-storage.yaml
   ```

   Run the following command which will fix the configuration for pods to start properly in OpenShift.

   ```
   sed -i '/capabilities/,+2 d' master-tigera-elasticsearch-storage.yaml
   ```

   Replace the running StatefulSet.

   ```
   oc replace -n calico-monitoring -f master-tigera-elasticsearch-storage.yaml
   ```

1. Reconfigure the Elasticsearch sysctl operator. The following will save the current configuration to
   `elasticsearch-operator-sysctl.yaml`.

   ```
   oc get ds elasticsearch-operator-sysctl -n default -o yaml --export > elasticsearch-operator-sysctl.yaml
   ```

   Run the following command which will fix the configuration for pods to start properly in OpenShift.

   ```
   sed -i '/hostPID/d' elasticsearch-operator-sysctl.yaml
   ```

   Replace the running DaemonSet

   ```
   oc replace -n default -f elasticsearch-operator-sysctl.yaml
   ```

{% endif %}
{% endif %}

{% if include.elasticsearch == "operator" %}
1.  Download the `kibana-dashboards.yaml` manifest.

    ```bash
    curl --compressed -O \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/kibana-dashboards.yaml
    ```

1. Use the following commands
   to set an environment variable called `REGISTRY` containing the location of the private registry and replace
   the paths in the manifest to refer to the private registry.

   ```bash
   REGISTRY=my-registry.com \
   sed -i -e "s?giantswarm?$REGISTRY/giantswarm?g" kibana-dashboards.yaml
   ```

1. Create a `ConfigMap` and start a job that will install the {{site.prodname}} Kibana indexes, views, and dashboards
   when the Kibana service endpoints come up.

   ```bash
   {{cli}} apply -f kibana-dashboards.yaml
   ```

{% else %}

1. Download the following JSON file containing the {[site.prodname}} Kibana indexes, views, and dashboards
   to your computer.

   ```bash
   curl --compressed -O \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/kibana-dashboard.json
   ```

1. Open the **Management** -> **Saved Objects** pane in Kibana and click **Import**. To complete the import, follow the
   [Kibana documentation](https://www.elastic.co/guide/en/kibana/current/managing-saved-objects.html#_import_objects).

{% endif %}

1. Open the **Management** -> **Index Patterns** pane in Kibana, select one of the imported index patterns and click the star to set it as the
   default pattern. Refer to the [Kibana documentation](https://www.elastic.co/guide/en/kibana/current/index-patterns.html#set-default-pattern)
   for more details.

1. If you wish to enforce application layer policies and secure workload-to-workload
   communications with mutually-authenticated TLS, continue to [Enabling application layer policy]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/app-layer-policy) (optional).
