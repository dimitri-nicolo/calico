{% if include.init == "systemd" or include.init == "kubernetes" %}

## <a name="install-cnx-mgr"></a>Installing the {{site.prodname}} Manager and API Server

{% endif %}

{% if include.init == "systemd" %}

1. Load the following manifest to Kubernetes to deploy dummy pods that 
   will be used for Prometheus targeting. You should ensure that this manifest 
   deploys one pod on each host running {{site.prodname}} that you wish to 
   monitor, adjusting the annotations and tolerations as needed.

   ```
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
   
   1. Replace `<ETCD_ENDPOINTS>` with the IP address of your etcd server.
     
      **Command**
      ```bash
      sed -i -e "s?<ETCD_ENDPOINTS>?<REPLACE_ME>?g" cnx-configmap.yaml
      ```

      **Example**
      ```bash
      sed -i -e "s?<ETCD_ENDPOINTS>?http://10.96.232.136:6666?g" cnx-configmap.yaml
      ```
         
      > **Tip**: You can specify more than one etcd server, using commas as delimiters.
      {: .alert .alert-success}
     
   1. Apply the manifest.
     
      ```bash
      kubectl apply -f cnx-configmap.yaml
      ```
   
{% endif %}

1. Download the manifest that corresponds to your datastore type and save the file 
   as cnx.yaml. That is how we will refer to it in later steps. 
   
   **etcd datastore**
   ```bash
   curl --compressed -o cnx.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml
   ```
   
   **Kubernetes API datastore**
   ```bash
   curl --compressed -o cnx.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-kdd.yaml
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="cnx" %}

{% if include.init == "openshift" %}

1. Update the login method to `OAuth`:

       sed -i -e 's/tigera.cnx-manager.authentication-type:.*$/tigera.cnx-manager.authentication-type: "OAuth"/g' cnx.yaml

1. Update the `OAuth` authority API endpoint with OpenShift master address.

   Example: If OpenShift master were at 127.0.0.1:8443, then the following command can be used to set up the parameter.

       sed -i -e 's/tigera.cnx-manager.oauth-authority:.*$/tigera.cnx-manager.oauth-authority: "https:\/\/127.0.0.1:8443\/oauth\/authorize"/g' cnx.yaml

1. Open the file in a text editor, and make the following modifications:

   In the `Deployment` named `cnx-apiserver`:

   - Uncomment the `ETCD_CA_CERT_FILE`, `ETCD_KEY_FILE`, and `ETCD_CERT_FILE` environment variables.
   - Uncomment the `volumeMount` named `etcd-certs`.
   - Uncomment the `volume` named `etcd-certs`. 

{% else %}

1. By default, {{site.prodname}} uses basic authentication. To use OpenID 
   Connect or OAuth, open cnx.yaml in your favorite text editor and
   modify the `ConfigMap` of `tigera-cnx-manager-config` as needed.
   
{% endif %}

1. If you want the {{site.prodname}} Manager to listen on a port other than
   30003 or you plan to set up a load balancer in front of it, edit the 
   `Service` object named `cnx-manager` as needed.  

1. Create a secret containing a TLS certificate and the private key used to
   sign it. The following command uses a self-signed certificate and key that
   should be present as part of your Kubernetes deployment to get you started. 

   ```bash
   kubectl create secret generic cnx-manager-tls \
   --from-file=cert=/etc/kubernetes/pki/apiserver.crt \
   --from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
   ```
   
   > **Note**: Web browsers will warn end users about the self-signed certificate 
   > used in the above command. To stop the warnings by using valid certificates 
   > instead, refer to [Securing {{site.prodname}} Manager with TLS]({{site.url}}/{{page.version}}/reference/cnx/securing-with-tls).
   > Refer to [Enabling TLS verification]({{site.url}}/{{page.version}}/reference/cnx/enabling-tls-verification) 
   > to achieve further security.
   {: .alert .alert-info}

1. Apply the manifest to install {{site.prodname}} Manager and the {{site.prodname}} API server.

   ```
   kubectl apply -f cnx.yaml
   ```

{% if include.init == "openshift" %}

1. Allow the {{site.prodname}} Manager to run as root:

       oadm policy add-scc-to-user anyuid system:serviceaccount:kube-system:cnx-manager

{% endif %}

1. Confirm that all of the pods are running with the following command.

   ```
   watch kubectl get pods --all-namespaces
   ```

   Wait until each pod has the `STATUS` of `Running`.

1. Apply the following manifest to set network policy that permits requests to {{site.prodname}}. 

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml
   ```
   
   > **Note**: You can also 
   > [view the manifest in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml){:target="_blank"}.
   {: .alert .alert-info}
