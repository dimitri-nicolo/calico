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

{% if include.init != "openshift" %}

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

{% else %}

1. Download the {{site.prodname}} manifest for etcd and save the file as cnx.yaml. That is how we will refer to it in later steps:

   ```bash
   curl --compressed -o cnx.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml
   ```

{% endif %}

{% include {{page.version}}/cnx-cred-sed.md yaml="cnx" %}

{% if include.init == "openshift" %}

1. Update the login method to `OAuth`:

       sed -i -e 's/tigera.cnx-manager.authentication-type:.*$/tigera.cnx-manager.authentication-type: "OAuth"/g' cnx.yaml

1. Update the `OAuth` authority API endpoint with your OpenShift master address. This address should be accessible from your browser.

   Example: If OpenShift master were at `https://master.openshift.example.com:8443`, then the following command could be used to set up the parameter.

       sed -i -e 's?tigera.cnx-manager.oauth-authority:.*$?tigera.cnx-manager.oauth-authority: "https://master.openshift.example.com:8443/oauth/authorize"?g' cnx.yaml

1. Open the file in a text editor, and make the following modifications:

   In the `Deployment` named `cnx-apiserver`:

   - Uncomment the `ETCD_CA_CERT_FILE`, `ETCD_KEY_FILE`, and `ETCD_CERT_FILE` environment variables in both the cnx-apiserver and cnx-queryserver containers.
   - Uncomment the `volumeMount` named `etcd-certs`.
   - Uncomment the `volume` named `etcd-certs`.

{% else %}

1. Refer to the bullet that corresponds to your chosen authentication method.

   - **Basic authentication**: Not recommended for a production system. If you want to use this method,
     you do not need to modify the manifest as it is the default selection. However, after completing
     the installation, complete the steps in [Basic authentication]({{site.url}}/{{page.version}}/reference/cnx/authentication#basic-authentication). Also refer to Kubernetes' [Static Password File](https://kubernetes.io/docs/admin/authentication/#static-password-file) discussion.

   - **OIDC**: Open the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
     by setting the value of `tigera.cnx-manager.authentication-type` to `OIDC`.
     Add the other necessary values in the manifest as per the comments. Refer to
     [OpenID Connect Tokens](https://kubernetes.io/docs/admin/authentication/#openid-connect-tokens){:target="_blank"}
     for more information. If you are using a Google identity provider, refer to
     [Google login]({{site.url}}/{{page.version}}/reference/cnx/authentication#google-login).

   - **OAuth**: Open the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
     by setting the value of `tigera.cnx-manager.authentication-type` to `OAuth`.
     Add the other necessary values in the manifest as per the comments.

   - **Token**: Open the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
     by setting the value of `tigera.cnx-manager.authentication-type` to `Token`.
     Refer to [Bearer tokens]({{site.url}}/{{page.version}}/reference/cnx/authentication#bearer-tokens)
     for more information. Also refer to Kubernetes' [Putting a bearer token in a request](https://kubernetes.io/docs/admin/authentication/#putting-a-bearer-token-in-a-request){:target="_blank"}
     for further details.<br>

{% endif %}

1. If you want the {{site.prodname}} Manager to listen on a port other than
   30003 or you plan to set up a load balancer in front of it, edit the
   `Service` object named `cnx-manager` as needed.  

1. Create a secret containing a TLS certificate and the private key used to
   sign it. The following command uses a self-signed certificate and key that
   should be present as part of your Kubernetes deployment to get you started.

{% if include.init == "openshift" %}

   ```bash
   oc create secret generic cnx-manager-tls \
   --from-file=cert=/etc/origin/master/master.server.crt \
   --from-file=key=/etc/origin/master/master.server.key -n kube-system
   ```

{% else %}

   ```bash
   kubectl create secret generic cnx-manager-tls \
   --from-file=cert=/etc/kubernetes/pki/apiserver.crt \
   --from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
   ```

{% endif %}

   > **Note**: Web browsers will warn end users about the self-signed certificate
   > used in the above command. To stop the warnings by using valid certificates
   > instead, refer to [{{site.prodname}} Manager connections]({{site.url}}/{{page.version}}/usage/encrypt-comms#cnx-manager-connections).
   {: .alert .alert-info}

1. Apply the manifest to install {{site.prodname}} Manager and the {{site.prodname}} API server.

{% if include.init == "openshift" %}

   ```
   oc apply -f cnx.yaml
   ```

{% else %}

   ```
   kubectl apply -f cnx.yaml
   ```

{% endif %}

{% if include.init == "openshift" %}

1. Allow the {{site.prodname}} Manager to run as root:

       oc adm policy add-scc-to-user anyuid system:serviceaccount:kube-system:cnx-manager

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
