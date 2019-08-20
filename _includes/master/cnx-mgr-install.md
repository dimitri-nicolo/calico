{% if include.init == "openshift" %}
  {% assign cli = "oc" %}
  {% assign manifestPath = "getting-started/openshift" %}
{% elsif include.platform == "eks" %}
  {% assign cli = "kubectl" %}
  {% assign cloudServiceInitials = "EKS" %}
  {% assign manifestPath = "getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-ecs" %}
{% elsif include.platform == "gke" %}
  {% assign cli = "kubectl" %}
  {% assign cloudServiceInitials = "GKE" %}
  {% assign manifestPath = "manifests/gke" %}
{% elsif include.platform == "aks" %}
  {% assign cli = "kubectl" %}
  {% assign cloudServiceInitials = "AKS" %}
  {% assign manifestPath = "manifests/aks" %}
{% else %}
  {% assign cli = "kubectl" %}
  {% assign manifestPath = "getting-started/kubernetes/installation/hosted/cnx/1.7" %}
{% endif %}

## Installing the {{site.prodname}} Manager

1. Download the {{site.prodname}} manifest.

   ```bash
   curl --compressed -O \
   {{site.url}}/{{page.version}}/{{manifestPath}}/cnx.yaml
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="cnx" %}

{% if include.init == "openshift" %}

1. Update the `OAuth` authority API endpoint with your OpenShift master address. This address should be accessible from your browser.

   Example: If OpenShift master were at `https://master.openshift.example.com:8443`, then the following command could be used to set up the parameter.

   ```bash
   sed -i -e 's?tigera.cnx-manager.oauth-authority:.*$?tigera.cnx-manager.oauth-authority: "https://master.openshift.example.com:8443/oauth/authorize"?g' cnx.yaml
   ```

{% elsif include.platform != "eks" and include.platform != "gke" and include.platform != "aks" %}

1. Refer to the bullet that corresponds to your chosen authentication method.

   - **Basic authentication**: Not recommended for a production system. If you want to use this method,
     you do not need to modify the manifest as it is the default selection. However, after completing
     the installation, complete the steps in [Basic authentication]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication#basic-authentication). Also refer to Kubernetes' [Static Password File](https://kubernetes.io/docs/admin/authentication/#static-password-file) discussion.

   - **OIDC**: Open the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
     by setting the value of `tigera.cnx-manager.authentication-type` to `OIDC`.
     Add the other necessary values in the manifest as per the comments. Refer to
     [OpenID Connect Tokens](https://kubernetes.io/docs/admin/authentication/#openid-connect-tokens){:target="_blank"}
     for more information. If you are using a Google identity provider, refer to
     [Google login]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication#google-login).

   - **OAuth**: Open the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
     by setting the value of `tigera.cnx-manager.authentication-type` to `OAuth`.
     Add the other necessary values in the manifest as per the comments.

   - **Token**: Open the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
     by setting the value of `tigera.cnx-manager.authentication-type` to `Token`.
     Refer to [Bearer tokens]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication#bearer-tokens)
     for more information. Also refer to Kubernetes' [Putting a bearer token in a request](https://kubernetes.io/docs/admin/authentication/#putting-a-bearer-token-in-a-request){:target="_blank"}
     for further details.<br>

{% endif %}

{% if include.init == "openshift" %}

1. Create a secret containing a TLS certificate and the private key used to
   sign it. The following commands re-use the certificate created for the cluster.

   ```bash
   oc create secret generic cnx-manager-tls \
   --from-file=cert=/etc/origin/master/master.server.crt \
   --from-file=key=/etc/origin/master/master.server.key -n calico-monitoring
   ```

{% elsif include.platform == "eks" or include.platform == "gke" or include.platform == "aks" %}

1. Create a secret containing a TLS certificate and the private key used to
   sign it. The following commands create a self-signed certificate and key.

   ```bash
   openssl req -out cnxmanager.csr -newkey rsa:2048 -nodes -keyout cnxmanager.key -subj "/CN=cnxmanager.cluster.local"
   cat <<EOF | kubectl create -f -
   apiVersion: certificates.k8s.io/v1beta1
   kind: CertificateSigningRequest
   metadata:
     name: cnxmanager.calico-monitoring
   spec:
     groups:
     - system:authenticated
     request: $(cat cnxmanager.csr | base64 | tr -d '\n')
     usages:
     - digital signature
     - key encipherment
     - server auth
   EOF
   kubectl certificate approve cnxmanager.calico-monitoring
   kubectl get csr cnxmanager.calico-monitoring -o jsonpath='{.status.certificate}' | base64 --decode > cnxmanager.crt
   kubectl create secret generic cnx-manager-tls --from-file=cert=./cnxmanager.crt --from-file=key=./cnxmanager.key -n calico-monitoring
   ```

{% else %}

1. Generate TLS certificates for the Tigera Secure EE Manager to use. The following example creates a self-signed certificate using OpenSSL, but you may generate them using any X.509-compatible tool or obtain them from your organization’s Certificate Authority.

   ```
   openssl req -x509 -newkey rsa:4096 \
                     -keyout manager.key \
                     -nodes \
                     -out manager.crt \
                     -subj "/CN=cnx-manager.calico-monitoring.svc" \
                     -days 3650
   ```

   Run the following command on the master node.

   ```bash
   kubectl create secret generic cnx-manager-tls \
   --from-file=cert=./manager.cert \
   --from-file=key=./manager.key -n calico-monitoring
   ```

{% endif %}

     > **Note**: Web browsers will warn end users about self-signed certificates.
     > To stop the warnings by using valid certificates
     > instead, refer to [{{site.prodname}} Manager connections]({{site.baseurl}}/{{page.version}}/security/comms/crypto-auth#{{site.prodnamedash}}-manager-connections).
     {: .alert .alert-info}

{% if include.platform == "aks" %}
1. Expose {{site.prodname}} Manager via a LoadBalancer. Open the cnx.yaml file and
   update the `Service` section named `cnx-manager` and replace `type: NodePort` with `type: LoadBalancer`.
{% endif %}

{% if include.platform != "docker-ee" %}
1. Edit the Kibana URL to point to your Kibana. Open the cnx.yaml file and
   modify the `ConfigMap` named `tigera-cnx-manager-config` by setting the
   value of `tigera.cnx-manager.kibana-url`
{% endif %}
{% if include.elasticsearch != "external" and include.platform != "aks" %}
   By default a NodePort is installed that serves Kibana on port 30601, so use
   the address of a node (for example a master).
{% endif %}

{% if include.upgrade %}
1. Uninstall {{site.prodname}} Manager from previous install.

   ```bash
   {{cli}} delete -n kube-system service cnx-manager
   {{cli}} delete -n kube-system networkpolicy.projectcalico.org allow-cnx.cnx-manager-access
   {{cli}} delete -n kube-system deployment cnx-manager
   {{cli}} delete -n kube-system configmap tigera-cnx-manager-config
   {{cli}} delete -n kube-system serviceaccount cnx-manager
   ```

1. Prior to {{site.prodname}} v2.4.0, {{site.prodname}} Manager users were granted access
   to Elasticsearch by a ClusterRole RBAC rule that allowed access to the appropriate
   Kubernetes services proxy resource:

   ```
   kind: ClusterRole
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     name: tigera-elasticsearch-access
   rules:
   - apiGroups: [""]
     resources: ["services/proxy"]
     resourceNames: ["https:elasticsearch-tigera-elasticsearch:8443"]
     verbs: ["get"]
   ```

   To ensure that {{site.prodname}} Manager users have the same level of access to
   Elasticsearch indices, ClusterRoles that grant access to Elasticsearch in the
   manner show above should be modified to give access to all resource names under
   of type `index` in the `lma.tigera.io` group.

   ```
   kubectl apply -f - <<EOF
   kind: ClusterRole
   apiVersion: rbac.authorization.k8s.io/v1beta1
   metadata:
     name: tigera-elasticsearch-access
   rules:
   - apiGroups: ["lma.tigera.io"]
     resources: ["index"]
     resourceNames: []
     verbs: ["get"]
   EOF
   ```

   If the ClusterRole `tigera-ui-user` or `network-admin` were not modified, then no
   additional step is required for Elasticsearch access.
{% endif %}

1. Apply the manifest to install the {{site.prodname}} Manager.

   ```bash
   {{cli}} apply -f cnx.yaml
   ```

1. Confirm that all of the pods are running with the following command.

   ```bash
   watch {{cli}} get pods --all-namespaces
   ```

   Wait until each pod has the `STATUS` of `Running`.

{% if include.platform == "eks" or include.platform == "gke" or include.platform == "aks" %}

1. To log into {{site.prodname}} Manager running in {{cloudServiceInitials}}, you'll need a token for a user
   with appropriate permissions on the cluster.

   The easiest way to create such a token is to create a service account, assign it permissions
   and get a token for it to use for login.  Update `USER` to change the name to give
   to the service account and update `NAMESPACE` to change the namespace where the
   service account is created. Create the namespace if needed.

   ```bash
   export USER=ui-user
   export NAMESPACE=ui-namespace
   {{cli}} create serviceaccount -n $NAMESPACE $USER
   kubectl get secret -n $NAMESPACE -o jsonpath='{.data.token}' $(kubectl -n $NAMESPACE get secret | grep $USER | awk '{print $1}') | base64 --decode
   ```

   Save the token - you'll use it to log in to {{site.prodname}} Manager.  Next we'll assign permissions to do so
   to it.  Use the value of `$USER` as `<USER>` and `$NAMESPACE` as `<NAMESPACE>` in the following step.

{% include {{page.version}}/cnx-grant-user-manager-permissions.md usertype="serviceaccount" %}
{% else %}
{% include {{page.version}}/cnx-grant-user-manager-permissions.md %}
{% endif %}

{% if include.platform == "aks" %}
1. Get `EXTERNAL-IP` of the LoadBalancer via which {{site.prodname}} Manager is exposed.
   ```bash
   kubectl get svc cnx-manager -n calico-monitoring
   ```
   
   Sign in by navigating to `https://<{{site.prodname}} Manager LoadBalancer EXTERNAL-IP>:9443` and login.
{% else %}
{% if include.platform != "docker-ee" %}
1. By default, {{site.prodname}} Manager is made accessible via a NodePort listening on port 30003.
   You can edit the `cnx.yaml` manifest if you want to change how {{site.prodname}} Manager is
   exposed.

{% if include.orch == "openshift" %}
   You may need to create an OpenShift route or Ingress if the NodePort is not accessible.
   Ensure that the Route is created with tls termination set to passthrough. Also, ensure that the host
   specified in the route is resolvable from within the cluster, and to update oauth-client.yaml with the
   hostname as specified in the route.

   Sign in by navigating to `https://<{{site.prodname}} Manager hostname specified in openshift route or ingress>` and login.
{% else %}
   You may need to create an ssh tunnel if the node is not accessible - for example:

   ```bash
   ssh <jumpbox> -L 127.0.0.1:30003:<kubernetes node>:30003
   ```

   Sign in by navigating to `https://<address of a Kubernetes node or 127.0.0.1 for ssh tunnel>:30003` and login.
{% endif %}
{% endif %}
{% endif %}

{% if include.platform == "eks" or include.platform == "gke" or include.platform == "aks" %}
   Log in to {{site.prodname}} Manager using the token you created earlier in the process.
{% endif %}
