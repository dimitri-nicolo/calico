---
title: Configure your own identity provider
description: Log in to Calico Enterprise Manager and Kibana using your own identity provider.
---

### Big picture

Configure an external identity provider (IdP), such as OIDC or Openshift, create a user with access to the manager UI, and log in to the {{site.prodname}} Manager UI and Kibana.

### Concepts

When configuring your cluster, you may be asked to provide information on the following inputs:

- **Client Id**: Id for exchanging data that are shared between the IdP and an application.
- **Client Secret**: Secret associated with the `client id` used by server applications for exchanging tokens.
- **Issuer URL**: URL where the IdP can be reached, based on the conventions of OAuth and OIDC.
- **Claims**: Every time your IdP issues a token for a valid user, these claims add metadata about the user. {{site.prodname}} uses this to determine the username.

### Before you begin...

Make sure you have installed {{site.prodname}} using one of the [installation guides]({{site.baseurl}}/getting-started/) and have set up [access to the Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).

### How to

> Note: For OpenShift, replace `kubectl` in the commands below with `oc`.
{: .alert .alert-info}

#### Configure your own Identity Provider

The {{site.prodname}} authentication method can be configured through the [Authentication API resource]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Authentication) named `tigera-secure`.

We currently support three identity providers: 
- **OIDC authentication**: The user identity is managed outside of the cluster by an OIDC authorization server.
- **Google OIDC authentication**: The user identity is managed by Google OIDC. Pick this option if you would like to make use of GSuite groups.
- **Openshift authentication**: The user identity is provided by the OpenShift OAuth server.

{% tabs %}
  <label:OIDC,active:true>
<%

1. Apply the Authentication CR to your cluster to let the operator configure your login. This example demonstrates the email claim. 
   This means that from the JWT that your IdP issues, the email field is used as the username to bind privileges to.

   ```
   apiVersion: operator.tigera.io/v1
   kind: Authentication
   metadata:
     name: tigera-secure
   spec:
     managerDomain: https://<domain-of-manager-ui>
     oidc:
       issuerURL: <your-idp-issuer>
       usernameClaim: email
   ```

1. Apply the secret to your cluster with your OIDC credentials. To obtain the values, please consult the documentation of your provider.

   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: tigera-oidc-credentials
     namespace: tigera-operator
   data:
     clientID: <your-base64-clientid>
     clientSecret: <your-base64-clientid-secret>
   ```

%>
  '<label:Google OIDC>'

<%
1. Apply the Authentication CR to your cluster to let the operator configure your login. This example demonstrates the email claim. 
   This means that from the JWT that your IdP creates, the email field is used as the username to bind privileges to. We recommend 
   to match the `issuerURL` and `usernameClaim` with the configuration of your kube-apiserver.

   ```
   apiVersion: operator.tigera.io/v1
   kind: Authentication
   metadata:
     name: tigera-secure
   spec:
     managerDomain: https://<domain-of-manager-ui>
     oidc:
       issuerURL: <your-idp-issuer>
       usernameClaim: email
   ```
1. (Optional) Google OIDC does not support the groups claim. However, {{site.prodname}} leverages [Dex IdP](https://dexidp.io/docs/connectors/google/) to add groups if you configure a service account.
   This account needs Domain-Wide Delegation and permission to access the `https://www.googleapis.com/auth/admin.directory.group.readonly` API scope.
   To get group fetching set up:
   - Follow the [instructions](https://developers.google.com/admin-sdk/directory/v1/guides/delegation) to set up a service account with Domain-Wide Delegation
       - During service account creation, a JSON key file will be created that contains authentication information for the service account.
       - When delegating the API scopes to the service account, delegate the `https://www.googleapis.com/auth/admin.directory.group.readonly` scope and **only this scope**.
   - Enable the [Admin SDK](https://console.developers.google.com/apis/library/admin.googleapis.com/) 
   - Use the `serviceAccountSecret` and `adminEmail` configuration options in the next step.
        - The contents of the JSON key file should be used as the `serviceAccountSecret`
        - For `adminEmail` choose a G Suite super user. The service account will impersonate this user when making calls to the admin API.

1. Apply the secret to your cluster with your OIDC credentials. 

   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: tigera-oidc-credentials
     namespace: tigera-operator
   data:
     clientID: <your-base64-clientid>
     clientSecret: <your-base64-clientid-secret>
     # If you created a service account in the previous step, include the following two fields.
     serviceAccountSecret: <your-base64-json-contents>
     adminEmail: <your-base64-gsuite-user>
   ```

%>
  '<label:Openshift>'

<%
1. Create values for some required variables. `MANAGER_URL` is the URL where {{site.prodname}} Manager will be accessed,
   `CLUSTER_DOMAIN` is the domain (excl. port) where your Openshift cluster is accessed and `CLIENT_SECRET` is a value of your choosing.
   ```bash
   MANAGER_URL=<manager-host>:<port>
   CLUSTER_DOMAIN=<domain-of-your-ocp-cluster>
   CLIENT_SECRET=<clientSecret>
   ```

1. Add an OAuthClient to your Openshift cluster.

   ```bash
   kubectl apply -f - <<EOF
   kind: OAuthClient
   apiVersion: oauth.openshift.io/v1
   metadata:
     # The name is used as the clientID by Dex.
     name: tigera-dex
   # The secret is used as the clientSecret by Dex
   secret: $CLIENT_SECRET
   # List of valid addresses for the callback. 
   redirectURIs:
    - "$MANAGER_URL/dex/callback" 
   grantMethod: prompt
   EOF
   ```

1. Apply the Authentication CR to your cluster to let the operator configure your login.

   ```bash
   kubectl apply -f - <<EOF
   apiVersion: operator.tigera.io/v1
   kind: Authentication
   metadata:
     name: tigera-secure
   spec:
     managerDomain: $MANAGER_URL
     openshift:
       issuerURL: https://api.$CLUSTER_DOMAIN:6443
   EOF
   ```

1. Obtain the certificates that are used to connect with openshift and store them in a file called `dex.pem`.
   ```bash
   echo | openssl s_client -servername oauth-openshift.apps.$CLUSTER_DOMAIN -connect oauth-openshift.apps.$CLUSTER_DOMAIN:443 |  sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > dex.pem
   echo | openssl s_client -servername api.$CLUSTER_DOMAIN -connect api.$CLUSTER_DOMAIN:6443 |  sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' >> dex.pem
   ```
   Alternatively, you can use the root CA of your cluster and store it in `dex.pem`.

1. Apply a secret to your cluster with your Openshift credentials.

   ```bash
   kubectl create secret generic tigera-openshift-credentials -n tigera-operator --from-file=rootCA=dex.pem --from-literal=clientID=tigera-dex --from-literal=clientSecret=$CLIENT_SECRET
   ```
   
%>
{% endtabs %}

**Grant user login privileges** 

For admin users apply this cluster role.
  ```bash
  kubectl create clusterrolebinding <user>-tigera-network-admin --user=<user> --clusterrole=tigera-network-admin
  ```

For regular users with view-only permissions apply this role.
  ```bash
  kubectl create clusterrolebinding <user>-tigera-ui-user --user=<user> --clusterrole=tigera-ui-user
  ```
> Note: Openshift users can also apply these cluster roles from the Openshift console. Navigate to "User Management" and then select "users".
{: .alert .alert-info}

#### Allow {{site.prodname}} URIs in your IdP
Most IdPs require redirect URIs to be allowed in order to redirect users at the end of the OAuth flow to the {{site.prodname}} Manager or to Kibana. 
Please consult your IdPs documentation for authorizing your domain for the respective origins and destinations.

**Authorized redirect URIs**
- `https://<host><port>/dex/callback`

### Above and beyond

- [Learn more about the default authentication options]({{site.baseurl}}/getting-started/cnx/authentication-quickstart)
- [Configure RBAC for tiered policies]({{site.baseurl}}/security/rbac-tiered-policies)
- [Configure RBAC for Elasticsearch]({{site.baseurl}}/security/logs/rbac-elasticsearch)
