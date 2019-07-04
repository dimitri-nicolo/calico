{% if include.init != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %}
  {% assign cli = "oc" %}
{% endif %}

## Installing the {{site.prodname}} API Server

{% if include.init == "systemd" %}

1. Load the following manifest to Kubernetes to deploy dummy pods that
   will be used for Prometheus targeting. You should ensure that this manifest
   deploys one pod on each host running {{site.prodname}} that you wish to
   monitor, adjusting the annotations and tolerations as needed.

   ```yaml
   apiVersion: extensions/v1beta1
   kind: DaemonSet
   metadata:
     name: node-exporter
     namespace: kube-system
     labels:
       k8s-app: calico-node
   spec:
     template:
       metadata:
         name: node-exporter
         labels:
           k8s-app: calico-node
         annotations:
           scheduler.alpha.kubernetes.io/critical-pod: ''
       spec:
         serviceAccountName: default
         containers:
         - image: busybox
           command: ["sleep", "10000000"]
           name: node-exporter
           ports:
           - containerPort: 9081
             hostPort: 9081
             name: scrape
         hostNetwork: true
         hostPID: true
         tolerations:
         - operator: Exists
           effect: NoSchedule
   ```
   > **Note**: Another option for monitoring is to set up and configure your own
   > Prometheus monitoring instead of using the monitoring provided in the next
   > steps, then it would not be necessary to load the above manifest.
   {: .alert .alert-info}


1. If you are using the etcd datastore:

   1. Download the [cnx-configmap.yaml file](hosted/cnx/1.7/cnx-configmap.yaml).

      ```bash
      curl --compressed -o cnx-configmap.yaml \
      {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-configmap.yaml
      ```

   1. Use the following commands to: set an environment variable called `ETCD_ENDPOINTS`
      containing the location of the private registry and replace `<ETCD_ENDPOINTS>` in the manifest
      with the location of your etcd cluster.

      ```bash
      ETCD_ENDPOINTS=10.90.89.100:2379,10.90.89.101:2379 \
      sed -i -e "s?<ETCD_ENDPOINTS>?$ETCD_ENDPOINTS?g" cnx-configmap.yaml
      ```

   1. Apply the manifest.

      ```bash
      kubectl apply -f cnx-configmap.yaml
      ```

{% endif %}

{% if include.init != "openshift" and include.net == "calico" %}

1. Download the manifest that corresponds to your datastore type and save the file
   as cnx-api.yaml. That is how we will refer to it in later steps.

   - **etcd datastore**
     ```bash
     curl --compressed -o cnx-api.yaml \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-api-etcd.yaml
     ```

   - **Kubernetes API datastore**
     ```bash
     curl --compressed -o cnx-api.yaml \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-api-kdd.yaml
     ```

{% elsif include.platform == "eks" %}

1. Download the EKS {{site.prodname}} manifest and save the file
   as cnx-api.yaml. That is how we will refer to it in later steps.

   ```bash
   curl --compressed -o cnx-api.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-ecs/cnx-api-kdd-eks.yaml
   ```

{% elsif include.platform == "gke" %}

1. Download the GKE {{site.prodname}} manifest and save the file
   as cnx-api.yaml. That is how we will refer to it in later steps.

   ```bash
   curl --compressed -o cnx-api.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-gke/cnx-api-kdd-gke.yaml
   ```

{% elsif include.init != "openshift" and include.net == "other" %}

1. Download the networking manifest for the Kubernetes API datastore and save the file
   as cnx-api.yaml. That is how we will refer to it in later steps.

   ```bash
   curl --compressed -o cnx-api.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-api-kdd.yaml
   ```

{% elsif include.init == "openshift" %}

1. Download the {{site.prodname}} manifest.

   ```bash
   curl --compressed -O {{site.url}}/{{page.version}}/getting-started/openshift/cnx-api.yaml
   ```

{% endif %}

{% if include.upgrade %}
   > **Note**: If you are upgrading from {{site.prodname}} v2.2 or earlier you will need 
   > to [upgrade to version 2.3](/v2.3/getting-started/kubernetes/upgrade/upgrade-tsee) before following
   > these intructions.
   {: .alert .alert-info}
{% endif %}

{% include {{page.version}}/cnx-cred-sed.md yaml="cnx-api" %}

1. Apply the manifest to install the {{site.prodname}} API server.

   ```bash
   {{cli}} apply -f cnx-api.yaml
   ```

1. Confirm that all of the pods are running with the following command.

   ```bash
   watch {{cli}} get pods -n kube-system
   ```

   Wait until each pod has the `STATUS` of `Running`.
