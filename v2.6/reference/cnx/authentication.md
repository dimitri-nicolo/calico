---
title: Configuring user authentication to Calico Enterprise Manager
redirect_from: latest/reference/cnx/authentication
canonical_url: https://docs.tigera.io/v2.3/reference/cnx/authentication
---

This document describes the authentication methods supported by {{site.prodname}}
Manager and how to set them up.

After setting up authentication, see [RBAC on tiered policies](rbac-tiered-policies)
for information on how to control what resources each user can access.

{{site.prodname}} doesn't have its own authentication and authorization system, it delegates
to Kubernetes.  Detailed information on how to set up each authentication
method is provided by the [Kubernetes authentication guide](https://kubernetes.io/docs/admin/authentication/).

The {{site.prodname}} Manager web interface allows users to select the authentication method
to use, but Kubernetes must be configured to support the chosen method.
Select **Menu** on any sign-in page to change the login method in the web application.

### Google login

Google login allows users to log in using their Google accounts.  To use this
method, you need to [setup Google OAuth 2.0](https://developers.google.com/identity/protocols/OpenIDConnect),
and [configure the Kubernetes API server to use it](https://kubernetes.io/docs/admin/authentication/#configuring-the-api-server).

To use Google login, the web server component of {{site.prodname}} Manager needs
to have a well known DNS name at which users access the application.  The sample
manifests in this documentation create a `NodePort` for the web server
serving on port 30003, but you may wish to set up connectivity differently.

1. [Setup Google OAuth 2.0](https://developers.google.com/identity/protocols/OpenIDConnect).
   - Ensure the redirect URIs are set to `https://<{{site.prodname}} Manager name>:<port>/login/oidc/callback`.
   - Note down the client ID.

2. [Configure the Kubernetes API server to use it](https://kubernetes.io/docs/admin/authentication/#configuring-the-api-server).
   - Example command:
    ```
    sed -i "/- kube-apiserver/a\    - --oidc-issuer-url=https://accounts.google.com\n    - --oidc-username-claim=email\n    - --oidc-client-id=<client_ID_from_above>" /etc/kubernetes/manifests/kube-apiserver.yaml
    ```

3. Configure {{site.prodname}} Manager to use it
   - Set the client ID in the tigera-cnx-manager-config ConfigMap (referenced
     in the installation instructions) and authentication-type to OIDC.

4. If {{site.prodname}} Manager has already been deployed, restart the web server (achieved by deleting the pod).

   ```
   kubectl delete pod cnx-manager-<hash> -n kube-system
   ```

[Configure Kubernetes RBAC bindings](rbac-tiered-policies) depending on the settings used for `--oidc-username/groups-claim` and `--oidc-username/groups-prefix` on the API server.

Or, as for a quickstart step for the email address you want to login with, bind the email address with the `cluster-admin` role to give full access.
   ```
   kubectl create clusterrolebinding oidc-user-permissive-binding \
   --clusterrole=cluster-admin \
   --user=<email_address>
   ```

### Basic authentication

Basic authentication allows users to configure Kubernetes with a list of username/passwords.
It is intended for testing purposes, and has significant limitations—notably
the Kubernetes API server must be restarted after making any changes.

Consult the [Kubernetes docs](https://kubernetes.io/docs/admin/authentication/#static-password-file)
to configure this login mode.

On a kubeadm system, you can use the following steps to set up basic authentication.

1. Use the following command to create a file `basic_auth.csv` containing
   a set of credentials.

   ```
   sudo sh -c "echo '<password>,<username>,1' > /etc/kubernetes/pki/basic_auth.csv"
   ```

1. Add a reference to the `basic_auth.csv` file in `kube-apiserver.yaml`.

   ```
   sudo sed -i \
   "/- kube-apiserver/a\    - --basic-auth-file=/etc/kubernetes/pki/basic_auth.csv" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

   > **Note**: We created the basic_auth.csv under /etc/kubernetes/pki because that volume is
   mounted by default on the kube-apiserver pod with a kubeadm installation.
   {: .alert .alert-info}

1. Restart kube-apiserver to pick up new settings:

   ```
   sudo systemctl restart kubelet
   ```

1. Select the **Login via username and password** option in the {{site.prodname}} 
   Manager web interface.

1. [Configure Kubernetes RBAC bindings](rbac-tiered-policies) using the username and groups defined in the
   basic authentication CSV file.



### Bearer tokens

The **Login via static token** option tells {{site.prodname}} Manager to pass the token through
to Kubernetes. The [Kubernetes docs](https://kubernetes.io/docs/admin/authentication/#static-token-file)
describe how to set up this method.

Like basic authentication, RBAC bindings use the username and groups defined in the token
file.
