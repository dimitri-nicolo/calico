{% if include.init == "systemd" %}

### Installing Tigera CNX

1. If you are using the etcd datastore, open the 
   [cnx-configmap.yaml file](hosted/cnx/1.7/cnx-configmap.yaml),
   copy its contents, save it in a new file, and replace `<ETCD_ENDPOINTS>`
   with the IP address of your etcd datastore. Then apply the manifest.
   
   ```bash
   kubectl apply -f cnx-configmap.yaml
   ```

1. Open the manifest that corresponds to your datastore type.
   - [etcd](hosted/cnx/1.7/cnx-etcd.yaml){:target="_blank"}
   - [Kubernetes API datastore](hosted/cnx/1.7/cnx-kdd.yaml){:target="_blank"}

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.

{% endif %}

{% include {{page.version}}/cnx-cred-sed.md yaml="cnx" %}

{% if include.orchestrator == "openshift" %}

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

1. If you want the CNX Manager to listen on a port other than
   30003 or you plan to set up a load balancer in front of it, edit the 
   `Service` object named `cnx-manager` as needed.  

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   {{site.prodname}} Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the {{site.prodname}} Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` service.

1. Store those credentials as `cert` and `key` in a secret named
   `cnx-manager-tls`.  For example:

   ```
   kubectl create secret generic cnx-manager-tls --from-file=cert=/path/to/certificate --from-file=key=/path/to/key -n kube-system
   ```

1. Optionally [enable]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/cnx#enabling-tls-verification-for-a-kubernetes-extension-api-server) TLS verification for CNX API Server.

1. Apply the manifest to install {{site.prodname}} Manager and the {{site.prodname}} API server.

   ```
   kubectl apply -f cnx.yaml
   ```

{% if include.orchestrator == "openshift" %}

1. Allow cnx-manager to run as root:

       oadm policy add-scc-to-user anyuid system:serviceaccount:kube-system:cnx-manager

{% endif %}

1. Configure authorization to allow {{site.prodname}} Manager users to edit policies.  Consult the
   [{{site.prodname}} Manager]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies)
   documents for advice on configuring this.  The authentication method you
   chose when setting up the cluster defines what format you need to use for
   usernames in the role bindings.

1. Optionally add network policy to ensure requests to CNX are permitted.  By default Kubernetes doesn't
   install any network policy, and therefore CNX Manager is accessible, but it is easy to
   unintentionally block it when editing policy so this is a recommended step.  Download the
   [cnx-policy.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml)
   file and apply it.  The file doesn't require any customization, but contains some comments
   suggesting ways to make the policy more fine grained if you know where CNX Manager will be
   accessed from or the addresses of your Kubernetes API Servers.

   ```
   kubectl apply -f cnx-policy.yaml
   ```
